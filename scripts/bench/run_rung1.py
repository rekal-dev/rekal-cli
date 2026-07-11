#!/usr/bin/env python3
"""run_rung1.py <outdir> [--transcripts DIR] — run rung-1 systems.

Systems (docs/research/03-benchmark.md §3):
  b5  Rekal full hybrid (config as-is)
  b3  BM25-only  (weights {bm25:1, lsa:0, nomic:0})
  b4  neural-only (weights {bm25:0, lsa:0, nomic:1})
  b1  grep-rank over raw transcript JSONL (--transcripts DIR); the
      non-agentic DCI proxy: sessions ranked by rg term-hit counts.

Weight ablations rewrite .rekal/config.json in place and ALWAYS restore the
original (backup kept beside it until success). Query-time weights: no
reindex between systems. Writes run-<system>.jsonl: {qid, ranked, seconds}.
"""
import argparse
import collections
import json
import pathlib
import re
import shutil
import subprocess
import sys
import time

TOPK = 10


def rekal_search(query: str) -> list[str]:
    out = subprocess.run(
        ["rekal", "-n", str(TOPK), query], capture_output=True, text=True, timeout=120
    )
    if out.returncode != 0:
        return []
    try:
        payload = json.loads(out.stdout)
    except json.JSONDecodeError:
        return []
    ranked: list[str] = []
    for r in payload.get("results", []):
        ranked.append(r["session_id"])
        for child in r.get("children", []) or []:
            if child["session_id"] not in ranked:
                ranked.append(child["session_id"])
    return ranked[:TOPK]


def grep_rank(query: str, transcripts: pathlib.Path) -> list[str]:
    terms = [t for t in re.findall(r"[a-zA-Z0-9]+", query.lower()) if len(t) > 2][:12]
    scores: collections.Counter[str] = collections.Counter()
    for term in terms:
        out = subprocess.run(
            ["rg", "-ci", "--no-messages", term, str(transcripts)],
            capture_output=True, text=True, timeout=300,
        )
        for line in out.stdout.splitlines():
            path, _, count = line.rpartition(":")
            if not path.endswith(".jsonl"):
                continue
            sid = pathlib.Path(path).stem
            try:
                scores[sid] += int(count)
            except ValueError:
                pass
    return [sid for sid, _ in scores.most_common(TOPK)]


def run_system(name: str, queries: list[dict], search, outdir: pathlib.Path) -> None:
    rows = []
    for q in queries:
        t0 = time.time()
        ranked = search(q["query"])
        rows.append({"qid": q["qid"], "ranked": ranked, "seconds": round(time.time() - t0, 3)})
        print(f"[{name}] {len(rows)}/{len(queries)}", end="\r", file=sys.stderr)
    (outdir / f"run-{name}.jsonl").write_text("".join(json.dumps(r) + "\n" for r in rows))
    print(f"[{name}] done: {len(rows)} queries", file=sys.stderr)


class WeightOverride:
    """Temporarily set recall weights in .rekal/config.json; always restore."""

    def __init__(self, weights: dict | None):
        self.weights = weights
        self.path = pathlib.Path(".rekal/config.json")
        self.backup = self.path.with_suffix(".json.benchbak")

    def __enter__(self):
        if self.weights is None:
            return self
        original = {}
        if self.path.exists():
            shutil.copy2(self.path, self.backup)
            original = json.loads(self.path.read_text() or "{}")
        original["weights"] = self.weights
        self.path.parent.mkdir(exist_ok=True)
        self.path.write_text(json.dumps(original, indent=2))
        return self

    def __exit__(self, *exc):
        if self.weights is None:
            return
        if self.backup.exists():
            shutil.move(self.backup, self.path)
        else:
            self.path.unlink(missing_ok=True)


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("outdir", type=pathlib.Path)
    ap.add_argument("--transcripts", type=pathlib.Path, help="raw JSONL dir for B1 grep-rank")
    ap.add_argument("--systems", default="b5,b3,b4,b1")
    args = ap.parse_args()

    queries = [json.loads(l) for l in (args.outdir / "queries.jsonl").read_text().splitlines()]
    systems = args.systems.split(",")

    ablations = {
        "b5": None,
        "b3": {"bm25": 1.0, "lsa": 0.0, "nomic": 0.0},
        "b4": {"bm25": 0.0, "lsa": 0.0, "nomic": 1.0},
    }
    for name in systems:
        if name in ablations:
            with WeightOverride(ablations[name]):
                run_system(name, queries, rekal_search, args.outdir)
        elif name == "b1":
            if not args.transcripts:
                print("[b1] skipped: pass --transcripts DIR", file=sys.stderr)
                continue
            run_system(name, queries, lambda q: grep_rank(q, args.transcripts), args.outdir)
        else:
            sys.exit(f"unknown system {name!r}")


if __name__ == "__main__":
    main()
