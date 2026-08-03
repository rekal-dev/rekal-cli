#!/usr/bin/env python3
"""Record the README demo: a real agent recalling a real decision.

The demo is the product loop, not a typed-command tour. It:

  1. builds a throwaway git repo and plants one earlier agent session in
     Claude Code's transcript format - a webhook retry policy where a fixed
     delay was proposed and rejected;
  2. runs the real `rekal init`, so the real skill is installed;
  3. commits, so the real post-commit hook captures that session;
  4. runs a real headless Claude Code session asking whether a fixed delay is
     a good idea, and records what the agent actually does.

Nothing below a `$` in the output is written by hand. The agent's tool calls,
the rekal output they produced, and the answer text are all captured from the
live run. Only the corpus is invented, because a demo needs a story a reader
can follow in fifteen seconds.

    python3 scripts/demo/record.py --rekal ./rekal

Writes scripts/demo/cast.json (captured run, reviewable in a diff) and
docs/assets/demo.svg. A live agent is not deterministic: re-recording yields a
different run, which is the point - it is a recording, not a mock.
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
import textwrap
import uuid
import xml.sax.saxutils as sax
from datetime import datetime, timedelta, timezone

HERE = pathlib.Path(__file__).resolve().parent
REPO_ROOT = HERE.parent.parent

T0 = datetime(2026, 7, 14, 10, 12, tzinfo=timezone.utc)
PRIOR_SESSION = "3f6b1c02-9d41-4e77-b8a2-5c1e0f7a4d33"

PROMPT = (
    "We're adding retries to the webhook dispatcher. Should we just use a "
    "fixed 5s delay? Check project memory before answering."
)

# The decision the earlier session settled, including the option that lost.
PRIOR_TURNS = [
    ("human", "Webhook dispatch needs retries. Simplest version is a constant wait between attempts, three tries and give up. Want me to wire that up?"),
    ("assistant", "I can add the constant-wait version to dispatcher.go, roughly twenty lines. Before I do: is the receiving endpoint shared across tenants?"),
    ("steer", "no, a fixed 5s delay stampedes the downstream on recovery. Every hook that failed during an outage retries at the same instant and we DDoS ourselves the moment the endpoint comes back. Use exponential backoff with jitter."),
    ("assistant", "Switching to exponential backoff with full jitter: base 500ms, factor 2, cap 60s, each delay randomized in [0, computed]. Full jitter spreads a retry storm across the window instead of synchronizing it into a spike."),
    ("steer", "also cap the attempts. Five, then dead-letter, anything past five never lands and just burns queue throughput."),
    ("assistant", "Done. Five attempts, then the event is published to the dead-letter topic with the last response status attached for triage."),
]


def sh(cmd, cwd=None, check=True, env=None, timeout=600):
    merged = dict(os.environ)
    merged.update(env or {})
    proc = subprocess.run(
        cmd, cwd=cwd, env=merged, shell=isinstance(cmd, str),
        capture_output=True, text=True, timeout=timeout,
    )
    if check and proc.returncode != 0:
        sys.exit(f"command failed: {cmd}\n{proc.stdout}\n{proc.stderr}")
    return proc


def transcript(session_id: str, repo: pathlib.Path) -> str:
    """Claude Code JSONL. A 'steer' turn is a queue-operation, which the
    parser classifies as human_steering: a human correcting the agent."""
    rows = []
    for index, (kind, text) in enumerate(PRIOR_TURNS):
        stamp = (T0 + timedelta(minutes=index)).isoformat().replace("+00:00", "Z")
        if kind == "steer":
            rows.append({"type": "queue-operation", "operation": "enqueue",
                         "sessionId": session_id, "timestamp": stamp, "content": text})
            continue
        role = "user" if kind == "human" else "assistant"
        rows.append({
            "type": role, "sessionId": session_id, "timestamp": stamp,
            "uuid": str(uuid.uuid5(uuid.NAMESPACE_OID, f"{session_id}:{index}")),
            "cwd": str(repo), "gitBranch": "main", "isSidechain": False,
            "userType": "external", "version": "2.1.197",
            "message": {"role": role, "content": text},
        })
    return "\n".join(json.dumps(r) for r in rows) + "\n"


def claude_session_dir(repo: pathlib.Path) -> pathlib.Path:
    """Where Claude Code keeps transcripts for a repo, and where rekal reads
    them from. Mirrors session.FindSessionDir / SanitizeRepoPath."""
    config = pathlib.Path(os.environ.get("CLAUDE_CONFIG_DIR") or (pathlib.Path.home() / ".claude"))
    return config / "projects" / re.sub(r"[^a-zA-Z0-9]", "-", str(repo))


def record(rekal: str, claude: str, workdir: pathlib.Path) -> dict:
    repo = workdir / "payments-api"
    repo.mkdir(parents=True)

    sh("git init -q -b main", cwd=repo)
    sh("git config user.email dev@team.dev", cwd=repo)
    sh("git config user.name 'Dana Ellis'", cwd=repo)
    sh("git config commit.gpgsign false", cwd=repo)

    session_dir = claude_session_dir(repo)
    session_dir.mkdir(parents=True, exist_ok=True)

    (repo / "README.md").write_text("# payments-api\n\nOutbound webhook delivery.\n")
    sh("git add -A", cwd=repo)
    sh("git commit -qm 'initial commit'", cwd=repo)

    steps: list[dict] = []

    init = sh([rekal, "init"], cwd=repo)
    steps.append({"kind": "shell", "command": "rekal init",
                  "output": [line for line in init.stdout.strip().splitlines()
                             if line.strip().endswith("initialized.")] or ["Rekal initialized."]})

    # Plant the earlier session, then commit: the post-commit hook captures it.
    (session_dir / f"{PRIOR_SESSION}.jsonl").write_text(transcript(PRIOR_SESSION, repo))
    (repo / "dispatcher.go").write_text(
        "package main\n\n// retryPolicy: exponential backoff, full jitter.\n"
        "// base 500ms, factor 2, cap 60s, 5 attempts, then dead-letter.\n"
    )
    sh("git add -A", cwd=repo)
    commit = sh(["git", "commit", "-m", "feat(dispatch): retry webhooks with exponential backoff"], cwd=repo)
    steps.append({"kind": "shell",
                  "command": "git commit -m 'feat(dispatch): retry webhooks with exponential backoff'",
                  "output": commit.stdout.strip().splitlines()[:1],
                  "note": "post-commit hook captures the session"})

    # The real thing: a fresh agent, no memory of that conversation.
    run = sh([claude, "-p", PROMPT, "--output-format", "stream-json", "--verbose",
              "--allowedTools", "Bash,Skill,Read,Glob,Grep"], cwd=repo, check=False, timeout=900)

    agent: list[dict] = []
    pending: dict[str, dict] = {}
    answer = ""
    for line in run.stdout.splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") == "assistant":
            for block in event["message"].get("content", []):
                if block["type"] == "tool_use":
                    entry = {"tool": block["name"], "input": block["input"], "result": []}
                    pending[block["id"]] = entry
                    agent.append(entry)
                elif block["type"] == "text" and block["text"].strip():
                    answer = block["text"].strip()
        elif event.get("type") == "user":
            for block in event["message"].get("content", []) if isinstance(event.get("message"), dict) else []:
                if isinstance(block, dict) and block.get("type") == "tool_result":
                    target = pending.get(block.get("tool_use_id"))
                    if target is None:
                        continue
                    content = block.get("content")
                    if isinstance(content, list):
                        content = "".join(c.get("text", "") for c in content if isinstance(c, dict))
                    target["result"] = str(content or "").strip().splitlines()
        elif event.get("type") == "result" and event.get("result"):
            answer = str(event["result"]).strip()

    shutil.rmtree(session_dir, ignore_errors=True)
    return {"steps": steps, "prompt": PROMPT, "agent": agent, "answer": answer,
            "exit": run.returncode}


# ---------------------------------------------------------------------------
# SVG rendering — a terminal frame, animated with CSS only.
# ---------------------------------------------------------------------------

CHAR_W, LINE_H, PAD_X, PAD_Y = 7.6, 20, 20, 50
TYPE_RATE, LINE_DELAY, PAUSE, END_PAUSE = 0.035, 0.09, 1.1, 4.5

BG, CHROME, FG, DIM = "#12141c", "#1b1e29", "#c9d1d9", "#767e8a"
PROMPT_C, CMD_C, ACCENT, TOOL, GOOD = "#59d499", "#ffffff", "#7ab7ff", "#c8a2f0", "#59d499"


def clip(text: str, width: int) -> str:
    text = text.rstrip()
    return text if len(text) <= width else text[: width - 1] + "…"


def wrap(text: str, width: int, indent: str = "") -> list[str]:
    text = text.rstrip()
    if len(text) <= width:
        return [text]
    return textwrap.wrap(text, width=width, subsequent_indent=indent) or [text]


def build_rows(cast: dict, cols: int) -> list[tuple[str, str]]:
    rows: list[tuple[str, str]] = []
    for step in cast["steps"]:
        rows.append(("cmd", step["command"] + (f"   # {step['note']}" if step.get("note") else "")))
        for line in step["output"]:
            rows.append(("out", line))
        rows.append(("gap", ""))

    rows.append(("cmd", f'claude "{cast["prompt"].split(" Check project")[0]}"'))
    rows.append(("gap", ""))

    # One frame cannot hold a whole agent run. Show the Skill load and the
    # distinct rekal invocations, first output lines only; cast.json keeps the
    # unabridged record, including any call skipped here.
    seen: set[str] = set()
    shown = 0
    for call in cast["agent"]:
        if call["tool"] == "Skill":
            rows.append(("tool", f"Skill({call['input'].get('skill', '')})"))
            continue
        if call["tool"] != "Bash" or shown >= 2:
            continue
        raw = (call["input"].get("command") or "").splitlines()
        command = raw[0] if raw else ""
        if not command.startswith("rekal"):
            continue
        # --json is the same answer in raw form; the digest already showed it.
        if "--json" in command:
            continue
        # Cut the shell the agent wrapped around it, then take the first
        # pipeline stage: `rekal X --explain || rekal X` and `rekal X` are one
        # call as far as the story goes.
        command = re.split(r"\s*(?:\|\||\||&&|2>)", command)[0].strip()
        key = re.sub(r"\s*--\S+", "", command).strip()
        if key in seen:
            continue
        seen.add(key)
        shown += 1
        rows.append(("tool", f"Bash({clip(command, cols - 12)})"))
        for line in [l for l in call["result"] if l.strip()][:3]:
            rows.append(("res", clip(line, cols - 6)))

    rows.append(("gap", ""))
    # The answer, down to the lines that carry the verdict and the policy.
    body = [p.strip() for p in cast["answer"].split("\n") if p.strip()]
    emitted = 0
    for index, para in enumerate(body):
        if emitted >= 7:
            rows.append(("res", "…"))
            break
        for piece in wrap(re.sub(r"[*`]", "", para), cols - 4, "  "):
            rows.append(("ans" if index == 0 else "ans2", piece))
            emitted += 1
    return rows


def render_svg(cast: dict, cols: int = 100) -> str:
    rows = build_rows(cast, cols)
    while rows and rows[-1][0] == "gap":
        rows.pop()

    timings, clock = [], 0.4
    for kind, text in rows:
        if kind == "cmd":
            span = len(text) * TYPE_RATE
            timings.append((clock, clock + span))
            clock += span + 0.4
        elif kind == "gap":
            timings.append((clock, clock))
            clock += PAUSE
        else:
            timings.append((clock, clock))
            clock += LINE_DELAY
    total = clock + END_PAUSE

    width = int(PAD_X * 2 + cols * CHAR_W)
    height = int(PAD_Y + len(rows) * LINE_H + 22)

    out = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
        f'viewBox="0 0 {width} {height}" role="img" '
        f'aria-label="A Claude Code session recalling a rejected webhook retry policy with rekal" '
        f'font-family="ui-monospace,SFMono-Regular,Menlo,Consolas,monospace" font-size="12.5">',
        "<style>",
        f".r{{opacity:0;animation-duration:{total:.2f}s;animation-iteration-count:infinite;"
        "animation-timing-function:steps(1,end);animation-fill-mode:both}",
        "@media (prefers-reduced-motion:reduce){.r{opacity:1;animation:none}.ty{animation:none;width:auto}}",
    ]
    for index, (start, _) in enumerate(timings):
        pct = max(0.0, min(99.8, start / total * 100))
        out.append(f"@keyframes a{index}{{0%,{pct:.3f}%{{opacity:0}}{pct + 0.01:.3f}%,100%{{opacity:1}}}}")
        out.append(f".r{index}{{animation-name:a{index}}}")
    for index, ((kind, text), (start, end)) in enumerate(zip(rows, timings)):
        if kind == "cmd" and text:
            p0, p1 = start / total * 100, end / total * 100
            out.append(f"@keyframes t{index}{{0%,{p0:.3f}%{{width:0}}{p1:.3f}%,100%{{width:{len(text) * CHAR_W:.1f}px}}}}")
            out.append(f".ty{index}{{animation:t{index} {total:.2f}s infinite linear both}}")
    out.append("</style>")

    out.append(f'<rect width="{width}" height="{height}" rx="9" fill="{BG}"/>')
    out.append(f'<rect width="{width}" height="32" rx="9" fill="{CHROME}"/>')
    out.append(f'<rect y="23" width="{width}" height="9" fill="{CHROME}"/>')
    for i, color in enumerate(("#ff5f57", "#febc2e", "#28c840")):
        out.append(f'<circle cx="{20 + i * 18}" cy="16" r="5.5" fill="{color}"/>')
    out.append(f'<text x="{width / 2}" y="20" fill="{DIM}" text-anchor="middle" font-size="11.5">payments-api</text>')

    for index, ((kind, text), _) in enumerate(zip(rows, timings)):
        if kind == "gap":
            continue
        y = PAD_Y + index * LINE_H
        esc = sax.escape(text)
        if kind == "cmd":
            out.append(f'<g class="r r{index}">')
            out.append(f'<text x="{PAD_X}" y="{y}" fill="{PROMPT_C}">$</text>')
            out.append(f'<clipPath id="c{index}"><rect class="ty{index}" x="{PAD_X + 14}" y="{y - 13}" height="{LINE_H}" width="0"/></clipPath>')
            out.append(f'<text x="{PAD_X + 14}" y="{y}" fill="{CMD_C}" clip-path="url(#c{index})" xml:space="preserve">{esc}</text>')
            out.append("</g>")
            continue
        if kind == "tool":
            out.append(f'<text class="r r{index}" x="{PAD_X}" y="{y}" fill="{TOOL}" xml:space="preserve">● {esc}</text>')
            continue
        if kind == "res":
            fill = ACCENT if text.strip().startswith(("INJECT", "KNOWLEDGE")) else DIM
            out.append(f'<text class="r r{index}" x="{PAD_X + 16}" y="{y}" fill="{fill}" xml:space="preserve">{esc}</text>')
            continue
        if kind in ("ans", "ans2"):
            fill = GOOD if kind == "ans" else FG
            weight = ' font-weight="600"' if kind == "ans" else ""
            out.append(f'<text class="r r{index}" x="{PAD_X}" y="{y}" fill="{fill}"{weight} xml:space="preserve">{esc}</text>')
            continue
        out.append(f'<text class="r r{index}" x="{PAD_X}" y="{y}" fill="{FG}" xml:space="preserve">{esc}</text>')

    out.append("</svg>")
    return "\n".join(out)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--rekal", default=shutil.which("rekal") or "rekal")
    ap.add_argument("--claude", default=shutil.which("claude") or "claude")
    ap.add_argument("--cast", type=pathlib.Path, default=HERE / "cast.json")
    ap.add_argument("--svg", type=pathlib.Path, default=REPO_ROOT / "docs/assets/demo.svg")
    ap.add_argument("--workdir", type=pathlib.Path, default=None,
                    help="where to build the throwaway repo (default: a temp dir)")
    ap.add_argument("--from-cast", action="store_true",
                    help="re-render the SVG from the committed cast, without running anything")
    args = ap.parse_args()

    if args.from_cast:
        cast = json.loads(args.cast.read_text())
    else:
        rekal = shutil.which(args.rekal) or args.rekal
        if not pathlib.Path(rekal).exists():
            sys.exit(f"no rekal binary at {rekal} (pass --rekal)")
        version = sh([rekal, "version"]).stdout.strip()
        workdir = args.workdir or pathlib.Path(os.environ.get("TMPDIR", "/tmp")) / "rekal-demo"
        shutil.rmtree(workdir, ignore_errors=True)
        workdir.mkdir(parents=True)
        cast = record(rekal, shutil.which(args.claude) or args.claude, workdir)
        cast["recorded_from"] = version
        args.cast.write_text(json.dumps(cast, indent=2) + "\n")
        print(f"cast: {args.cast} ({version}, agent made {len(cast['agent'])} tool calls)")

    args.svg.parent.mkdir(parents=True, exist_ok=True)
    args.svg.write_text(render_svg(cast))
    print(f"svg:  {args.svg}")


if __name__ == "__main__":
    main()
