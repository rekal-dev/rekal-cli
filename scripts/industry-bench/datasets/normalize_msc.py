#!/usr/bin/env python3
"""WS-A: normalize MSC (gonced8/multi-session_chat) to SCHEMA.md.

MSC is multi-session dialogue with persona annotations — no official QA.
For full-tier adapter regression we synthesize persona-fact questions:
one question per persona bullet from init_personas / session-0 personas,
answer = the bullet text, evidence = s1.

These questions are **synthetic** (see LICENSE-NOTE-msc.md). Headline
paper numbers must not treat them as an official MSC score.
"""

from __future__ import annotations

import argparse
import json
from datetime import datetime, timedelta, timezone
from pathlib import Path


def normalize_row(row: dict, split: str) -> dict:
    cid = f"msc-{split}-{row['id']}"
    sessions_out = []
    base = datetime(2023, 1, 1, tzinfo=timezone.utc) + timedelta(days=int(row["id"]) % 365)
    speaker_order: list[str] = []

    for i, sess in enumerate(row.get("sessions") or []):
        dialogue = sess.get("dialogue") or []
        if not dialogue:
            continue
        for utt in dialogue:
            sp = utt.get("speaker") or "Speaker 1"
            if sp not in speaker_order:
                speaker_order.append(sp)
        first = speaker_order[0] if speaker_order else "Speaker 1"
        turns = []
        for utt in dialogue:
            sp = utt.get("speaker") or first
            role = "user" if sp == first else "assistant"
            turns.append({"role": role, "text": f"{sp}: {(utt.get('text') or '').strip()}"})
        sid = f"s{i + 1}"
        date = (base + timedelta(days=i)).strftime("%Y-%m-%dT%H:%M:%SZ")
        sessions_out.append({"session_id": sid, "date": date, "turns": turns})

    facts = []
    for p in row.get("init_personas") or []:
        for t in p.get("text") or []:
            if t.strip():
                facts.append((p.get("speaker") or "Speaker", t.strip()))
    if sessions_out and not facts and row.get("sessions"):
        for p in (row["sessions"][0].get("personas") or []):
            for t in p.get("text") or []:
                if t.strip():
                    facts.append((p.get("speaker") or "Speaker", t.strip()))

    questions = []
    for qi, (speaker, fact) in enumerate(facts[:20]):
        questions.append(
            {
                "question_id": f"q{qi + 1}",
                "category": "persona-fact",
                "question": f"According to their persona, what did {speaker} say: {fact[:80]}?",
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
                    conv = normalize_row(json.loads(line), split)
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
