# Real session fixtures

These are sanitized copies of real Claude Code session transcripts captured
on a developer machine, not hand-written JSON. They exist to close a gap
that has shipped two bugs despite a fully green test suite (a migration bug
and a checkpoint/index-update bug — both only caught by manually validating
against real transcripts; see `docs/REVIEW_2026-07-03.md`, finding P1-2).
Hand-written fixtures only ever contain the fields their author remembered
to type; real transcripts carry harness noise a synthetic fixture easily
omits by accident (`isMeta` wrappers, `queue-operation` entries, sidechain
duplicates, real subagent `meta.json` sidecars, deeply nested
`tool_use`/`tool_result` chains).

## What's here

The directory layout mirrors `~/.claude/projects/<sanitized-repo-path>/`
exactly, because `session.discoverSessionRefs` depends on that shape:

```
<trunk-session>.jsonl
<trunk-session>/subagents/agent-<id>.jsonl
<trunk-session>/subagents/agent-<id>.meta.json
```

Every `.jsonl` line and every `.meta.json` file has been scrubbed with
`scripts/gen-session-fixtures.go`, which applies the same secret-redaction
and path/username-anonymization the production capture path applies
(`cmd/rekal/cli/scrub`), and truncates long string values (tool output,
file contents) to keep the fixture set small — structure matters here, not
payload size.

`cmd/rekal/cli/session/real_fixtures_test.go` runs the real discovery+parse
pipeline against these fixtures.

## Refreshing

To pull a fresh batch from your own machine's Claude Code history for this
repo:

```sh
go run scripts/gen-session-fixtures.go \
  ~/.claude/projects/-Users-<you>-path-to-rekal-cli \
  cmd/rekal/cli/session/testdata/real \
  60
```

The last argument caps lines copied per `.jsonl` file (keep it small —
this directory is meant to stay a few hundred KB, not a full history dump).

**Before committing a refresh:** diff the output and skim it by hand. The
scrubber redacts known secret/path patterns; it does not guarantee every
possible sensitive value is caught, and it doesn't know what in your own
conversation history might be project-sensitive beyond generic PII. Treat
this the same as any other "do I want this in git forever" review.

Prefer *adding* new fixture files over blindly overwriting the whole
directory when the goal is capturing one specific new shape (e.g. a queue
cancel/remove operation, a teammates run, a new harness format) — that way
existing regression coverage doesn't silently shrink.
