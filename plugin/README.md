# Rekal Memory — Claude Code plugin

The setup half of Rekal. Two commands, one vendored installer, nothing else.

```
/plugin marketplace add rekal-dev/rekal-cli
/plugin install rekal@rekal-dev
```

| Command | Scope | Does |
|---|---|---|
| `/rekal:install` | once per machine | installs the `rekal` binary |
| `/rekal:init` | once per repository | runs `rekal init` — store, hooks, transport branch, recall skill |

Both are also model-invoked: a `rekal` command that reports `command not found`
routes to the first, `not initialized` to the second. Both confirm before
touching your machine or repo.

`bin/rekal-install` is a byte-identical copy of the project's
[`scripts/install.sh`](../scripts/install.sh), pinned by a test. Shipping it here
means setup runs code that came with the reviewed, SHA-pinned plugin rather than
piping a live URL into a shell.

## What is not here

The recall skill — substrate triage, the ledger workflow gate, the reference
pages. That is embedded in the binary and installed by `rekal init` into
`.claude/skills/rekal/`, versioned with the binary that answers its commands.

A plugin tracks this repository's `main`; an installed binary is whatever version
the user has. Shipping the recall skill in both would put two copies in context
at once and let the newer one describe flags the user's binary does not have. One
owner, no divergence — see
[`docs/design/plugin-distribution.md`](../docs/design/plugin-distribution.md).

Rekal itself: [github.com/rekal-dev/rekal-cli](https://github.com/rekal-dev/rekal-cli)
