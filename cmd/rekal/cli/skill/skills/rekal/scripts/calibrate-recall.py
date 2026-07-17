#!/usr/bin/env python3
"""Propose (and optionally apply) local recall calibration.

Agent-facing: diagnose miss patterns from smoke manifests or scoring lineage,
emit a profile JSON (gates + weights), optionally write into the current
repo's .rekal/config.json (weights only) and .rekal/calibration/.

Does not modify data.db. Does not change shipped hunt-gate defaults.
Gate overrides are emitted as REKAL_HUNT_* env suggestions.

See docs/design/skill-adaptive-recall.md.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
from collections import Counter
from pathlib import Path


CODING_GATES = {
    "CONF_MIN": 0.70,
    "CONF_SOFT": 0.68,
    "GAP_MIN": 0.04,
    "MASS_MIN": 3.5,
    "KNOWLEDGE_MIN": 0.40,
}

CHAT_GATES = {
    "CONF_MIN": 0.55,
    "CONF_SOFT": 0.50,
    "GAP_MIN": 0.03,
    "MASS_MIN": 0.0,
    "KNOWLEDGE_MIN": 0.90,
}

# Query-time weights (search.DefaultWeights) — chat leans a bit more LSA for
# weak-lexical multi-hop; coding keeps shipped defaults.
CODING_WEIGHTS = {
    "bm25": 0.35,
    "lsa": 0.10,
    "semantic": 0.55,
    "steering_boost": 1.3,
    "summary_boost": 1.15,
    "subagent_downweight": 0.7,
    "facet_boost": 0.3,
}

CHAT_WEIGHTS = {
    "bm25": 0.30,
    "lsa": 0.20,
    "semantic": 0.50,
    "steering_boost": 1.0,
    "summary_boost": 1.0,
    "subagent_downweight": 1.0,
    "facet_boost": 0.1,
}

MULTI_HOP_WEIGHTS = {
    "bm25": 0.25,
    "lsa": 0.25,
    "semantic": 0.50,
    "steering_boost": 1.2,
    "summary_boost": 1.1,
    "subagent_downweight": 0.8,
    "facet_boost": 0.2,
}


def load_manifests(paths: list[Path]) -> list[dict]:
    rows = []
    for p in paths:
        if p.is_dir():
            for m in p.glob("**/manifest.json"):
                rows.append(json.loads(m.read_text()))
            for q in p.glob("**/per_question.jsonl"):
                for line in q.read_text().splitlines():
                    if line.strip():
                        rows.append({"_per_q": json.loads(line)})
        elif p.suffix == ".json":
            rows.append(json.loads(p.read_text()))
        elif p.suffix == ".jsonl" or p.name.endswith(".jsonl"):
            for line in p.read_text().splitlines():
                if line.strip():
                    rows.append({"_per_q": json.loads(line)})
    return rows


def summarize_misses(rows: list[dict]) -> dict:
    reasons: Counter[str] = Counter()
    cats: Counter[str] = Counter()
    n_q = 0
    n_hit = 0
    for r in rows:
        if "_per_q" in r:
            q = r["_per_q"]
            n_q += 1
            reason = q.get("miss_reason") or ("hit" if q.get("evidence_at_5") else "unknown")
            reasons[reason] += 1
            cats[q.get("category") or "unknown"] += 1
            if reason == "hit" or q.get("evidence_at_5"):
                n_hit += 1
        elif "miss_reasons" in r:
            for k, v in (r.get("miss_reasons") or {}).items():
                reasons[k] += int(v)
            n_q += int(r.get("n_questions") or 0)
    return {
        "n_questions": n_q,
        "n_hit_proxy": n_hit,
        "miss_reasons": dict(reasons),
        "categories": dict(cats),
    }


def choose_profile(summary: dict) -> str:
    reasons = summary.get("miss_reasons") or {}
    cats = summary.get("categories") or {}
    multi = sum(v for k, v in cats.items() if "multi" in k or k == "multi-hop")
    deep = reasons.get("deep_rank_lt5", 0) + reasons.get("deep_rank_lt10", 0)
    true_miss = reasons.get("true_miss", 0)
    gate = reasons.get("gate_blocked", 0)
    total = max(1, sum(reasons.values()))

    if multi / max(1, sum(cats.values())) >= 0.25 or deep / total >= 0.25:
        return "multi-hop"
    if gate / total >= 0.15 or true_miss / total <= 0.05 and deep / total >= 0.1:
        # lots of deep ranks / few true misses → chat-like dilution
        return "chat"
    if not reasons:
        return "coding"
    return "chat" if deep + true_miss > reasons.get("hit", 0) * 0.3 else "coding"


def build_proposal(profile: str, summary: dict, source: str) -> dict:
    if profile == "chat":
        gates, weights = dict(CHAT_GATES), dict(CHAT_WEIGHTS)
    elif profile == "multi-hop":
        gates, weights = dict(CHAT_GATES), dict(MULTI_HOP_WEIGHTS)
        gates["CONF_MIN"] = 0.52
    else:
        gates, weights = dict(CODING_GATES), dict(CODING_WEIGHTS)

    body = {
        "profile": profile,
        "gate": gates,
        "weights": weights,
        "summary": summary,
        "source": source,
        "notes": (
            "Proposal only. Apply with --apply to write local .rekal/config.json weights "
            "and .rekal/calibration/active-profile.json. Export REKAL_HUNT_* for gates. "
            "Do not commit. Do not tune on a frozen test split."
        ),
    }
    raw = json.dumps({"gate": gates, "weights": weights, "profile": profile}, sort_keys=True)
    body["proposal_sha256"] = hashlib.sha256(raw.encode()).hexdigest()[:16]
    return body


def env_exports(gates: dict) -> str:
    lines = []
    for k, v in gates.items():
        lines.append(f"export REKAL_HUNT_{k}={v}")
    return "\n".join(lines)


def apply_local(git_root: Path, proposal: dict) -> None:
    rekal = git_root / ".rekal"
    if not rekal.is_dir():
        raise SystemExit(f"no .rekal under {git_root} — run rekal init first")
    cfg_path = rekal / "config.json"
    cfg = {}
    if cfg_path.exists():
        cfg = json.loads(cfg_path.read_text())
    cfg["weights"] = proposal["weights"]
    cfg_path.write_text(json.dumps(cfg, indent=2) + "\n")

    cal = rekal / "calibration"
    cal.mkdir(exist_ok=True)
    active = {
        "profile": proposal["profile"],
        "proposal_sha256": proposal["proposal_sha256"],
        "gate": proposal["gate"],
        "weights": proposal["weights"],
        "applied_from": proposal.get("source"),
    }
    (cal / "active-profile.json").write_text(json.dumps(active, indent=2) + "\n")
    (cal / "last-proposal.json").write_text(json.dumps(proposal, indent=2) + "\n")


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--from-smoke", type=Path, nargs="*", default=[],
                    help="manifest.json / per_question.jsonl / run dirs")
    ap.add_argument("--profile", choices=("coding", "chat", "multi-hop", "auto"), default="auto")
    ap.add_argument("--apply", action="store_true", help="write local .rekal/config.json weights")
    ap.add_argument("--git-root", type=Path, default=None)
    ap.add_argument("--print-env", action="store_true", help="print REKAL_HUNT_* exports")
    args = ap.parse_args()

    rows = load_manifests(args.from_smoke) if args.from_smoke else []
    summary = summarize_misses(rows) if rows else {"n_questions": 0, "miss_reasons": {}, "categories": {}}
    profile = choose_profile(summary) if args.profile == "auto" else args.profile
    source = ",".join(str(p) for p in args.from_smoke) if args.from_smoke else "defaults"
    proposal = build_proposal(profile, summary, source)

    print(json.dumps(proposal, indent=2))
    if args.print_env:
        print("\n# gate env (export before recall-route / hunt-gate):", file=sys.stderr)
        print(env_exports(proposal["gate"]), file=sys.stderr)

    if args.apply:
        root = args.git_root or Path(os.environ.get("GIT_ROOT") or Path.cwd())
        # resolve to git root if possible
        p = root.resolve()
        while p != p.parent and not (p / ".git").exists() and not (p / ".rekal").exists():
            p = p.parent
        apply_local(p, proposal)
        print(f"\napplied profile={proposal['profile']} sha={proposal['proposal_sha256']} → {p}/.rekal/", file=sys.stderr)


if __name__ == "__main__":
    main()
