#!/usr/bin/env python3
"""LoCoMo multi-lookup (agentic) retrieval evaluator — no LLM required.

Single-shot `rekal "question"` underestimates the skill: the skill instructs the
agent to search several ways and read multiple sessions. This models that with
deterministic query reformulation + reciprocal-rank fusion (RRF), so we can
measure the multi-lookup ceiling without an API key, then fold the winning
strategy into the shipped skill.

Variants per question (all deterministic):
  v0  original question
  v1  keyword query (content words only; drop stopwords + question words)
  v2..vk  clause splits on conjunctions / commas / "?"
  vt  temporal-emphasis variant (when temporal cue words present)

Fusion: RRF over each variant's ranked session list (k=60). Evidence rank is the
rank of the evidence session in the fused list.

  python3 eval_locomo_multilookup.py --limit 20 --workers 6 --set miss
  python3 eval_locomo_multilookup.py --limit 20 --workers 6 --set all
"""
from __future__ import annotations

import argparse
import glob
import json
import os
import re
import subprocess
import time
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
REKAL = str(REPO_ROOT / "rekal")
LOCO = Path.home() / "imb-locomo"
INPUT = REPO_ROOT / "scripts/industry-bench/datasets/data/locomo-conversations.jsonl"
WINNER_RUNS = REPO_ROOT / "scripts/industry-bench/runs/full/locomo-winner-20260718"
WEIGHTS = json.load(open(REPO_ROOT / "scripts/industry-bench/calibration/locomo-bm25-push.json"))["weights"]
WJ = json.dumps(WEIGHTS, separators=(",", ":"), sort_keys=True)

STOP = set("""a an the of to in on at for and or but if is are was were be been being do does did
have has had i you he she it we they me my your his her their our this that these those what when
where who whom which why how did do with about from into over under again then once will would can
could should may might must not no yes as by so than too very just also very there here them us""".split())
TEMPORAL_CUES = {"when", "before", "after", "first", "last", "date", "day", "month", "year", "recently", "ago", "earliest", "latest"}


def marker(evid: str, fp: str) -> bool:
    return bool(re.search(rf"-{re.escape(evid)}\.md$", fp))


def keywords(q: str) -> str:
    toks = re.findall(r"[A-Za-z0-9']+", q.lower())
    kept = [t for t in toks if t not in STOP and len(t) > 2]
    return " ".join(kept) if kept else q


def clauses(q: str) -> list[str]:
    parts = re.split(r"\b(?:and|or|but|then|,|;|\?)\b", q, flags=re.IGNORECASE)
    return [p.strip() for p in parts if len(p.strip().split()) >= 3]


def variants(q: str) -> list[str]:
    vs = [q, keywords(q)]
    vs.extend(clauses(q)[:2])
    if any(c in q.lower().split() for c in TEMPORAL_CUES):
        vs.append(keywords(q) + " date time when")
    seen, out = set(), []
    for v in vs:
        v = v.strip()
        if v and v.lower() not in seen:
            seen.add(v.lower())
            out.append(v)
    return out


def run_recall(repo: Path, query: str, limit: int, env: dict, retries: int = 6) -> list[str]:
    """Return ranked list of session file-basenames (retry on lock/rebuild)."""
    for i in range(retries):
        try:
            p = subprocess.run(
                [REKAL, "--weights", WJ, query, "--limit", str(limit), "--json"],
                cwd=repo, env=env, capture_output=True, text=True, timeout=90,
            )
        except subprocess.TimeoutExpired:
            time.sleep(1.0 * (i + 1))
            continue
        out = (p.stdout or "") + (p.stderr or "")
        if p.returncode == 0:
            s, e = p.stdout.find("{"), p.stdout.rfind("}")
            if s < 0:
                return []
            try:
                payload = json.loads(p.stdout[s : e + 1])
            except json.JSONDecodeError:
                return []
            files = []
            for r in payload.get("results") or []:
                fs = (r.get("session") or {}).get("files") or []
                files.append(str(fs[0]) if fs else "")
            return files
        low = out.lower()
        if any(x in low for x in ("another rekal process", "lock", "index not built", "rebuilding")):
            time.sleep(1.0 * (i + 1))
            continue
        return []
    return []


