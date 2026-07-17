#!/usr/bin/env python3
"""Minimal smoke scorers (WS-E). Official LongMemEval/LoCoMo judges come later.

exact_match_contains: gold answer string appears in hypothesis (casefold).
deliberately_wrong: hypothesis that cannot contain gold → score 0 (sanity).
"""

from __future__ import annotations

import argparse
import json
import sys


def contains_score(hypothesis: str, gold: str) -> float:
    if not gold:
        # abstention / empty gold: treat "don't know"/"unknown" as correct
        h = (hypothesis or "").casefold()
        return 1.0 if any(x in h for x in ("don't know", "do not know", "unknown", "i don't know")) else 0.0
    return 1.0 if gold.casefold() in (hypothesis or "").casefold() else 0.0


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--hypothesis", required=True)
    ap.add_argument("--gold", required=True)
    ap.add_argument("--sanity-wrong", action="store_true",
                    help="replace hypothesis with a wrong string; expect score 0")
    args = ap.parse_args()
    hyp = "COMPLETELY UNRELATED WRONG ANSWER XYZ" if args.sanity_wrong else args.hypothesis
    score = contains_score(hyp, args.gold)
    print(json.dumps({"hypothesis": hyp, "gold": args.gold, "score": score}))
    if args.sanity_wrong and score != 0.0:
        sys.exit("sanity-wrong expected score 0")
    if not args.sanity_wrong and args.gold and score != 1.0:
        # soft: print only; caller decides
        pass


if __name__ == "__main__":
    main()
