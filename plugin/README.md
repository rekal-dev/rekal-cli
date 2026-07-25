# Rekal Memory

Code has git. Every line, every change, every author — recorded forever. The
reasoning behind the code has nothing.

Your agent starts every session blank. It reads the code. It does not know why
the code looks that way, what was tried last week, or which approach the team
already explored and abandoned.

Rekal is the ledger for that. Four ideas, in order.

## Save the transcript

The session is the record. Rekal writes it raw and append-only.

No summaries. No distillation. Nothing derived. A summary is a lossy guess about
what will matter later, made before anyone knows the question — and it rots
quietly while you trust it. The transcript does not rot. It is what was actually
said.

Immutable once written. Nobody edits it, nobody deletes it. That is what makes it
worth sharing: if anyone could edit the record, nobody could trust it.

## Agent agnostic

The transcript is the substrate, not one vendor's memory format.

Rekal reads sessions from Claude Code, Cursor, Codex, Gemini, Copilot, Kiro, and
OpenCode, and answers from all of them together. Switch agents next quarter and
the ledger is still yours.

## Reason at query time

Maintained memory layers answer from summaries built before anyone asked the
question, and need a consolidation pipeline kept healthy. Rekal has neither.

Nothing is precomputed into "memories." Retrieval, ranking, routing, judgment —
all of it runs when you ask, against the real record.

This is lazy evaluation applied to memory. Nothing is derived until a question
needs it, so nothing derived can be stale. The index is a disposable accelerator
over the ledger; delete it and one command rebuilds it from truth.

The skill orients your agent and hands it the record. It does not script the
steps. Tools remove toil, never thought — the judgment stays with the agent, and
gets better as the agent does.

## Git is the transport

No server. No account. No external memory service. Nothing to operate, nothing to
breach, because there is nothing to connect to.

Embedding is local. The model ships inside the binary, embeddings are computed on
your machine and stored in `.rekal/`, and they never travel. Semantic search with
no tier to run or pay for.

Memory travels the way code travels: `git push`. Merged work reaches your team.
Unmerged work stays on your machine. The store is two files in `.rekal/`.

Thin on the wire, rich on the machine — strip what git already has, compress what
remains, compute indexes and embeddings locally and never ship them. Recall runs
on your machine, median around 150 ms.

## Install

```
/plugin marketplace add rekal-dev/rekal-cli
/plugin install rekal@rekal-dev
```

Then two commands. Each says what it will do and waits.

| Command | Scope |
|---|---|
| `/rekal:install` | once per machine — installs the `rekal` binary |
| `/rekal:init` | once per repository — store, git hook, transport branch, recall skill |

Both also fire on their own when a `rekal` command reports `command not found` or
`not initialized`.

Requires git, macOS or Linux. After that, commit as normal — a post-commit hook
captures the session. `rekal clean` removes everything, with no residue.

## What this plugin is

Setup only: two skills and the installer they run.

The recall skill — the one that answers questions — ships inside the binary and
is installed by `rekal init`, so it always matches the version running your
commands. Bundling it here too would load two copies at once and let the newer
one describe flags your binary does not have.

`bin/rekal-install` is a byte-identical copy of the project's
[`scripts/install.sh`](https://github.com/rekal-dev/rekal-cli/blob/main/scripts/install.sh),
pinned by a test, so setup runs code that shipped with this plugin rather than a
live URL piped into your shell.

[rekal.dev](https://rekal.dev) ·
[Source](https://github.com/rekal-dev/rekal-cli) ·
[Design](https://github.com/rekal-dev/rekal-cli/blob/main/docs/design/plugin-distribution.md) ·
[Paper](https://arxiv.org/abs/2607.14390) ·
Apache-2.0
