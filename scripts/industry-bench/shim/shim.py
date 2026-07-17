#!/usr/bin/env python3
"""WS-E harness shim: Rekal behind add/search/answer verbs.

Contract: docs/research/industry-bench/02-adapter-architecture.md §4.

This module is stdlib-only. It operates on an *already ingested* synthetic
repo (from sh_gen) for search/answer, and records token counts of every
retrieved context. `add` is a thin buffer that flushes via sh_gen on
session boundary — full online add is for harness integration; the smoke
path uses pre-ingested repos.

Token accounting: whitespace-split word count as a cheap proxy when no
tokenizer is configured; optional --tiktoken later. Manifests record the
counter name so numbers stay comparable within a run.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path

CAPTURE_KILL_SWITCHES = ("REKAL_BENCH", "REKAL_SKIP_CHECKPOINT")


def count_tokens(text: str) -> int:
    """Whitespace token proxy (documented in run manifests as token_counter=whitespace)."""
    if not text:
        return 0
    return len(text.split())


def rekal_env(repo: Path, claude_config: Path | None, rekal: Path) -> dict:
    env = dict(os.environ)
    for k in CAPTURE_KILL_SWITCHES:
        env.pop(k, None)
    env["PATH"] = f"{rekal.parent}{os.pathsep}{env.get('PATH', '')}"
    if claude_config is not None:
        env["CLAUDE_CONFIG_DIR"] = str(claude_config)
    return env


def run_rekal(rekal: Path, repo: Path, env: dict, args: list[str], retries: int = 8) -> str:
    """Run rekal in repo cwd; retry on DuckDB 'another rekal process' lock."""
    last = None
    for i in range(retries):
        p = subprocess.run(
            [str(rekal), *args],
            cwd=repo,
            env=env,
            capture_output=True,
            text=True,
        )
        out = (p.stdout or "") + (p.stderr or "")
        if p.returncode == 0:
            return p.stdout
        if "another rekal process" in out.lower() or "lock" in out.lower():
            time.sleep(1.5 * (i + 1))
            last = out
            continue
        raise RuntimeError(f"rekal {' '.join(args)} failed ({p.returncode}): {out[-2000:]}")
    raise RuntimeError(f"rekal locked after retries: {last[-1000:] if last else ''}")


def search(rekal: Path, repo: Path, query: str, limit: int = 5, env: dict | None = None) -> dict:
    """search(user_id, query) → contexts with token counts.

    Invokes stock `rekal <query>` (JSON). No mode classifier yet (WS-C);
    stock gates only — WS-E DoD allows this before C/D plug in.
    """
    if env is None:
        env = rekal_env(repo, None, rekal)
    raw = run_rekal(rekal, repo, env, [query, "--limit", str(limit)])
    # rekal may print version-update noise before/after JSON; extract object
    start = raw.find("{")
    end = raw.rfind("}")
    if start < 0 or end < 0:
        raise RuntimeError(f"no JSON in recall output: {raw[:500]}")
    payload = json.loads(raw[start : end + 1])

    contexts = []
    retrieved_tokens = 0
    for r in payload.get("results") or []:
        snippet = r.get("snippet") or ""
        tok = count_tokens(snippet)
        retrieved_tokens += tok
        contexts.append(
            {
                "session_id": r.get("session_id"),
                "score": r.get("score"),
                "confidence": r.get("confidence"),
                "mass": r.get("mass"),
                "snippet": snippet,
                "snippet_turn_index": r.get("snippet_turn_index"),
                "tokens": tok,
                "files": (r.get("session") or {}).get("files"),
            }
        )
    knowledge = []
    for k in payload.get("knowledge") or []:
        snip = k.get("snippet") or ""
        tok = count_tokens(snip)
        retrieved_tokens += tok
        knowledge.append({**k, "tokens": tok})

    return {
        "query": query,
        "contexts": contexts,
        "knowledge": knowledge,
        "retrieved_context_tokens": retrieved_tokens,
        "token_counter": "whitespace",
        "limit": limit,
    }


def answer_extractive(query: str, search_result: dict) -> dict:
    """Deterministic extractive 'answer' for smoke tests without an LLM.

    Returns the top session snippet as hypothesis. Official LongMemEval
    scoring needs an LLM judge later; this path lets the harness exercise
    ingest→search→answer→token columns without API keys.
    """
    ctx = search_result.get("contexts") or []
    if not ctx:
        text = "I don't know"
        source = None
    else:
        text = ctx[0]["snippet"]
        source = ctx[0].get("session_id")
    answer_tokens = count_tokens(text)
    return {
        "query": query,
        "hypothesis": text,
        "source_session_id": source,
        "answer_tokens": answer_tokens,
        "retrieved_context_tokens": search_result.get("retrieved_context_tokens", 0),
        "answer_path_tokens": search_result.get("retrieved_context_tokens", 0) + answer_tokens,
        "token_counter": "whitespace",
        "answer_mode": "extractive-top1",
    }


def load_questions(conversations_jsonl: Path, conversation_id: str | None = None) -> list:
    rows = []
    with open(conversations_jsonl) as f:
        for line in f:
            if not line.strip():
                continue
            c = json.loads(line)
            if conversation_id and c["conversation_id"] != conversation_id:
                continue
            for q in c["questions"]:
                rows.append(
                    {
                        "conversation_id": c["conversation_id"],
                        "question_id": q["question_id"],
                        "category": q["category"],
                        "question": q["question"],
                        "answer": q["answer"],
                        "evidence_session_ids": q.get("evidence_session_ids") or [],
                    }
                )
    return rows


def smoke_one(
    rekal: Path,
    workdir: Path,
    conversations_jsonl: Path,
    conversation_id: str,
    out_dir: Path,
) -> dict:
    """End-to-end smoke for one pre-ingested conversation's questions."""
    repo = workdir / conversation_id / "repo"
    if not (repo / ".rekal").exists():
        raise FileNotFoundError(f"missing ingested repo {repo}")
    questions = load_questions(conversations_jsonl, conversation_id)
    env = rekal_env(repo, workdir / conversation_id / "claude-config", rekal)
    per_q = []
    for q in questions:
        sr = search(rekal, repo, q["question"], limit=5, env=env)
        ans = answer_extractive(q["question"], sr)
        # Evidence hit? session marker files encode benchmark session id.
        hit = False
        for c in sr["contexts"]:
            files = c.get("files") or []
            for ev in q["evidence_session_ids"]:
                if any(ev in (f or "") for f in files):
                    hit = True
        if not hit:
            # also check snippet / session linkage via marker in files basename
            for c in sr["contexts"]:
                for ev in q["evidence_session_ids"]:
                    if ev in json.dumps(c):
                        hit = True
        per_q.append(
            {
                **q,
                "search": sr,
                "answer": ans,
                "evidence_in_top": hit,
                "top_score": (sr["contexts"][0]["score"] if sr["contexts"] else None),
                "top_confidence": (sr["contexts"][0]["confidence"] if sr["contexts"] else None),
            }
        )

    out_dir.mkdir(parents=True, exist_ok=True)
    manifest = {
        "conversation_id": conversation_id,
        "system": "rekal-stock",
        "calibration": "stock",
        "token_counter": "whitespace",
        "n_questions": len(per_q),
        "evidence_hit_rate": (
            sum(1 for r in per_q if r["evidence_in_top"]) / len(per_q) if per_q else 0.0
        ),
        "retrieved_context_tokens_mean": (
            sum(r["answer"]["retrieved_context_tokens"] for r in per_q) / len(per_q) if per_q else 0
        ),
        "answer_path_tokens_mean": (
            sum(r["answer"]["answer_path_tokens"] for r in per_q) / len(per_q) if per_q else 0
        ),
    }
    (out_dir / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    with open(out_dir / "per_question.jsonl", "w") as f:
        for r in per_q:
            f.write(json.dumps(r, ensure_ascii=False) + "\n")
    lines = [
        f"# smoke {conversation_id}",
        "",
        f"- questions: {manifest['n_questions']}",
        f"- evidence_in_top rate: {manifest['evidence_hit_rate']:.2f}",
        f"- retrieved_context_tokens mean: {manifest['retrieved_context_tokens_mean']:.1f}",
        f"- answer_path_tokens mean: {manifest['answer_path_tokens_mean']:.1f}",
        "",
        "| qid | category | evidence_hit | top_score | ctx_tok |",
        "|---|---|---:|---:|---:|",
    ]
    for r in per_q:
        lines.append(
            f"| {r['question_id']} | {r['category']} | {int(r['evidence_in_top'])} | "
            f"{r['top_score']} | {r['answer']['retrieved_context_tokens']} |"
        )
    (out_dir / "table.md").write_text("\n".join(lines) + "\n")
    print(json.dumps(manifest, indent=2))
    return manifest


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    sub = ap.add_subparsers(dest="cmd", required=True)

    p_search = sub.add_parser("search", help="recall search with token accounting")
    p_search.add_argument("--repo", type=Path, required=True)
    p_search.add_argument("--query", required=True)
    p_search.add_argument("--rekal", default="rekal")
    p_search.add_argument("--limit", type=int, default=5)

    p_smoke = sub.add_parser("smoke", help="smoke one ingested conversation")
    p_smoke.add_argument("--workdir", type=Path, required=True, help="sh_gen --out dir")
    p_smoke.add_argument("--conversation-id", required=True)
    p_smoke.add_argument("--input", type=Path, required=True, help="conversations.jsonl")
    p_smoke.add_argument("--out", type=Path, required=True, help="runs/... output dir")
    p_smoke.add_argument("--rekal", default="rekal")

    args = ap.parse_args()
    rekal = Path(args.rekal).resolve()
    if args.cmd == "search":
        env = rekal_env(args.repo.resolve(), None, rekal)
        print(json.dumps(search(rekal, args.repo.resolve(), args.query, args.limit, env), indent=2))
    elif args.cmd == "smoke":
        smoke_one(rekal, args.workdir.resolve(), args.input, args.conversation_id, args.out)


if __name__ == "__main__":
    main()
