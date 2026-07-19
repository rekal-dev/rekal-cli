#!/usr/bin/env python3
"""LoCoMo route-logic eval — no LLM.

For every LoCoMo question (minus known-bad), run stock recall in the ingested
conversation repo, pipe through the checkout's `route.py` under chat
calibration, and report gate + token cost (raw recall JSON vs route stdout).

Pass / fail (deterministic):
  * Gold silence (adversarial / abstention) → must SILENCE
  * Otherwise (answerable or open-domain that needs an answer) → must INJECT
  * KNOWLEDGE is always a fail on LoCoMo (marker prose, not dialogue memory)
  * route.py exit 2 is always a fail

  python3 scripts/industry-bench/eval_locomo_route.py
  python3 scripts/industry-bench/eval_locomo_route.py --limit 40 --workers 4
"""
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
IB = REPO_ROOT / "scripts" / "industry-bench"
ROUTE = REPO_ROOT / "cmd" / "rekal" / "cli" / "skill" / "skills" / "rekal" / "scripts" / "route.py"
REKAL = REPO_ROOT / "rekal"
LOCOMO_ROOT = Path.home() / "imb-locomo"
CONVS = IB / "datasets" / "data" / "locomo-conversations.jsonl"
KNOWN_BAD = IB / "datasets" / "locomo-known-bad.jsonl"
CHAT_CAL = IB / "calibration" / "locomo-route-eval.json"


def toks(s: str) -> int:
    try:
        import tiktoken

        return len(tiktoken.get_encoding("cl100k_base").encode(s))
    except Exception:
        return max(1, round(len(s) / 4))


def load_tasks(limit: int | None) -> list[dict]:
    bad = set()
    if KNOWN_BAD.exists():
        for line in KNOWN_BAD.read_text().splitlines():
            if not line.strip():
                continue
            r = json.loads(line)
            bad.add((r["conversation_id"], r["question_id"]))
    tasks = []
    for c in map(json.loads, CONVS.read_text().splitlines()):
        cid = c["conversation_id"]
        for q in c["questions"]:
            if (cid, q["question_id"]) in bad:
                continue
            evid = q.get("evidence_session_ids") or []
            cat = q.get("category") or ""
            gold = str(q.get("answer") or "").strip()
            gold_l = gold.lower()
            # Category 5 adversarials (and explicit unknown gold) must stay silent.
            gold_silence = cat == "adversarial" or gold_l in {
                "unknown",
                "n/a",
                "none",
                "no answer",
                "not enough information",
            }
            tasks.append(
                {
                    "id": f'{cid}:{q["question_id"]}',
                    "conv": cid,
                    "category": cat,
                    "question": q["question"],
                    "evidence": evid,
                    "answerable": bool(evid),
                    "gold_silence": gold_silence,
                    "want_gate": "SILENCE" if gold_silence else "INJECT",
                }
            )
    if limit is not None:
        tasks = tasks[:limit]
    return tasks


def chat_env(calibration: Path) -> tuple[dict, dict]:
    env = dict(os.environ)
    env["PATH"] = f"{REKAL.parent}{os.pathsep}{env.get('PATH', '')}"
    env["REKAL_BENCH"] = "1"  # never checkpoint into bench repos
    cal = json.loads(calibration.read_text())
    for k, v in (cal.get("gate") or {}).items():
        env[f"REKAL_HUNT_{k}"] = str(v)
    return env, cal.get("weights") or {}


def evidence_marker(evid: str, fp: str) -> bool:
    import re

    return bool(re.search(rf"-{re.escape(evid)}\.md$", fp or ""))


def has_evidence(results: list, evid: list[str], k: int) -> bool:
    if not evid:
        return False
    for r in results[:k]:
        files = (r.get("session") or {}).get("files") or []
        for e in evid:
            if any(evidence_marker(e, str(f or "")) for f in files):
                return True
    return False


