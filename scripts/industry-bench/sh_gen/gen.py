#!/usr/bin/env python3
"""sh-gen: synthetic-history generator (WS-B).

Turns a benchmark corpus in the interchange format (datasets/SCHEMA.md) into
per-conversation synthetic git repos whose commits carry the benchmark
sessions through Rekal's production checkpoint pipeline. Contract:
docs/research/industry-bench/02-adapter-architecture.md §1.

Stdlib only. Deterministic given the input (session/turn UUIDs are uuid5 of
stable keys; commit dates come from the dataset).
"""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

# Substrings that make session.SkipCapture treat a transcript cwd as bench
# pollution (cmd/rekal/cli/session/bench.go). A workdir under such a path
# would be silently skipped by checkpoint, so refuse it up front.
FORBIDDEN_PATH_PARTS = ("/scripts/bench", "rekal-bench", "/.rekal-bench")

# Env vars that make SkipCapture true for EVERY payload. They protect a
# developer's real repos while harness scripts run under a coding agent;
# they must never leak into the ingest environment.
CAPTURE_KILL_SWITCHES = ("REKAL_BENCH", "REKAL_SKIP_CHECKPOINT")

UUID_NS = uuid.uuid5(uuid.NAMESPACE_URL, "rekal-industry-bench")


def sanitize_repo_path(repo_path: str) -> str:
    """Port of session.SanitizeRepoPath (cmd/rekal/cli/session/find.go):
    every non-alphanumeric character becomes a dash."""
    return re.sub(r"[^a-zA-Z0-9]", "-", repo_path)


def ingest_env(base: dict, claude_config_dir: Path, rekal_dir: Path) -> dict:
    env = dict(base)
    for k in CAPTURE_KILL_SWITCHES:
        env.pop(k, None)
    env["CLAUDE_CONFIG_DIR"] = str(claude_config_dir)
    env["PATH"] = f"{rekal_dir}{os.pathsep}{env.get('PATH', '')}"
    return env


def run(cmd, cwd, env, check=True):
    p = subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True)
    if check and p.returncode != 0:
        raise RuntimeError(
            f"command failed ({p.returncode}): {' '.join(cmd)}\n"
            f"stdout: {p.stdout[-2000:]}\nstderr: {p.stderr[-2000:]}"
        )
    return p


def iso(dt: datetime) -> str:
    return dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.000Z")


def session_jsonl_lines(conv_id, session, repo_path):
    """Emit Claude Code JSONL entries for one benchmark session.

    Shapes accepted by session.ParseTranscript (cmd/rekal/cli/session/
    parse.go): plain-string message content is valid for both roles;
    timestamps are per-entry RFC3339; sessionId/gitBranch/cwd are read from
    the first line carrying them.
    """
    sid = str(uuid.uuid5(UUID_NS, f"{conv_id}:{session['session_id']}"))
    base = datetime.fromisoformat(session["date"].replace("Z", "+00:00"))
    lines = []
    for i, turn in enumerate(session["turns"]):
        role = turn["role"]
        if role not in ("user", "assistant"):
            raise ValueError(f"{conv_id}/{session['session_id']}: bad role {role!r}")
        entry = {
            "type": role,
            "uuid": str(uuid.uuid5(UUID_NS, f"{conv_id}:{session['session_id']}:{i}")),
            "sessionId": sid,
            "timestamp": iso(base + timedelta(seconds=60 * i)),
            "gitBranch": "main",
            "cwd": repo_path,
            "message": {"role": role, "content": turn["text"]},
        }
        lines.append(json.dumps(entry, ensure_ascii=False))
    return sid, lines


