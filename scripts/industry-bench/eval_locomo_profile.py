#!/usr/bin/env python3
"""Fast LoCoMo profile evaluator (recall ceiling + all-seeds).

For a given weights JSON, run one rekal recall per answerable question at a deep
limit and compute, by marker-file match against retrieved session files:
  - any@k   : at least one evidence session in top-k (recall)
  - all@k   : every evidence session in top-k  ("get all seeds")

Parallelism is across conversations (distinct repos → no DuckDB index lock
contention); questions within a conversation run sequentially. rekal calls retry
on transient lock / index-rebuild states, mirroring shim.run_rekal.

  python3 eval_locomo_profile.py --weights '{"bm25":0.7,...}' --limit 50 --label bm25-heavy
"""
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import time
from collections import Counter, defaultdict
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
REKAL = str(REPO_ROOT / "rekal")
LOCO = Path.home() / "imb-locomo"
INPUT = REPO_ROOT / "scripts/industry-bench/datasets/data/locomo-conversations.jsonl"
KS = (5, 10, 20, 50)


def marker(evid: str, fp: str) -> bool:
    return bool(re.search(rf"-{re.escape(evid)}\.md$", fp))


def load_by_conv() -> dict[str, list[dict]]:
    by = defaultdict(list)
    for line in open(INPUT):
        line = line.strip()
        if not line:
            continue
        conv = json.loads(line)
        cid = conv["conversation_id"]
        for q in conv.get("questions") or []:
            ev = q.get("evidence_session_ids") or []
            if ev:
                by[cid].append({"q": q["question"], "ev": ev, "cat": q.get("category")})
    return by


def run_rekal(repo: Path, wj: str, query: str, limit: int, env: dict, retries: int = 8) -> dict:
    for i in range(retries):
        p = subprocess.run(
            [REKAL, "--weights", wj, query, "--limit", str(limit), "--json"],
            cwd=repo, env=env, capture_output=True, text=True, timeout=120,
        )
        out = (p.stdout or "") + (p.stderr or "")
        if p.returncode == 0:
            s, e = p.stdout.find("{"), p.stdout.rfind("}")
            if s >= 0:
                return json.loads(p.stdout[s : e + 1])
            return {"results": []}
        low = out.lower()
        if any(t in low for t in ("another rekal process", "lock", "index not built", "rebuilding")):
            time.sleep(1.0 * (i + 1))
            continue
        return {"results": []}
    return {"results": []}


def eval_conv(cid: str, items: list[dict], wj: str, limit: int, env: dict) -> list[dict]:
    repo = LOCO / cid / "repo"
    res = []
    if not repo.exists():
        return [{"cat": it["cat"], "ev_n": len(it["ev"]), "ranks": []} for it in items]
    for it in items:
        payload = run_rekal(repo, wj, it["q"], limit, env)
        # rank position of each evidence session (min rank per ev id)
        ev_ranks = {}
        for i, r in enumerate(payload.get("results") or []):
            files = (r.get("session") or {}).get("files") or []
            for evid in it["ev"]:
                if evid in ev_ranks:
                    continue
                if any(marker(evid, str(fp)) for fp in files):
                    ev_ranks[evid] = i
        res.append({"cat": it["cat"], "ev_n": len(it["ev"]), "ranks": [ev_ranks.get(e) for e in it["ev"]]})
    return res


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--weights", required=True)
    ap.add_argument("--limit", type=int, default=50)
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--label", default="profile")
    args = ap.parse_args()

    weights = json.loads(args.weights)
    wj = json.dumps(weights, separators=(",", ":"), sort_keys=True)
    env = dict(os.environ)
    env["PATH"] = f"{Path(REKAL).parent}{os.pathsep}{env.get('PATH', '')}"
    for k in ("REKAL_DISABLE_CAPTURE", "REKAL_NO_CAPTURE"):
        env.pop(k, None)

    by = load_by_conv()
    all_rows: list[dict] = []
    with ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {ex.submit(eval_conv, cid, items, wj, args.limit, env): cid for cid, items in by.items()}
        for fut in futs:
            all_rows.extend(fut.result())

    n = len(all_rows)

    def any_at(k: int) -> float:
        return sum(1 for r in all_rows if any(x is not None and x < k for x in r["ranks"])) / n

    def all_at(k: int) -> float:
        return sum(1 for r in all_rows if r["ranks"] and all(x is not None and x < k for x in r["ranks"])) / n

    multi = [r for r in all_rows if r["ev_n"] > 1]
    def all_at_multi(k: int) -> float:
        if not multi:
            return 0.0
        return sum(1 for r in multi if all(x is not None and x < k for x in r["ranks"])) / len(multi)

    print(json.dumps({
        "label": args.label,
        "weights": weights,
        "limit": args.limit,
        "n_answerable": n,
        "n_multi_evidence": len(multi),
        "any@k": {str(k): round(any_at(k), 4) for k in KS},
        "all_seeds@k": {str(k): round(all_at(k), 4) for k in KS},
        "all_seeds@k_multi_only": {str(k): round(all_at_multi(k), 4) for k in KS},
    }, indent=2))


if __name__ == "__main__":
    main()
