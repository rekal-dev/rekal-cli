#!/usr/bin/env python3
"""run_rung3.py <outdir> — tokens-to-context proxy for rung 3.

Compares two drill strategies an agent can use after recall lands on the
gold session (docs/research/03-benchmark.md §2, rung 3):

  window   raw drill — a WINDOW-turn slice around snippet_turn_index
           (rekal query --session <id> --offset <i-2> --limit 5)
  summary  summary-first — exactly the turn summary_turn_index points at
           (rekal query --session <id> --offset <idx> --limit 1); the
           per-result pointer makes this a single precise fetch

This is a cost/coverage PROXY, not LLM-judged answer quality (that is rung
2): tokens = bytes/4 of the drill output an agent would ingest; coverage =
fraction of the gold label's distinctive content words present in the
drilled text. The comparison is paired — only queries whose gold session
appears in recall's top results, and the summary column only where a
summary exists. Writes rung3.jsonl and prints a markdown table. Stdlib only;
all content stays local.
"""
import hashlib
import json
import pathlib
import re
import subprocess
import sys

TOPK = 10
WINDOW = 5
STOPWORDS = frozenset(
    "the and for with that this from have has was were are but not you your "
    "our can will would should into over under when what how why where "
    "them then than they there here also been being does did just like "
    "some more most much many each other same only very".split()
)


def run_rekal(args: list[str]) -> str:
    out = subprocess.run(["rekal", *args], capture_output=True, text=True, timeout=120)
    return out.stdout if out.returncode == 0 else ""


def recall(query: str) -> list[dict]:
    raw = run_rekal(["-n", str(TOPK), query])
    try:
        results = json.loads(raw).get("results", [])
    except json.JSONDecodeError:
        return []
    flat: list[dict] = []
    for r in results:
        flat.append(r)
        flat.extend(r.get("children", []) or [])
    return flat[:TOPK]


def drill(sid: str, offset: int, limit: int) -> str:
    return run_rekal(
        ["query", "--session", sid, "--offset", str(max(0, offset)), "--limit", str(limit)]
    )


def tokens(text: str) -> int:
    return (len(text) + 3) // 4


def content_words(text: str) -> set[str]:
    return {
        w for w in re.findall(r"[a-z0-9]+", text.lower())
        if len(w) > 3 and w not in STOPWORDS
    }


def coverage(gold_text: str, drilled: str) -> float:
    gold = content_words(gold_text)
    if not gold:
        return 0.0
    return len(gold & content_words(drilled)) / len(gold)


def load_gold_sources(outdir: pathlib.Path) -> dict[str, str]:
    """Rebuild qid -> gold source text from labels (gen_queries.py's qid)."""
    sources: dict[str, str] = {}
    for task in ("t1", "t2", "t3"):
        path = outdir / f"labels-{task}.jsonl"
        if not path.exists():
            continue
        for line in path.read_text().splitlines():
            label = json.loads(line)
            qid = hashlib.sha1((task + json.dumps(label.get("gold"))).encode()).hexdigest()[:12]
            sources[qid] = " ".join(str(v) for v in label.get("source", {}).values())
    return sources


def main() -> None:
    outdir = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".")
    queries = [json.loads(l) for l in (outdir / "queries.jsonl").read_text().splitlines()]
    sources = load_gold_sources(outdir)

    rows, skipped = [], 0
    for q in queries:
        gold_text = sources.get(q["qid"], "")
        if not gold_text:
            skipped += 1
            continue
        hit = next((r for r in recall(q["query"]) if r["session_id"] in set(q["gold"])), None)
        if hit is None:
            skipped += 1  # retrieval miss — drill comparison is undefined
            continue

        sid = hit["session_id"]
        win = drill(sid, hit.get("snippet_turn_index", 0) - WINDOW // 2, WINDOW)
        row = {
            "qid": q["qid"], "task": q["task"], "session": sid,
            "window_tokens": tokens(win), "window_coverage": round(coverage(gold_text, win), 4),
        }
        if "summary_turn_index" in hit:
            summ = drill(sid, hit["summary_turn_index"], 1)
            row["summary_tokens"] = tokens(summ)
            row["summary_coverage"] = round(coverage(gold_text, summ), 4)
        rows.append(row)
        print(f"[rung3] {len(rows)} scored / {skipped} skipped", end="\r", file=sys.stderr)

    (outdir / "rung3.jsonl").write_text("".join(json.dumps(r) + "\n" for r in rows))
    print(f"\n[rung3] wrote {len(rows)} rows ({skipped} skipped)", file=sys.stderr)

    def table(rows: list[dict], title: str) -> None:
        if not rows:
            return
        paired = [r for r in rows if "summary_tokens" in r]
        mean = lambda xs: sum(xs) / len(xs) if xs else 0.0
        print(f"\n### {title} (n={len(rows)}, with-summary={len(paired)})\n")
        print("| strategy | mean tokens | mean coverage | coverage / 1k tokens |")
        print("|---|---|---|---|")
        for name, tk, cv, subset in (
            ("window (all)", "window_tokens", "window_coverage", rows),
            ("window (paired)", "window_tokens", "window_coverage", paired),
            ("summary (paired)", "summary_tokens", "summary_coverage", paired),
        ):
            mt, mc = mean([r[tk] for r in subset]), mean([r[cv] for r in subset])
            eff = (mc / mt * 1000) if mt else 0.0
            print(f"| {name} | {mt:.0f} | {mc:.3f} | {eff:.3f} |")

    for task in ("t1", "t2", "t3"):
        table([r for r in rows if r["task"] == task], f"rung 3 — {task}")
    table(rows, "rung 3 — pooled")


if __name__ == "__main__":
    main()
