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
      [--model <m>] [--judge-model <m>] [--phase both|answer|judge]

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
    """Drop only what interferes with a local `claude -p`: a foreign config
    dir and forced model overrides. Gateway vars (ANTHROPIC_BASE_URL,
    ANTHROPIC_AUTH_TOKEN, ANTHROPIC_API_KEY, HTTPS_PROXY, …) are KEPT — a
    proxied claude cannot reach its endpoint without them, and stripping the
    whole ANTHROPIC* namespace made every proxied run fail as 'error'."""
    drop = {"CLAUDE_CONFIG_DIR", "ANTHROPIC_MODEL", "ANTHROPIC_SMALL_FAST_MODEL"}
    return {k: v for k, v in os.environ.items() if k not in drop}


def run_claude(prompt: str, cwd: str, model: str, timeout: int, agentic: bool) -> tuple[str, str, dict]:
    """Returns (status, text, usage). status: ok | limit | error | timeout.

    usage: {"input_tokens","output_tokens","cache_read_tokens","cache_creation_tokens",
            "total_cost_usd","num_turns","duration_ms"} — zeros when unavailable.
    """
    # --model omitted when empty: the claude CLI (or the gateway) picks its
    # configured default — required on single-model gateway endpoints.
    cmd = [CLAUDE, "-p"] + (["--model", model] if model else []) + ["--output-format", "json"]
    if agentic:
        cmd += ["--dangerously-skip-permissions", "--allowed-tools", "Bash,Read,Grep,Glob"]
    empty_usage = {"input_tokens": 0, "output_tokens": 0, "cache_read_tokens": 0,
                   "cache_creation_tokens": 0, "total_cost_usd": 0.0, "num_turns": 0,
                   "duration_ms": 0}
    try:
        # Prompt goes via stdin: greedy multi-value flags (--allowed-tools) would
        # otherwise swallow a positional prompt argument.
        p = subprocess.run(cmd, cwd=cwd, env=clean_env(), input=prompt,
                           capture_output=True, text=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        return "timeout", "", empty_usage
    out = (p.stdout or "").strip()
    err = (p.stderr or "").strip()
    if p.returncode == 0 and out:
        try:
            d = json.loads(out)
            u = d.get("usage") or {}
            usage = {
                "input_tokens": int(u.get("input_tokens") or 0),
                "output_tokens": int(u.get("output_tokens") or 0),
                "cache_read_tokens": int(u.get("cache_read_input_tokens") or 0),
                "cache_creation_tokens": int(u.get("cache_creation_input_tokens") or 0),
                "total_cost_usd": float(d.get("total_cost_usd") or 0.0),
                "num_turns": int(d.get("num_turns") or 0),
                "duration_ms": int(d.get("duration_ms") or 0),
            }
            text = (d.get("result") or "").strip()
            if d.get("is_error") and LIMIT_PATTERNS.search(text):
                return "limit", text[-500:], usage
            return ("ok" if text else "error"), text, usage
        except (json.JSONDecodeError, TypeError, ValueError):
            return "ok", out, empty_usage  # non-JSON fallback: keep the raw text
    blob = out + "\n" + err
    if LIMIT_PATTERNS.search(blob):
        return "limit", blob[-500:], empty_usage
    return "error", blob[-500:], empty_usage


def wait_for_refresh(model: str) -> None:
    """Block until a 1-token probe succeeds — the usage window has refreshed."""
    n = 0
    while True:
        n += 1
        log(f"limit hit — probe {n}: sleeping {PROBE_INTERVAL_S}s before retry")
        time.sleep(PROBE_INTERVAL_S)
        status, _, _ = run_claude("Reply with exactly: OK", cwd=str(Path.home()),
                                  model=model, timeout=60, agentic=False)
        if status == "ok":
            log("window refreshed — resuming")
            return


ANSWER_PROMPT = """You have the Rekal memory skill. Answer ONE question using ONLY this repo's Rekal memory — no outside knowledge.

The skill (follow it faithfully): read {skill_dir}/SKILL.md and {skill_dir}/references/hunt.md first.
Binary: {rekal}
Recall:  {rekal} --weights '{weights}' -n <N> "<query>"   (prints INJECT/KNOWLEDGE/SILENCE + a candidate digest; add --json for raw .results[])
Drill:   {rekal} query --session <id> --offset <k> --limit 5   (or --role human; --full last resort)
SQL:     {rekal} query "SELECT ..."   (tables: sessions, turns(session_id, ts, role, content, turn_index))

Apply the skill's time-axis, enumeration, attribution and depth-judgment rules.
If the memory genuinely does not contain the answer, reply exactly: I DON'T KNOW — NOT IN MEMORY

Question: {question}

Reply with ONLY the final short answer (one or two sentences), no preamble."""

# Compact skill card for small executors: the full skill files overwhelm weak
# models — they read two long documents and then under-execute the loop
# (observed: 17/22 haiku errors were wrongful abstentions with evidence present
# in memory). The card inlines the operative rules so the model's effort goes
# to navigation, not doc comprehension.
ANSWER_PROMPT_CARD = """Answer ONE question about the people in this repo's Rekal memory (a searchable ledger of past conversations). Every FACT about these people must come from the memory; you may combine those facts with general world knowledge to reason to a conclusion, but never invent personal facts.

TOOLS
Search: {rekal} --weights '{weights}' -n 20 "<query>"
  → prints INJECT/KNOWLEDGE/SILENCE + a candidate digest (session ids, confidences, snippets); add --json for raw .results[]
Read a session around a hit: {rekal} query --session <id> --offset <turn-2> --limit 5
Whole session (small): {rekal} query --session <id> --full
SQL over all turns: {rekal} query "SELECT ts, session_id, content FROM turns WHERE content ILIKE '%word%' ORDER BY ts"
  (tables: sessions, turns(session_id, ts, role, content, turn_index))

THE LOOP — you MUST complete it before ever saying you don't know:
1. Search the question as asked. Read the digest past #1 — the answer is often at rank 3-10.
2. No answer yet? Reformulate at least twice: keyword-only (drop question words), synonyms, split compound questions. Search each. A session that surfaces under two phrasings is almost always the answer.
3. Still nothing? Widen: -n 50. Large histories bury evidence deep.
4. Drill the top 2-3 candidates (turn window around the snippet) — snippets are keyholes; the answer text is usually in the surrounding turns, not the snippet.
5. Time questions ("when/how long/which month/what order"): use SQL. Anchor dates first (the event you can date), scan the window with ts BETWEEN, ORDER BY ts. Event time ≠ mention time ("last week I…" shifts it). COUNT before LIMIT; page — never trust a truncated list. When the speaker used a relative phrase, KEEP it relative in your answer: "yesterday" on Oct 21 → "October 20"; "a few days ago" on Aug 19 → "a few days before August 19" (do NOT flatten it to the session date).
6. "All/every/what order/what activities/besides X" questions: the gold answer is a COMPLETE list. Enumerate every distinct item across ALL sessions (SQL ILIKE over turns is the reliable way), dedupe, and list them all. A partial list is graded wrong. Do not stop after the first session that mentions two items.
7. "My/I" preference questions: answer from the user's own assertion ("I bought/made/love"), not from things merely discussed.
8. Multi-hop questions (the answer needs TWO facts joined — "who did X tell about Y", "what do A and B both own", "what does X do besides Y"): search each entity/fact SEPARATELY, collect every mention (people, items, activities) across ALL matching sessions, then combine. One search cannot answer a two-fact question — the two facts almost never sit in the same session. Enumerate mentions per entity with SQL if needed.
9. Recommendation/judgment questions ("what would X enjoy", "would this suit X", "does X employ many people"): first retrieve what memory establishes about X (preferences, facts, context), then REASON over it with general world knowledge to conclude. Memory gives the premises; you supply the inference (e.g. memory says X loves Star Wars and visits Ireland → world knowledge names the Star Wars filming locations in Ireland). Do not abstain just because the conclusion itself is not stated verbatim in memory — abstain only if memory lacks even the premises.
10. FALSE-PREMISE TRAP: if the memory shows the question's premise is wrong — the event/attribute belongs to a DIFFERENT person, or the event never happened at the stated time ("What did Deborah design…" when it was Jolene who designed it) — the correct answer is "I DON'T KNOW — NOT IN MEMORY". Do NOT answer with the corrected fact or name the right person; a question built on a false premise has no answer in memory.

ABSTAIN ("I DON'T KNOW — NOT IN MEMORY") ONLY IF: (a) rule 10 applies — the premise is false, OR (b) you did steps 1-4 (reformulated twice, widened to -n 50, drilled top candidates) and everything came back genuinely unrelated. Wrongful abstention on an answerable question is as bad as a wrong answer. But NEVER invent: if the loop truly finds nothing, say so.

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
    ap.add_argument("--model", default="",
                    help="executor model; omit to use the claude CLI's configured default")
    ap.add_argument("--judge-model", default="",
                    help="judge model; omit to use the claude CLI's configured default")
    ap.add_argument("--phase", choices=("both", "answer", "judge"), default="both")
    ap.add_argument("--prompt-style", choices=("skill-files", "card"), default="card",
                    help="card = compact inlined skill (better for small models)")
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
            tmpl = ANSWER_PROMPT_CARD if args.prompt_style == "card" else ANSWER_PROMPT
            prompt = tmpl.format(skill_dir=SKILL_DIR, rekal=REKAL,
                                 weights=WEIGHTS, question=t["question"])
            while True:
                status, text, usage = run_claude(prompt, cwd=str(repo), model=args.model,
                                                 timeout=ANSWER_TIMEOUT_S, agentic=True)
                if status == "limit":
                    wait_for_refresh(args.model)
                    continue
                break
            rec = {"id": t["id"], "category": t.get("category"), "status": status,
                   "answer": text if status == "ok" else "",
                   "error": "" if status == "ok" else text,
                   "usage": usage,
                   "ts": datetime.now(timezone.utc).isoformat()}
            with open(answers_f, "a") as f:
                f.write(json.dumps(rec) + "\n")
            log(f"[{len(done_answers) + i + 1}/{len(tasks)}] {t['id']} -> {status} "
                f"(in={usage['input_tokens']} out={usage['output_tokens']} "
                f"cache={usage['cache_read_tokens']} turns={usage['num_turns']} "
                f"${usage['total_cost_usd']:.4f})")

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
                status, text, usage = run_claude(prompt, cwd=str(Path.home()), model=args.judge_model,
                                                 timeout=JUDGE_TIMEOUT_S, agentic=False)
                if status == "limit":
                    wait_for_refresh(args.judge_model)
                    continue
                break
            verdict = "CORRECT" if status == "ok" and "CORRECT" in text.upper() and "INCORRECT" not in text.upper() else \
                      ("INCORRECT" if status == "ok" else f"ERROR:{status}")
            with open(judged_f, "a") as f:
                f.write(json.dumps({"id": a["id"], "category": a.get("category"),
                                    "is_abstention": is_abs, "verdict": verdict,
                                    "usage": usage}) + "\n")
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
        # token/cost rollup across both phases
        tot = collections.Counter()
        n_ans = 0
        if answers_f.exists():
            for l in open(answers_f):
                if not l.strip():
                    continue
                u = json.loads(l).get("usage") or {}
                n_ans += 1
                for k in ("input_tokens", "output_tokens", "cache_read_tokens",
                          "cache_creation_tokens", "num_turns"):
                    tot[f"answer_{k}"] += int(u.get(k) or 0)
                tot["answer_cost_musd"] += int(float(u.get("total_cost_usd") or 0) * 1e6)
        for r in rows:
            u = r.get("usage") or {}
            for k in ("input_tokens", "output_tokens"):
                tot[f"judge_{k}"] += int(u.get(k) or 0)
            tot["judge_cost_musd"] += int(float(u.get("total_cost_usd") or 0) * 1e6)
        log(f"TOKENS answer-phase: in={tot['answer_input_tokens']} out={tot['answer_output_tokens']} "
            f"cache_read={tot['answer_cache_read_tokens']} cache_create={tot['answer_cache_creation_tokens']} "
            f"agent_turns={tot['answer_num_turns']} cost=${tot['answer_cost_musd'] / 1e6:.4f}"
            + (f" (per-q in={tot['answer_input_tokens'] // max(n_ans, 1)} "
               f"out={tot['answer_output_tokens'] // max(n_ans, 1)})" if n_ans else ""))
        log(f"TOKENS judge-phase:  in={tot['judge_input_tokens']} out={tot['judge_output_tokens']} "
            f"cost=${tot['judge_cost_musd'] / 1e6:.4f}")
        log(f"TOKENS total cost:   ${(tot['answer_cost_musd'] + tot['judge_cost_musd']) / 1e6:.4f}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
