# Rekal — Claude Code plugin

The bootstrap half of Rekal. This plugin ships **one setup skill and nothing
else**: it explains what Rekal is, installs the binary, and runs `rekal init`.

```
/plugin marketplace add rekal-dev/rekal-cli
/plugin install rekal@rekal-dev
```

The recall skill — substrate triage, the ledger workflow gate, the reference
pages — is **not** here. It is embedded in the binary and installed by
`rekal init` into `.claude/skills/rekal/`, versioned with the binary that
answers its commands.

That split is deliberate. A plugin tracks this repository's `main`; an installed
binary is whatever version the user has. Shipping the recall skill here would
put two copies in context at once and let the newer one describe flags the
user's binary does not have. One owner, no divergence — see
[`docs/design/plugin-distribution.md`](../docs/design/plugin-distribution.md).

Rekal itself: [github.com/rekal-dev/rekal-cli](https://github.com/rekal-dev/rekal-cli)
