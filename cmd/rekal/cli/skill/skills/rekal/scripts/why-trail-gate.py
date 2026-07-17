#!/usr/bin/env python3
"""Gate WHY synthesis on gather size.

Input: newline-delimited gather rows (TSV/CSV/anything) on stdin, or a JSON
array / {"rows":[…]} / {"results":[…]}.

Exit:
  0 — PASS n=<count>  (enough trail to synthesize)
  1 — SILENCE n=<count> reason=under_gathered
  2 — error

Default minimum: 10 rows. Override: WHY_TRAIL_MIN=N
"""
from __future__ import annotations

import json
import os
import sys

DEFAULT_MIN = 10


def count_rows(raw: str) -> int:
    raw = raw.strip()
    if not raw:
        return 0
    if raw[0] in "{[":
        data = json.loads(raw)
        if isinstance(data, list):
            return len(data)
        if isinstance(data, dict):
            for key in ("rows", "results", "turns"):
                if isinstance(data.get(key), list):
                    return len(data[key])
            return 0
    return sum(1 for line in raw.splitlines() if line.strip())


def main() -> int:
    if len(sys.argv) > 2 or (len(sys.argv) == 2 and sys.argv[1] in ("-h", "--help")):
        print("usage: why-trail-gate.py [gather.txt|gather.json]  # else stdin", file=sys.stderr)
        return 2
    try:
        raw = open(sys.argv[1], encoding="utf-8").read() if len(sys.argv) == 2 else sys.stdin.read()
        n = count_rows(raw)
    except (OSError, json.JSONDecodeError) as e:
        print(f"SILENCE n=0 reason=parse_error:{e}", file=sys.stderr)
        return 2

    try:
        minimum = int(os.environ.get("WHY_TRAIL_MIN", str(DEFAULT_MIN)))
    except ValueError:
        minimum = DEFAULT_MIN

    if n >= minimum:
        print(f"PASS n={n} min={minimum}")
        return 0
    print(f"SILENCE n={n} min={minimum} reason=under_gathered")
    return 1


if __name__ == "__main__":
    sys.exit(main())
