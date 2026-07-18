#!/usr/bin/env python3
"""WS-A: normalize BEAM raw HF rows to SCHEMA.md.

Mohammadta/BEAM (current HF shape):
  chat: list[session], session = list[{role, content, time_anchor, ...}]
  probing_questions: Python-literal str of {category: [qdicts]}

Also accepts flatter shapes (messages / conversation / dialogue) and
chunks long flat dialogues into ~20-turn sessions for sh-gen.
"""

from __future__ import annotations

import argparse
import ast
import json
import re
from datetime import datetime, timedelta, timezone
from pathlib import Path


def turns_from_messages(msgs: list) -> list:
    out = []
    for m in msgs:
        if isinstance(m, dict):
            role = (m.get("role") or m.get("speaker") or "user").lower()
            text = m.get("content") or m.get("text") or ""
        elif isinstance(m, str):
            role, text = "user", m
        else:
            continue
        if role in ("human", "user", "speaker1", "speaker_1"):
            role = "user"
        elif role in ("assistant", "gpt", "bot", "speaker2", "speaker_2"):
            role = "assistant"
        else:
            role = "user" if not out or out[-1]["role"] == "assistant" else "assistant"
        if text:
            # Strip BEAM turn-index suffixes like " ->-> 1,1"
            text = re.sub(r"\s*->->\s*[\d,]+\s*$", "", str(text)).strip()
            out.append({"role": role, "text": text})
    return out


def chunk_sessions(turns: list, chunk: int = 20) -> list:
    base = datetime(2024, 1, 1, tzinfo=timezone.utc)
    sessions = []
    for i in range(0, len(turns), chunk):
        part = turns[i : i + chunk]
        if not part:
            continue
        sessions.append(
            {
                "session_id": f"s{len(sessions) + 1}",
                "date": (base + timedelta(days=len(sessions))).strftime("%Y-%m-%dT%H:%M:%SZ"),
                "turns": part,
            }
        )
    return sessions


def to_iso_date(raw: str, fallback: datetime) -> str:
    """Best-effort parse of BEAM time anchors like 'March-15-2024' → ISO Zulu."""
    if isinstance(raw, str) and raw.strip():
        s = raw.strip().replace("_", "-")
        for fmt in ("%B-%d-%Y", "%b-%d-%Y", "%Y-%m-%d", "%m-%d-%Y", "%B %d, %Y", "%B %d %Y"):
            try:
                dt = datetime.strptime(s, fmt).replace(tzinfo=timezone.utc)
                return dt.strftime("%Y-%m-%dT%H:%M:%SZ")
            except ValueError:
                continue
    return fallback.strftime("%Y-%m-%dT%H:%M:%SZ")


def sessions_from_chat(chat) -> list:
    """Mohammadta/BEAM: chat is list[session], session is list[message dict]."""
    if not isinstance(chat, list) or not chat:
        return []
    # Nested sessions: [[{role,content}, ...], ...]
    if isinstance(chat[0], list):
        base = datetime(2024, 1, 1, tzinfo=timezone.utc)
        sessions = []
        for i, sess in enumerate(chat):
            turns = turns_from_messages(sess if isinstance(sess, list) else [])
            if not turns:
                continue
            fallback = base + timedelta(days=i)
            ta = None
            if isinstance(sess, list) and sess and isinstance(sess[0], dict):
                ta = sess[0].get("time_anchor")
            date = to_iso_date(ta, fallback)
            sessions.append({"session_id": f"s{len(sessions) + 1}", "date": date, "turns": turns})
        return sessions
    # Flat message list
    if isinstance(chat[0], dict):
        return chunk_sessions(turns_from_messages(chat))
    return []


