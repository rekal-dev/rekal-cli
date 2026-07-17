#!/usr/bin/env python3
"""WS-A: normalize BEAM raw HF rows to SCHEMA.md.

BEAM schema varies by release; we detect common fields:
  messages / conversation / dialogue → sessions of turns
  questions / probes → questions with evidence if present

If a row is one long dialogue, we split into sessions of ~20 turns
(dated daily) so sh-gen can checkpoint incrementally (--fast helps).
"""

from __future__ import annotations

import argparse
import json
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
            out.append({"role": role, "text": str(text)})
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


def normalize_row(row: dict, idx: int, tier: str) -> dict | None:
    cid = str(row.get("id") or row.get("conversation_id") or f"beam-{tier}-{idx}")
    msgs = row.get("messages") or row.get("conversation") or row.get("dialogue") or row.get("chat")
    if isinstance(msgs, str):
        try:
            msgs = json.loads(msgs)
        except json.JSONDecodeError:
            msgs = [{"role": "user", "text": msgs}]
    if not msgs and "sessions" in row:
        # already session-shaped
        sessions = row["sessions"]
    elif msgs:
        turns = turns_from_messages(msgs if isinstance(msgs, list) else [])
        sessions = chunk_sessions(turns)
    else:
        return None

    questions = []
    for qi, q in enumerate(row.get("questions") or row.get("probes") or row.get("qa") or []):
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
                    "category": str(q.get("category") or q.get("ability") or "beam"),
                    "question": q.get("question") or q.get("query") or "",
                    "answer": q.get("answer") or q.get("gold") or "",
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
