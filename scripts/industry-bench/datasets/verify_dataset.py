#!/usr/bin/env python3
"""WS-A: verify a normalized conversations.jsonl against paper-reported sizes
and SCHEMA.md invariants. Stdlib only.

Usage:
  python3 verify_dataset.py longmemeval-s
  python3 verify_dataset.py locomo
  python3 verify_dataset.py toy
  python3 verify_dataset.py --file path/to/conversations.jsonl [--expect-json '{...}']
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import Counter
from pathlib import Path

# Expected sizes from the published papers / upstream READMEs.
# LongMemEval cleaned release: 500 instances (github.com/xiaowu0162/LongMemEval).
# LoCoMo locomo10.json: 10 conversations (snap-research/locomo).
# Session/turn totals are pinned from a verified download so regressions
# in the normalizer are loud; bump intentionally when upstream cleans data.
EXPECT = {
    "longmemeval-s": {
        "file": "data/longmemeval-s-conversations.jsonl",
        "conversations": 500,
        "questions": 500,
        "abstention": 30,
        "sessions": 23867,
        "turns": 246750,
    },
    "locomo": {
        "file": "data/locomo-conversations.jsonl",
        "conversations": 10,
        "questions": 1986,
        "sessions": 272,
        "turns": 5882,
    },
    "toy": {
        "file": "toy/conversations.jsonl",
        "conversations": 1,
        "questions": 4,
        "sessions": 3,
        "turns": 12,
        "abstention": 1,
    },
}

REQUIRED_CONV = ("conversation_id", "sessions", "questions")
REQUIRED_SESS = ("session_id", "date", "turns")
REQUIRED_TURN = ("role", "text")
REQUIRED_Q = ("question_id", "category", "question", "answer", "evidence_session_ids")
DATE_PREFIX = None  # RFC3339 checked loosely below


def load_jsonl(path: Path) -> list:
    rows = []
    with open(path) as f:
        for i, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError as e:
                sys.exit(f"{path}:{i}: JSON error: {e}")
    return rows


def check_schema(convs: list) -> list[str]:
    errs = []
    seen_cids = set()
    for ci, c in enumerate(convs):
        for k in REQUIRED_CONV:
            if k not in c:
                errs.append(f"conv[{ci}]: missing {k}")
        cid = c.get("conversation_id")
        if cid in seen_cids:
            errs.append(f"duplicate conversation_id {cid!r}")
        seen_cids.add(cid)
        sessions = c.get("sessions") or []
        if not sessions:
            errs.append(f"{cid}: empty sessions")
        prev_date = ""
        for s in sessions:
            for k in REQUIRED_SESS:
                if k not in s:
                    errs.append(f"{cid}: session missing {k}")
            d = s.get("date", "")
            if not (isinstance(d, str) and d.endswith("Z") and "T" in d):
                errs.append(f"{cid}/{s.get('session_id')}: bad date {d!r}")
            if d < prev_date:
                errs.append(f"{cid}: sessions not chronological ({prev_date} → {d})")
            prev_date = d
            for t in s.get("turns") or []:
                for k in REQUIRED_TURN:
                    if k not in t:
                        errs.append(f"{cid}: turn missing {k}")
                if t.get("role") not in ("user", "assistant"):
                    errs.append(f"{cid}: bad role {t.get('role')!r}")
        for q in c.get("questions") or []:
            for k in REQUIRED_Q:
                if k not in q:
                    errs.append(f"{cid}: question missing {k}")
            ev = q.get("evidence_session_ids")
            if not isinstance(ev, list):
                errs.append(f"{cid}/{q.get('question_id')}: evidence_session_ids not a list")
            elif q.get("category") == "abstention" and ev:
                errs.append(
                    f"{cid}/{q.get('question_id')}: abstention must have empty evidence"
                )
    return errs


def summarize(convs: list) -> dict:
    cats = Counter()
    n_abs = 0
    for c in convs:
        for q in c["questions"]:
            cats[q["category"]] += 1
            if q["category"] == "abstention":
                n_abs += 1
    return {
        "conversations": len(convs),
        "sessions": sum(len(c["sessions"]) for c in convs),
        "turns": sum(len(s["turns"]) for c in convs for s in c["sessions"]),
        "questions": sum(len(c["questions"]) for c in convs),
        "abstention": n_abs,
        "categories": dict(cats),
    }


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("name", nargs="?", help="dataset key: longmemeval-s | locomo | toy")
    ap.add_argument("--file", type=Path, help="explicit conversations.jsonl path")
    ap.add_argument(
        "--expect-json",
        default=None,
        help='JSON object overriding expected counts, e.g. \'{"conversations":500}\'',
    )
    ap.add_argument(
        "--skip-count-check",
        action="store_true",
        help="schema-only (useful while pinning a new upstream release)",
    )
    args = ap.parse_args()

    here = Path(__file__).parent
    expect = {}
    if args.name:
        if args.name not in EXPECT:
            sys.exit(f"unknown dataset {args.name!r}; known: {', '.join(EXPECT)}")
        expect = dict(EXPECT[args.name])
        path = here / expect.pop("file")
    elif args.file:
        path = args.file
    else:
        ap.print_help()
        sys.exit(2)

    if args.expect_json:
        expect.update(json.loads(args.expect_json))

    if not path.exists():
        sys.exit(f"missing {path} (run the matching get_*.sh + normalize_*.py first)")

    convs = load_jsonl(path)
    errs = check_schema(convs)
    summary = summarize(convs)
    print(f"file: {path}")
    print(
        f"counts: conversations={summary['conversations']} sessions={summary['sessions']} "
        f"turns={summary['turns']} questions={summary['questions']} "
        f"abstention={summary['abstention']}"
    )
    print(f"categories: {summary['categories']}")

    if errs:
        print(f"SCHEMA FAIL ({len(errs)}):")
        for e in errs[:40]:
            print(f"  - {e}")
        if len(errs) > 40:
            print(f"  ... and {len(errs) - 40} more")
        sys.exit(1)
    print("schema: ok")

    if args.skip_count_check or not expect:
        sys.exit(0)

    mismatches = []
    for k, want in expect.items():
        got = summary.get(k)
        if got != want:
            mismatches.append(f"{k}: got {got}, want {want}")
    if mismatches:
        print("COUNT FAIL (vs paper / pinned download):")
        for m in mismatches:
            print(f"  - {m}")
        sys.exit(1)
    print("counts: ok (match expected)")
    sys.exit(0)


if __name__ == "__main__":
    main()
