#!/usr/bin/env python3
"""Propose a recall profile from industry-bench smoke/full run dirs.

Wraps the skill script so bench agents and WS-D share one proposer.
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--runs", type=Path, nargs="+", required=True,
                    help="smoke or full run directories (contain manifest / per_question)")
    ap.add_argument("--profile", default="auto")
    ap.add_argument("--out", type=Path, required=True)
    args = ap.parse_args()

    skill = (
        Path(__file__).resolve().parents[3]
        / "cmd/rekal/cli/skill/skills/rekal/scripts/calibrate-recall.py"
    )
    # parents: calibration -> industry-bench -> scripts -> repo root = 3? 
    # __file__ = scripts/industry-bench/calibration/propose_from_smokes.py
    # parent0=calibration, 1=industry-bench, 2=scripts, 3=repo root
    repo = Path(__file__).resolve().parents[3]
    skill = repo / "cmd/rekal/cli/skill/skills/rekal/scripts/calibrate-recall.py"

    cmd = [sys.executable, str(skill), "--profile", args.profile]
    for r in args.runs:
        cmd.extend(["--from-smoke", str(r)])
    out = subprocess.check_output(cmd, text=True)
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(out if out.endswith("\n") else out + "\n")
    print(f"wrote {args.out}")


if __name__ == "__main__":
    main()
