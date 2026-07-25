# Rekal Memory

**Your agent starts every session blank. Rekal gives it your team's reasoning — why this approach, what got tried, what got thrown away — recalled from git in ~150 ms.**

[rekal.dev](https://rekal.dev) · [GitHub](https://github.com/rekal-dev/rekal-cli) · [Paper (arXiv:2607.14390)](https://arxiv.org/abs/2607.14390)

---

Every AI session settles decisions. Then it ends, and the reasoning is gone. Next
week a different agent proposes the thing you already rejected, and nobody
remembers why it was rejected.

Rekal captures each session at commit time, stores it **raw in git**, and indexes
it **locally**. No server. No vector database. No subscription. The store is two
files in `.rekal/` that travel with your repo.

```
$ rekal "should webhook retries use a fixed delay?"

INJECT top=0.81 gap=0.28 2 seeds
  01JNQX8F2K9M conf=0.81 t14 [reached 3×] "no, a fixed 5s delay stampedes the
  downstream on recovery. Use exponential backoff with jitter…"
```

## The numbers

| | |
|---|---|
| **90.6%** | LoCoMo accuracy — **98.6%** top-20 recall |
| **86.6%** | LongMemEval accuracy — **99%** top-20 recall |
| **~150 ms** | median local recall |
| **158×** | thinner on the wire than the raw transcript |
| **0** | memory servers, vector DB tiers, SaaS subscriptions |

Retrieval runs locally over git, with no memory layers and no external memory
service. On the accuracy rows the answering agent is GPT-5 Sol — Rekal supplies
the memory, the model supplies the answer.
[Full tables and methodology →](https://github.com/rekal-dev/rekal-cli#benchmarks)

## Install

```
/plugin marketplace add rekal-dev/rekal-cli
/plugin install rekal@rekal-dev
```

Then two commands, each asking before it touches anything:

| Command | Scope | Does |
|---|---|---|
| `/rekal:install` | once per machine | installs the `rekal` binary |
| `/rekal:init` | once per repository | store, git hooks, transport branch, recall skill |

Both also fire on their own when a `rekal` command reports `command not found`
or `not initialized` — so your agent recovers without you remembering the
command.

Requires git, macOS or Linux.

## After setup

Commit as normal. Your session is captured by a `post-commit` hook. In any later
session — yours or a teammate's — the agent asks Rekal first and gets the
decision *and the reason the alternative was rejected*.

Merged work travels to your team over plain `git push`. Unmerged work stays
local. `rekal clean` removes everything, no residue.

Works alongside Claude Code, Cursor, Copilot, Codex, Gemini, Kiro, and OpenCode.

## What this plugin is, precisely

Setup only: two skills and the installer they run. The **recall skill** — the one
that actually answers questions — is embedded in the `rekal` binary and installed
by `rekal init`, so it always matches the version running your commands.

`bin/rekal-install` is a byte-identical copy of the project's
[`scripts/install.sh`](https://github.com/rekal-dev/rekal-cli/blob/main/scripts/install.sh),
pinned by a test. Setup runs code that shipped with this reviewed plugin rather
than piping a live URL into your shell.

Why the recall skill isn't bundled here:
[`docs/design/plugin-distribution.md`](https://github.com/rekal-dev/rekal-cli/blob/main/docs/design/plugin-distribution.md).

Apache-2.0 · [rekal.dev](https://rekal.dev)