def ingest_conversation(conv, out_dir: Path, rekal: Path, email: str, fast: int):
    conv_id = conv["conversation_id"]
    conv_dir = out_dir / conv_id
    repo = (conv_dir / "repo").resolve()
    config = conv_dir / "claude-config"
    if conv_dir.exists():
        shutil.rmtree(conv_dir)
    repo.mkdir(parents=True)
    config.mkdir(parents=True)
    repo_path = str(repo)

    env = ingest_env(os.environ, config, rekal.parent)

    run(["git", "init", "-q", "-b", "main"], repo, env)
    run(["git", "config", "user.email", email], repo, env)
    run(["git", "config", "user.name", "Industry Bench"], repo, env)
    run([str(rekal), "init"], repo, env)
    # rekal init leaves setup files (hooks are outside the tree; CLAUDE.md,
    # .claude/skills, .gitignore are in it). Commit them once so session
    # commits stay one-marker-file-sized. Backdate to before the first
    # session so the time axis stays monotone.
    first = datetime.fromisoformat(conv["sessions"][0]["date"].replace("Z", "+00:00"))
    init_date = iso(first - timedelta(days=1))
    env_init = {**env, "GIT_AUTHOR_DATE": init_date, "GIT_COMMITTER_DATE": init_date}
    run(["git", "add", "-A"], repo, env_init)
    run(["git", "commit", "-q", "--allow-empty", "-m", "init"], repo, env_init)

    session_dir = Path(config) / "projects" / sanitize_repo_path(repo_path)
    session_dir.mkdir(parents=True, exist_ok=True)
    (repo / "sessions").mkdir(exist_ok=True)

    expected = {"sessions": 0, "turns": 0}
    batch = []
    sessions = conv["sessions"]
    for n, session in enumerate(sessions):
        sid, lines = session_jsonl_lines(conv_id, session, repo_path)
        (session_dir / f"{sid}.jsonl").write_text("\n".join(lines) + "\n")
        marker = repo / "sessions" / f"{session['date'][:10]}-{session['session_id']}.md"
        marker.write_text(
            f"# {session['session_id']}\n\ndate: {session['date']}\n"
            f"benchmark_session_id: {session['session_id']}\n"
        )
        expected["sessions"] += 1
        expected["turns"] += len(session["turns"])
        batch.append(session)
        last = n == len(sessions) - 1
        if len(batch) >= max(fast, 1) or last:
            date = iso(datetime.fromisoformat(batch[-1]["date"].replace("Z", "+00:00")))
            cenv = {**env, "GIT_AUTHOR_DATE": date, "GIT_COMMITTER_DATE": date}
            run(["git", "add", "-A"], repo, cenv)
            ids = ", ".join(s["session_id"] for s in batch)
            run(["git", "commit", "-q", "-m", f"session {batch[-1]['date'][:10]} ({ids})"], repo, cenv)
            # The post-commit hook already ran checkpoint; run it once more
            # explicitly so a hook misfire is loud, not silent (idempotent
            # by content hash).
            run([str(rekal), "checkpoint"], repo, cenv)
            batch = []
    return repo, env, expected


def q(rekal, repo, env, sql):
    """Run rekal query and return the raw stdout (JSON-ish)."""
    p = run([str(rekal), "query", sql], repo, env)
    return p.stdout


def count(rekal, repo, env, sql):
    out = q(rekal, repo, env, sql)
    nums = re.findall(r"\d+", out)
    if not nums:
        raise RuntimeError(f"no count in output of {sql!r}: {out[:400]}")
    return int(nums[-1])


