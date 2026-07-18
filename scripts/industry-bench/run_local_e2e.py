#!/usr/bin/env python3
"""Resilient local end-to-end benchmark runner: local Claude (haiku) executes
the Rekal skill agentically per question, then judges answers vs gold.

Designed for subscription token limits: every result is checkpointed to disk
the moment it lands, and when the usage window is exhausted the runner parses
the error, sleeps until the window refreshes (probing cheaply), and continues
where it left off. Safe to kill/relaunch at any time — done ids are skipped.

Usage:
  python3 run_local_e2e.py --tasks tasks.jsonl --gold gold.jsonl \
      --repos-root ~/imb-lme-m/flat --out runs/local-e2e-<tag> \
      [--model haiku] [--judge-model haiku] [--phase both|answer|judge]

Tasks jsonl: {"id","conv","question","category"} per line.
Gold jsonl:  {"id","gold",...} per line (never shown to the answerer).
"""
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

CLAUDE = os.environ.get("CLAUDE_BIN", str(Path.home() / ".local/bin/claude"))
REKAL = os.environ.get("REKAL", str(Path(__file__).resolve().parents[2] / "rekal"))
WEIGHTS = '{"bm25":0.55,"lsa":0.1,"semantic":0.35,"facet_boost":0.05}'
SKILL_DIR = Path(__file__).resolve().parents[2] / "cmd/rekal/cli/skill/skills/rekal"

# Signals that the subscription window is exhausted (vs a real error).
LIMIT_PATTERNS = re.compile(
    r"usage limit|rate.?limit|limit reached|quota|too many requests|429|"
    r"overloaded|capacity|try again later|out of.*(tokens|credits)",
    re.IGNORECASE,
)
PROBE_INTERVAL_S = int(os.environ.get("PROBE_INTERVAL_S", "900"))  # 15 min
ANSWER_TIMEOUT_S = int(os.environ.get("ANSWER_TIMEOUT_S", "600"))
JUDGE_TIMEOUT_S = int(os.environ.get("JUDGE_TIMEOUT_S", "120"))


def log(msg: str) -> None:
    print(f"[{datetime.now(timezone.utc).strftime('%H:%M:%SZ')}] {msg}", flush=True)


def clean_env() -> dict:
    env = {k: v for k, v in os.environ.items()
           if not (k.startswith("ANTHROPIC") or k == "CLAUDE_CONFIG_DIR")}
    return env


