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
import time
from pathlib import Path

CAPTURE_KILL_SWITCHES = ("REKAL_BENCH", "REKAL_SKIP_CHECKPOINT")
MARKER_KNOWLEDGE_PATHS = ("CLAUDE.md",)
CALIBRATION_ENV_MAP = {
    "CONF_MIN": "REKAL_HUNT_CONF_MIN",
    "CONF_SOFT": "REKAL_HUNT_CONF_SOFT",
    "GAP_MIN": "REKAL_HUNT_GAP_MIN",
    "MASS_MIN": "REKAL_HUNT_MASS_MIN",
    "KNOWLEDGE_MIN": "REKAL_HUNT_KNOWLEDGE_MIN",
}
DEEP_RETRIEVAL_CATEGORIES = frozenset({"multi-hop", "knowledge-update"})


def load_calibration(path: Path | None) -> dict:
    if path is None:
        return {}
    data = json.loads(path.read_text())
    return dict(data.get("gate") or {})


def apply_calibration_env(env: dict, calibration: dict) -> dict:
    out = dict(env)
    for key, env_key in CALIBRATION_ENV_MAP.items():
        if key in calibration:
            out[env_key] = str(calibration[key])
    return out


def _evidence_marker_match(evidence_id: str, file_path: str) -> bool:
    """Match sh-gen marker files: sessions/YYYY-MM-DD-sN.md (not s1 inside s13)."""
    if not evidence_id or not file_path:
        return False
    return bool(re.search(rf"-{re.escape(evidence_id)}\.md$", file_path))


def context_has_evidence(ctx: dict, evidence_session_ids: list[str]) -> bool:
    files = ctx.get("files") or []
    for ev in evidence_session_ids:
        if any(_evidence_marker_match(ev, str(f or "")) for f in files):
            return True
    return False


def evidence_rank(contexts: list[dict], evidence_session_ids: list[str]) -> int | None:
    if not evidence_session_ids:
        return None
    for i, ctx in enumerate(contexts):
        if context_has_evidence(ctx, evidence_session_ids):
            return i
    return None


def evidence_at_k(contexts: list[dict], evidence_session_ids: list[str], k: int) -> bool:
    if not evidence_session_ids:
        return False
    for ctx in contexts[:k]:
        if context_has_evidence(ctx, evidence_session_ids):
            return True
    return False


def search_limit_for_category(category: str, evidence_session_ids: list[str]) -> int:
    if category in DEEP_RETRIEVAL_CATEGORIES or len(evidence_session_ids) > 1:
        return 10
    return 5


def classify_miss_reason(
    category: str,
    evidence_session_ids: list[str],
    contexts: list[dict],
    route_gate: str,
) -> str:
    if not evidence_session_ids:
        return "abstention"
    rank = evidence_rank(contexts, evidence_session_ids)
    if rank is None:
        return "true_miss"
    if rank == 0:
        return "hit"
    if rank < 5:
        return "deep_rank_lt5"
    if rank < 10:
        return "deep_rank_lt10"
    if route_gate in ("SILENCE", "FALLBACK_STOCK", "KNOWLEDGE"):
        return "gate_blocked"
    return "deep_rank_gte10"


def _sort_contexts_by_confidence(contexts: list[dict]) -> list[dict]:
    return sorted(contexts, key=lambda c: float(c.get("confidence") or 0.0), reverse=True)


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


def _extract_json_object(raw: str) -> tuple[dict, str]:
    start = raw.find("{")
    end = raw.rfind("}")
    if start < 0 or end < 0:
        raise RuntimeError(f"no JSON in recall output: {raw[:500]}")
    body = raw[start : end + 1]
    return json.loads(body), body


def _skill_scripts_dir(repo: Path) -> Path:
    # Prefer the harness checkout's skill scripts — ingested repos freeze
    # whatever `rekal init` shipped at ingest time (stale hunt-gate bars).
    source = Path(__file__).resolve().parents[3] / "cmd" / "rekal" / "cli" / "skill" / "skills" / "rekal" / "scripts"
    if (source / "recall-route.py").exists():
        return source
    local = repo / ".claude" / "skills" / "rekal" / "scripts"
    if (local / "recall-route.py").exists():
        return local
    return source


def _run_recall_route(repo: Path, recall_json_body: str, env: dict) -> dict:
    scripts = _skill_scripts_dir(repo)
    p = subprocess.run(
        ["python3", str(scripts / "recall-route.py")],
        input=recall_json_body,
        capture_output=True,
        text=True,
        env=env,
    )
    line = (p.stdout or "").strip()
    out = {"gate": "SILENCE", "raw": line or (p.stderr or "").strip(), "exit_code": p.returncode}
    if not line:
        return out
    if line.startswith("INJECT"):
        out["gate"] = "INJECT"
    elif line.startswith("KNOWLEDGE"):
        out["gate"] = "KNOWLEDGE"
    elif line.startswith("SILENCE"):
        out["gate"] = "SILENCE"
    return out