def _keywords(q: str) -> str:
    stop = set(
        """a an the of to in on at for and or but if is are was were be been being do does did
        have has had i you he she it we they me my your his her their our this that these those what when
        where who whom which why how with about from into over under again then once will would can
        could should may might must not no yes as by so than too very just also there here them us""".split()
    )
    import re

    toks_ = re.findall(r"[A-Za-z0-9']+", q.lower())
    kept = [t for t in toks_ if t not in stop and len(t) > 2]
    return " ".join(kept) if kept else q


def _recall_route(repo: Path, query: str, env: dict, weights: dict, limit: int) -> dict:
    """One recall→route pass. Returns gate/payload/token fields or error."""
    wj = json.dumps(weights, separators=(",", ":"), sort_keys=True)
    args = [query, "--limit", str(limit)]
    if wj and wj != "{}":
        args = ["--weights", wj, *args]

    last_err = ""
    raw = ""
    for attempt in range(6):
        p = subprocess.run(
            [str(REKAL), *args],
            cwd=repo,
            env=env,
            capture_output=True,
            text=True,
            timeout=120,
        )
        out = (p.stdout or "") + (p.stderr or "")
        if p.returncode == 0 and "{" in (p.stdout or ""):
            raw = p.stdout
            break
        low = out.lower()
        if "lock" in low or "another rekal" in low or "rebuilding" in low or "index not built" in low:
            time.sleep(1.2 * (attempt + 1))
            last_err = out[-400:]
            continue
        return {"error": f"recall rc={p.returncode}: {out[-400:]}", "gate": "ERROR", "route_rc": 2}
    else:
        return {"error": f"recall retries: {last_err}", "gate": "ERROR", "route_rc": 2}

    try:
        s, e = raw.find("{"), raw.rfind("}")
        body = raw[s : e + 1]
        payload = json.loads(body)
    except json.JSONDecodeError as err:
        return {"error": f"json: {err}", "gate": "ERROR", "route_rc": 2}

    rp = subprocess.run(
        ["python3", str(ROUTE)],
        input=body,
        capture_output=True,
        text=True,
        env=env,
        timeout=30,
    )
    rout = (rp.stdout or "").strip()
    line1 = rout.splitlines()[0] if rout else ""
    if line1.startswith("INJECT"):
        gate = "INJECT"
    elif line1.startswith("KNOWLEDGE"):
        gate = "KNOWLEDGE"
    elif line1.startswith("SILENCE"):
        gate = "SILENCE"
    else:
        gate = "ERROR"

    results = payload.get("results") or []
    inclusive = gate == "INJECT" and any(l.startswith("KNOWLEDGE") for l in rout.splitlines()[1:])
    return {
        "gate": gate,
        "route_rc": rp.returncode,
        "inclusive_knowledge": inclusive,
        "n_results": len(results),
        "top_conf": float(results[0].get("confidence") or 0) if results else 0.0,
        "evidence@5": has_evidence(results, [], 5),  # filled by caller
        "results": results,
        "raw_tok": toks(body),
        "route_tok": toks(rout),
        "rout": rout,
        "query_used": query,
        "error": None,
    }