def rrf_fuse(rankings: list[list[str]], k: int = 60) -> list[str]:
    scores: dict[str, float] = {}
    for ranking in rankings:
        for rank, fp in enumerate(ranking):
            if not fp:
                continue
            scores[fp] = scores.get(fp, 0.0) + 1.0 / (k + rank + 1)
    return [fp for fp, _ in sorted(scores.items(), key=lambda x: x[1], reverse=True)]


def ev_rank(ranked: list[str], ev: list[str]) -> int | None:
    for i, fp in enumerate(ranked):
        if any(marker(e, fp) for e in ev):
            return i
    return None


def load_all() -> list[dict]:
    out = []
    for line in open(INPUT):
        line = line.strip()
        if not line:
            continue
        conv = json.loads(line)
        cid = conv["conversation_id"]
        for q in conv.get("questions") or []:
            if q.get("evidence_session_ids"):
                out.append({"conv": cid, "q": q["question"], "ev": q["evidence_session_ids"], "cat": q.get("category")})
    return out


def load_miss() -> list[dict]:
    out = []
    for f in glob.glob(str(WINNER_RUNS / "*-skill-winner-20260718/per_question.jsonl")):
        conv = Path(f).parent.name.split("-skill-")[0]
        for line in open(f):
            line = line.strip()
            if not line:
                continue
            d = json.loads(line)
            ev = d.get("evidence_session_ids") or []
            if ev and d.get("evidence_at_5") in (0, 0.0, False):
                out.append({"conv": conv, "q": d["question"], "ev": ev, "cat": d.get("category")})
    return out


def eval_item(item: dict, limit: int, env: dict) -> dict:
    repo = LOCO / item["conv"] / "repo"
    if not repo.exists():
        return {"single": None, "multi": None, "cat": item["cat"]}
    vs = variants(item["q"])
    single = run_recall(repo, item["q"], limit, env)
    rankings = [single]
    for v in vs[1:]:
        rankings.append(run_recall(repo, v, limit, env))
    fused = rrf_fuse(rankings)
    return {
        "single": ev_rank(single, item["ev"]),
        "multi": ev_rank(fused, item["ev"]),
        "n_queries": len(rankings),
        "cat": item["cat"],
    }


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--limit", type=int, default=20)
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--set", choices=["miss", "all"], default="miss")
    args = ap.parse_args()

    env = dict(os.environ)
    env["PATH"] = f"{Path(REKAL).parent}{os.pathsep}{env.get('PATH', '')}"
    for k in ("REKAL_DISABLE_CAPTURE", "REKAL_NO_CAPTURE"):
        env.pop(k, None)

    items = load_miss() if args.set == "miss" else load_all()
    results = [None] * len(items)
    with ThreadPoolExecutor(max_workers=args.workers) as ex:
        for i, r in enumerate(ex.map(lambda it: eval_item(it, args.limit, env), items)):
            results[i] = r

    def hit(field, k):
        return sum(1 for r in results if r[field] is not None and r[field] < k)

    n = len(items)
    from collections import Counter
    recovered5 = [r for r in results if (r["single"] is None or r["single"] >= 5) and (r["multi"] is not None and r["multi"] < 5)]
    print(json.dumps({
        "set": args.set, "n": n, "limit": args.limit,
        "single_ev@5": round(hit("single", 5) / n, 4),
        "multi_ev@5": round(hit("multi", 5) / n, 4),
        "single_ev@10": round(hit("single", 10) / n, 4),
        "multi_ev@10": round(hit("multi", 10) / n, 4),
        "single_ev@20": round(hit("single", 20) / n, 4),
        "multi_ev@20": round(hit("multi", 20) / n, 4),
        "recovered_into_top5_by_multilookup": len(recovered5),
        "recovered_by_cat": dict(Counter(r["cat"] for r in recovered5)),
    }, indent=2))


if __name__ == "__main__":
    main()
