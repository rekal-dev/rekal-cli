#!/usr/bin/env python3
"""run_rung2.py <outdir> — judged answer quality (rung 2).

For a stratified sample of queries, each system assembles context, an ANSWER
model answers from it, and a JUDGE model grades the answer against the gold
turn (bias controls: distinct answer/judge models, and the judge sees the
reference — 03-benchmark.md §4). Reports judged accuracy and context tokens
per system.

Systems (automated here): b0 no-memory, b1 grep over transcripts, b5 Rekal
hybrid recall+drill. B6 (skills-driven, agentic) is the agent-in-the-loop
rung and is run manually (06-eval-strategy.md §4b), not here.

Env: BENCH_ANSWER_LLM and BENCH_JUDGE_LLM — stdin->stdout commands, and
distinct from each other and from BENCH_LLM (query generation). Writes
rung2.jsonl (per query×system) and rung2.md (the paper's judged table).
Stdlib only.
"""
import argparse
import collections
import json
import os
import pathlib
import re
import subprocess
import sys

from score import fmt  # mean [95% CI]

HERE = pathlib.Path(__file__).parent
ANSWER_PROMPT = (HERE / "prompts" / "answer.txt").read_text()
JUDGE_PROMPT = (HERE / "prompts" / "judge.txt").read_text()
CTX_CHARS = 6000              # ~1500 tokens of assembled context
GRADE = {"CORRECT": 1.0, "PARTIAL": 0.5, "WRONG": 0.0}


def llm(env_var: str, prompt: str) -> str:
    cmd = os.environ.get(env_var)
    if not cmd:
        sys.exit(f"set {env_var} to a stdin->stdout model command")
    out = subprocess.run(cmd, shell=True, input=prompt,
                         capture_output=True, text=True, timeout=180)
    return out.stdout.strip()


def sql(query: str) -> list[dict]:
    out = subprocess.run(["rekal", "query", query],
                         capture_output=True, text=True, timeout=120)
    return [json.loads(l) for l in out.stdout.splitlines() if l.strip()]


def recall_full(query: str, k: int) -> list[dict]:
    out = subprocess.run(["rekal", "-n", str(k), query],
                         capture_output=True, text=True, timeout=120)
    try:
        return json.loads(out.stdout).get("results", [])[:k]
    except json.JSONDecodeError:
        return []


def window(sid: str, ti: int, radius: int = 3) -> str:
    rows = sql(f"SELECT content FROM turns WHERE session_id = '{sid}' "
               f"AND turn_index BETWEEN {ti - radius} AND {ti + radius} "
               "ORDER BY turn_index")
    return "\n".join(r.get("content", "") for r in rows)


def reference(gold: list[str], turn_index) -> str:
    """Ground truth the judge grades against: the gold turn's neighborhood
    (T2) or the gold session's decision-bearing turns."""
    if turn_index is not None and gold:
        return window(gold[0], int(turn_index))[:CTX_CHARS]
    sids = ",".join(f"'{s}'" for s in gold)
    rows = sql(f"SELECT content FROM turns WHERE session_id IN ({sids}) "
               "AND role IN ('human_steering','summary','human') "
               "ORDER BY length(content) DESC LIMIT 6")
    return "\n".join(r.get("content", "") for r in rows)[:CTX_CHARS]


def context_b5(query: str, k: int) -> str:
    parts = []
    for r in recall_full(query, k):
        sid = r.get("session_id")
        ti = r.get("snippet_turn_index")
        if sid and ti is not None:
            parts.append(window(sid, int(ti)))
        elif r.get("snippet"):
            parts.append(r["snippet"])
    return "\n---\n".join(parts)[:CTX_CHARS]


def context_b1(query: str, transcripts: pathlib.Path) -> str:
    terms = [t for t in re.findall(r"[a-zA-Z0-9]+", query.lower()) if len(t) > 3][:8]
    if not terms:
        return ""
    out = subprocess.run(
        ["rg", "-i", "--no-filename", "--no-line-number", "-m", "3",
         "|".join(terms), str(transcripts)],
        capture_output=True, text=True, timeout=120)
    return out.stdout[:CTX_CHARS]


def stratified(queries: list[dict], n: int) -> list[dict]:
    by_task = collections.defaultdict(list)
    for q in queries:
        by_task[q.get("task", "?")].append(q)
    per = max(1, n // max(len(by_task), 1))
    out = []
    for qs in by_task.values():
        out.extend(qs[:per])
    return out[:n]


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("outdir", type=pathlib.Path)
    ap.add_argument("--systems", default="b0,b1,b5")
    ap.add_argument("--sample", type=int, default=150)
    ap.add_argument("--k", type=int, default=1, help="top sessions for b5 context")
    ap.add_argument("--transcripts", type=pathlib.Path, help="raw JSONL dir for b1")
    args = ap.parse_args()

    queries = [json.loads(l) for l in
               (args.outdir / "queries.jsonl").read_text().splitlines()
               if l.strip()]
    sample = stratified([q for q in queries if q.get("split") != "dev"], args.sample)
    systems = args.systems.split(",")

    rows = []
    for i, q in enumerate(sample):
        ref = reference(q["gold"], q.get("turn_index"))
        for sysname in systems:
            if sysname == "b0":
                ctx = ""
            elif sysname == "b1":
                ctx = context_b1(q["query"], args.transcripts) if args.transcripts else ""
            elif sysname == "b5":
                ctx = context_b5(q["query"], args.k)
            else:
                sys.exit(f"unknown system {sysname!r}")
            answer = llm("BENCH_ANSWER_LLM",
                         ANSWER_PROMPT.format(query=q["query"], context=ctx or "(none)"))
            verdict = llm("BENCH_JUDGE_LLM", JUDGE_PROMPT.format(
                query=q["query"], reference=ref, answer=answer)).upper()
            grade = next((GRADE[g] for g in GRADE if g in verdict), 0.0)
            rows.append({"qid": q["qid"], "task": q.get("task"), "system": sysname,
                        "grade": grade, "ctx_tokens": len(ctx) // 4})
        print(f"[rung2] {i + 1}/{len(sample)}", end="\r", file=sys.stderr)

    (args.outdir / "rung2.jsonl").write_text(
        "".join(json.dumps(r) + "\n" for r in rows))

    lines = ["## Rung 2 — judged answer quality (test split; mean [95% CI])",
             "", "| system | n | judged acc | mean ctx tokens |",
             "|---|---|---|---|"]
    for sysname in systems:
        srows = [r for r in rows if r["system"] == sysname]
        if not srows:
            continue
        acc = fmt([r["grade"] for r in srows])
        toks = round(sum(r["ctx_tokens"] for r in srows) / len(srows))
        lines.append(f"| {sysname} | {len(srows)} | {acc} | {toks} |")
    (args.outdir / "rung2.md").write_text("\n".join(lines) + "\n")
    print("\n" + "\n".join(lines))


if __name__ == "__main__":
    main()