def run_one(task: dict, env: dict, weights: dict, limit: int) -> dict:
    repo = LOCOMO_ROOT / task["conv"] / "repo"
    if not repo.is_dir():
        return {**task, "error": f"missing repo {repo}", "gate": "ERROR", "route_rc": 2}

    t0 = time.time()
    primary = _recall_route(repo, task["question"], env, weights, limit)
    if primary.get("error"):
        return {**task, **primary}

    # Skill: widen before SILENCE — keyword reformulation (multi-lookup v1).
    lookups = [primary]
    if primary["gate"] == "SILENCE":
        kw = _keywords(task["question"])
        if kw and kw.lower() != task["question"].lower():
            alt = _recall_route(repo, kw, env, weights, limit)
            if not alt.get("error"):
                lookups.append(alt)

    # Prefer first non-SILENCE; else keep primary.
    chosen = primary
    for L in lookups:
        if L["gate"] in ("INJECT", "KNOWLEDGE"):
            chosen = L
            break

    evid = task["evidence"]
    results = chosen.get("results") or []
    return {
        **task,
        "gate": chosen["gate"],
        "route_rc": chosen["route_rc"],
        "inclusive_knowledge": chosen.get("inclusive_knowledge"),
        "n_results": chosen.get("n_results"),
        "top_conf": chosen.get("top_conf"),
        "evidence@5": has_evidence(results, evid, 5),
        "evidence@20": has_evidence(results, evid, 20),
        "raw_tok": sum(L.get("raw_tok") or 0 for L in lookups),
        "route_tok": sum(L.get("route_tok") or 0 for L in lookups),
        "lookups": len(lookups),
        "query_used": chosen.get("query_used"),
        "elapsed_s": round(time.time() - t0, 3),
        "error": None,
    }


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--limit", type=int, default=None, help="cap questions (dev sample)")
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--n", type=int, default=20, help="rekal -n / --limit")
    ap.add_argument("--out", type=Path, default=IB / "runs" / "locomo-route-inclusive")
    ap.add_argument(
        "--calibration",
        type=Path,
        default=CHAT_CAL,
        help="gate/weights JSON (default: locomo-route-eval.json)",
    )
    args = ap.parse_args()
    calibration = args.calibration

    if not REKAL.exists():
        print(f"missing binary: {REKAL} (build first)", file=sys.stderr)
        return 2
    if not ROUTE.exists():
        print(f"missing route.py: {ROUTE}", file=sys.stderr)
        return 2
    if not LOCOMO_ROOT.is_dir():
        print(f"missing LoCoMo repos: {LOCOMO_ROOT}", file=sys.stderr)
        return 2
    if not calibration.is_file():
        print(f"missing calibration: {calibration}", file=sys.stderr)
        return 2

    tasks = load_tasks(args.limit)
    env, weights = chat_env(calibration)
    print(f"tasks={len(tasks)} workers={args.workers} limit={args.n}")
    print(f"chat gates: CONF_MIN={env.get('REKAL_HUNT_CONF_MIN')} "
          f"CONF_SOFT={env.get('REKAL_HUNT_CONF_SOFT')} GAP_MIN={env.get('REKAL_HUNT_GAP_MIN')}")

    rows: list[dict] = []
    with ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {ex.submit(run_one, t, env, weights, args.n): t for t in tasks}
        done = 0
        for fut in as_completed(futs):
            rows.append(fut.result())
            done += 1
            if done % 50 == 0 or done == len(tasks):
                print(f"  … {done}/{len(tasks)}", flush=True)

    rows.sort(key=lambda r: r["id"])
    args.out.mkdir(parents=True, exist_ok=True)
    out_path = args.out / "route.jsonl"
    with out_path.open("w") as f:
        for r in rows:
            f.write(json.dumps(r) + "\n")

    # Aggregate
    from collections import Counter

    def route_ok(r: dict) -> bool:
        """INJECT when an answer is expected; SILENCE when gold is abstention.
        KNOWLEDGE always fails on LoCoMo (synthetic marker prose)."""
        if r.get("gate") == "ERROR" or r.get("route_rc") == 2:
            return False
        want = r.get("want_gate") or ("SILENCE" if r.get("gold_silence") else "INJECT")
        return r.get("gate") == want

    for r in rows:
        r["pass"] = route_ok(r)

    gates = Counter(r["gate"] for r in rows)
    want_inj = [r for r in rows if r.get("want_gate") == "INJECT"]
    want_sil = [r for r in rows if r.get("want_gate") == "SILENCE"]
    ans = [r for r in rows if r.get("answerable")]
    passed = [r for r in rows if r.get("pass")]
    failed = [r for r in rows if not r.get("pass")]
    fail_want_inj = [r for r in failed if r.get("want_gate") == "INJECT"]
    fail_want_sil = [r for r in failed if r.get("want_gate") == "SILENCE"]
    errors = [r for r in rows if r["gate"] == "ERROR" or r.get("route_rc") == 2]
    inj = [r for r in rows if r["gate"] == "INJECT"]
    knowledge_fails = [r for r in failed if r.get("gate") == "KNOWLEDGE"]
    evid20 = sum(1 for r in ans if r.get("evidence@20"))
    evid20_inj = sum(1 for r in ans if r["gate"] == "INJECT" and r.get("evidence@20"))
    inclusive = sum(1 for r in inj if r.get("inclusive_knowledge"))

    raw_tok = sum(r.get("raw_tok") or 0 for r in rows)
    route_tok = sum(r.get("route_tok") or 0 for r in rows)

    summary = {
        "n": len(rows),
        "gates": dict(gates),
        "pass": len(passed),
        "fail": len(failed),
        "pass_rate": round(len(passed) / max(1, len(rows)), 4),
        "want_inject": len(want_inj),
        "want_inject_pass": sum(1 for r in want_inj if r.get("pass")),
        "want_inject_pass_rate": round(sum(1 for r in want_inj if r.get("pass")) / max(1, len(want_inj)), 4),
        "want_silence": len(want_sil),
        "want_silence_pass": sum(1 for r in want_sil if r.get("pass")),
        "want_silence_pass_rate": round(sum(1 for r in want_sil if r.get("pass")) / max(1, len(want_sil)), 4),
        "knowledge_as_fail": len(knowledge_fails),
        "errors": len(errors),
        "inject": len(inj),
        "inject_with_inclusive_knowledge": inclusive,
        "evidence@20_answerable": evid20,
        "evidence@20_rate": round(evid20 / max(1, len(ans)), 4),
        "evidence@20_among_inject": evid20_inj,
        "raw_tok_total": raw_tok,
        "route_tok_total": route_tok,
        "token_saved_pct": round(100 * (1 - route_tok / max(1, raw_tok)), 2),
        "token_ratio_raw_over_route": round(raw_tok / max(1, route_tok), 2),
        "avg_raw_tok": round(raw_tok / max(1, len(rows)), 1),
        "avg_route_tok": round(route_tok / max(1, len(rows)), 1),
        "chat_calibration": str(calibration),
        "route": str(ROUTE),
        "rule": "want_silence→SILENCE; else→INJECT; KNOWLEDGE always fail",
    }
    (args.out / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    # rewrite jsonl with pass flags
    with out_path.open("w") as f:
        for r in rows:
            f.write(json.dumps(r) + "\n")

    print("\n=== LoCoMo route summary ===")
    for k, v in summary.items():
        print(f"  {k}: {v}")

    if fail_want_inj[:15]:
        print("\n=== want INJECT but got other (max 15) ===")
        for r in fail_want_inj[:15]:
            print(
                f"  {r['id']} gate={r['gate']} cat={r['category']} "
                f"top_conf={r.get('top_conf')} ev@20={r.get('evidence@20')} "
                f"q={r['question'][:55]!r}"
            )

    if fail_want_sil[:15]:
        print("\n=== want SILENCE but got other (max 15) ===")
        for r in fail_want_sil[:15]:
            print(
                f"  {r['id']} gate={r['gate']} cat={r['category']} "
                f"top_conf={r.get('top_conf')} q={r['question'][:55]!r}"
            )

    if errors:
        print(f"\n=== ERRORS ({len(errors)}) ===")
        for r in errors[:10]:
            print(f"  {r['id']}: {r.get('error')}")

    ok = not failed
    print(
        "\n"
        + (
            "PASS: every question matched want_gate (INJECT / SILENCE)"
            if ok
            else f"FAIL: {len(failed)}/{len(rows)} mismatches"
        )
    )
    print(f"wrote {out_path}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
