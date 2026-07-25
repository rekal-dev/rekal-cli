---
name: rekal-setup
description: >
  Use when the user wants to set up Rekal in a repository, asks what Rekal is,
  or when a `rekal` command reports it is not installed or not initialized.
  Rekal is git-backed memory of prior AI sessions — why a change was made, what
  was tried, what was rejected. This skill only installs it. Once `rekal init`
  has run, the binary installs the full recall skill and that one takes over.
---

# Rekal — get it running

This skill has one job: get Rekal installed and initialized. It does not answer
questions from memory. The binary installs a second, much richer skill that
does; after `rekal init`, route recall questions there, not here.

## What it is, in one paragraph

Every AI session settles things — why this approach, what got tried and thrown
away. Then the session ends and the reasoning is gone. Rekal captures each
session at commit time, stores it raw in git, indexes it locally, and shares it
across the team when the work merges. No server, no vector-database tier: the
store is two files in `.rekal/`. A later agent asks `rekal "<question>"` and
gets the conversation behind the change.

## Setup changes the user's machine and repository. Ask first — every step.

The install pipes a script from the internet into a shell. `rekal init` writes
git hooks, an orphan branch, and a line in `CLAUDE.md`. These are the user's
decisions, not yours. State plainly what each step does, then wait.

If the user declines, stop. Say Rekal is not available here and answer from the
tree. Declining is a complete answer, not a problem to route around.

## 1. Install the binary

Requires git, macOS or Linux. One binary — CLI, database, embedding model, all
embedded.

```bash
curl -fsSL https://raw.githubusercontent.com/rekal-dev/rekal-cli/main/scripts/install.sh | bash
```

Installs to `~/.local/bin`; `--target <dir>` overrides. Verify with `rekal
version`.

If the shell still reports `command not found`, the install directory is not on
`PATH`. Tell the user which directory to add and let them edit their own shell
profile.

## 2. Initialize the repository

From the repository root:

```bash
rekal init
```

That writes `.rekal/` (the store), a post-commit hook that captures sessions, an
orphan branch `rekal/<email>` for transport, the full recall skill under
`.claude/skills/rekal/`, and one marker-tagged line in `CLAUDE.md`. It also
writes the equivalent line for any other agent it detects — `AGENTS.md`,
`GEMINI.md`, `.github/copilot-instructions.md`, `.kiro/steering/rekal.md`. User
content is never touched.

`rekal clean` removes all of it, with no residue.

## 3. Hand off

After `rekal init`, the binary owns the recall skill. Tell the user setup is
done and what to expect:

- A fresh store holds nothing. Recall stays silent until the next commit — that
  silence is correct, not a broken install.
- If the team already shares a ledger, `rekal sync` pulls it now.

From here, recall questions go to the installed `rekal` skill. This one is done.
