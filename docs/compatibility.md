# Compatibility

What a version number promises, and what it does not.

Rekal follows [semantic versioning](https://semver.org/spec/v2.0.0.html). The
question semver leaves open is *what* the version covers, because a CLI has
several surfaces and they do not deserve the same guarantee. This page answers
that. It takes effect at 1.0.0.

## Covered by the version

Breaking any of these requires a major version.

### Your data

`data.db` is append-only and migrated forward, and this is the strongest
guarantee Rekal makes. A newer binary reads a store written by any older one,
migrating the schema in place on open. Rekal has no path that mutates or
deletes a captured turn — not a flag, not a subcommand, not a repair mode. If a
future version needs a shape the old rows cannot express, it adds columns or
tables; it does not rewrite history.

A **downgrade** is not covered. An older binary may not understand a newer
store.

### The wire format

Frames on a `rekal/<email>` branch carry a magic string and a format version.
The branch is append-only: publishing adds frames and never rewrites the body,
and no code path force-pushes. A reader tolerates frames it was not built for
by skipping them, which permits partial reading but does not guarantee lossless
re-export by an older writer. For native memory, upgrade every writer of a
memory-bearing branch: an old full rebuild can omit unknown records. Changing
the meaning of an existing frame field is breaking.

### Command and flag surface

Command names, flag names and their short forms, positional argument order, and
the meaning of each. A flag that exists keeps its meaning.

Removing or renaming a flag is breaking. A rename ships the old name as a
hidden alias for at least one major version — `--re-export` → `--rebuild` is
the precedent.

### Exit codes

Agents branch on these, so they are a contract:

| Code | Meaning |
|---|---|
| `0` | Useful ranked/prose results; unreleased main also succeeds for nonempty literal or memory results |
| `1` | No useful text-search results, or a runtime error |

### Default output shape

Recall's default output is the seed digest as text; `--json` is structured.
Adding a field to the JSON is not breaking; removing or repurposing one is.
Adding a line *kind* to the digest is not breaking; changing the header grammar
(`INJECT top= gap= N seeds`) is.

### Configuration

`.rekal/config.json` and the global file are additive. An unknown key is
ignored, not an error, so a config written for a newer binary does not break an
older one. Removing a key or changing a default's meaning is breaking; changing
a default's *value* is not, and ranking-weight defaults will move as retrieval
improves.

## Not covered by the version

These change in any release, including patches.

- **`index.db`** — the schema, its tables and its contents. It is a disposable
  projection of retained ledgers, current repository prose and configured local
  sources. Its schema is migrated on open; an empty index is rebuilt inline.
  `rekal index` regenerates it from available sources. Keep native local-note
  files if you need their contents: deleting them removes the rebuild source.
  Never depend on the index schema as a stable interface.
- **The SQL schema reachable through `rekal query --sql`.** That command is raw
  read-only access to the databases, offered for analysis and debugging. Table
  and column names are documented in `--help` and in
  [docs/db/](db/) as a convenience, not as an interface. Scripts that
  parse `data.db` tables directly will break; this is the one place where
  "documented" does not mean "stable".
- **Ranking.** Scores, ordering, which sessions surface for a query, and the
  weights that produce them. Retrieval quality is the product; freezing it
  would freeze the product. Absolute `confidence` and `mass` keep their
  meaning (absolute, corpus-independent) even as values move.
- **The recall graph and its `[reached N×]` hint.** Local, personal, derived.
- **The installed agent skill** under `.claude/skills/rekal/`. It is owned by
  the binary, refreshed on upgrade, and its internal structure is not an
  interface. Edit it and `rekal init` will overwrite your edits.
- **Log and progress output on stderr**, and the wording of any message.
  Parse exit codes and `--json`, never prose.
- **Anything behind `REKAL_HUNT_*`.** Research escape hatches, not
  configuration.

## Deprecation

A deprecated flag or command keeps working, hidden from `--help`, for at least
one major version, and its removal is a `!`-marked entry in
[CHANGELOG.md](../CHANGELOG.md). Nothing is removed in a minor or patch release
except a genuine defect — as when `push --force` was removed before 1.0,
because a flag that discards published frames contradicted the
append-only guarantee the wire format is supposed to *be*.

## Pre-1.0

Everything before 1.0.0 predates this page. `0.2.x` moved fast and broke flags;
that is what a `0.x` is for. The append-only store and wire designs are not new
at 1.0 — schema migration on open and skip-unknown frame decoding have been in
the codebase throughout — but they were never written down as a promise until
now, and this page is where they become one.

## Planned v2.0 search interface (unreleased)

The unreleased interface uses ordinary search for ranked context plus literal
conversation enumeration, and `rekal query --ref` for bounded source reads.
Default text output changes; JSON adds `ref` and `literal` fields. Existing
`find`, session queries, SQL and evidence commands remain supported.

This interface change does not rewrite commits, append to commit bodies, or
require a force push. Canonical references use session IDs and memory revision
IDs, not disposable `sN` numbering. Rebuilding the index from the same retained
sources preserves these references. Local note snapshots may expire.

No special reindex is required solely for search/read routing. An absent or stale
index still needs normal rebuilding/sync; this change does not automatically
backfill historical passage vectors. Memory wire compatibility and the need to
upgrade all memory-bearing branch writers remain separate requirements. Follow
the [upgrade guide](upgrading.md); the version proposal does not indicate a release.
