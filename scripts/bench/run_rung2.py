#!/usr/bin/env python3
"""run_rung2.py <outdir> — judged answer quality (rung 2).

For a stratified sample of queries, each system assembles context, an ANSWER
model answers from it, and a JUDGE model grades the answer against the gold
turn (bias controls: distinct answer/judge models, and the judge sees the
reference — 03-benchmark.md §4). Reports judged accuracy, per task and pooled,
with the CORRECT/PARTIAL/WRONG breakdown.

Answer context is deliberately generous (tokens are not the metric here — that
is rung 3): the question is whether the system surfaced the answer at all, not
how few tokens it used. Systems (automated): b0 no-memory, b1 grep over
transcripts, b5 Rekal hybrid recall+drill. B6 (skills-driven, agentic) is the
agent-in-the-loop rung, run manually (06-eval-strategy.md §4b).

Env: BENCH_ANSWER_LLM and BENCH_JUDGE_LLM — stdin->stdout commands, distinct
from each other and from BENCH_LLM (query generation). Writes rung2.jsonl and
rung2.md. Stdlib only.
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
REF_CHARS = 4000                 # gold reference the judge grades against
GRADE = {"CORRECT": 1.0, "PARTIAL": 0.5, "WRONG": 0.0}


def llm(env_var: str, prompt: str) -> str:
    cmd = os.environ.get(env_var)
    if not cmd:
        sys.exit(f"set {env_var} to a stdin->stdout model command")
    out = subprocess.run(cmd, shell=True, input=prompt,
                         capture_output=True, text=True, timeout=240)
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


def window(sid: str, ti: int, radius: int) -> str:
    rows = sql(f"SELECT content FROM turns WHERE session_id = '{sid}' "
               f"AND turn_index BETWEEN {ti - radius} AND {ti + radius} "
               "ORDER BY turn_index")
    return "\n".join(r.get("content", "") for r in rows)


def reference(gold: list[str], turn_index) -> str:
    """Ground truth the judge grades against: the gold turn's neighborhood
    (T2) or the gold session's decision-bearing turns."""
    if turn_index is not None and gold:
        return window(gold[0], int(turn_index), 3)[:REF_CHARS]
    sids = ",".join(f"'{s}'" for s in gold)
    rows = sql(f"SELECT content FROM turns WHERE session_id IN ({sids}) "
               "AND role IN ('human_steering','summary','human') "
               "ORDER BY length(content) DESC LIMIT 6")
    return "\n".join(r.get("content", "") for r in rows)[:REF_CHARS]


def context_b5(query: str, k: int, radius: int, cap: int) -> str:
    parts = []
    for r in recall_full(query, k):
        sid, ti = r.get("session_id"), r.get("snippet_turn_index")
        if sid and ti is not None:
            parts.append(window(sid, int(ti), radius))
        elif r.get("snippet"):
            parts.append(r["snippet"])
    return "\n---\n".join(parts)[:cap]


def context_b1(query: str, transcripts: pathlib.Path, cap: int) -> str:
    terms = [t for t in re.findall(r"[a-zA-Z0-9]+", query.lower()) if len(t) > 3][:8]
    if not terms:
        return ""
    out = subprocess.run(
        ["rg", "-i", "--no-filename", "--no-line-number", "-m", "6",
         "|".join(terms), str(transcripts)],
        capture_output=True, text=True, timeout=120)
    return out.stdout[:cap]


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
    ap.add_argument("--sample", type=int, default=200)
    ap.add_argument("--k", type=int, default=3, help="top sessions for b5 context")
    ap.add_argument("--radius", type=int, default=5, help="turns each side of the match")
    ap.add_argument("--ctx-chars", type=int, default=24000,
                    help="answer-context budget (generous by design; ~6k tokens)")
    ap.add_argument("--transcripts", type=pathlib.Path, help="raw JSONL dir for b1")
    args = ap.parse_args()

    queries = [json.loads(l) for l in
               (args.outdir / "queries.jsonl").read_text().splitlines() if l.strip()]
    sample = stratified([q for q in queries if q.get("split") != "dev"], args.sample)
    systems = args.systems.split(",")

    rows = []
    for i, q in enumerate(sample):
        ref = reference(q["gold"], q.get("turn_index"))
        for sysname in systems:
            if sysname == "b0":
                ctx = ""
            elif sysname == "b1":
                ctx = context_b1(q["query"], args.transcripts, args.ctx_chars) if args.transcripts else ""
            elif sysname == "b5":
                ctx = context_b5(q["query"], args.k, args.radius, args.ctx_chars)
            else:
                sys.exit(f"unknown system {sysname!r}")
            answer = llm("BENCH_ANSWER_LLM",
                         ANSWER_PROMPT.format(query=q["query"], context=ctx or "(none)"))
            verdict = llm("BENCH_JUDGE_LLM", JUDGE_PROMPT.format(
                query=q["query"], reference=ref, answer=answer)).upper()
            label = next((g for g in GRADE if g in verdict), "WRONG")
            rows.append({"qid": q["qid"], "task": q.get("task"), "system": sysname,
                        "grade": GRADE[label], "verdict": label,
                        "ctx_tokens": len(ctx) // 4})
        print(f"[rung2] {i + 1}/{len(sample)}", end="\r", file=sys.stderr)

    (args.outdir / "rung2.jsonl").write_text(
        "".join(json.dumps(r) + "\n" for r in rows))

    lines = ["## Rung 2 — judged answer quality (test split; mean [95% CI])", ""]
    for sysname in systems:
        srows = [r for r in rows if r["system"] == sysname]
        if not srows:
            continue
        dist = collections.Counter(r["verdict"] for r in srows)
        lines += [
            f"### {sysname}  "
            f"(CORRECT {dist['CORRECT']} / PARTIAL {dist['PARTIAL']} / WRONG {dist['WRONG']}; "
            f"mean ctx {round(sum(r['ctx_tokens'] for r in srows) / len(srows))} tok)",
            "| task | n | judged acc |", "|---|---|---|"]
        by_task: dict[str, list[dict]] = collections.defaultdict(list)
        for r in srows:
            by_task[r["task"]].append(r)
        for task in sorted(by_task):
            ms = by_task[task]
            lines.append(f"| {task} | {len(ms)} | {fmt([m['grade'] for m in ms])} |")
        lines.append(f"| **all** | {len(srows)} | {fmt([r['grade'] for r in srows])} |")
        lines.append("")
    (args.outdir / "rung2.md").write_text("\n".join(lines) + "\n")
    print("\n".join(lines))


if __name__ == "__main__":
    main()
