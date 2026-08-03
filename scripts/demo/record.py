#!/usr/bin/env python3
"""Record the README demo from a real rekal binary.

Builds a throwaway git repo with synthetic agent transcripts, lets the real
post-commit hook capture them, then runs real `rekal` commands and records
their real stdout. Nothing in the resulting cast is hand-written: every byte
of output below a `$` line came out of the binary named by --rekal.

The *corpus* is invented (a fictional payments-api and a webhook-retry
decision) because a demo needs a story a reader can follow in fifteen seconds.
The *output* is not. Re-run this against any build to see what that build
actually prints.

    python3 scripts/demo/record.py --rekal ./rekal

Writes scripts/demo/cast.json (the captured transcript, reviewable in diffs)
and docs/assets/demo.svg (the animated terminal for the README).
"""

from __future__ import annotations

import argparse
import json
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import uuid
import xml.sax.saxutils as sax
from datetime import datetime, timedelta, timezone

HERE = pathlib.Path(__file__).resolve().parent
REPO_ROOT = HERE.parent.parent

# Fixed clock and ids so a re-record diffs cleanly against the committed cast.
T0 = datetime(2026, 7, 14, 10, 12, tzinfo=timezone.utc)
SESSION_IDS = [
    "3f6b1c02-9d41-4e77-b8a2-5c1e0f7a4d33",
    "b1d84e57-2a6f-4c19-9e05-77c3ab8f1120",
]


def sh(cmd, cwd=None, env=None, check=True):
    merged = dict(os.environ)
    merged.update(env or {})
    proc = subprocess.run(
        cmd,
        cwd=cwd,
        env=merged,
        shell=isinstance(cmd, str),
        capture_output=True,
        text=True,
    )
    if check and proc.returncode != 0:
        sys.exit(f"command failed: {cmd}\n{proc.stdout}\n{proc.stderr}")
    return proc


def ts(minutes: int) -> str:
    return (T0 + timedelta(minutes=minutes)).isoformat().replace("+00:00", "Z")


def transcript(session_id: str, repo: pathlib.Path, turns) -> str:
    """Render turns into Claude Code's JSONL transcript format.

    A ("steer", text) turn becomes a queue-operation entry, which the parser
    classifies as human_steering — a human correcting the agent mid-course.
    That is the role the demo is about.
    """
    out = []
    for index, (kind, text) in enumerate(turns):
        if kind == "steer":
            out.append(
                {
                    "type": "queue-operation",
                    "operation": "enqueue",
                    "sessionId": session_id,
                    "timestamp": ts(index),
                    "content": text,
                }
            )
            continue
        out.append(
            {
                "type": "user" if kind == "human" else "assistant",
                "sessionId": session_id,
                "timestamp": ts(index),
                "uuid": str(uuid.uuid5(uuid.NAMESPACE_OID, f"{session_id}:{index}")),
                "cwd": str(repo),
                "gitBranch": "main",
                "isSidechain": False,
                "userType": "external",
                "version": "2.1.197",
                "message": {"role": "user" if kind == "human" else "assistant", "content": text},
            }
        )
    return "\n".join(json.dumps(entry) for entry in out) + "\n"