def parse_probing(raw) -> list:
    """BEAM probing_questions: often a Python-literal string of {category: [qdicts]}."""
    if raw is None:
        return []
    if isinstance(raw, str):
        try:
            raw = json.loads(raw)
        except json.JSONDecodeError:
            try:
                raw = ast.literal_eval(raw)
            except (SyntaxError, ValueError):
                return []
    out = []
    if isinstance(raw, dict):
        for cat, items in raw.items():
            if not isinstance(items, list):
                continue
            for q in items:
                if isinstance(q, dict):
                    out.append({**q, "category": q.get("category") or cat})
                elif isinstance(q, str):
                    out.append({"question": q, "category": cat})
        return out
    if isinstance(raw, list):
        return raw
    return []


def question_answer(q: dict) -> str:
    for k in ("answer", "ideal_answer", "ideal_response", "ideal_summary", "gold"):
        v = q.get(k)
        if v:
            return str(v)
    return ""


def normalize_row(row: dict, idx: int, tier: str) -> dict | None:
    cid = str(row.get("id") or row.get("conversation_id") or f"beam-{tier}-{idx}")
    msgs = row.get("messages") or row.get("conversation") or row.get("dialogue") or row.get("chat")
    if isinstance(msgs, str):
        try:
            msgs = json.loads(msgs)
        except json.JSONDecodeError:
            msgs = [{"role": "user", "text": msgs}]
    if not msgs and "sessions" in row:
        sessions = row["sessions"]
    elif msgs is not None and row.get("chat") is msgs:
        # Prefer BEAM nested-session shape
        sessions = sessions_from_chat(msgs)
        if not sessions:
            return None
    elif msgs:
        # Flat messages (or mis-detected nested → sessions_from_chat handles both)
        if isinstance(msgs, list) and msgs and isinstance(msgs[0], list):
            sessions = sessions_from_chat(msgs)
        else:
            turns = turns_from_messages(msgs if isinstance(msgs, list) else [])
            sessions = chunk_sessions(turns)
    else:
        return None

    q_src = (
        row.get("questions")
        or row.get("probes")
        or row.get("qa")
        or parse_probing(row.get("probing_questions"))
    )
    if isinstance(q_src, str):
        q_src = parse_probing(q_src)

    questions = []
    for qi, q in enumerate(q_src or []):
        if isinstance(q, str):
            questions.append(
                {
                    "question_id": f"q{qi + 1}",
                    "category": "beam",
                    "question": q,
                    "answer": "",
                    "evidence_session_ids": [],
                }
            )
        elif isinstance(q, dict):
            questions.append(
                {
                    "question_id": str(q.get("id") or f"q{qi + 1}"),
                    "category": str(q.get("category") or q.get("ability") or q.get("question_type") or "beam"),
                    "question": q.get("question") or q.get("query") or "",
                    "answer": question_answer(q),
                    "evidence_session_ids": q.get("evidence_session_ids") or [],
                }
            )

    if not sessions:
        return None
    return {
        "conversation_id": f"beam-{tier}-{cid}",
        "sessions": sessions,
        "questions": questions,
        "extra": {"benchmark": "beam", "tier": tier},
    }


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--tier", default="128k")
    ap.add_argument("--data-dir", type=Path, default=None)
    ap.add_argument("--chunk-turns", type=int, default=20)
    args = ap.parse_args()
    here = Path(__file__).resolve().parent
    data = args.data_dir or here / "data"
    raw = data / f"beam-{args.tier}-raw" / "conversations.jsonl"
    out = data / f"beam-{args.tier}-conversations.jsonl"
    if not raw.exists():
        raise SystemExit(f"missing {raw} — run get_beam.sh {args.tier}")

    n = 0
    with open(raw) as f, open(out, "w") as o:
        for i, line in enumerate(f):
            if not line.strip():
                continue
            conv = normalize_row(json.loads(line), i, args.tier)
            if not conv:
                continue
            # re-chunk if needed
            if args.chunk_turns and conv["sessions"] and len(conv["sessions"]) == 1:
                turns = conv["sessions"][0]["turns"]
                if len(turns) > args.chunk_turns:
                    conv["sessions"] = chunk_sessions(turns, args.chunk_turns)
            o.write(json.dumps(conv, ensure_ascii=False) + "\n")
            n += 1
    print(f"wrote {out}: conversations={n}")


if __name__ == "__main__":
    main()
