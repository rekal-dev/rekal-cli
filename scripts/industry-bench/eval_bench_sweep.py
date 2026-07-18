#!/usr/bin/env python3
"""Benchmark-agnostic fast weight-sweep evaluator.

Reads questions once from an input jsonl, maps each conversation to its ingested
+ indexed repo (flat `<root>/<conv>/repo` or sharded `<root>/shard-*/<conv>/repo`),
and for each named weight profile runs one `rekal --weights <p> "<q>" -n <limit>`
per answerable question, scoring evidence rank by marker-file match
(`-<evidence_id>.md`). Avoids the shim's per-conv full-file rescan, so it scales
to large inputs (e.g. LME-M's 2.5 GB / 482-session conversations).

  python3 eval_bench_sweep.py --input <jsonl> --root ~/imb-lme-m \
      --profiles profiles.json --limit 20 --workers 6 --ids-file dev-ids.txt
"""
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import time
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
REKAL = str(REPO_ROOT / "rekal")


def marker(evid: str, fp: str) -> bool:
    return bool(re.search(rf"-{re.escape(evid)}\.md$", fp))


def build_repo_index(root: Path) -> dict[str, Path]:
    idx: dict[str, Path] = {}
    for repo in root.glob("*/repo"):
        if (repo / ".rekal").exists():
            idx[repo.parent.name] = repo
    for repo in root.glob("shard-*/*/repo"):
        if (repo / ".rekal").exists():
            idx[repo.parent.name] = repo
    return idx


def load_questions(input_jsonl: Path, ids: set[str] | None) -> list[dict]:
    out = []
    with open(input_jsonl) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            c = json.loads(line)
            cid = c["conversation_id"]
            if ids is not None and cid not in ids:
                continue
            for q in c.get("questions") or []:
                ev = q.get("evidence_session_ids") or []
                if ev:
                    out.append({"conv": cid, "q": q["question"], "ev": ev, "cat": q.get("category")})
    return out


def recall_rank(repo: Path, query: str, ev: list[str], wj: str, limit: int, env: dict, retries: int = 6) -> int | None:
    for i in range(retries):
        try:
            p = subprocess.run(
                [REKAL, "--weights", wj, query, "--limit", str(limit)],
                cwd=repo, env=env, capture_output=True, text=True, timeout=120,
            )
        except subprocess.TimeoutExpired:
            time.sleep(1.0 * (i + 1))
            continue
        out = (p.stdout or "") + (p.stderr or "")
        if p.returncode == 0:
            s, e = p.stdout.find("{"), p.stdout.rfind("}")
            if s < 0:
                return None
            try:
                payload = json.loads(p.stdout[s : e + 1])
            except json.JSONDecodeError:
                return None
            for rank, r in enumerate(payload.get("results") or []):
                files = (r.get("session") or {}).get("files") or []
                if any(marker(evid, str(fp)) for fp in files for evid in ev):
                    return rank
            return None
        low = out.lower()
        if any(x in low for x in ("another rekal process", "lock", "index not built", "rebuilding")):
            time.sleep(1.0 * (i + 1))
            continue
        return None
    return None


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--input", type=Path, required=True)
    ap.add_argument("--root", type=Path, required=True)
    ap.add_argument("--profiles", type=Path, required=True, help="JSON: {name: weights_dict}")
    ap.add_argument("--limit", type=int, default=20)
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--ids-file", type=Path, default=None)
    ap.add_argument("--out", type=Path, default=None)
    args = ap.parse_args()

    ids = None
    if args.ids_file:
        ids = {x.strip() for x in args.ids_file.read_text().splitlines() if x.strip()}

    repo_index = build_repo_index(args.root)
    questions = [q for q in load_questions(args.input, ids) if q["conv"] in repo_index]
    profiles = json.loads(args.profiles.read_text())

    env = dict(os.environ)
    env["PATH"] = f"{Path(REKAL).parent}{os.pathsep}{env.get('PATH', '')}"
    for k in ("REKAL_DISABLE_CAPTURE", "REKAL_NO_CAPTURE"):
        env.pop(k, None)

    n = len(questions)
    print(f"# sweep: {n} answerable questions, {len(profiles)} profiles, limit={args.limit}")
    results = {}
    for name, weights in profiles.items():
        wj = json.dumps(weights, separators=(",", ":"), sort_keys=True)

        def work(q):
            return (recall_rank(repo_index[q["conv"]], q["q"], q["ev"], wj, args.limit, env), q["cat"])

        ranks = [None] * n
        with ThreadPoolExecutor(max_workers=args.workers) as ex:
            for i, r in enumerate(ex.map(work, questions)):
                ranks[i] = r

        def rate(k):
            return sum(1 for r, _ in ranks if r is not None and r < k) / n if n else 0.0

        miss5 = Counter(c for r, c in ranks if not (r is not None and r < 5))
        results[name] = {
            "ev@5": round(rate(5), 4), "ev@10": round(rate(10), 4), "ev@20": round(rate(20), 4),
            "found@limit": round(sum(1 for r, _ in ranks if r is not None) / n, 4) if n else 0.0,
            "ev@5_miss_by_cat": dict(miss5),
        }
        print(f"  {name:24} ev@5={results[name]['ev@5']:.4f} ev@10={results[name]['ev@10']:.4f} "
              f"ev@20={results[name]['ev@20']:.4f} found@{args.limit}={results[name]['found@limit']:.4f}")

    ranked = sorted(results.items(), key=lambda kv: kv[1]["ev@5"], reverse=True)
    print("\n# ranked by ev@5")
    for name, r in ranked:
        print(f"  {name:24} ev@5={r['ev@5']:.4f}")
    payload = {"n": n, "limit": args.limit, "profiles": results, "winner": ranked[0][0] if ranked else None}
    if args.out:
        args.out.write_text(json.dumps(payload, indent=2) + "\n")
        print(f"\nwrote {args.out}")


if __name__ == "__main__":
    main()