# The story: a retry policy is settled, including the option that lost and why.
# Session two exists so recall has to rank, not just find.
SESSIONS = [
    (
        SESSION_IDS[0],
        "feat(dispatch): retry webhooks with exponential backoff and jitter",
        "dispatcher.go",
        "package main\n\n// retryPolicy: exponential backoff, full jitter.\n"
        "// base 500ms, factor 2, cap 60s, 5 attempts, then dead-letter.\n",
        [
            ("human", "Webhook dispatch needs retries. Simplest version is a constant wait between attempts - three tries and give up. Want me to wire that up?"),
            ("assistant", "I can add the constant-wait version to dispatcher.go, roughly twenty lines. Before I do: is the receiving endpoint shared across tenants?"),
            ("steer", "no, a fixed 5s delay stampedes the downstream on recovery. Every hook that failed during an outage retries at the same instant and we DDoS ourselves the moment the endpoint comes back. Use exponential backoff with jitter."),
            ("assistant", "Switching to exponential backoff with full jitter: base 500ms, factor 2, cap 60s, each delay randomized in [0, computed]. Full jitter spreads a retry storm across the window instead of synchronizing it into a spike."),
            ("steer", "also cap the attempts. Five, then dead-letter - anything past five never lands and just burns queue throughput."),
            ("assistant", "Done. Five attempts, then the event is published to the dead-letter topic with the last response status attached for triage."),
        ],
    ),
    (
        SESSION_IDS[1],
        "feat(dispatch): key webhook deliveries by idempotency token",
        "idempotency.go",
        "package main\n\n// Delivery keys are stable across retries.\n",
        [
            ("human", "If we retry a delivery the receiver can process it twice. How do we make that safe?"),
            ("assistant", "Attach an idempotency key derived from the event id, stable across retries, and let the receiver dedupe on it. I'll thread it through the dispatcher."),
            ("steer", "keep the key stable across retries specifically - regenerating per attempt defeats the whole point and the receiver sees each retry as a new event."),
            ("assistant", "Right. The key is computed once when the event is enqueued and carried on every attempt, including after a dead-letter replay."),
        ],
    ),
]

# The commands the demo runs. Every one is executed for real.
STEPS = [
    ("git commit -m 'feat(dispatch): retry webhooks with exponential backoff'", "commit"),
    ("rekal \"should webhook retries use a fixed delay?\"", "recall"),
    ("rekal query -s {sid} --role human_steering", "drill"),
]


def build(rekal: str, workdir: pathlib.Path) -> list[dict]:
    home = workdir / "claude-home"
    repo = workdir / "payments-api"
    repo.mkdir(parents=True)
    env = {"CLAUDE_CONFIG_DIR": str(home)}

    sh("git init -q -b main", cwd=repo)
    sh("git config user.email dev@team.dev", cwd=repo)
    sh("git config user.name 'Dana Ellis'", cwd=repo)
    sh("git config commit.gpgsign false", cwd=repo)

    session_dir = home / "projects" / re.sub(r"[^a-zA-Z0-9]", "-", str(repo))
    session_dir.mkdir(parents=True)

    (repo / "README.md").write_text("# payments-api\n\nOutbound webhook delivery.\n")
    sh("git add -A", cwd=repo)
    sh("git commit -qm 'initial commit'", cwd=repo, env=env)
    sh([rekal, "init"], cwd=repo, env=env)

    cast: list[dict] = []

    # Write every transcript up front, then commit each change. The post-commit
    # hook captures whatever the agent has said by that point - same as real use.
    for session_id, subject, path, body, turns in SESSIONS:
        (session_dir / f"{session_id}.jsonl").write_text(transcript(session_id, repo, turns))
        (repo / path).write_text(body)
        sh("git add -A", cwd=repo)
        commit = sh(["git", "commit", "-m", subject], cwd=repo, env=env)
        if session_id == SESSION_IDS[0]:
            cast.append(
                {
                    "command": f"git commit -m '{subject}'",
                    "output": commit.stdout.strip().splitlines()[:2],
                    "note": "post-commit hook captures the session",
                }
            )

    recall_cmd = ["should webhook retries use a fixed delay?"]
    recall = sh([rekal, *recall_cmd], cwd=repo, env=env, check=False)
    cast.append(
        {
            "command": 'rekal "should webhook retries use a fixed delay?"',
            "output": recall.stdout.strip().splitlines(),
            "exit": recall.returncode,
        }
    )

    sid = next(
        (tok for tok in recall.stdout.split() if re.fullmatch(r"s\d+", tok)),
        "s1",
    )
    drill = sh([rekal, "query", "-s", sid, "--role", "human_steering"], cwd=repo, env=env, check=False)
    lines = drill.stdout.strip().splitlines()
    cast.append(
        {
            "command": f"rekal query -s {sid} --role human_steering",
            "output": lines,
            "exit": drill.returncode,
        }
    )
    return cast