def run_claude(prompt: str, cwd: str, model: str, timeout: int, agentic: bool) -> tuple[str, str]:
    """Returns (status, text). status: ok | limit | error | timeout."""
    cmd = [CLAUDE, "-p", "--model", model]
    if agentic:
        cmd += ["--dangerously-skip-permissions", "--allowed-tools", "Bash,Read,Grep,Glob"]
    try:
        # Prompt goes via stdin: greedy multi-value flags (--allowed-tools) would
        # otherwise swallow a positional prompt argument.
        p = subprocess.run(cmd, cwd=cwd, env=clean_env(), input=prompt,
                           capture_output=True, text=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        return "timeout", ""
    out = (p.stdout or "").strip()
    err = (p.stderr or "").strip()
    if p.returncode == 0 and out:
        return "ok", out
    blob = out + "\n" + err
    if LIMIT_PATTERNS.search(blob):
        return "limit", blob[-500:]
    return "error", blob[-500:]


def wait_for_refresh(model: str) -> None:
    """Block until a 1-token probe succeeds — the usage window has refreshed."""
    n = 0
    while True:
        n += 1
        log(f"limit hit — probe {n}: sleeping {PROBE_INTERVAL_S}s before retry")
        time.sleep(PROBE_INTERVAL_S)
        status, _ = run_claude("Reply with exactly: OK", cwd=str(Path.home()),
                               model=model, timeout=60, agentic=False)
        if status == "ok":
            log("window refreshed — resuming")
            return


ANSWER_PROMPT = """You have the Rekal memory skill. Answer ONE question using ONLY this repo's Rekal memory — no outside knowledge.

The skill (follow it faithfully): read {skill_dir}/SKILL.md and {skill_dir}/references/hunt.md first.
Binary: {rekal}
Recall:  {rekal} --weights '{weights}' -n <N> "<query>"   (JSON; .results[])
Route:   pipe recall JSON through {skill_dir}/scripts/recall-route.py — work from its digest
Drill:   {rekal} query --session <id> --offset <k> --limit 5   (or --role human; --full last resort)
SQL:     {rekal} query "SELECT ..."   (tables: sessions, turns(session_id, ts, role, content, turn_index))

Apply the skill's time-axis, enumeration, attribution and depth-judgment rules.
If the memory genuinely does not contain the answer, reply exactly: I DON'T KNOW — NOT IN MEMORY

Question: {question}

Reply with ONLY the final short answer (one or two sentences), no preamble."""

JUDGE_PROMPT = """You are a strict grader. Compare the answer to gold on MEANING (same key fact/value/entity; wording may differ).
Abstention questions: is_abstention={is_abstention}. If true, a refusal like "I DON'T KNOW — NOT IN MEMORY" is CORRECT and a fabricated specific answer is INCORRECT. If false, a wrong or missing key fact (or a wrongful refusal) is INCORRECT.

Question: {question}
Gold: {gold}
Answer: {answer}

Reply with exactly one word: CORRECT or INCORRECT"""


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--tasks", required=True)
    ap.add_argument("--gold", required=True)
    ap.add_argument("--repos-root", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--model", default="haiku")
    ap.add_argument("--judge-model", default="haiku")
    ap.add_argument("--phase", choices=("both", "answer", "judge"), default="both")
    args = ap.parse_args()

    out = Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    answers_f = out / "answers.jsonl"
    judged_f = out / "judged.jsonl"

    tasks = [json.loads(l) for l in open(args.tasks) if l.strip()]
    gold = {json.loads(l)["id"]: json.loads(l) for l in open(args.gold) if l.strip()}
    done_answers = set()
    if answers_f.exists():
        done_answers = {json.loads(l)["id"] for l in open(answers_f) if l.strip()}
    repos_root = Path(os.path.expanduser(args.repos_root))

    # Phase 1 — answer
    if args.phase in ("both", "answer"):
        todo = [t for t in tasks if t["id"] not in done_answers]
        log(f"answer phase: {len(todo)} to do ({len(done_answers)} already done)")
        for i, t in enumerate(todo):
            repo = repos_root / t["conv"] / "repo"
            if not (repo / ".rekal").is_dir():
                log(f"skip {t['id']}: no repo")
                continue
            prompt = ANSWER_PROMPT.format(skill_dir=SKILL_DIR, rekal=REKAL,
                                          weights=WEIGHTS, question=t["question"])
            while True:
                status, text = run_claude(prompt, cwd=str(repo), model=args.model,
                                          timeout=ANSWER_TIMEOUT_S, agentic=True)
                if status == "limit":
                    wait_for_refresh(args.model)
                    continue
                break
            rec = {"id": t["id"], "category": t.get("category"), "status": status,
                   "answer": text if status == "ok" else "",
                   "error": "" if status == "ok" else text,
                   "ts": datetime.now(timezone.utc).isoformat()}
            with open(answers_f, "a") as f:
                f.write(json.dumps(rec) + "\n")
            log(f"[{len(done_answers) + i + 1}/{len(tasks)}] {t['id']} -> {status}")

    # Phase 2 — judge
    if args.phase in ("both", "judge"):
        answers = {}
        if answers_f.exists():
            for l in open(answers_f):
                if l.strip():
                    r = json.loads(l)
                    if r.get("status") == "ok":
                        answers[r["id"]] = r
        done_judged = set()
        if judged_f.exists():
            done_judged = {json.loads(l)["id"] for l in open(judged_f) if l.strip()}
        todo = [a for i, a in answers.items() if i not in done_judged]
        log(f"judge phase: {len(todo)} to grade ({len(done_judged)} already done)")
        tmap = {t["id"]: t for t in tasks}
        for a in todo:
            g = gold.get(a["id"], {})
            t = tmap.get(a["id"], {})
            is_abs = (t.get("category") == "abstention") or not (g.get("evidence") or g.get("evidence_session_ids"))
            prompt = JUDGE_PROMPT.format(question=t.get("question", ""), gold=g.get("gold", ""),
                                         answer=a["answer"], is_abstention=is_abs)
            while True:
                status, text = run_claude(prompt, cwd=str(Path.home()), model=args.judge_model,
                                          timeout=JUDGE_TIMEOUT_S, agentic=False)
                if status == "limit":
                    wait_for_refresh(args.judge_model)
                    continue
                break
            verdict = "CORRECT" if status == "ok" and "CORRECT" in text.upper() and "INCORRECT" not in text.upper() else \
                      ("INCORRECT" if status == "ok" else f"ERROR:{status}")
            with open(judged_f, "a") as f:
                f.write(json.dumps({"id": a["id"], "category": a.get("category"),
                                    "is_abstention": is_abs, "verdict": verdict}) + "\n")
            log(f"judged {a['id']} -> {verdict}")

        # summary
        rows = [json.loads(l) for l in open(judged_f) if l.strip()] if judged_f.exists() else []
        ok = sum(1 for r in rows if r["verdict"] == "CORRECT")
        log(f"SUMMARY: {ok}/{len(rows)} correct")
        import collections
        by = collections.defaultdict(lambda: [0, 0])
        for r in rows:
            by[r.get("category") or "?"][1] += 1
            if r["verdict"] == "CORRECT":
                by[r.get("category") or "?"][0] += 1
        for c, (k, n) in sorted(by.items()):
            log(f"  {c}: {k}/{n}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
