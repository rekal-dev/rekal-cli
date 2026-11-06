# Changelog

Notable changes to Rekal. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versions follow [semantic versioning](https://semver.org/spec/v2.0.0.html) under the
scope defined in [docs/compatibility.md](docs/compatibility.md).

Per-release notes for every `0.1.x` and `0.2.x` build are on the
[Releases page](https://github.com/rekal-dev/rekal-cli/releases). This file
starts at 1.0; it records what 1.0 ships and everything after.

## [Unreleased]

### Fixed

- Checkpoint caches Copilot and Codex transcript repo metadata by file size and
  nanosecond mtime, including sessions belonging to other repositories. Warm
  discovery still enumerates files but avoids reopening unchanged transcripts.
- Incremental indexing appends missing turn/tool rows instead of rewriting a
  growing session's complete history. Historical checkpoint/file links are
  preserved and repaired; stale session vectors still invalidate. Existing
  index bloat can be reclaimed with `rekal index`.
- Codex discovers and streams compressed `.jsonl.zst` rollouts, hashes decoded
  contents, and keeps the same capture identity when Codex materializes a
  compressed rollout on resume.
- Copilot rejects oversized-record scan failures instead of recording a
  successful partial capture. Recent Claude compact-summary, progress and
  harness-message shapes have explicit compatibility coverage.

## [1.0.3] - 2026-11-06

### Fixed

- Codex capture honors `CODEX_HOME` for both repo-scoped discovery and
  cross-repo import, reads nested metadata IDs and response-only message
  transcripts, preserves message timestamps, avoids mirrored-message
  duplication, reports oversized-record errors, and excludes similarly named
  sibling repositories. Shell tool calls also accept the `cmd` argument.
- The installer extracts release archives without restoring foreign owners,
  fixing installs in containers with restricted UID/GID mappings. The vendored
  setup plugin carries the same fix (plugin 0.3.5).
- The industry benchmark smoke shim explicitly requests JSON, uses the shipped
  binary's digest instead of removed route scripts, accepts legitimate SILENCE
  exits, preserves engine order and inclusive knowledge, and discloses its two
  diagnostic recall calls. Python regressions now run in the release test gate.
- Cloud setup instructions use the current CLI, explain existing LFS hooks and
  Codex exports, and verify embedding models without waiting on a persistent
  lock file.
- The installer resolves the release version off the `releases/latest`
  redirect instead of the GitHub API. Unauthenticated API calls are capped at
  60 an hour per IP, so a shared office egress could exhaust the budget before
  a user typed anything, and some networks reach `github.com` and
  `raw.githubusercontent.com` but never allowlist `api.github.com` — both
  ended the install at "Could not fetch latest version from GitHub". The
  redirect uses the same host the download already needs. When resolution does
  fail it now reports the HTTP code each host returned and names
  `REKAL_VERSION`, which skips the lookup.
- README install commands work as written. `--target` is shown with the
  `bash -s --` a piped shell requires, and `REKAL_VERSION` in front of the
  shell that runs the script rather than in front of `curl`, where it applied
  to the download and never reached the script — silently installing the
  latest version instead of the pinned one. The commands now name a version so
  a first install does not depend on the lookup at all.

## [1.0.2] - 2026-08-04

### Fixed

- `rekal push --rebuild` refuses when the remote branch holds checkpoints this
  machine's `data.db` does not — it no longer silently deletes another
  machine's already-shared frames.
- The merged-only export gate judges ancestry against local `main` when it is
  ahead of a stale `origin/main`, so a checkpoint ships on the same `git push`
  that merges it rather than waiting for the next one.
- Fish installs create `~/.config/fish/` before appending the PATH export, so
  a machine that never configured fish still gets a working profile line.

## [1.0.1] - 2026-08-04

### Fixed

- Installer no longer refuses machines without Claude Code — Cursor, Codex,
  Gemini, Copilot, Kiro and OpenCode users can follow the documented
  `curl | bash` path.
- Headless installs (CI, containers, no controlling tty) no longer crash on
  the PATH-prompt step; they silently append instead.
- Drilling a session missing from both databases reports a plain "session not
  found" instead of a raw SQL driver string.
- `rekal sync --help` documents that a plain sync also checkpoints and pushes
  the caller's own local work before fetching.

## [1.0.0] - 2026-08-04

First stable 1.0 release. The store format, wire format, command surface and
exit codes are covered by the version number under
[docs/compatibility.md](docs/compatibility.md). Ranking and `index.db` remain
deliberately unfrozen.

### Capabilities

- **Capture.** Sessions from Claude Code, Cursor, Codex, Gemini, OpenCode,
  Copilot and Kiro, snapshotted by a `post-commit` hook into an append-only
  `data.db`. Re-capturing a growing conversation appends to its existing
  session rather than storing it again. No capture runs during a rebase.
- **Redaction.** Secret redaction (pattern plus Shannon entropy) and
  home-directory path anonymization run before anything is written to the
  database, so nothing unscrubbed can reach the wire later.
- **Sharing.** Merged work only. A checkpoint reaches the orphan branch
  `rekal/<email>` when its commit is an ancestor of the default branch, or its
  branch landed as a patch-equivalent squash. Unmerged spikes stay local.
- **Recall.** Hybrid BM25 + LSA + deep-semantic ranking with additive facet,
  recency and recall-graph layers, returning a compact seed digest by default
  and raw JSON under `--json`.
- **Knowledge layer.** Tracked prose at HEAD chunked and searched alongside
  sessions, answering "what is the convention now" with a file and line pointer
  instead of an old conversation.
- **Recall graph.** Query-time edges from a session to the query that reached
  it, local-only and never pushed, surfacing as a `[reached N× drilled N×]`
  hint and an opt-out ranking boost.
- **Single binary.** Database, embedding model, inference engine, full-text
  extension and compression dictionary all embedded. No runtime, no service, no
  API key.

### Changed since 0.2.x

- `rekal push --force` is **removed**. The append-only wire format is a
  structural guarantee, not a policy, and a flag that overwrites a branch would
  demote it to the latter. No code path force-pushes; discarding a ref remains
  an ordinary git operation.
- `rekal push --re-export` is renamed `--rebuild`. The old name still works as
  a hidden alias and will be removed no earlier than 2.0.
- `rekal push` reports a publication failure as a warning by default, and as an
  error only under `--strict`, so a diverged memory branch cannot abort the
  user's `git push`.
- Short flags across the command surface (`-p`, `-c`, `-a`, `-A`, `-n`, `-j`,
  `-e`, `-s`, `-r`, `-o`, `-F`, `-q`, `-i`).
- `rekal query --session` accepts the short handle from a digest (`s14`) as
  well as a full ULID.
- `rekal query --sql` runs on a read-only handle and accepts a single
  statement.
- Zero-argument subcommands reject extra words instead of silently answering a
  different question; `rekal -- <query>` forces recall past a command-name
  collision.
- `rekal init` runs a structural index rebuild so the first recall does not
  silently pay for an empty `index.db`.

### Documentation

- A recorded demo driven by a real agent, generated by
  `scripts/demo/record.py` rather than staged.
- README corrected against the shipped binary: `rekal clean` residue, the seed
  digest format, platform support, and the download and on-disk size.
- Compatibility contract published in
  [docs/compatibility.md](docs/compatibility.md).

[Unreleased]: https://github.com/rekal-dev/rekal-cli/compare/v1.0.2...HEAD
[1.0.2]: https://github.com/rekal-dev/rekal-cli/releases/tag/v1.0.2
[1.0.1]: https://github.com/rekal-dev/rekal-cli/releases/tag/v1.0.1
[1.0.0]: https://github.com/rekal-dev/rekal-cli/releases/tag/v1.0.0