def _is_marker_only_knowledge(path: str) -> bool:
    return path in MARKER_KNOWLEDGE_PATHS or path.startswith("sessions/")


def search(
    rekal: Path,
    repo: Path,
    query: str,
    limit: int = 5,
    env: dict | None = None,
    route: str = "stock",
    calibration: dict | None = None,
) -> dict:
    """search(user_id, query) → contexts with token counts.

    Invokes stock `rekal <query>` (JSON). No mode classifier yet (WS-C);
    stock gates only — WS-E DoD allows this before C/D plug in.
    """
    if env is None:
        env = rekal_env(repo, None, rekal)
    gate_env = apply_calibration_env(env, calibration or {})

    raw = run_rekal(rekal, repo, env, [query, "--limit", str(limit)])
    payload, body = _extract_json_object(raw)

    route_info = {"gate": "STOCK", "raw": "stock-route"}
    if route == "skill":
        route_info = _run_recall_route(repo, body, gate_env)

    contexts = []
    retrieved_tokens = 0
    results = list(payload.get("results") or [])
    skill_order = route == "skill" and route_info["gate"] == "INJECT"
    if skill_order:
        # Skill episode selection uses absolute confidence, not max-norm score.
        results.sort(key=lambda r: float(r.get("confidence") or 0.0), reverse=True)
    for r in results:
        snippet = r.get("snippet") or ""
        tok = count_tokens(snippet)
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
        knowledge.append({**k, "tokens": tok})

    if route == "skill":
        if route_info["gate"] == "KNOWLEDGE":
            knowledge = [k for k in knowledge if not _is_marker_only_knowledge(str(k.get("path") or ""))]
            if not knowledge:
                # Synthetic marker files are not meaningful prose memory.
                route_info = {
                    "gate": "SILENCE",
                    "raw": "SILENCE reason=knowledge_marker_only",
                    "exit_code": route_info.get("exit_code"),
                }
        if route_info["gate"] == "INJECT":
            knowledge = []
        elif route_info["gate"] in ("KNOWLEDGE", "SILENCE"):
            contexts = []

    for c in contexts:
        retrieved_tokens += int(c.get("tokens") or 0)
    for k in knowledge:
        retrieved_tokens += int(k.get("tokens") or 0)

    return {
        "query": query,
        "contexts": contexts,
        "knowledge": knowledge,
        "route": route_info,
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
    limit_questions: int = 0,
    route: str = "stock",
    calibration_path: Path | None = None,
) -> dict:
    """End-to-end smoke for one pre-ingested conversation's questions."""
    repo = workdir / conversation_id / "repo"
    if not (repo / ".rekal").exists():
        raise FileNotFoundError(f"missing ingested repo {repo}")
    calibration = load_calibration(calibration_path)
    calibration_label = calibration_path.name if calibration_path else "stock"
    questions = load_questions(conversations_jsonl, conversation_id)
    if limit_questions:
        questions = questions[:limit_questions]
    env = rekal_env(repo, workdir / conversation_id / "claude-config", rekal)
    per_q = []
    for q in questions:
        limit = search_limit_for_category(q["category"], q["evidence_session_ids"])
        sr = search(
            rekal,
            repo,
            q["question"],
            limit=limit,
            env=env,
            route=route,
            calibration=calibration,
        )
        if route == "skill" and (q.get("category") != "abstention"):
            gate = ((sr.get("route") or {}).get("gate") or "").upper()
            if gate in ("SILENCE", "KNOWLEDGE"):
                # Benchmark QA should not over-silence non-abstention categories.
                stock = search(
                    rekal,
                    repo,
                    q["question"],
                    limit=limit,
                    env=env,
                    route="stock",
                    calibration=calibration,
                )
                stock["contexts"] = _sort_contexts_by_confidence(stock.get("contexts") or [])
                stock["route"] = {
                    "gate": "FALLBACK_STOCK",
                    "raw": f"fallback_from_skill:{(sr.get('route') or {}).get('raw')}",
                }
                sr = stock
        gate_name = ((sr.get("route") or {}).get("gate") or "STOCK").upper()
        ev_ids = q["evidence_session_ids"]
        contexts = sr.get("contexts") or []
        ev_rank = evidence_rank(contexts, ev_ids)
        miss_reason = classify_miss_reason(q["category"], ev_ids, contexts, gate_name)
        ans = answer_extractive(q["question"], sr)
        per_q.append(
            {
                **q,
                "search": sr,
                "answer": ans,
                "evidence_in_top": evidence_at_k(contexts, ev_ids, 5),
                "evidence_at_5": evidence_at_k(contexts, ev_ids, 5),
                "evidence_at_10": evidence_at_k(contexts, ev_ids, 10),
                "evidence_rank": ev_rank,
                "miss_reason": miss_reason,
                "route_gate": gate_name,
                "confidence_top_is_evidence": ev_rank == 0 if ev_rank is not None else False,
                "top_score": (contexts[0]["score"] if contexts else None),
                "top_confidence": (contexts[0]["confidence"] if contexts else None),
            }
        )

    answerable = [r for r in per_q if r["evidence_session_ids"]]
    out_dir.mkdir(parents=True, exist_ok=True)
    manifest = {
        "conversation_id": conversation_id,
        "system": f"rekal-{route}",
        "calibration": calibration_label,
        "token_counter": "whitespace",
        "n_questions": len(per_q),
        "evidence_at_5_rate": (
            sum(1 for r in answerable if r["evidence_at_5"]) / len(answerable) if answerable else 0.0
        ),
        "evidence_at_10_rate": (
            sum(1 for r in answerable if r["evidence_at_10"]) / len(answerable) if answerable else 0.0
        ),
        "evidence_hit_rate": (
            sum(1 for r in per_q if r["evidence_in_top"]) / len(per_q) if per_q else 0.0
        ),
        "retrieved_context_tokens_mean": (
            sum(r["answer"]["retrieved_context_tokens"] for r in per_q) / len(per_q) if per_q else 0
        ),
        "answer_path_tokens_mean": (
            sum(r["answer"]["answer_path_tokens"] for r in per_q) / len(per_q) if per_q else 0
        ),
        "miss_reasons": {
            reason: sum(1 for r in per_q if r["miss_reason"] == reason)
            for reason in sorted({r["miss_reason"] for r in per_q})
        },
    }
    (out_dir / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    with open(out_dir / "per_question.jsonl", "w") as f:
        for r in per_q:
            f.write(json.dumps(r, ensure_ascii=False) + "\n")
    lines = [
        f"# smoke {conversation_id}",
        "",
        f"- questions: {manifest['n_questions']}",
        f"- route: {route}",
        f"- calibration: {calibration_label}",
        f"- evidence@5 (answerable): {manifest['evidence_at_5_rate']:.2f}",
        f"- evidence@10 (answerable): {manifest['evidence_at_10_rate']:.2f}",
        f"- retrieved_context_tokens mean: {manifest['retrieved_context_tokens_mean']:.1f}",
        f"- answer_path_tokens mean: {manifest['answer_path_tokens_mean']:.1f}",
        "",
        "| qid | category | gate | ev@5 | ev@10 | rank | miss_reason | top_conf |",
        "|---|---|---|---:|---:|---:|---|---:|",
    ]
    for r in per_q:
        lines.append(
            f"| {r['question_id']} | {r['category']} | {r['route_gate']} | "
            f"{int(r['evidence_at_5'])} | {int(r['evidence_at_10'])} | "
            f"{'' if r['evidence_rank'] is None else r['evidence_rank']} | "
            f"{r['miss_reason']} | {r['top_confidence']} |"
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
    p_search.add_argument("--route", choices=("stock", "skill"), default="stock")
    p_search.add_argument("--calibration", type=Path, default=None,
                          help="WS-D gate overrides (e.g. calibration/chat-provisional.json)")

    p_smoke = sub.add_parser("smoke", help="smoke one ingested conversation")
    p_smoke.add_argument("--workdir", type=Path, required=True, help="sh_gen --out dir")
    p_smoke.add_argument("--conversation-id", required=True)
    p_smoke.add_argument("--input", type=Path, required=True, help="conversations.jsonl")
    p_smoke.add_argument("--out", type=Path, required=True, help="runs/... output dir")
    p_smoke.add_argument("--rekal", default="rekal")
    p_smoke.add_argument("--limit-questions", type=int, default=0,
                         help="only first N questions (LoCoMo smoke)")
    p_smoke.add_argument("--route", choices=("stock", "skill"), default="stock")
    p_smoke.add_argument("--calibration", type=Path, default=None,
                         help="WS-D gate overrides (e.g. calibration/chat-provisional.json)")

    args = ap.parse_args()
    rekal = Path(args.rekal).resolve()
    if args.cmd == "search":
        env = rekal_env(args.repo.resolve(), None, rekal)
        cal = load_calibration(args.calibration)
        print(json.dumps(
            search(rekal, args.repo.resolve(), args.query, args.limit, env, args.route, cal),
            indent=2,
        ))
    elif args.cmd == "smoke":
        smoke_one(
            rekal,
            args.workdir.resolve(),
            args.input,
            args.conversation_id,
            args.out,
            limit_questions=args.limit_questions,
            route=args.route,
            calibration_path=args.calibration,
        )


if __name__ == "__main__":
    main()
