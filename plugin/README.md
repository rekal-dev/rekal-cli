# Rekal Memory

**Carry the reasoning forward.** Rekal gives coding agents a Git-linked record
of earlier decisions, rejected approaches and source evidence across sessions.

Capture supports Claude Code, Cursor, Codex, Gemini, Copilot, Kiro and OpenCode.
The binary captures conversations at commit time and shares eligible records on
`rekal/<email>` branches. Search locally, then read the relevant source. The agent
interprets the evidence; Rekal does not generate an answer on its behalf.

This plugin installs the published binary. Native Claude project-note capture,
literal pagination and the common `query --ref` reader are **unreleased main**
features, not part of v1.0.2. See the [release status](../CHANGELOG.md).

## Agent memory and original evidence

Main can capture existing Claude project memory notes alongside conversations.
Notes remain attributed revisions; they do not replace captured transcript text.
Rekal does not control distillation or write back into native agent files.
Explicit personal sources remain local to the search index.

## Git transport, local retrieval

The default embedding model ships inside the binary. No hosted memory service
is required. Git transport uses your remote; optional HTTP embeddings send text
to the endpoint you configure. Release checks contact GitHub without sending
conversation text. See [security boundaries](../SECURITY.md).

Repository records follow a merged-commit export gate. Local search indexes and
personal-note snapshots are not published. See [usage](../docs/usage.md) and
[upgrade compatibility](../docs/upgrading.md).

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
captures the session. `rekal clean` removes local setup and its store; it does
not retract copies already published to Git or held by teammates.

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
