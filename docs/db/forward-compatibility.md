# Forward compatibility: mixed rekal versions on one data.db

**Status:** decided and implemented (2026-07). Applies to `data.db` and,
to a lesser degree, `index.db`.

## Context

`data.db` is append-only and shared via git (SOUL.md: "distributed through
git... shared across the team"). A team does not upgrade `rekal` in
lockstep — someone commits a `data.db` written by a newer rekal version,
and a teammate on an older build pulls it and runs `rekal recall`. Until
now, migrations only handled the well-tested backward direction (an older
DB opened by a newer binary — see `schema_migrate_test.go`). Nothing
handled, or even detected, the forward direction.

Because every migration to date has been purely additive (new nullable
columns, new tables), an older binary opening a newer DB would _usually_
keep working by accident — it just ignores columns it doesn't select. But
nothing guaranteed that stays true, and nothing would tell the user why
something broke if a future change ever needed to be non-additive (a
renamed column, a changed meaning, a required migration step). The failure
mode would be a confusing raw SQL/type error three layers into a query,
not something a user could act on — which fails SOUL.md's character
directly ("honest... say what happened, say what to do, stop").

## Decision

Stamp both DBs with an integer schema version (`schema_meta.schema_version`
for `data.db`, reusing the existing `index_state.schema_version` key/value
row for `index.db`). `MigrateDataSchema`/`MigrateIndexSchema` — the single
choke point every command already routes through before touching either
DB — check the stored version against this build's
`CurrentDataSchemaVersion`/`CurrentIndexSchemaVersion` before doing
anything else. If the DB's version is higher than the build understands,
migration stops immediately and returns `db.SchemaVersionError`, whose
message is a complete, actionable sentence:

```
data.db was written by a newer rekal (schema v2) — this build only understands up to v1. Upgrade rekal to continue.
```

If the DB's version is lower or equal, migration proceeds exactly as
before (existing additive `addColumnIfMissing` calls), then stamps the
current version at the end.

A DB with no stamp at all (every DB written before this change) reads as
version 0 — always `<=` any current version, so every existing `data.db`
and `index.db` in the wild keeps opening cleanly. This is the same
"missing means oldest, never a hard failure" convention `addColumnIfMissing`
already uses for individual columns.

## Why this and not something else

- **A version check the older binary runs itself is the only version check
  that can actually stop it.** There's no central registry of "what's the
  latest schema" to consult — the DB itself has to carry that fact so any
  binary, old or new, can compare against what it knows.
- **One integer, not per-table/per-column granularity.** SOUL.md: "we don't
  add options where a good default exists." A single monotonic version
  covering the whole schema is enough to answer "can I safely proceed?" —
  finer-grained versioning (per-table, per-migration) would add real
  complexity for a question that only ever needs a yes/no answer.
- **`index.db` gets the same treatment even though it's lower-stakes.**
  `index.db` is local-only and always rebuildable, so a version mismatch
  there can only happen after a local downgrade, never from a teammate's
  data — there's no sharing risk. It's still checked, because "a clear
  error beats a confusing one" applies just as much locally; the fix is
  cheap (reuse the table `index_state` already has) and keeps the two DBs'
  migration code symmetric instead of one being defended and one not.
- **The version check lives inside `MigrateDataSchema`/`MigrateIndexSchema`,
  not duplicated at each of the ~8 call sites that invoke them.** Every
  command — `checkpoint`, `push`, `query`, `recall`, `sync`, `index` —
  already calls one of these two functions before touching either DB. That
  is the one choke point; adding the check there means every command gets
  the protection automatically, with no per-command wiring and no risk of
  a future command forgetting to check.

## Trade-off accepted: the error message isn't wrapped in the SilentError/
`rekal:`-voice pattern

Most user-facing messages in this codebase go through the `SilentError`
pattern (`fmt.Fprintln(cmd.ErrOrStderr(), "rekal: ...")` +
`NewSilentError(err)`) so the final output is exactly the polished
`rekal: <message>` voice SOUL.md's Voice section describes. Doing that here
would mean adding a type-switch on `*db.SchemaVersionError` at every one of
the ~8 call sites that call `MigrateDataSchema`/`MigrateIndexSchema`, each
constructing its own `fmt.Fprintln` + `NewSilentError` — the exact
per-call-site duplication the choke-point design above was chosen to avoid.

Instead, `SchemaVersionError.Error()` returns the complete, plain-English,
actionable message itself, and callers keep their existing
`fmt.Errorf("migrate data schema: %w", err)` wrapping unchanged. The
message the user sees has one extra `migrate data schema: ` prefix ahead of
the actual explanation — for example:

```
migrate data schema: data.db was written by a newer rekal (schema v2) — this build only understands up to v1. Upgrade rekal to continue.
```

This is a real, deliberate trade-off: consistency with the strict
`rekal:`-prefixed voice, in exchange for zero call-site changes and zero
risk of a future command's error path forgetting to translate this
specific error type. Given the message underneath is still a single clear
sentence naming the problem and the fix, this reads acceptably even with
the wrapping prefix, and matches how other "deep" errors already surface
in this codebase today (e.g. `checkpoint.go`'s
`"data DB is corrupt or unreadable: %w"`). If this ever becomes visibly
worse — for example if wrapping accumulates past one level at some call
site — revisit by having `Run()` in `root.go` special-case
`errors.As(err, &db.SchemaVersionError{})` centrally instead of at each
call site, which would fix every command in one place without the
per-call-site duplication this design avoided.

## What this does not cover

- **Non-additive migrations still need a real plan when one actually
  happens.** This mechanism only guarantees an old binary *fails clearly*
  instead of *failing confusingly* — it does not make a breaking schema
  change safe to ship. The day a migration needs to rename or repurpose a
  column, `CurrentDataSchemaVersion` bumps, and that is the trigger to
  design the actual compatibility story for that specific change (a new
  column alongside the old one, a translation view, etc.) — this document
  only covers detection, not every future migration's safety.
- **A newer binary reading an older DB** is the existing, already-tested
  direction (`TestMigrateDataSchema_OldDataDB`,
  `TestMigrateIndexSchema_OldIndexDB`) — unaffected by this change.
