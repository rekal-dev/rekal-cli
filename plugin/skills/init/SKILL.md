---
name: init
description: >
  Turn on Rekal memory in the current repository by running `rekal init`. Use
  when the user asks to initialize or set up Rekal here, or when a `rekal`
  command reported the repository is not initialized. Once per repository. Do
  not offer this merely because a repo lacks a `.rekal/` store — most repos do
  not want one.
---

# Initialize Rekal in this repository

Run once per repository. The binary must already be installed — if `rekal` is
not on `PATH`, use the `install` skill first.

## Do not volunteer this

This skill is loaded in every repository the user opens. Almost none of them
should be initialized. Act when the user asks, or when a `rekal` command has
actually reported the repo is not initialized. A repository simply lacking
`.rekal/` is not a reason to suggest anything.

## Ask first

`rekal init` changes the user's repository. Name what it writes, then wait:

- `.rekal/` — the store (two files: an append-only `data.db`, a local `index.db`)
- a `post-commit` / `pre-push` git hook that captures sessions at commit time
- an orphan branch `rekal/<their-email>` used to transport memory over git
- the full recall skill under `.claude/skills/rekal/`
- one marker-tagged line in `CLAUDE.md`, plus the equivalent line in any other
  agent rules file it detects (`AGENTS.md`, `GEMINI.md`,
  `.github/copilot-instructions.md`, `.kiro/steering/rekal.md`)

User content is never rewritten — only the marker line is added. `rekal clean`
removes everything, with no residue.

If the user declines, stop and say so plainly.

## Run it

From the repository root:

```bash
rekal init
```

## After

Say what to expect, because the first impression is misleading:

- **A fresh store is empty.** Recall stays silent until the next commit. That
  silence is correct, not a broken install — memory accumulates from here.
- **If the team already shares a ledger**, `rekal sync` pulls it now and recall
  has something to find immediately.

From this point the binary owns memory. It installed the full recall skill,
which is the one that answers questions — richer than anything in this plugin,
and versioned with the binary whose commands it runs. Route recall questions
there. This skill is done until the next repository.
