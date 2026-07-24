#!/usr/bin/env python3
"""Session-level recall probe (NOT an official benchmark metric).

For each answerable question of one ingested conversation, run stock
`rekal "<question>"` and check whether the top-ranked session is one of the
question's evidence sessions. Session identity is recovered from the
result's `files` list (each synthetic commit touches exactly
`sessions/<date>-<sid>.md`; contract in 02-adapter-architecture §1).

This is a WS-B/WS-A plumbing probe: it establishes that real questions
retrieve against a real ingested corpus and gives a directional
session-recall@k floor with stock (uncalibrated) gates. Official metrics
come from WS-E's scoring wrappers; do not quote these numbers as LoCoMo
results.
"""

import argparse
import json
import re
import subprocess
import sys
import time
from collections import defaultdict
from pathlib import Path

MARKER_RE = re.compile(r"sessions/\d{4}-\d{2}-\d{2}-(s\d+)\.md")


def rekal_recall(rekal, repo, env, question, retries=8):
    for _ in range(retries):
        p = subprocess.run([rekal, question], cwd=repo, env=env,
                           capture_output=True, text=True)
        if "another rekal process" not in (p.stdout + p.stderr):
            break
        time.sleep(1)
    try:
        return json.loads(p.stdout)
    except json.JSONDecodeError:
        raise RuntimeError(f"recall did not return JSON: {p.stdout[:300]} {p.stderr[:300]}")


def result_session_ids(result):
    out = []
    for f in result.get("session", {}).get("files", []):
        m = MARKER_RE.search(f)
        if m:
            out.append(m.group(1))
    return out


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--corpus", required=True, help="interchange conversations.jsonl")
    ap.add_argument("--workdir", required=True, help="sh-gen --out dir")
    ap.add_argument("--conversation", required=True)
    ap.add_argument("--rekal", required=True)
    ap.add_argument("--limit", type=int, default=0, help="max questions (0 = all)")
    ap.add_argument("--out", default=None, help="write per-question jsonl here")
    args = ap.parse_args()

    conv = None
    with open(args.corpus) as f:
        for line in f:
            c = json.loads(line)
            if c["conversation_id"] == args.conversation:
                conv = c
                break
    if conv is None:
        sys.exit(f"conversation {args.conversation} not in {args.corpus}")

    conv_dir = Path(args.workdir) / args.conversation
    repo = conv_dir / "repo"
    import os
    env = dict(os.environ)
    for k in ("REKAL_BENCH", "REKAL_SKIP_CHECKPOINT"):
        env.pop(k, None)
    env["CLAUDE_CONFIG_DIR"] = str(conv_dir / "claude-config")

    questions = [q for q in conv["questions"] if q["evidence_session_ids"]]
    if args.limit:
        questions = questions[: args.limit]

    rows = []
    hits1 = hits3 = 0
    by_cat = defaultdict(lambda: [0, 0])
    t0 = time.time()
    for q in questions:
        res = rekal_recall(args.rekal, repo, env, q["question"])
        ranked = [s for r in res.get("results", []) for s in result_session_ids(r)]
        gold = set(q["evidence_session_ids"])
        top1 = bool(ranked) and ranked[0] in gold
        top3 = any(s in gold for s in ranked[:3])
        hits1 += top1
        hits3 += top3
        by_cat[q["category"]][0] += top1
        by_cat[q["category"]][1] += 1
        rows.append({
            "question_id": q["question_id"], "category": q["category"],
            "gold_sessions": sorted(gold), "ranked_sessions": ranked[:5],
            "top1_hit": top1, "top3_hit": top3,
            "top1_confidence": (res.get("results") or [{}])[0].get("confidence"),
        })
    dt = time.time() - t0

    n = len(questions)
    print(f"conversation={args.conversation} answerable_questions={n} "
          f"wall={dt:.0f}s ({dt/max(n,1):.1f}s/q)")
    print(f"session-recall@1 = {hits1}/{n} = {hits1/max(n,1):.2f}")
    print(f"session-recall@3 = {hits3}/{n} = {hits3/max(n,1):.2f}")
    for cat, (h, t) in sorted(by_cat.items()):
        print(f"  {cat:16s} recall@1 {h}/{t} = {h/max(t,1):.2f}")

    if args.out:
        with open(args.out, "w") as f:
            for r in rows:
                f.write(json.dumps(r) + "\n")
        print(f"per-question rows -> {args.out}")


if __name__ == "__main__":
    main()
