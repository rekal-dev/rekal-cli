---
name: init
description: >
  Turn on Rekal memory in the current repository by running `rekal init`. Use
  when the user asks to initialize or set up Rekal here, or when a `rekal`
  command reported the repository is not initialized. Also use to refresh setup
  after a binary update. Do not offer this merely because a repo lacks a `.rekal/` store — most repos do
  not want one.
---

# Initialize Rekal in this repository

Run in each repository; rerun to refresh managed skills, hooks and instruction
lines. The binary must already be installed — if `rekal` is not on `PATH`, use
the `install` skill first.

## Do not volunteer this

This skill is loaded in every repository the user opens. Almost none of them
should be initialized. Act when the user asks, or when a `rekal` command has
actually reported the repo is not initialized. A repository simply lacking
`.rekal/` is not a reason to suggest anything.

## Explain the setup

When the user requests setup, run it within that authorization. Otherwise confirm
before initializing a repository. Explain what `rekal init` writes:

- `.rekal/` — the store (two files: an append-only `data.db`, a local `index.db`)
- `post-commit` / `pre-push` hooks for capture on commit and sharing on push
- an orphan branch `rekal/<their-email>` used to transport memory over git
- the full recall skill under `.claude/skills/rekal/`
- one marker-tagged line in `CLAUDE.md`, plus the equivalent line in any other
  agent rules file it detects (`AGENTS.md`, `GEMINI.md`,
  `.github/copilot-instructions.md`, `.kiro/steering/rekal.md`)

Existing user instructions are preserved; the managed marker line is added or
refreshed. Existing custom hooks are preserved; read any integration notices.
`rekal clean` removes local repository setup, not copies already shared remotely.

If the user declines, stop and say so plainly.

## Run it

From the repository root:

```bash
rekal init
```

## After

Explain what is ready:

- **Existing context can be available immediately.** Init captures discoverable
  sessions and supported memory notes and builds the local index. If there is no
  history yet, work and commit normally; an empty result is expected.
- **Native notes are observed, not rewritten.** Claude project notes are
  discovered by default; other sources are configurable. Explicitly local notes
  stay local. Eligible repository records can travel through Git.
- **If the team already shares a ledger**, `rekal sync` pulls it now and recall
  has something to find immediately.

The binary installs the full recall skill, versioned with its commands. Use
that skill for context questions: search, read the returned reference, and follow
`more:` when needed. The agent decides how to use the evidence.
