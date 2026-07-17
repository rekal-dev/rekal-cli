#!/usr/bin/env python3
"""WS-A: normalize LongMemEval (cleaned) to the interchange format
(SCHEMA.md). Stdlib only, deterministic.

Mapping decisions (recorded here because they are judgement calls):

- Each LongMemEval *instance* (one question + its haystack) becomes one
  interchange conversation. conversation_id = question_id.
- Haystack session ids are kept verbatim as session_id (e.g.
  sharegpt_…, answer_…). evidence_session_ids = answer_session_ids.
- Turn field rename: content → text. role is already user|assistant.
  has_answer (evidence-turn label) is preserved under turn-level extra
  only if present — but SCHEMA turns are {role,text} only, so has_answer
  goes into the session's optional extra.evidence_turn_indices (0-based).
- Dates like "2023/05/20 (Sat) 02:21" parse to RFC3339 UTC. The weekday
  token is ignored; missing seconds → :00.
- question_type is the benchmark taxonomy and becomes category, EXCEPT
  when question_id ends with "_abs": category is forced to "abstention"
  (paper's fifth ability) and the original question_type is kept in
  extra.question_type. Answer stays as published (often empty / "unknown"
  phrasing); evidence_session_ids stays [] for abstention so the silence
  gate is the intended path.
- question_date is stored in extra.question_date (RFC3339); it is not a
  session.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

DATE_RE = re.compile(
    r"^(\d{4})/(\d{2})/(\d{2})\s+\([A-Za-z]+\)\s+(\d{1,2}):(\d{2})(?:\s*)?$"
)


def parse_date(s: str) -> str:
    m = DATE_RE.match(s.strip())
    if not m:
        raise ValueError(f"unparseable LongMemEval date: {s!r}")
    year, month, day, hh, mm = m.groups()
    dt = datetime(
        int(year), int(month), int(day), int(hh), int(mm), 0, tzinfo=timezone.utc
    )
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


def normalize_instance(inst: dict) -> dict:
    qid = inst["question_id"]
    qtype = inst["question_type"]
    is_abs = qid.endswith("_abs")
    category = "abstention" if is_abs else qtype

    ids = inst["haystack_session_ids"]
    dates = inst["haystack_dates"]
    sessions_raw = inst["haystack_sessions"]
    if not (len(ids) == len(dates) == len(sessions_raw)):
        raise ValueError(
            f"{qid}: haystack length mismatch "
            f"ids={len(ids)} dates={len(dates)} sessions={len(sessions_raw)}"
        )

    sessions = []
    for sid, date_s, turns_raw in zip(ids, dates, sessions_raw):
        evidence_idxs = []
        turns = []
        for i, t in enumerate(turns_raw):
            role = t["role"]
            if role not in ("user", "assistant"):
                raise ValueError(f"{qid}/{sid}: bad role {role!r}")
            text = t.get("content")
            if text is None:
                raise ValueError(f"{qid}/{sid}: turn {i} missing content")
            turns.append({"role": role, "text": text})
            if t.get("has_answer"):
                evidence_idxs.append(i)
        sess = {
            "session_id": sid,
            "date": parse_date(date_s),
            "turns": turns,
        }
        if evidence_idxs:
            sess["extra"] = {"evidence_turn_indices": evidence_idxs}
        sessions.append(sess)
    sessions.sort(key=lambda s: (s["date"], s["session_id"]))

    evidence = list(inst.get("answer_session_ids") or [])
    if is_abs:
        evidence = []

    question = {
        "question_id": qid,
        "category": category,
        "question": inst["question"],
        "answer": "" if inst.get("answer") is None else str(inst["answer"]),
        "evidence_session_ids": evidence,
        "extra": {
            "question_type": qtype,
            "question_date": parse_date(inst["question_date"]),
            "abstention": is_abs,
        },
    }

    return {
        "conversation_id": qid,
        "sessions": sessions,
        "questions": [question],
        "extra": {"benchmark": "longmemeval", "source_question_type": qtype},
    }


def normalize(data: list) -> list:
    return [normalize_instance(inst) for inst in data]


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--input",
        default=None,
        help="longmemeval_*_cleaned.json (default: data/longmemeval_s_cleaned.json)",
    )
    ap.add_argument(
        "--output",
        default=None,
        help="conversations.jsonl (default: data/longmemeval-s-conversations.jsonl)",
    )
    ap.add_argument(
        "--variant",
        choices=("s", "m", "oracle"),
        default="s",
        help="default input/output filenames when --input/--output omitted",
    )
    args = ap.parse_args()

    here = Path(__file__).parent
    defaults = {
        "s": ("longmemeval_s_cleaned.json", "longmemeval-s-conversations.jsonl"),
        "m": ("longmemeval_m_cleaned.json", "longmemeval-m-conversations.jsonl"),
        "oracle": ("longmemeval_oracle.json", "longmemeval-oracle-conversations.jsonl"),
    }
    din, dout = defaults[args.variant]
    inp = Path(args.input) if args.input else here / "data" / din
    outp = Path(args.output) if args.output else here / "data" / dout

    data = json.load(open(inp))
    if not isinstance(data, list):
        sys.exit(f"{inp}: expected JSON array")
    convs = normalize(data)

    n_sess = sum(len(c["sessions"]) for c in convs)
    n_turn = sum(len(s["turns"]) for c in convs for s in c["sessions"])
    n_q = sum(len(c["questions"]) for c in convs)
    n_abs = sum(1 for c in convs for q in c["questions"] if q["category"] == "abstention")

    outp.parent.mkdir(parents=True, exist_ok=True)
    with open(outp, "w") as f:
        for c in convs:
            f.write(json.dumps(c, ensure_ascii=False) + "\n")
    print(
        f"wrote {outp}: conversations={len(convs)} sessions={n_sess} "
        f"turns={n_turn} questions={n_q} abstention={n_abs}"
    )

    raw_sess = sum(len(x["haystack_sessions"]) for x in data)
    raw_turn = sum(len(s) for x in data for s in x["haystack_sessions"])
    ok = (len(convs), n_sess, n_turn, n_q) == (len(data), raw_sess, raw_turn, len(data))
    print(
        f"cross-check vs raw: instances {len(data)} sessions {raw_sess} "
        f"turns {raw_turn} -> {'ok' if ok else 'MISMATCH'}"
    )
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