# --------------------------------------------------------------------------
# SVG rendering
# --------------------------------------------------------------------------

CHAR_W = 8.4
LINE_H = 21
PAD_X = 22
PAD_Y = 52
TYPE_RATE = 0.045          # seconds per character
OUTPUT_LINE_DELAY = 0.10   # seconds between output lines appearing
STEP_PAUSE = 1.5           # seconds to read a completed step
END_PAUSE = 4.0            # seconds on the finished frame before looping

BG = "#12141c"
CHROME = "#1c1f2b"
FG = "#c9d1d9"
DIM = "#7d8590"
PROMPT = "#59d499"
CMD = "#ffffff"
ACCENT = "#7ab7ff"
WARN = "#f0a35e"


def wrap(text: str, width: int) -> list[str]:
    if len(text) <= width:
        return [text]
    out, line = [], ""
    for word in text.split(" "):
        if line and len(line) + 1 + len(word) > width:
            out.append(line)
            line = word
        else:
            line = f"{line} {word}".strip()
    if line:
        out.append(line)
    return out


def classify(line: str) -> str:
    stripped = line.strip()
    if stripped.startswith(("INJECT", "KNOWLEDGE")):
        return "verdict"
    if stripped.startswith("SILENCE"):
        return "warn"
    if re.match(r"^s\d+ conf=", stripped):
        return "seed"
    if stripped.startswith(("[", "turn ", "---")):
        return "dim"
    return "fg"


