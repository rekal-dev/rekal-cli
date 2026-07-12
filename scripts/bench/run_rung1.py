#!/usr/bin/env python3
"""run_rung1.py <outdir> [--transcripts DIR] — run rung-1 systems.

Systems (docs/research/03-benchmark.md §3):
  b5  Rekal full hybrid (config as-is)
  b3  BM25-only  (weights {bm25:1, lsa:0, nomic:0})
  b4  neural-only (weights {bm25:0, lsa:0, nomic:1})
  b1  grep-rank over raw transcript JSONL (--transcripts DIR); the
      non-agentic DCI proxy: sessions ranked by rg term-hit counts.
  b4x neural-only with a substituted embedding model (B4' in the paper)
  b5x full hybrid with a substituted embedding model (B5' in the paper)
      Both need --embedding-config FILE: the JSON for the config's
      "embedding" section (endpoint/model/api_key_env; see config.go).
      The swap is config-only but vectors live in the index, so each
      direction costs one `rekal index` — the (model, content_hash) embed
      cache makes the swap back re-embed nothing. Tests prediction P9.

Weight ablations rewrite .rekal/config.json in place and ALWAYS restore the
original (backup kept beside it until success). Query-time weights: no
reindex between systems. Writes run-<system>.jsonl: {qid, ranked, seconds}.

B1 id-space join: transcript filenames are the harness's session UUIDs while
gold labels are Rekal ULIDs, so b1 first builds sidmap.json — for each gold
session, a distinctive turn substring is matched into the transcripts dir
with rg -F to find its file. Ranked transcripts then translate to ULIDs
(unmapped ones stay in the ranking as competitors, so recall is not
inflated); subagent transcripts nested under a trunk's directory fold into
the trunk. Coverage is reported in sidmap-report.json — unmapped GOLD
sessions deflate b1, so read its metrics as a lower bound at <100% coverage.
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


def rekal_sql(sql: str) -> list[dict]:
    out = subprocess.run(
        ["rekal", "query", "--index", sql], capture_output=True, text=True, timeout=120
    )
    if out.returncode != 0:
        return []
    rows = []
    for line in out.stdout.splitlines():
        try:
            rows.append(json.loads(line))
        except json.JSONDecodeError:
            continue
    return rows


def transcript_id(path: str, root: pathlib.Path) -> str | None:
    """Trunk transcript id for a hit: the stem of a top-level <uuid>.jsonl,
    or the first-level directory name for subagent files nested under it."""
    try:
        rel = pathlib.Path(path).resolve().relative_to(root.resolve())
    except ValueError:
        return None
    return rel.stem if len(rel.parts) == 1 else rel.parts[0]


# Substrings that survive both scrubbing and JSON string encoding verbatim:
# no quotes/backslashes/newlines/non-ASCII, and no "/" (scrub anonymizes
# file paths before insert, so pathy text differs between index and raw).
SAFE_RUN = re.compile(r"[A-Za-z0-9 ,.;:()\[\]{}=+*_<>'!?%&#@^~|-]{40,}")


def build_sidmap(
    gold_sids: set[str], transcripts: pathlib.Path, outdir: pathlib.Path
) -> dict[str, list[str]]:
    """Map transcript id (filename UUID) -> Rekal session ULIDs, for the gold
    sessions. A trunk and its subagent sessions share one transcript id, so
    the value is a list. Cached in sidmap.json; coverage in sidmap-report.json.
    """
    cache = outdir / "sidmap.json"
    if cache.exists():
        return json.loads(cache.read_text())
    sidmap: dict[str, list[str]] = {}
    unmapped: list[str] = []
    for i, sid in enumerate(sorted(gold_sids)):
        print(f"[sidmap] {i + 1}/{len(gold_sids)}", end="\r", file=sys.stderr)
        rows = rekal_sql(
            f"SELECT content FROM turns_ft WHERE session_id = '{sid}' "
            "AND role IN ('human','human_steering','assistant') "
            "ORDER BY length(content) DESC LIMIT 6"
        )
        needles = []
        for row in rows:
            for m in SAFE_RUN.finditer(row.get("content", "")):
                needles.append(m.group(0).strip()[:80])
        hit = None
        for needle in needles[:8]:
            out = subprocess.run(
                ["rg", "-lF", "--no-messages", "--", needle, str(transcripts)],
                capture_output=True, text=True, timeout=300,
            )
            ids = {
                t
                for t in (transcript_id(p, transcripts) for p in out.stdout.splitlines())
                if t
            }
            if len(ids) == 1:  # unique match; boilerplate hits many files and is skipped
                hit = ids.pop()
                break
        if hit:
            sidmap.setdefault(hit, []).append(sid)
        else:
            unmapped.append(sid)
    cache.write_text(json.dumps(sidmap, indent=1))
    coverage = 1 - len(unmapped) / max(len(gold_sids), 1)
    (outdir / "sidmap-report.json").write_text(
        json.dumps({"gold_sessions": len(gold_sids), "mapped": len(gold_sids) - len(unmapped),
                    "coverage": round(coverage, 3), "unmapped": unmapped}, indent=1)
    )
    print(f"[sidmap] coverage {coverage:.1%} ({len(unmapped)} gold sessions unmapped; "
          "b1 metrics are a lower bound below 100%)", file=sys.stderr)
    return sidmap


def grep_rank(
    query: str, transcripts: pathlib.Path, sidmap: dict[str, list[str]]
) -> list[str]:
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
            tid = transcript_id(path, transcripts)
            if not tid:
                continue
            try:
                scores[tid] += int(count)
            except ValueError:
                pass
    ranked: list[str] = []
    for tid, _ in scores.most_common():
        # Translate to Rekal ULIDs; unmapped transcripts stay in the ranking
        # under their own id so they still displace gold as competitors.
        for sid in sidmap.get(tid, [tid]):
            if sid not in ranked:
                ranked.append(sid)
        if len(ranked) >= TOPK:
            break
    return ranked[:TOPK]


def run_system(name: str, queries: list[dict], search, outdir: pathlib.Path) -> None:
    rows = []
    for q in queries:
        t0 = time.time()
        ranked = search(q["query"])
        rows.append({"qid": q["qid"], "ranked": ranked, "seconds": round(time.time() - t0, 3)})
        print(f"[{name}] {len(rows)}/{len(queries)}", end="\r", file=sys.stderr)
    (outdir / f"run-{name}.jsonl").write_text("".join(json.dumps(r) + "\n" for r in rows))
    print(f"[{name}] done: {len(rows)} queries", file=sys.stderr)


class EmbeddingOverride:
    """Temporarily set the embedding backend in .rekal/config.json and
    reindex; always restore the config and reindex back. Distinct backup
    suffix so it nests outside WeightOverride."""

    def __init__(self, embedding: dict | None):
        self.embedding = embedding
        self.path = pathlib.Path(".rekal/config.json")
        self.backup = self.path.with_suffix(".json.embedbak")

    def _reindex(self) -> None:
        print("[embedding] rekal index (regenerating vectors)...", file=sys.stderr)
        subprocess.run(["rekal", "index"], check=True, timeout=7200)

    def __enter__(self):
        if self.embedding is None:
            return self
        original = {}
        if self.path.exists():
            shutil.copy2(self.path, self.backup)
            original = json.loads(self.path.read_text() or "{}")
        original["embedding"] = self.embedding
        self.path.parent.mkdir(exist_ok=True)
        self.path.write_text(json.dumps(original, indent=2))
        self._reindex()
        return self

    def __exit__(self, *exc):
        if self.embedding is None:
            return
        if self.backup.exists():
            shutil.move(self.backup, self.path)
        else:
            self.path.unlink(missing_ok=True)
        self._reindex()


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
    ap.add_argument("--embedding-config", type=pathlib.Path,
                    help='JSON for config\'s "embedding" section; enables b4x/b5x (P9)')
    args = ap.parse_args()

    queries = [json.loads(l) for l in (args.outdir / "queries.jsonl").read_text().splitlines()]
    systems = args.systems.split(",")

    ablations = {
        "b5": None,
        "b3": {"bm25": 1.0, "lsa": 0.0, "nomic": 0.0},
        "b4": {"bm25": 0.0, "lsa": 0.0, "nomic": 1.0},
    }
    emb_systems = [s for s in systems if s in ("b4x", "b5x")]
    for name in systems:
        if name in ablations:
            with WeightOverride(ablations[name]):
                run_system(name, queries, rekal_search, args.outdir)
        elif name == "b1":
            if not args.transcripts:
                print("[b1] skipped: pass --transcripts DIR", file=sys.stderr)
                continue
            gold_sids = {sid for q in queries for sid in q["gold"]}
            sidmap = build_sidmap(gold_sids, args.transcripts, args.outdir)
            run_system(
                name, queries, lambda q: grep_rank(q, args.transcripts, sidmap), args.outdir
            )
        elif name in emb_systems:
            continue  # grouped below: one embedding swap covers both
        else:
            sys.exit(f"unknown system {name!r}")

    if emb_systems:
        if not args.embedding_config:
            print("[b4x/b5x] skipped: pass --embedding-config FILE", file=sys.stderr)
            return
        embedding = json.loads(args.embedding_config.read_text())
        with EmbeddingOverride(embedding):
            for name in emb_systems:
                weights = ablations["b4"] if name == "b4x" else None
                with WeightOverride(weights):
                    run_system(name, queries, rekal_search, args.outdir)


if __name__ == "__main__":
    main()
