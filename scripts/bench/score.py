#!/usr/bin/env python3
"""score.py <outdir> — score run-*.jsonl against queries.jsonl gold labels.

Metrics per docs/research/03-benchmark.md §4: MRR, Recall@{1,5,10}, nDCG@10,
with 95% bootstrap CIs (1000 resamples over queries). Reports the test split
by task and pooled, as a markdown table ready for the paper's Table 3.
Stdlib only.
"""
import json
import math
import pathlib
import random
import sys


def metrics_for(ranked: list[str], gold: set[str]) -> dict:
    first = next((i for i, sid in enumerate(ranked) if sid in gold), None)
    mrr = 1.0 / (first + 1) if first is not None else 0.0
    rec = {k: (1.0 if any(s in gold for s in ranked[:k]) else 0.0) for k in (1, 5, 10)}
    dcg = sum(1.0 / math.log2(i + 2) for i, sid in enumerate(ranked[:10]) if sid in gold)
    ideal = sum(1.0 / math.log2(i + 2) for i in range(min(len(gold), 10)))
    ndcg = dcg / ideal if ideal > 0 else 0.0
    return {"mrr": mrr, "r@1": rec[1], "r@5": rec[5], "r@10": rec[10], "ndcg@10": ndcg}


def bootstrap_ci(values: list[float], n: int = 1000) -> tuple[float, float]:
    if not values:
        return (0.0, 0.0)
    rng = random.Random(42)
    means = sorted(
        sum(rng.choices(values, k=len(values))) / len(values) for _ in range(n)
    )
    return means[int(0.025 * n)], means[int(0.975 * n)]


def fmt(values: list[float]) -> str:
    if not values:
        return "—"
    mean = sum(values) / len(values)
    lo, hi = bootstrap_ci(values)
    return f"{mean:.3f} [{lo:.3f},{hi:.3f}]"


def main() -> None:
    outdir = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".")
    queries = {
        q["qid"]: q
        for q in map(json.loads, (outdir / "queries.jsonl").read_text().splitlines())
    }

    for run_path in sorted(outdir.glob("run-*.jsonl")):
        system = run_path.stem.removeprefix("run-")
        per_task: dict[str, list[dict]] = {}
        seconds: list[float] = []
        for line in run_path.read_text().splitlines():
            row = json.loads(line)
            q = queries.get(row["qid"])
            if not q or q.get("split") == "dev":
                continue
            per_task.setdefault(q["task"], []).append(
                metrics_for(row["ranked"], set(q["gold"]))
            )
            seconds.append(row.get("seconds", 0.0))

        print(f"\n## {system}  (test split; mean [95% CI]; "
              f"median latency {sorted(seconds)[len(seconds)//2] if seconds else 0:.2f}s)")
        print("| task | n | MRR | R@1 | R@5 | R@10 | nDCG@10 |")
        print("|---|---|---|---|---|---|---|")
        pooled: list[dict] = []
        for task in sorted(per_task):
            ms = per_task[task]
            pooled.extend(ms)
            cols = [fmt([m[k] for m in ms]) for k in ("mrr", "r@1", "r@5", "r@10", "ndcg@10")]
            print(f"| {task} | {len(ms)} | " + " | ".join(cols) + " |")
        if pooled:
            cols = [fmt([m[k] for m in pooled]) for k in ("mrr", "r@1", "r@5", "r@10", "ndcg@10")]
            print(f"| **all** | {len(pooled)} | " + " | ".join(cols) + " |")


if __name__ == "__main__":
    main()
