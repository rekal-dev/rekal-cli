#!/usr/bin/env python3
"""gen_queries.py <outdir> — turn mined labels into natural-language queries.

For each label, asks $BENCH_LLM (any command reading a prompt on stdin and
printing one paraphrased question on stdout, e.g. `claude -p`) to phrase the
developer question, then enforces the leakage control from
docs/research/03-benchmark.md: 4-gram Jaccard overlap between query and
source material must be <= 0.30 (one aggressive-paraphrase retry, else the
label is skipped and logged). Tags a deterministic 10% dev split.

Writes queries.jsonl: {qid, task, split, query, gold, turn_index?}.
Stdlib only; all content stays local.
"""
import hashlib
import json
import os
import pathlib
import re
import subprocess
import sys

MAX_JACCARD = 0.30

PROMPTS = {
    "t1": (
        "A developer is about to ask their project-memory tool how a past "
        "change was made. The change: commit subject {subject!r}, touching "
        "files {files}. Write ONE natural question they would ask (about the "
        "how/why of that change), without quoting the subject verbatim. "
        "Output only the question."
    ),
    "t2": (
        "During an AI coding session a human corrected the agent with this "
        "steering message:\n---\n{content}\n---\nWrite ONE natural question a "
        "developer would later ask to rediscover that decision, WITHOUT "
        "reusing its distinctive phrases. Output only the question."
    ),
    "t3": (
        "A team once explored an approach with this intent (from an "
        "abandoned branch):\n---\n{intent}\n---\nWrite ONE natural question "
        'of the form "have we tried / should we do X?" that this history '
        "answers, in fresh wording. Output only the question."
    ),
    "t4": (
        "Two past AI coding sessions in this repo touched overlapping files.\n"
        "Session A raised:\n---\n{s1_snippet}\n---\nSession B raised:\n---\n"
        "{s2_snippet}\n---\nShared files: {files}.\nWrite ONE natural developer "
        "question whose answer genuinely requires facts from BOTH sessions — "
        "not answerable from either one alone — in fresh wording that quotes "
        "neither. Output only the question."
    ),
}

# T4 multi-hop validation: a question only qualifies if it needs both halves.
VALIDATE_T4 = (
    "Question: {q}\n\nContext A:\n{a}\n\nContext B:\n{b}\n\nCan this question be "
    "fully answered using ONLY Context A, or using ONLY Context B? Reply with "
    "exactly one token: A (A alone suffices), B (B alone suffices), or BOTH "
    "(needs both)."
)


def ngrams(text: str, n: int = 4) -> set:
    toks = re.findall(r"[a-z0-9]+", text.lower())
    return {" ".join(toks[i : i + n]) for i in range(len(toks) - n + 1)}


def jaccard(a: set, b: set) -> float:
    if not a or not b:
        return 0.0
    return len(a & b) / len(a | b)


def llm(prompt: str) -> str:
    cmd = os.environ.get("BENCH_LLM")
    if not cmd:
        sys.exit("set BENCH_LLM to a stdin->stdout paraphrase command (e.g. 'claude -p')")
    out = subprocess.run(
        cmd, shell=True, input=prompt, capture_output=True, text=True, timeout=120
    )
    return out.stdout.strip().splitlines()[-1].strip() if out.stdout.strip() else ""


def source_text(label: dict) -> str:
    src = label.get("source", {})
    return " ".join(str(v) for v in src.values())


def main() -> None:
    outdir = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".")
    queries, skipped = [], []

    for task in ("t1", "t2", "t3", "t4"):
        path = outdir / f"labels-{task}.jsonl"
        if not path.exists():
            continue
        for line in path.read_text().splitlines():
            label = json.loads(line)
            src = label.get("source", {})
            prompt = PROMPTS[task].format(
                subject=src.get("subject", ""),
                files=", ".join(src.get("files", [])[:8]),
                content=src.get("content", ""),
                intent=src.get("intent", ""),
                s1_snippet=src.get("s1_snippet", ""),
                s2_snippet=src.get("s2_snippet", ""),
            )
            src_grams = ngrams(source_text(label))
            query = ""
            for attempt in range(2):
                q = llm(prompt if attempt == 0 else prompt + "\nParaphrase more aggressively; share no distinctive phrase with the source.")
                if q and jaccard(ngrams(q), src_grams) <= MAX_JACCARD:
                    query = q
                    break
            qid = hashlib.sha1((task + json.dumps(label.get("gold"))).encode()).hexdigest()[:12]
            if not query:
                skipped.append({"qid": qid, "task": task, "reason": "leakage-or-empty"})
                continue
            # T4 only counts if the question genuinely needs both sessions.
            if task == "t4":
                verdict = llm(VALIDATE_T4.format(
                    q=query, a=src.get("s1_snippet", ""), b=src.get("s2_snippet", "")))
                if "BOTH" not in verdict.upper():
                    skipped.append({"qid": qid, "task": task, "reason": "not-multi-hop"})
                    continue
            rec = {
                "qid": qid,
                "task": task,
                "split": "dev" if int(qid, 16) % 10 == 0 else "test",
                "query": query,
                "gold": label["gold"],
            }
            if "turn_index" in label:
                rec["turn_index"] = label["turn_index"]
            queries.append(rec)

    (outdir / "queries.jsonl").write_text("".join(json.dumps(q) + "\n" for q in queries))
    (outdir / "skipped.jsonl").write_text("".join(json.dumps(s) + "\n" for s in skipped))
    by_task = {}
    for q in queries:
        by_task[q["task"]] = by_task.get(q["task"], 0) + 1
    print(f"queries: {len(queries)} {by_task}, skipped: {len(skipped)}", file=sys.stderr)


if __name__ == "__main__":
    main()
