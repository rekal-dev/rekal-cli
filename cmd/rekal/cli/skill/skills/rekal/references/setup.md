# Setup — rekal is not ready here

You reached this because a rekal command could not run. Three states, three
fixes. Diagnose from the error, do one thing, retry the original command.

Setup changes the user's machine or repository. **Ask before each step.** If the
user declines, stop using this skill: say memory is not available here, and
answer from the tree.

## `rekal: command not found`

The binary is not installed. It is one binary — CLI, database, embedding model,
compression dictionary, all embedded. Requires git, macOS or Linux.

```bash
curl -fsSL https://raw.githubusercontent.com/rekal-dev/rekal-cli/main/scripts/install.sh | bash
```

This downloads and runs a script from the internet. Say that plainly when you
ask. Installs to `~/.local/bin`; `--target <dir>` overrides.

Verify with `rekal version`. If the shell still reports command not found, the
install directory is not on `PATH` — tell the user which directory to add and
let them edit their own shell profile.

## `rekal: not initialized (run rekal init)`

The binary is there; this repository has no store. From the repository root:

```bash
rekal init
```

That writes `.rekal/` (the store), a post-commit hook that captures sessions, an
orphan branch `rekal/<email>` for transport, the skill under `.claude/skills/`,
and one marker-tagged line in `CLAUDE.md`. `rekal clean` removes all of it.

## Initialized, but recall says nothing

A fresh store holds no memory. Recall is silent because there is nothing to
recall — that silence is correct, not a broken install. Memory starts at the
next commit.

If the team already shares a ledger, pull it before concluding anything:

```bash
rekal sync
```

## When not to set up

If the user asked a question and rekal simply is not here, do not install
anything. Answer from the tree and say memory is not available in this
repository. Setup is a decision the user makes, never a fallback you take.
