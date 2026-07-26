# Contributing to Rekal

Thanks for wanting to help.

This file covers what a contribution has to satisfy. For how to build, test, and
release, see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

## Read the soul first

[SOUL.md](SOUL.md) is not a mission statement. It is the review bar.

Every design decision in this repository was made by running it through the
eight questions at the end of that file, and a pull request is reviewed the same
way:

1. Does it preserve immutability?
2. Does the intent stay next to the code?
3. Is it thin on the wire?
4. Does the data stay within git and the local machine?
5. Is it simple — zero dependencies, zero config?
6. Is it transparent — can the user see and remove everything?
7. Does the agent get what it needs?
8. Does it enable the agent's reasoning rather than replace it?

If a change answers *no* to any of these, it will be sent back — not because the
idea is bad, but because there is usually another way to get the same result.
Propose it in an issue first and we will look for that way together.

### What will be declined

Stated up front so nobody spends a weekend on it:

- **A server, daemon, or hosted component** as part of the product. (The local
  single-flight nomic daemon is an implementation detail of model loading, not
  a service.)
- **Telemetry, analytics, or crash reporting.** Any of it. We don't phone home.
- **A required API key or network call.** Rekal works fully offline. External
  backends may exist only as opt-in configuration that is off by default.
- **A mutation, consolidation, or forgetting path over `data.db`.** The ledger
  is append-only. Memory that can be rewritten cannot be trusted, and trust is
  what makes it shareable.
- **A corpus-tuned constant frozen into a skill script.** A number read off one
  dataset overfits the next. Gate on a signal the engine calibrates to be
  corpus-invariant, or report the raw signal and let the agent judge.
- **A new rule in the skill where a function would do.** Every rule is an
  unwritten spec for a tool that doesn't exist yet. Close the gap by moving
  capability down into the binary.

## Before you open a pull request

### Run the checks

```bash
mise run fmt && mise run lint && mise run test:ci
```

CI runs the same three. `./scripts/install-hooks.sh` installs a pre-push hook
that runs them for you.

### Update the docs in the same commit

This repository treats stale docs as worse than no docs. If your change touches
commands, packages, files, or behavior, update in the same change:

- **[CLAUDE.md](CLAUDE.md)** — the architecture and directory map. This is a
  standing rule, not a nicety.
- **`--help` text** — when a command's behavior changes.
- **[docs/spec/command/](docs/spec/command/)** — when a command spec changes.

### Write tests

- **Unit tests** live next to the source in `_test.go`, same package. Always
  use `t.Parallel()`. No network, no heavy I/O.
- **Integration tests** live in `cmd/rekal/cli/integration_test/` behind
  `//go:build integration`, use the `TestEnv` pattern for isolated temp repos,
  and test the public API only.
- Bug fixes should come with the test that fails without them. Several commits
  in this history are named for exactly that.

### Match the surrounding code

Write lint-compliant Go on the first attempt. Handle every error explicitly.
Follow the patterns already in [CLAUDE.md](CLAUDE.md) — `SilentError` for
already-reported errors, the shared preconditions (`EnsureGitRoot`,
`EnsureInitDone`) on every command except `init` and `clean`, the standard
command constructor shape.

### CLI output voice

Short sentences. Plain words. Say what happened, say what to do, stop.

```
rekal: not a git repository (run this inside a project)
rekal: captured 3 sessions, 847 turns
rekal: no sessions match "JWT expiry" in src/auth/
```

No exclamation marks. No emoji. No "oops."

## Commits and pull requests

Commit subjects follow `type(scope): what changed`, in the imperative, with the
*reason* preferred over the mechanism:

```
fix(checkpoint): skip capture while a rebase is replaying commits
fix(sync): keep the longest arriving copy of a session, not the first
test(index): pin that turns stay searchable after a session is re-indexed
docs: clarify skill auto-refresh vs rekal init
```

Types in use: `feat`, `fix`, `docs`, `test`, `refactor`. Scopes are package or
command names, comma-separated when a change spans two.

Keep a pull request to one concern. In the description, say what changed and
why; if it touches anything in the eight questions above, say which one and how
it still holds.

## Contributing to the research

The benchmark harness under `scripts/bench/` and `scripts/industry-bench/` has
stricter rules than the code, because numbers travel further than commits. The
honesty rules are binding — see
[docs/research/industry-bench/04-procedures.md](docs/research/industry-bench/04-procedures.md).
The ones that catch people most often:

- Vendor numbers appear only in a clearly labeled *self-reported* column. Our
  table contains only runs we executed ourselves.
- No averaging across benchmarks that measure different constructs.
- Report the gate's false-silence rate alongside its wins.
- **Rekal core is frozen during a benchmark run.** Needing to change core to
  make a benchmark pass is a finding to write down, not a fix to apply.

## Reporting things

- **Bugs and feature requests** — [open an issue](https://github.com/rekal-dev/rekal-cli/issues).
- **Security vulnerabilities** — do not open an issue. Follow
  [SECURITY.md](SECURITY.md).
- **Conduct concerns** — [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
- **Questions** — the [Discord](https://discord.gg/hDMj8zHH2).

## License

By contributing, you agree that your contributions are licensed under
[Apache-2.0](LICENSE), the same as the rest of the project. There is no CLA.
