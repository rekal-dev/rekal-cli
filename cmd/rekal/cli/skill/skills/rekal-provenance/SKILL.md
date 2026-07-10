---
name: rekal-provenance
description: |
  Use this skill to answer "how and why was this change made?" in a repo with
  Rekal (.rekal/ exists). It walks from an artifact — a file, a function, a
  commit — back to the AI session that produced it, then to the human intent
  behind it. Reach for it when reading unfamiliar code, onboarding to a large
  codebase, reviewing a diff, or doing archaeology on why something exists.
---

# Rekal — Change Provenance

Git tells you *what* changed and *who* typed the command. Rekal tells you
*why* — the conversation that produced the change: the intent, the options
weighed, the correction that reshaped it. This skill turns a line of code into
its origin story.

The funnel is always the same: **artifact → commit → session → intent.**
Each hop narrows from "a file exists" to "a person wanted this, for this
reason." Walk it top to bottom; stop as soon as you have the answer.

## When to Use

- Reviewing a diff or PR and the *why* is not obvious from the code
- Onboarding — you need the reasoning behind a subsystem, not just its shape
- A change looks wrong or surprising; before "fixing" it, learn why it is so
- Archaeology: "who decided this and what were they trying to do?"

## The funnel

### Hop 1 — Anchor on the artifact

Start from what you can see. Get the commits that touched it.

```bash
git log --oneline -15 -- path/to/file.go      # commits that changed this file
git log --oneline -15 -L :FuncName:path.go    # commits that changed one function
git blame -L 120,160 -- path/to/file.go       # the exact commit behind these lines
```

Pick the commit(s) that introduced the code you care about — usually the most
recent non-trivial one, or the one `blame` points at.

### Hop 2 — Commit → producing session

Rekal links every checkpoint's commit to the session(s) that produced it
(`checkpoint_sessions`). Ask it directly:

```bash
rekal --commit <sha>                          # sessions that produced this commit
rekal --commit <sha> --file path/to/file.go   # narrow to sessions touching this file
```

Each hit's `session_id` is the thread of work behind the commit. If a commit
maps to several sessions (a long feature branch), prefer the one whose `files`
or snippet matches the code you're tracing.

**If `--commit` returns nothing** the change predates Rekal, or landed via a
squash that renamed the sha. Fall back to the file:

```bash
rekal --file 'path/to/file\.go' "<what the code does>"
```

### Hop 3 — Session → intent

Now read the thread, cheapest turns first. Intent lives in the human turns;
the shape of the decision lives in the assistant's reasoning.

```bash
rekal query --session <id> --role human            # what was asked for (intent)
rekal query --session <id> --role human_steering   # course-corrections mid-work
rekal query --session <id> --offset <n> --limit 5  # zoom to the snippet turn
```

`human_steering` turns are the highest-signal part of a provenance trace: they
are the moments the human redirected the agent, which is exactly where a design
took its real shape. Read these before `--full`.

### Hop 4 — Emit the why-chain

Summarize what you found as a short chain, not a transcript dump:

```
file.go:134 (rate-limit retry)
  ← commit a1b2c3d "harden webhook delivery"
  ← session 01JNQ… (agent, feat/webhooks branch)
  ← intent: "deliveries were dropping under load"
  ← decision: exponential backoff chosen over fixed delay after the
    human steered away from a fixed 5s ("that stampedes on recovery")
```

That chain is the deliverable. It is what git alone can never give you.

## Large-codebase scale

In a monorepo, an unscoped search is noise. Always anchor provenance with a
path or commit so recall stays on the subsystem you're tracing:

```bash
rekal --file 'services/billing/' "proration"      # scope to one service tree
rekal --commit <sha>                               # scope to one change
```

To find which session *authored* a file (not just mentioned it), the index
records produced files with authorship-bearing change types (`A`/`M` from git,
`T` from Write/Edit tool calls):

```bash
rekal query --index "SELECT session_id, change_type FROM files_index \
  WHERE file_path LIKE '%billing/proration%' ORDER BY change_type"
```

`A` (added) and `M` (modified) are strong authorship; `T` means a tool wrote or
edited it during the session. Prefer sessions with `A`/`M`/`T` over ones that
merely read the file.

## Guidelines

- Walk the funnel in order; each hop is cheaper to read than the next is to
  fetch. Do not jump to `--full`.
- One commit can have many sessions and one session many commits — use `--file`
  to disambiguate.
- Report the why-chain, not the raw session. The chain is the value; the
  transcript is the source.
- If provenance dead-ends (pre-Rekal code, no matching session), say so plainly
  rather than inventing a rationale.
