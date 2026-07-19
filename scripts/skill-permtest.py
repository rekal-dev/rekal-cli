#!/usr/bin/env python3
"""Permutation / property test for the Rekal skill routes.

Plan
====
The skill's routing surface is `rekal <flags> "<query>" | route.py`. Its
correctness is not a fixed expected string (results are corpus-dependent) but a
set of INVARIANTS that must hold across the permutation of inputs. We permute:

  * query ARCHETYPE  — real-domain / why / convention / code-symbol / junk /
                       analytical / thin
  * FLAGS            — plain / -n {1,5,100,0,-1} / --explain / --file {hit,miss}
                       / --role human_steering
  * SEMANTIC state   — daemon warm (this run) vs the cold `semantic:warming`
                       signal (checked separately)

and assert invariants per cell. Executed against the live corpus (74 sessions,
540 knowledge chunks, full nomic semantic).

Invariants
==========
  I1  recall emits parseable JSON for every non-error query
  I2  route.py exit code matches its verdict (0 INJECT/KNOWLEDGE, 1 SILENCE)
  I3  verdict is on line 1 and is one of INJECT / KNOWLEDGE / SILENCE
  I4  route.py never dies (exit 2) on real recall JSON
  I5  archetype laws:
        junk        -> never INJECT (no confident episode)
        thin        -> SILENCE
        real-domain -> not SILENCE (episode or knowledge answers)
  I6  determinism: same (query,flags) twice -> identical verdict (warm)
  I7  -n 0 -> empty result set -> SILENCE ; -n -1 -> recall rejects (exit!=0)
  I8  daemon warm -> no `SEMANTIC warming` note
"""
from __future__ import annotations

import json
import os
import subprocess
import sys

# Usage: run from a git repo where Rekal is initialized and the index is
# populated (rekal init && rekal sync), with the nomic daemon warm for the
# semantic invariants. Prefers the installed skill copy, falls back to source.
_CANDIDATES = (
    ".claude/skills/rekal/scripts/route.py",
    "cmd/rekal/cli/skill/skills/rekal/scripts/route.py",
)
ROUTE = next((p for p in _CANDIDATES if os.path.exists(p)), _CANDIDATES[0])

# Archetype -> representative queries. Junk topics are deliberately NOT ones
# discussed in this session (avoid self-capture contamination).
ARCHETYPES = {
    "real-domain": [
        "merged-only sharing export gate",
        "knowledge layer chunk vector cosine",
        "facet FTS index recall boost",
        "scrub redact secrets before insert",
        "LSA query projection cache",
    ],
    "why": [
        "why split data.db and index.db",
        "why saturating bm25 confidence not max-normalized",
    ],
    "convention": [
        "worktree shared store main worktree",
        "two-tier config precedence local global",
    ],
    "code-symbol": ["func QuerySessionContent", "PopulateFacetText", "EnsureKnowledgeSchema"],
    "junk": [
        "boiling point of mercury in kelvin",
        "migratory patterns of arctic terns",
        "how to tune a violin by ear",
        "rules of cricket lbw dismissal",
        "beef bourguignon braising time",
    ],
    "analytical": ["how many sessions total and date range", "list every human steering turn"],
    "thin": ["", " ", "a"],
}


def run(args: list[str]) -> tuple[int, str, str]:
    p = subprocess.run(["rekal", *args], capture_output=True, text=True, timeout=60)
    return p.returncode, p.stdout, p.stderr


def route(recall_json: str) -> tuple[int, str]:
    p = subprocess.run(["python3", ROUTE], input=recall_json, capture_output=True, text=True, timeout=30)
    return p.returncode, (p.stdout or p.stderr)


def verdict_of(out: str) -> str:
    line1 = out.splitlines()[0] if out.strip() else ""
    for v in ("INJECT", "KNOWLEDGE", "SILENCE"):
        if line1.startswith(v):
            return v
    return "?"


FAILS: list[str] = []
N = 0


def check(cond: bool, label: str) -> None:
    global N
    N += 1
    if not cond:
        FAILS.append(label)


