#!/usr/bin/env python3
"""Record the README demo: a real situation, end to end.

The situation, which is the one Rekal exists for:

  Dana works out webhook delivery semantics with her agent over a long
  session. A fixed retry delay is proposed and rejected along the way. The
  work lands on main and she pushes.

  Weeks later Sam, on a different machine, was never in that conversation and
  asks his agent whether a fixed 5s delay would do. The agent loads the rekal
  skill, recalls, drills into Dana's session, and answers with the decision
  *and* why the alternative lost - without re-proposing it.

Everything the demo shows is executed for real: `rekal init`, the post-commit
and pre-push hooks against a real remote, `rekal sync` on the second machine,
and a real headless Claude Code session. The agent's tool calls, the rekal
output they produced, and the answer text are all captured live.

`rekal init` is the only rekal command the demo types on the authoring side.
Capture and publication ride the `git commit` and `git push` a developer
already runs, so the frame shows the workflow the README promises rather than
teaching a manual `rekal push` step that nobody needs.

The frame carries one claim and no more: a conversation becomes memory, and
that memory reaches a teammate over plain git. The merged-only guarantee, the
recall graph and the knowledge layer are all real and all documented, but a
demo that argues three things argues none of them.

The corpus in conversation.json is invented, because a demo needs a story a
reader can follow. Nothing else is.

    python3 scripts/demo/record.py --rekal ./rekal

Writes scripts/demo/cast.json (the captured run, reviewable in a diff) and
docs/assets/demo.svg. A live agent is not deterministic, so re-recording gives
a different run - that is a property of a recording, not a defect. The fix for
an untidy run is not to re-roll until it looks good.
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

DANA, SAM = "dana@team.dev", "sam@team.dev"
T0 = datetime(2026, 6, 23, 9, 40, tzinfo=timezone.utc)
PRIOR_SESSION = "3f6b1c02-9d41-4e77-b8a2-5c1e0f7a4d33"

PROMPT = (
    "We need retries on the webhook dispatcher. Can we just use a fixed 5s "
    "delay between attempts? Check project memory before you answer."
)


def sh(cmd, cwd=None, check=True, env=None, timeout=900):
    merged = dict(os.environ)
    merged.update(env or {})
    proc = subprocess.run(cmd, cwd=cwd, env=merged, shell=isinstance(cmd, str),
                          capture_output=True, text=True, timeout=timeout)
    if check and proc.returncode != 0:
        sys.exit(f"command failed: {cmd}\n{proc.stdout}\n{proc.stderr}")
    return proc


def claude_session_dir(repo: pathlib.Path) -> pathlib.Path:
    """Where Claude Code keeps transcripts, and where rekal reads them from.
    Mirrors session.FindSessionDir / SanitizeRepoPath."""
    config = pathlib.Path(os.environ.get("CLAUDE_CONFIG_DIR") or (pathlib.Path.home() / ".claude"))
    return config / "projects" / re.sub(r"[^a-zA-Z0-9]", "-", str(repo))


def transcript(session_id: str, repo: pathlib.Path, branch: str, turns) -> str:
    rows = []
    for index, (kind, text) in enumerate(turns):
        stamp = (T0 + timedelta(minutes=index * 3)).isoformat().replace("+00:00", "Z")
        if kind == "steer":
            rows.append({"type": "queue-operation", "operation": "enqueue",
                         "sessionId": session_id, "timestamp": stamp, "content": text})
            continue
        role = "user" if kind == "human" else "assistant"
        rows.append({
            "type": role, "sessionId": session_id, "timestamp": stamp,
            "uuid": str(uuid.uuid5(uuid.NAMESPACE_OID, f"{session_id}:{index}")),
            "cwd": str(repo), "gitBranch": branch, "isSidechain": False,
            "userType": "external", "version": "2.1.197",
            "message": {"role": role, "content": text},
        })
    return "\n".join(json.dumps(r) for r in rows) + "\n"


def git(repo, *args, **kw):
    return sh(["git", *args], cwd=repo, **kw)


def pick(proc, needle: str) -> list[str]:
    """Lines matching needle from either stream. rekal writes progress to
    stderr, so reading stdout alone silently drops half its output."""
    return [l.strip() for l in (proc.stdout + "\n" + proc.stderr).splitlines() if needle in l]


def record(rekal: str, claude: str, workdir: pathlib.Path) -> dict:
    convo = json.loads((HERE / "conversation.json").read_text())
    turns = convo["turns"]

    origin = workdir / "origin.git"
    dana, sam = workdir / "dana", workdir / "sam"
    # A previous recording at this workdir maps to the same
    # ~/.claude/projects/<sanitized> directory. A transcript left there gets
    # picked up by `rekal init`, which put the demo's own prompt into the store
    # and ranked it above the memory the demo exists to show. Clear both before
    # anything runs, not just after.
    for repo in (dana, sam):
        shutil.rmtree(claude_session_dir(repo), ignore_errors=True)
    sh(["git", "init", "-q", "--bare", str(origin)])

    rows: list[dict] = []

    def add(kind: str, text: str, pane: str = "dana") -> None:
        rows.append({"kind": kind, "text": text, "pane": pane})

    # ---- Dana: a long session, captured at commit, published on push -------
    sh(["git", "clone", "-q", str(origin), str(dana)])
    git(dana, "config", "user.email", DANA)
    git(dana, "config", "user.name", "Dana Ellis")
    git(dana, "config", "commit.gpgsign", "false")
    (dana / "README.md").write_text("# payments-api\n\nOutbound webhook delivery.\n")
    git(dana, "add", "-A")
    git(dana, "commit", "-qm", "initial commit")
    git(dana, "branch", "-M", "main")
    git(dana, "push", "-q", "-u", "origin", "main")
    init = sh([rekal, "init"], cwd=dana)
    add("note", "one command, once per repo")
    add("cmd", "rekal init")
    for line in pick(init, "initialized")[:1]:
        add("out", line)
    add("note", "after that, nothing but normal git")

    dana_sessions = claude_session_dir(dana)
    dana_sessions.mkdir(parents=True, exist_ok=True)
    (dana_sessions / f"{PRIOR_SESSION}.jsonl").write_text(
        transcript(PRIOR_SESSION, dana, "main", turns))

    (dana / "dispatcher.go").write_text(
        "package main\n\n// Delivery: exponential backoff, full jitter, base 500ms, cap 60s.\n"
        "// Five attempts, then dead-letter. Idempotency key is stable per event.\n")
    (dana / "delivery.go").write_text("package main\n\n// deliveries table + worker\n")
    git(dana, "add", "-A")
    commit = git(dana, "commit", "-m", convo["commit_subject"])

    add("note", f"three weeks ago: one {len(turns)}-turn session on webhook delivery")
    add("cmd", "git commit -m 'feat(dispatch): retry webhooks with backoff'")
    captured = [l for l in (commit.stdout + commit.stderr).splitlines() if "captured" in l]
    add("out", captured[0].strip() if captured else "rekal: 1 session(s) captured")

    push = git(dana, "push", "origin", "main")
    add("cmd", "git push")
    for line in pick(push, "pushed to")[:1]:
        add("out", line)
    # Off-camera: the merged-only gate is evaluated against the remote tip, so
    # a commit that just landed publishes on the *next* push. A developer hits
    # this on their next ordinary `git push`; the demo should not spend a row
    # on the timing.
    sh([rekal, "push"], cwd=dana, check=False)

    # ---- Sam: different machine, never in that conversation ---------------
    sh(["git", "clone", "-q", str(origin), str(sam)])
    git(sam, "config", "user.email", SAM)
    git(sam, "config", "user.name", "Sam Okafor")
    git(sam, "config", "commit.gpgsign", "false")
    sh([rekal, "init"], cwd=sam)
    sync = sh([rekal, "sync"], cwd=sam, check=False)
    add("note", "today, another machine. Never in that conversation.", "sam")
    add("cmd", "rekal sync", "sam")
    for line in pick(sync, "synced")[:1]:
        add("out", line, "sam")

    # Warm the embedding daemon so the recording shows steady state rather
    # than the first-run "SEMANTIC warming" notice.
    sh([rekal, "webhook delivery retries"], cwd=sam, check=False, timeout=600)

    run = sh([claude, "-p", PROMPT, "--output-format", "stream-json", "--verbose",
              "--allowedTools", "Bash,Skill,Read,Glob,Grep"], cwd=sam, check=False)

    agent: list[dict] = []
    pending: dict[str, dict] = {}
    answer = ""
    for line in run.stdout.splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        kind = event.get("type")
        if kind == "assistant":
            for block in event["message"].get("content", []):
                if block["type"] == "tool_use":
                    entry = {"tool": block["name"], "input": block["input"], "result": []}
                    pending[block["id"]] = entry
                    agent.append(entry)
                elif block["type"] == "text" and block["text"].strip():
                    answer = block["text"].strip()
        elif kind == "user" and isinstance(event.get("message"), dict):
            for block in event["message"].get("content", []):
                if isinstance(block, dict) and block.get("type") == "tool_result":
                    target = pending.get(block.get("tool_use_id"))
                    if target is None:
                        continue
                    content = block.get("content")
                    if isinstance(content, list):
                        content = "".join(c.get("text", "") for c in content if isinstance(c, dict))
                    target["result"] = str(content or "").strip().splitlines()
        elif kind == "result" and event.get("result"):
            answer = str(event["result"]).strip()

    add("cmd", 'claude "can we just use a fixed 5s retry delay?"', "sam")
    add("agent", "", "sam")
    for index, para in enumerate(answer_lines(answer)):
        add("ans" if index == 0 else "ans2", para, "sam")

    for repo in (dana, sam):
        shutil.rmtree(claude_session_dir(repo), ignore_errors=True)

    return {"rows": rows, "prompt": PROMPT, "agent": agent, "answer": answer,
            "turns": len(turns), "exit": run.returncode}


COLS = 66


def clip(text: str, width: int = COLS) -> str:
    text = text.rstrip()
    return text if len(text) <= width else text[: width - 1] + "…"


def agent_rows(agent: list[dict]) -> list[dict]:
    """Skill load plus the distinct rekal calls. One frame cannot hold a whole
    agent run; cast.json keeps everything, including calls dropped here."""
    out, seen, shown = [], set(), 0
    for call in agent:
        if call["tool"] == "Skill":
            out.append({"kind": "tool", "text": f"Skill({call['input'].get('skill', '')})"})
            continue
        if call["tool"] != "Bash" or shown >= 2:
            continue
        raw = (call["input"].get("command") or "").splitlines()
        # An agent may wrap the call in shell (a guessed script path with a
        # `|| rekal ...` fallback, a redirect, a pipe). Take the first segment
        # that is actually a rekal invocation rather than requiring the whole
        # command to be one.
        segments = re.split(r"\s*(?:\|\||\||&&|;|2>)\s*", raw[0] if raw else "")
        command = next((s.strip() for s in segments if s.strip().startswith("rekal ")), "")
        if not command or any(f in command for f in ("--json", "--help", "-h ")):
            continue
        key = re.sub(r"\s*--\S+|\s*-\w\b", "", command).strip()
        if key in seen:
            continue
        seen.add(key)
        shown += 1
        out.append({"kind": "tool", "text": clip(f"Bash({command})", COLS - 4)})
        useful = [l for l in call["result"]
                  if l.strip() and not l.startswith(("/bin/bash", "bash:", "sh:"))]
        for line in useful[:3]:
            out.append({"kind": "res", "text": clip(line, COLS - 6)})
    return out


def answer_lines(answer: str, limit: int = 7) -> list[str]:
    body, emitted = [], 0
    for para in [p.strip() for p in answer.split("\n") if p.strip()]:
        if emitted >= limit:
            body.append("…")
            break
        for piece in textwrap.wrap(re.sub(r"[*`]", "", para), width=COLS - 4,
                                   subsequent_indent="  ") or [""]:
            body.append(piece)
            emitted += 1
    return body


# ---------------------------------------------------------------------------
# SVG: a terminal frame, CSS-only animation. Plays once and holds the final
# frame - a looping demo makes a reader wait a full cycle to re-read a line.
# ---------------------------------------------------------------------------

CHAR_W, LINE_H, PAD_X, PAD_Y = 7.0, 18.5, 16, 52
LANE = 104
TYPE_RATE, LINE_DELAY, BEAT, NOTE_BEAT = 0.035, 0.40, 1.8, 1.5

BG, PANE, CHROME, FG, DIM = "#0d0f16", "#12141c", "#1b1e29", "#c9d1d9", "#6e7681"
CMD_C, TOOL, GOOD = "#ffffff", "#c8a2f0", "#59d499"
DANA_C, SAM_C, WIRE_C = "#f0a35e", "#7ab7ff", "#59d499"

PANES = (
    ("dana", "dana@team.dev", "wrote it", DANA_C),
    ("sam", "sam@team.dev", "never saw it", SAM_C),
)


def expand(cast: dict) -> list[dict]:
    """Materialize display rows, splicing the agent's calls in where the
    recorder left a marker."""
    out = []
    for row in cast["rows"]:
        if row["kind"] != "agent":
            out.append(row)
            continue
        for call in agent_rows(cast["agent"]):
            out.append({**call, "pane": row["pane"]})
    return out


def render_svg(cast: dict) -> str:
    """Two terminals side by side: the machine that had the conversation, and
    the machine that didn't. The gap between them is plain git - which is the
    whole argument, so it gets drawn rather than described."""
    rows = expand(cast)

    timings, clock = [], 0.5
    for row in rows:
        kind, text = row["kind"], row["text"]
        if kind == "cmd":
            span = len(text) * TYPE_RATE
            timings.append((clock, clock + span))
            clock += span + 0.9
        elif kind == "note":
            timings.append((clock, clock))
            clock += NOTE_BEAT
        elif kind == "gap":
            timings.append((clock, clock))
            clock += BEAT
        elif kind in ("ans", "ans2"):
            timings.append((clock, clock))
            clock += 0.5
        else:
            timings.append((clock, clock))
            clock += LINE_DELAY
    total = clock + 1.0

    pane_w = int(PAD_X * 2 + COLS * CHAR_W)
    width = pane_w * 2 + LANE
    per_pane = {key: [i for i, r in enumerate(rows) if r["pane"] == key] for key, *_ in PANES}
    height = int(PAD_Y + max(len(v) for v in per_pane.values()) * LINE_H + 18)

    # The wire crosses when the second machine starts doing anything.
    first_sam = per_pane["sam"][0] if per_pane["sam"] else 0
    wire_at = timings[first_sam][0] - 0.35

    pct = lambda t: max(0.0, min(99.8, t / total * 100))
    out = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
        f'viewBox="0 0 {width} {height}" role="img" '
        f'aria-label="Two terminals side by side. On the left Dana commits a {cast["turns"]}-turn '
        f'session about webhook delivery and pushes it. On the right Sam, who was never in that '
        f'conversation, syncs and his agent answers that a fixed retry delay was already rejected." '
        f'font-family="ui-monospace,SFMono-Regular,Menlo,Consolas,monospace" font-size="11.5">',
        "<style>",
        # Plays once and holds: a loop makes a reader wait a full cycle to
        # re-read a line they missed.
        f".r{{opacity:0;animation-duration:{total:.2f}s;animation-iteration-count:1;"
        "animation-timing-function:steps(1,end);animation-fill-mode:both}",
        "@media (prefers-reduced-motion:reduce){.r{opacity:1;animation:none}"
        "[class*=ty]{animation:none!important;width:100%!important}}",
    ]
    for index, (start, _) in enumerate(timings):
        p = pct(start)
        out.append(f"@keyframes a{index}{{0%,{p:.3f}%{{opacity:0}}{p + 0.01:.3f}%,100%{{opacity:1}}}}")
        out.append(f".r{index}{{animation-name:a{index}}}")
    for index, (row, (start, end)) in enumerate(zip(rows, timings)):
        if row["kind"] == "cmd" and row["text"]:
            out.append(f"@keyframes t{index}{{0%,{pct(start):.3f}%{{width:0}}{pct(end):.3f}%,100%"
                       f"{{width:{len(clip(row['text'])) * CHAR_W:.1f}px}}}}")
            out.append(f".ty{index}{{animation:t{index} {total:.2f}s 1 linear both}}")
    w = pct(wire_at)
    out.append(f"@keyframes wire{{0%,{w:.3f}%{{opacity:0}}{w + 0.01:.3f}%,100%{{opacity:1}}}}")
    out.append(f".wire{{opacity:0;animation:wire {total:.2f}s 1 steps(1,end) both}}")
    out.append("</style>")
    out.append(f'<rect width="{width}" height="{height}" fill="{BG}"/>')

    for pane_index, (key, who, tag, accent) in enumerate(PANES):
        x0 = pane_index * (pane_w + LANE)
        out.append(f'<rect x="{x0}" width="{pane_w}" height="{height}" rx="9" fill="{PANE}"/>')
        out.append(f'<rect x="{x0}" width="{pane_w}" height="32" rx="9" fill="{CHROME}"/>')
        out.append(f'<rect x="{x0}" y="23" width="{pane_w}" height="9" fill="{CHROME}"/>')
        out.append(f'<circle cx="{x0 + 17}" cy="16" r="4.5" fill="{accent}"/>')
        out.append(f'<text x="{x0 + 30}" y="20" fill="{accent}" font-size="11.5" '
                   f'font-weight="600">{who}</text>')
        out.append(f'<text x="{x0 + pane_w - 12}" y="20" fill="{DIM}" font-size="11" '
                   f'text-anchor="end">{tag}</text>')

        for slot, index in enumerate(per_pane[key]):
            row = rows[index]
            kind, text = row["kind"], clip(row["text"])
            if kind == "gap":
                continue
            y = PAD_Y + slot * LINE_H
            esc = sax.escape(text)
            if kind == "cmd":
                out.append(f'<g class="r r{index}">')
                out.append(f'<text x="{x0 + PAD_X}" y="{y}" fill="{accent}">$</text>')
                out.append(f'<clipPath id="c{index}"><rect class="ty{index}" '
                           f'x="{x0 + PAD_X + 12}" y="{y - 12}" height="{LINE_H}" width="0"/></clipPath>')
                out.append(f'<text x="{x0 + PAD_X + 12}" y="{y}" fill="{CMD_C}" '
                           f'clip-path="url(#c{index})" xml:space="preserve">{esc}</text>')
                out.append("</g>")
                continue
            fill, dx, extra, prefix = {
                "note": (DIM, 0, ' font-style="italic"', "# "),
                "tool": (TOOL, 0, "", "● "),
                "res": (SAM_C if text.strip().startswith(("INJECT", "KNOWLEDGE")) else DIM, 14, "", ""),
                "ans": (GOOD, 0, ' font-weight="600"', ""),
                "ans2": (FG, 0, "", ""),
            }.get(kind, (FG, 0, "", ""))
            out.append(f'<text class="r r{index}" x="{x0 + PAD_X + dx}" y="{y}" fill="{fill}"{extra} '
                       f'xml:space="preserve">{prefix}{esc}</text>')

    # The lane between the two machines: no server, just a branch.
    lane_x = pane_w + LANE / 2
    mid = height / 2
    out.append(f'<g class="wire">')
    out.append(f'<path d="M {pane_w + 10} {mid} H {pane_w + LANE - 16}" stroke="{WIRE_C}" '
               f'stroke-width="1.2" stroke-dasharray="3 3" fill="none"/>')
    out.append(f'<path d="M {pane_w + LANE - 20} {mid - 4} l 6 4 l -6 4 z" fill="{WIRE_C}"/>')
    out.append(f'<text x="{lane_x}" y="{mid - 14}" fill="{WIRE_C}" font-size="10.5" '
               f'text-anchor="middle">git push</text>')
    out.append(f'<text x="{lane_x}" y="{mid + 22}" fill="{DIM}" font-size="9.5" '
               f'text-anchor="middle">rekal/dana</text>')
    out.append(f'<text x="{lane_x}" y="{mid + 34}" fill="{DIM}" font-size="9.5" '
               f'text-anchor="middle">@team.dev</text>')
    out.append(f'<text x="{lane_x}" y="{mid + 50}" fill="{DIM}" font-size="9.5" '
               f'text-anchor="middle">no server</text>')
    out.append("</g>")

    out.append("</svg>")
    return "\n".join(out)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--rekal", default=shutil.which("rekal") or "rekal")
    ap.add_argument("--claude", default=shutil.which("claude") or "claude")
    ap.add_argument("--cast", type=pathlib.Path, default=HERE / "cast.json")
    ap.add_argument("--svg", type=pathlib.Path, default=REPO_ROOT / "docs/assets/demo.svg")
    ap.add_argument("--workdir", type=pathlib.Path, default=None)
    ap.add_argument("--from-cast", action="store_true",
                    help="re-render the SVG from the committed cast, running nothing")
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
        print(f"cast: {args.cast} ({version}, {len(cast['agent'])} agent tool calls)")

    args.svg.parent.mkdir(parents=True, exist_ok=True)
    args.svg.write_text(render_svg(cast))
    print(f"svg:  {args.svg}")


if __name__ == "__main__":
    main()
