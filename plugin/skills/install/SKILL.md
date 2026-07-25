---
name: install
description: >
  Install the Rekal binary on this machine. Use when `rekal` is not on PATH —
  a command reported `command not found` — or when the user asks to install
  Rekal. Once per machine, not per repository; to set up a repo that already
  has the binary, use the init skill instead.
---

# Install Rekal

Rekal is git-backed memory of prior AI sessions: why a change was made, what was
tried, what was rejected. This installs the binary. Setting up a repository is a
separate step — the `init` skill.

## Ask first

The installer downloads a release archive and writes an executable to the user's
machine. Say that plainly, then wait for an answer.

If the user declines, stop. Say Rekal is not available and answer from the tree.
Declining is a complete answer.

## Run it

The plugin ships the installer on `PATH`:

```bash
rekal-install
```

That is a vendored copy of the project's own `scripts/install.sh` — it detects
the platform, downloads the matching release from GitHub, and verifies it
against the published checksums. Installs to `~/.local/bin`; pass
`--target <dir>` to override.

Requires git, macOS or Linux. The archive is large (~160 MB — the binary embeds
an embedding model), so the download takes a moment.

## Verify

```bash
rekal version
```

If the shell reports `command not found` after a successful install, the install
directory is not on `PATH`. Tell the user which directory to add and let them
edit their own shell profile. Do not edit it for them.

## Next

The binary is installed but no repository uses it yet. If the user wants memory
in the current repo, continue with the `init` skill. Ask before running it —
that is a separate decision about a separate thing.