def cell(query: str, flags: list[str]) -> dict | None:
    rc, out, err = run([*flags, query])
    tag = f'{flags}"{query[:24]}"'
    # -n -1 is expected to be rejected (I7); handle before JSON parse.
    if "-1" in flags:
        check(rc != 0, f"I7 -n -1 rejected {tag}")
        return None
    if rc != 0:
        FAILS.append(f"I1 recall failed rc={rc} {tag}: {err[:60]}")
        return None
    try:
        data = json.loads(out)
    except json.JSONDecodeError:
        FAILS.append(f"I1 non-JSON {tag}")
        return None
    rrc, rout = route(out)
    v = verdict_of(rout)
    check(v in ("INJECT", "KNOWLEDGE", "SILENCE"), f"I3 verdict-token {tag} got {v!r}")
    check(rrc != 2, f"I4 route crash {tag}")
    if v in ("INJECT", "KNOWLEDGE"):
        check(rrc == 0, f"I2 exit {tag} v={v} rc={rrc}")
    elif v == "SILENCE":
        check(rrc == 1, f"I2 exit {tag} v={v} rc={rrc}")
    check("SEMANTIC warming" not in rout, f"I8 warm-no-note {tag}")
    return {"verdict": v, "results": len(data.get("results") or []), "route_rc": rrc}


def main() -> int:
    print("=== Matrix A: archetype × plain ===")
    verdicts: dict[str, list[str]] = {}
    for arch, qs in ARCHETYPES.items():
        vs = []
        for q in qs:
            r = cell(q, [])
            if r is None:
                continue
            vs.append(r["verdict"])
            # I5 archetype laws
            if arch == "junk":
                check(r["verdict"] != "INJECT", f'I5 junk-not-inject "{q}"')
            if arch == "thin":
                check(r["verdict"] == "SILENCE", f'I5 thin-silence "{q!r}"')
            if arch == "real-domain":
                check(r["verdict"] != "SILENCE", f'I5 real-not-silence "{q}"')
        verdicts[arch] = vs
        print(f"  {arch:12} -> {vs}")

    print("=== Matrix B: real query × flags ===")
    base = "merged-only sharing export gate"
    for flags in ([], ["-n", "1"], ["-n", "5"], ["-n", "100"], ["-n", "0"],
                  ["--explain"], ["--file", "route"], ["--file", "zzznotafile"],
                  ["-n", "-1"]):
        r = cell(base, flags)
        if r is None:
            print(f"  {str(flags):28} -> (rejected/none)")
            continue
        print(f"  {str(flags):28} -> {r['verdict']:9} results={r['results']}")
        if flags == ["-n", "0"]:
            check(r["results"] == 0 and r["verdict"] == "SILENCE", "I7 -n0 empty-silence")

    print("=== Matrix C: determinism (real + junk, x2) ===")
    for q in ["knowledge layer chunk vector cosine", "how to tune a violin by ear"]:
        a = cell(q, [])
        b = cell(q, [])
        if a and b:
            check(a["verdict"] == b["verdict"], f'I6 determinism "{q[:20]}" {a["verdict"]}vs{b["verdict"]}')
            print(f'  "{q[:28]:28}" -> {a["verdict"]} == {b["verdict"]}')

    print("=== Matrix D: drill route × --role (query --session) ===")
    # Pick a real session with the most turns to drill.
    rc, out, _ = run(["query", "SELECT id FROM sessions ORDER BY captured_at DESC LIMIT 1"])
    sid = ""
    try:
        sid = json.loads(out.splitlines()[0])["id"] if out.strip() else ""
    except (json.JSONDecodeError, KeyError, IndexError):
        pass
    if sid:
        for role in ("human", "assistant", "human_steering", "summary", ""):
            args = ["query", "--session", sid] + (["--role", role] if role else [])
            p = subprocess.run(["rekal", *args], capture_output=True, text=True, timeout=30)
            ok = p.returncode == 0
            try:
                turns = len((json.loads(p.stdout).get("turns") or [])) if p.stdout.strip() else 0
                # I-drill: turns must be an array, never JSON null.
                check("\"turns\": null" not in p.stdout, f"I-drill turns-not-null role={role or 'all'}")
            except json.JSONDecodeError:
                turns = -1
            check(ok, f"I-drill role={role or 'all'} rc={p.returncode}")
            check(turns >= 0, f"I-drill role={role or 'all'} valid-json")
            print(f"  --role {role or '(all)':14} -> rc={p.returncode} turns={turns}")
        # --role without --session must be rejected (the validation we found).
        p = subprocess.run(["rekal", "query", "--role", "human"], capture_output=True, text=True, timeout=30)
        check(p.returncode != 0, "I-drill role-requires-session rejected")
        print(f"  --role human (no --session) -> rc={p.returncode} (expect !=0)")

    print(f"\n=== RESULT: {N - len(FAILS)}/{N} invariant checks passed ===")
    if FAILS:
        print("FAILURES:")
        for f in FAILS:
            print("  ✗", f)
        return 1
    print("ALL INVARIANTS HOLD")
    return 0


if __name__ == "__main__":
    sys.exit(main())
