#!/usr/bin/env python3
"""mine_wild.py <outdir> — in-the-wild retrieval eval, self-labeled by real
agent behavior (06-eval-strategy.md §4c). No synthetic queries, no LLM.

The ledger records every real `rekal "<q>"` an agent ran and every
`rekal query --session <sid>` it ran next. A recall followed by a drill is an
implicit relevance label (LRAT): the drilled session is what the agent judged
worth reading for that real query. We pair them per session and emit a
queries file that runs through run_rung1.py / score.py unchanged — the realest
possible retrievability test, graded against actual behavior.

Writes <outdir>/wild/queries.jsonl ({qid, task:"wild", split:"test", query,
gold:[sid], cross_repo}) and wild-meta.json. cross_repo flags drills into an
imported (origin-labeled) session — direct in-the-wild evidence for §4d.
Stdlib only.
"""
import hashlib
import json
import pathlib
import re
import subprocess
import sys

SUBCOMMANDS = {"query", "init", "clean", "index", "log", "push", "sync",
               "version", "checkpoint", "help"}
ULID = re.compile(r"\b([0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{26})\b")
QUOTED = re.compile(r"\"([^\"]+)\"|'([^']+)'")


def q(sql: str) -> list[dict]:
    out = subprocess.run(
        ["rekal", "query", "--index", sql], capture_output=True, text=True, timeout=300
    )
    if out.returncode != 0:
        sys.exit(f"rekal query failed:\n{out.stderr}")
    return [json.loads(l) for l in out.stdout.splitlines() if l.strip()]


def recall_query(cmd: str) -> str | None:
    """Extract the query text from a `rekal <query>` recall, or None if the
    command is a subcommand (query/init/...) rather than a recall."""
    m = re.match(r"\s*rekal\b(.*)", cmd, re.DOTALL)
    if not m:
        return None
    rest = m.group(1).strip()
    if not rest:
        return None
    first = rest.split(maxsplit=1)[0]
    if first.strip("\"'") in SUBCOMMANDS:
        return None
    # Prefer the quoted query (keeps intra-query hyphens); else drop leading
    # flags and their numeric values and take the remainder.
    qm = QUOTED.search(rest)
    if qm:
        return (qm.group(1) or qm.group(2)).strip() or None
    out, expect_val = [], False
    for tok in rest.split():
        if tok.startswith("-"):
            expect_val = tok in ("-n", "--limit")
            continue
        if expect_val and tok.isdigit():
            expect_val = False
            continue
        expect_val = False
        out.append(tok)
    return " ".join(out).strip() or None


def drilled_sid(cmd: str) -> str | None:
    """The session id from `rekal query --session <sid>`, else None."""
    if "query" not in cmd or "--session" not in cmd:
        return None
    m = ULID.search(cmd.split("--session", 1)[1])
    return m.group(1) if m else None


def main() -> None:
    outdir = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".")
    wild = outdir / "wild"
    wild.mkdir(parents=True, exist_ok=True)

    rows = q("SELECT session_id AS sid, call_order AS ord, cmd_prefix AS cmd "
             "FROM tool_calls_index WHERE cmd_prefix ILIKE 'rekal%' "
             "ORDER BY session_id, call_order")
    origins = {r["sid"]: r.get("origin") for r in q(
        "SELECT session_id AS sid, origin FROM session_facets")}

    # Per session, pair each drill with the most recent preceding recall.
    pairs: list[tuple[str, str]] = []
    by_session: dict[str, list[dict]] = {}
    for r in rows:
        by_session.setdefault(r["sid"], []).append(r)
    for calls in by_session.values():
        last_query: str | None = None
        for c in sorted(calls, key=lambda x: int(x["ord"])):
            cmd = c.get("cmd") or ""
            sid = drilled_sid(cmd)
            if sid is not None:
                if last_query and sid in origins:  # drilled a known session
                    pairs.append((last_query, sid))
                continue
            rq = recall_query(cmd)
            if rq:
                last_query = rq

    seen, queries, cross = set(), [], 0
    for query, sid in pairs:
        key = (query, sid)
        if key in seen:
            continue
        seen.add(key)
        is_cross = bool(origins.get(sid))  # non-null origin = imported repo
        cross += is_cross
        qid = hashlib.sha1(f"wild{query}{sid}".encode()).hexdigest()[:12]
        queries.append({"qid": qid, "task": "wild", "split": "test",
                        "query": query, "gold": [sid], "cross_repo": is_cross})

    (wild / "queries.jsonl").write_text(
        "".join(json.dumps(x) + "\n" for x in queries))
    meta = {"real_query_drill_pairs": len(queries),
            "cross_repo_drills": cross,
            "recall_invocations_seen": sum(
                1 for r in rows if recall_query(r.get("cmd") or ""))}
    (wild / "wild-meta.json").write_text(json.dumps(meta, indent=2))
    print(f"wild: {len(queries)} real query→drill pairs "
          f"({cross} into imported/cross-repo sessions). "
          f"Run: run_rung1.py {wild} && score.py {wild}", file=sys.stderr)


if __name__ == "__main__":
    main()