def render_svg(cast: list[dict], cols: int = 96) -> str:
    rows: list[tuple] = []
    for step in cast:
        rows.append(("cmd", step["command"], step.get("note")))
        for line in step["output"]:
            for piece in wrap(line.rstrip(), cols - 2):
                rows.append(("out", piece, None))
        rows.append(("gap", "", None))
    if rows and rows[-1][0] == "gap":
        rows.pop()

    # Timeline
    timings = []
    clock = 0.4
    for kind, text, _ in rows:
        if kind == "cmd":
            duration = len(text) * TYPE_RATE
            timings.append((clock, clock + duration))
            clock += duration + 0.35
        elif kind == "out":
            timings.append((clock, clock))
            clock += OUTPUT_LINE_DELAY
        else:
            timings.append((clock, clock))
            clock += STEP_PAUSE
    total = clock + END_PAUSE

    width = int(PAD_X * 2 + cols * CHAR_W)
    height = int(PAD_Y + len(rows) * LINE_H + 28)

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
        f'viewBox="0 0 {width} {height}" font-family="ui-monospace,SFMono-Regular,Menlo,Consolas,monospace" font-size="13.5">',
        "<style>",
        f"  .r{{opacity:0;animation-duration:{total:.2f}s;animation-iteration-count:infinite;animation-timing-function:steps(1,end);animation-fill-mode:both}}",
        "  @media (prefers-reduced-motion: reduce){.r{opacity:1;animation:none}.cur{display:none}}",
    ]

    for index, (start, _) in enumerate(timings):
        pct = max(0.0, min(99.9, start / total * 100))
        parts.append(
            f"  @keyframes a{index}{{0%,{pct:.3f}%{{opacity:0}}{min(pct + 0.001, 100):.3f}%,100%{{opacity:1}}}}"
        )
        parts.append(f"  .r{index}{{animation-name:a{index}}}")

    # Typing reveal for command rows.
    for index, ((kind, text, _), (start, end)) in enumerate(zip(rows, timings)):
        if kind != "cmd" or not text:
            continue
        p0 = start / total * 100
        p1 = end / total * 100
        full = len(text) * CHAR_W
        parts.append(
            f"  @keyframes t{index}{{0%,{p0:.3f}%{{width:0}}{p1:.3f}%,100%{{width:{full:.1f}px}}}}"
        )
        parts.append(
            f"  .t{index}{{animation:t{index} {total:.2f}s infinite linear both}}"
        )
    parts.append("</style>")

    parts.append(f'<rect width="{width}" height="{height}" rx="10" fill="{BG}"/>')
    parts.append(f'<rect width="{width}" height="34" rx="10" fill="{CHROME}"/>')
    parts.append(f'<rect y="24" width="{width}" height="10" fill="{CHROME}"/>')
    for i, color in enumerate(("#ff5f57", "#febc2e", "#28c840")):
        parts.append(f'<circle cx="{22 + i * 19}" cy="17" r="6" fill="{color}"/>')
    parts.append(
        f'<text x="{width / 2}" y="22" fill="{DIM}" text-anchor="middle" font-size="12">'
        "payments-api — rekal</text>"
    )

    for index, ((kind, text, note), (start, end)) in enumerate(zip(rows, timings)):
        y = PAD_Y + index * LINE_H
        if kind == "gap":
            continue
        if kind == "cmd":
            parts.append(f'<g class="r r{index}">')
            parts.append(f'<text x="{PAD_X}" y="{y}" fill="{PROMPT}">$</text>')
            # Typed reveal: a clipping rect grows across the command text.
            parts.append(f'<clipPath id="c{index}"><rect class="t{index}" x="{PAD_X + 16}" y="{y - 14}" height="{LINE_H}" width="0"/></clipPath>')
            parts.append(
                f'<text x="{PAD_X + 16}" y="{y}" fill="{CMD}" clip-path="url(#c{index})">{sax.escape(text)}</text>'
            )
            if note:
                note_x = PAD_X + 16 + (len(text) + 2) * CHAR_W
                parts.append(f'<text x="{note_x}" y="{y}" fill="{DIM}" font-size="12"># {sax.escape(note)}</text>')
            parts.append("</g>")
            continue

        role = classify(text)
        color = {"verdict": ACCENT, "warn": WARN, "seed": FG, "dim": DIM, "fg": FG}[role]
        weight = ' font-weight="600"' if role == "verdict" else ""
        parts.append(
            f'<text class="r r{index}" x="{PAD_X}" y="{y}" fill="{color}"{weight} xml:space="preserve">{sax.escape(text)}</text>'
        )

    parts.append("</svg>")
    return "\n".join(parts)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--rekal", default=shutil.which("rekal") or "rekal",
                    help="rekal binary to record (default: first on PATH)")
    ap.add_argument("--cast", type=pathlib.Path, default=HERE / "cast.json")
    ap.add_argument("--svg", type=pathlib.Path, default=REPO_ROOT / "docs/assets/demo.svg")
    ap.add_argument("--from-cast", action="store_true",
                    help="re-render the SVG from the committed cast without running rekal")
    args = ap.parse_args()

    if args.from_cast:
        cast = json.loads(args.cast.read_text())
    else:
        rekal = shutil.which(args.rekal) or args.rekal
        if not pathlib.Path(rekal).exists():
            sys.exit(f"no rekal binary at {rekal} (pass --rekal)")
        version = sh([rekal, "version"]).stdout.strip()
        with tempfile.TemporaryDirectory(prefix="rekal-demo-") as tmp:
            cast = build(rekal, pathlib.Path(tmp))
        cast.insert(0, {"recorded_from": version})
        args.cast.write_text(json.dumps(cast, indent=2) + "\n")
        print(f"cast: {args.cast} (from {version})")

    steps = [step for step in cast if "command" in step]
    args.svg.parent.mkdir(parents=True, exist_ok=True)
    args.svg.write_text(render_svg(steps))
    print(f"svg:  {args.svg}")


if __name__ == "__main__":
    main()
