#!/usr/bin/env python3
"""WS-A: normalize MSC (gonced8/multi-session_chat) to SCHEMA.md.

MSC is multi-session dialogue with persona annotations — no official QA.
For full-tier adapter regression we synthesize persona-fact questions:
one question per persona bullet from session 0 / init_personas, answer =
the bullet text, evidence = s1 (first dialogue session).

These questions are **synthetic** (see LICENSE-NOTE-msc.md). Headline
paper numbers must not treat them as an official MSC score.
"""

from __future__ import annotations

import argparse
import json
from datetime import datetime, timedelta, timezone
from pathlib import Path


def role_for(speaker: str, speaker_map: dict[str, str]) -> str:
    return speaker_map.setdefault(speaker, "user" if len(speaker_map) == 0 else "assistant")


def normalize_row(row: dict, split: str) -> dict:
    cid = f"msc-{split}-{row['id']}"
    speakers: dict[str, str] = {}
    sessions_out = []
    base = datetime(2023, 1, 1, tzinfo=timezone.utc) + timedelta(days=int(row["id"]) % 365)

    for i, sess in enumerate(row.get("sessions") or []):
        sid = f"s{i + 1}"
        date = (base + timedelta(days=i)).strftime("%Y-%m-%dT%H:%M:%SZ")
        turns = []
        for j, utt in enumerate(sess.get("dialogue") or []):
            sp = utt.get("speaker") or "Speaker 1"
            role = role_for(sp, speakers)
            text = f"{sp}: {(utt.get('text') or '').strip()}"
            turns.append({"role": role if role in ("user", "assistant") else "user", "text": text})
            # SCHEMA only allows user|assistant — map second speaker to assistant
            if role not in ("user", "assistant"):
                turns[-1]["role"] = "assistant" if len(speakers) > 1 else "user"
        # Fix roles: first unique speaker → user, second → assistant
        order = list(speakers.keys())
        fixed = []
        for utt in sess.get("dialogue") or []:
            sp = utt.get("speaker") or order[0]
            role = "user" if sp == order[0] else "assistant"
            fixed.append({"role": role, "text": f"{sp}: {(utt.get('text') or '').strip()}"})
        if not fixed:
            continue
        sessions_out.append({"session_id": sid, "date": date, "turns": fixed})

    questions = []
    facts = []
    for p in row.get("init_personas") or []:
        for t in p.get("text") or []:
            if t.strip():
                facts.append((p.get("speaker") or "Speaker", t.strip()))
    if sessions_out and not facts:
        for p in (row["sessions"][0].get("personas") or []):
            for t in p.get("text") or []:
                if t.strip():
                    facts.append((p.get("speaker") or "Speaker", t.strip()))

    for qi, (speaker, fact) in enumerate(facts[:20]):  # cap per conversation
        questions.append(
            {
                "question_id": f"q{qi + 1}",
                "category": "persona-fact",
                "question": f"What did {speaker} say about themselves regarding: {fact[:60]}?",
                "answer": fact,
                "evidence_session_ids": ["s1"] if sessions_out else [],
                "extra": {"synthetic": True, "benchmark": "msc"},
            }
        )

    return {
        "conversation_id": cid,
        "sessions": sessions_out,
        "questions": questions,
        "extra": {"benchmark": "msc", "split": split, "source_id": row.get("id")},
    }


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--data-dir", type=Path, default=None)
    ap.add_argument("--output", type=Path, default=None)
    ap.add_argument("--splits", default="train,valid,test")
    ap.add_argument("--limit", type=int, default=0, help="max conversations (0=all)")
    args = ap.parse_args()
    here = Path(__file__).resolve().parent
    data_dir = args.data_dir or here / "data"
    out = args.output or data_dir / "msc-conversations.jsonl"

    n = 0
    with open(out, "w") as fout:
        for split in args.splits.split(","):
            path = data_dir / f"msc-{split}.jsonl"
            if not path.exists():
                raise SystemExit(f"missing {path} — run get_msc.sh first")
            with open(path) as f:
                for line in f:
                    if not line.strip():
                        continue
                    row = json.loads(line)
                    conv = normalize_row(row, split)
                    if not conv["sessions"]:
                        continue
                    fout.write(json.dumps(conv, ensure_ascii=False) + "\n")
                    n += 1
                    if args.limit and n >= args.limit:
                        break
            if args.limit and n >= args.limit:
                break
    print(f"wrote {out}: conversations={n}")


if __name__ == "__main__":
    main()