def verify_conversation(conv, repo, env, rekal, expected, fast=1):
    """Ingest-verification per 04-procedures §3 (adapted: captured_at is
    ingestion time by design — the benchmark time axis is turns.ts and the
    backdated commit dates)."""
    failures = []

    def check(name, got, want):
        ok = got == want
        if not ok:
            failures.append(f"{name}: got {got}, want {want}")
        print(f"  {'ok ' if ok else 'FAIL'} {name}: {got}" + ("" if ok else f" (want {want})"))

    n_sessions = count(rekal, repo, env, "SELECT count(*) FROM sessions")
    n_turns = count(rekal, repo, env, "SELECT count(*) FROM turns")
    # one init commit + session commits; only session commits carry sessions
    n_cp_links = count(rekal, repo, env, "SELECT count(*) FROM checkpoint_sessions")
    n_linked_sessions = count(
        rekal, repo, env, "SELECT count(DISTINCT session_id) FROM checkpoint_sessions"
    )
    check("sessions", n_sessions, expected["sessions"])
    check("turns", n_turns, expected["turns"])
    check("checkpoint_sessions links", n_cp_links, expected["sessions"])
    check("distinct linked sessions", n_linked_sessions, expected["sessions"])

    # Temporal axis: turns.ts must span the benchmark dates.
    first = conv["sessions"][0]["date"][:10]
    last = conv["sessions"][-1]["date"][:10]
    span = q(rekal, repo, env, "SELECT min(ts), max(ts) FROM turns")
    for d, name in ((first, "first benchmark date in turns.ts"), (last, "last benchmark date in turns.ts")):
        ok = d in span
        if not ok:
            failures.append(f"{name}: {d} not in {span.strip()[:200]}")
        print(f"  {'ok ' if ok else 'FAIL'} {name}")

    # Commit backdating: git log dates must match benchmark dates.
    # With --fast N, sessions are batched into one commit dated at the batch's
    # last session, so only batch-boundary sessions produce a dated commit.
    log = run(["git", "log", "--format=%aI %s", "--reverse"], repo, env).stdout
    sessions = conv["sessions"]
    step = max(fast, 1)
    boundary_idx = set(range(step - 1, len(sessions), step)) | {len(sessions) - 1}
    for n, s in enumerate(sessions):
        if n not in boundary_idx:
            continue
        ok = any(line.startswith(s["date"][:10]) and s["session_id"] in line for line in log.splitlines())
        if not ok:
            failures.append(f"commit for {s['session_id']} not dated {s['date'][:10]}")
        print(f"  {'ok ' if ok else 'FAIL'} commit backdated for {s['session_id']}")

    return failures


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--input", required=True, help="conversations.jsonl (interchange format)")
    ap.add_argument("--out", required=True, help="workdir root (one subdir per conversation)")
    ap.add_argument("--rekal", default=shutil.which("rekal"), help="path to rekal binary")
    ap.add_argument("--email", default="bench@industry.local")
    ap.add_argument("--limit", type=int, default=0, help="ingest only the first N conversations")
    ap.add_argument("--offset", type=int, default=0, help="skip the first N conversations (with --limit, a shard)")
    ap.add_argument("--fast", type=int, default=1,
                    help="sessions per commit (BEAM tiers only; default 1 = contract mode)")
    ap.add_argument("--index", action="store_true", help="run 'rekal index' after ingest")
    ap.add_argument("--verify", action="store_true", help="run ingest verification; non-zero exit on mismatch")
    args = ap.parse_args()

    if not args.rekal:
        sys.exit("no rekal binary found (PATH or --rekal)")
    rekal = Path(args.rekal).resolve()
    out_dir = Path(args.out).resolve()
    lowered = str(out_dir).lower()
    for part in FORBIDDEN_PATH_PARTS:
        if part in lowered:
            sys.exit(f"--out path contains {part!r}, which SkipCapture treats as bench pollution; "
                     "choose a different workdir")
    out_dir.mkdir(parents=True, exist_ok=True)

    conversations = []
    with open(args.input) as f:
        for line in f:
            if line.strip():
                conversations.append(json.loads(line))
    if args.offset:
        conversations = conversations[args.offset :]
    if args.limit:
        conversations = conversations[: args.limit]

    all_failures = []
    for conv in conversations:
        print(f"[{conv['conversation_id']}] ingesting {len(conv['sessions'])} sessions")
        repo, env, expected = ingest_conversation(conv, out_dir, rekal, args.email, args.fast)
        if args.index:
            run([str(rekal), "index"], repo, env)
        if args.verify:
            failures = verify_conversation(conv, repo, env, rekal, expected, args.fast)
            all_failures.extend(f"{conv['conversation_id']}: {f}" for f in failures)

    if all_failures:
        print("\nVERIFICATION FAILED:")
        for f in all_failures:
            print(f"  - {f}")
        sys.exit(1)
    print(f"\ndone: {len(conversations)} conversation(s) into {out_dir}")


if __name__ == "__main__":
    main()
