#!/usr/bin/env python3
"""tune_weights.py <outdir> [--step 0.1] — dev-tune the layer mix, then
validate the winner on TEST against the incumbent.

RHO discipline (docs/research/03-benchmark.md §5): the grid search sees the
DEV split only. The single dev winner and the incumbent (whatever
.rekal/config.json currently resolves to, or the built-in defaults) are then
both run on TEST, and the verdict is SHIP only if the winner beats the
incumbent there — measured against the CURRENT index state, so a re-tag or
reindex invalidates old sweeps (delete tune-results.jsonl to restart).

Grid: all (bm25, lsa, nomic) on the simplex with the given step. Boost knobs
(steering/summary/subagent) are left at config/default values — this tunes
the layer mix only. Each combo costs one dev-query sweep (query-time weights,
no reindex). Progress is appended to tune-results.jsonl, so an interrupted
sweep resumes where it left off.

Outputs: tune-results.jsonl (one line per combo), tune-verdict.md (the
dev table, the test comparison with a paired-bootstrap CI on the MRR
difference, and the SHIP/REJECT line).
"""
import argparse
import json
import pathlib
import random
import sys

from run_rung1 import WeightOverride, rekal_search
from score import metrics_for


def mrrs(queries: list[dict]) -> list[float]:
    out = []
    for i, q in enumerate(queries):
        out.append(metrics_for(rekal_search(q["query"]), set(q["gold"]))["mrr"])
        print(f"  {i + 1}/{len(queries)}", end="\r", file=sys.stderr)
    return out


def grid(step: float) -> list[dict]:
    n = round(1 / step)
    combos = []
    for b in range(n + 1):
        for l in range(n + 1 - b):
            combos.append(
                {"bm25": round(b * step, 2), "lsa": round(l * step, 2),
                 "nomic": round(1 - (b + l) * step, 2)}
            )
    return combos


def paired_bootstrap_diff(a: list[float], b: list[float], n: int = 1000) -> tuple[float, float]:
    """95% CI on mean(a - b), resampling query indices (paired)."""
    diffs = [x - y for x, y in zip(a, b)]
    rng = random.Random(42)
    means = sorted(
        sum(rng.choices(diffs, k=len(diffs))) / len(diffs) for _ in range(n)
    )
    return means[int(0.025 * n)], means[int(0.975 * n)]


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("outdir", type=pathlib.Path)
    ap.add_argument("--step", type=float, default=0.1)
    args = ap.parse_args()

    queries = [json.loads(l) for l in (args.outdir / "queries.jsonl").read_text().splitlines()]
    dev = [q for q in queries if q.get("split") == "dev"]
    test = [q for q in queries if q.get("split") != "dev"]
    if not dev:
        sys.exit("no dev split in queries.jsonl (gen_queries.py tags 10% as dev)")
    print(f"dev n={len(dev)} (small: dev ranking is noisy — the TEST gate is "
          f"what decides), test n={len(test)}", file=sys.stderr)

    results_path = args.outdir / "tune-results.jsonl"
    done: dict[str, dict] = {}
    if results_path.exists():
        for line in results_path.read_text().splitlines():
            row = json.loads(line)
            done[json.dumps(row["weights"], sort_keys=True)] = row

    combos = grid(args.step)
    for i, w in enumerate(combos):
        key = json.dumps(w, sort_keys=True)
        if key in done:
            continue
        print(f"[dev {i + 1}/{len(combos)}] {w}", file=sys.stderr)
        with WeightOverride(w):
            vals = mrrs(dev)
        row = {"weights": w, "dev_mrr": round(sum(vals) / len(vals), 4), "n": len(vals)}
        done[key] = row
        with results_path.open("a") as f:
            f.write(json.dumps(row) + "\n")

    ranked = sorted(done.values(), key=lambda r: -r["dev_mrr"])
    winner = ranked[0]["weights"]
    print(f"\ndev winner: {winner} (dev MRR {ranked[0]['dev_mrr']})", file=sys.stderr)

    print("[test] incumbent (config as-is)", file=sys.stderr)
    inc_vals = mrrs(test)
    print("[test] dev winner", file=sys.stderr)
    with WeightOverride(winner):
        win_vals = mrrs(test)

    inc_mrr = sum(inc_vals) / len(inc_vals)
    win_mrr = sum(win_vals) / len(win_vals)
    lo, hi = paired_bootstrap_diff(win_vals, inc_vals)
    ship = win_mrr > inc_mrr
    lines = [
        "# Weight tuning (dev-tuned, test-validated)",
        "",
        f"Dev split n={len(dev)}, test n={len(test)}, grid step {args.step}.",
        "",
        "Top 5 on dev:",
        "",
        "| bm25 | lsa | nomic | dev MRR |",
        "|---|---|---|---|",
        *(f"| {r['weights']['bm25']} | {r['weights']['lsa']} | {r['weights']['nomic']} "
          f"| {r['dev_mrr']} |" for r in ranked[:5]),
        "",
        "Test validation (winner vs incumbent):",
        "",
        "| system | test MRR |",
        "|---|---|",
        f"| incumbent (config as-is) | {inc_mrr:.4f} |",
        f"| dev winner {winner} | {win_mrr:.4f} |",
        "",
        f"Paired bootstrap 95% CI on MRR difference (winner − incumbent): "
        f"[{lo:.4f}, {hi:.4f}]",
        "",
        f"**Verdict: {'SHIP' if ship else 'REJECT'}**"
        + ("" if ship else " — incumbent stands (RHO rule: no adoption without a test-split win)")
        + (" — note: CI includes 0; a win this small may not replicate" if ship and lo <= 0 else ""),
    ]
    (args.outdir / "tune-verdict.md").write_text("\n".join(lines) + "\n")
    print("\n".join(lines))


if __name__ == "__main__":
    main()
