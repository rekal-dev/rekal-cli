# Capture and incremental-index audit — 6 November 2026

Baseline: `b069d2dc014a7f5df804ac0e863f1f2cb399cd4b` (after PR #88).
Linux amd64, Go 1.25.6, DuckDB from the repository lockfile, `GOMAXPROCS=4`.
This is a capture/storage compatibility check, not an answer-quality benchmark.

## Findings and changes

| Finding | Result |
| --- | --- |
| Copilot opens every session's events file to determine its repo on every checkpoint | Cache positive and negative CWD metadata by adapter, physical path, actual file size and nanosecond mtime. Load metadata once per checkpoint; batch only changed records. Directory enumeration/stat remains proportional to inventory. |
| Codex has the same global repo-matching scan | Use the same optional discovery cache, preserving all CWDs recorded by session metadata and turn contexts. |
| Incremental index deletes/reinserts the whole growing session | Insert missing immutable turn/tool IDs. Replace only one bounded facet row per affected session and invalidate its stale vectors. Avoid rewriting unchanged commit messages. |
| Incremental index drops a session's older file links | Insert missing links from all historical checkpoints. This also repairs affected sessions whose links older versions lost. |
| Recent Codex rollouts can be compressed | Discover and stream `.jsonl.zst` in both active and archived stores. Hash decoded contents and share the logical plain path's append mapping across compression/resume. |
| Copilot ignores a scanner failure after valid records | Reject partial parses on oversized records or read failures; do not mark them captured. The 10 MiB per-record limit remains. |
| Recent Claude compact summaries include display flags and extra records | Regression covers `isVisibleInTranscriptOnly`, `isCompactSummary`, `compact_boundary`, progress records and injected `isMeta` messages. Summary stays a summary, genuine user input stays human, harness records are filtered. |

The discovery table is local metadata alongside `checkpoint_state`, not a new
wire record. A cache read failure falls back to parsing; a failed transcript
read is not persisted as a negative result. Deleting this cache costs one cold
scan, not lost conversation history. No source conversation row is rewritten.

## Measured discovery work

Run:

```bash
GOMAXPROCS=4 go test ./cmd/rekal/cli -run '^$' \
  -bench '^BenchmarkSessionDiscoveryCache$' -benchtime=3x -count=1
```

1,000 Copilot session directories, all belonging to another repo; short synthetic
headers on this host. Includes metadata load, directory walk and file stats;
excludes cache flush, parsing matching sessions, indexing and embeddings.

| State | Transcript opens per discovery | Time per discovery |
| --- | ---: | ---: |
| Uncached | 1,000 | 28.61 ms |
| Warm | 0 | 5.90 ms |
| Four signatures invalidated | 4 | 4.81 ms |

The small timing difference between warm and four-changed is noise. The useful
invariant is the file-open count, not an extrapolation to a user's cold disk.
Separate regression tests overwrite an existing transcript, leave its parent
directory mtime unchanged, and confirm cache invalidation, including equal-size
replacement and a previously nonmatching session that now matches.

## Measured index growth

Compared baseline and fixed code with the same isolated synthetic store: one
session, 1,000 starting turns, then 20 captures appending 10 turns each. Each
capture adds a distinct checkpoint/file link. Both code paths open/close the
index per capture and run the production incremental index including FTS.

| After 21 captures | Baseline | Fixed |
| --- | ---: | ---: |
| Live turn rows | 1,200 | 1,200 |
| Stored `id` slots (`pragma_storage_info`) | 23,100 | 1,200 |
| First surviving turn's row ID | 21,900 | 0 |
| Historical file links | 1 | 21 |
| Index file bytes | 12,333,056 | 7,352,320 |

Elapsed time was 7.68 s / 7.37 s; this small run does not establish a throughput
gain. File size depends on content/compression and FTS still rebuilds snapshots.
The regression in `db/incremental_append_test.go` pins preserved turn identity,
correct growing row counts, idempotence, complete file links, refreshed facets
and invalidated stale vectors. Existing physical bloat is reclaimed by one
`rekal index`; the change prevents the whole-session rewrite cause but cannot
promise constant storage as the corpus grows.

## Upstream compatibility evidence

- [OpenAI Codex compression implementation](https://github.com/openai/codex/blob/b1a0dff15854d5f62a9c0931431d70f6148da9e6/codex-rs/rollout/src/compression.rs)
  and [recorder tests](https://github.com/openai/codex/blob/b1a0dff15854d5f62a9c0931431d70f6148da9e6/codex-rs/rollout/src/recorder_tests.rs)
  establish the compressed physical representation and materialization before
  append. This does not imply compression is enabled in every Codex install.
- [Claude Code official package metadata](https://registry.npmjs.org/@anthropic-ai%2fclaude-code/latest)
  reported version **2.1.318** during this audit. Reviewed the vendor changelog
  and current hook documentation. The package is a native-binary wrapper; this
  audit did not run an authenticated 2.1.318 conversation, so it is not a blanket
  end-to-end certification of that version.
- [Observed Claude 2.1.220 compaction records](https://github.com/anthropics/claude-code/issues/82509)
  provide the display-flag shape used by the regression (synthetic contents).
- [Claude hook input contract](https://code.claude.com/docs/en/hooks#common-input-fields)
  documents asynchronous transcript writes and `last_assistant_message` on
  Stop/SubagentStop. A git post-commit hook captures flushed transcript text;
  later text is appended on the next checkpoint. This patch does not register
  a new Claude Stop hook or synthesize text not yet present in the transcript.
- `CLAUDE_CONFIG_DIR` and nested subagent transcript discovery were already
  supported; the existing adapter tests remain in the validation gate.

## Validation

Focused adapter/cache/index regressions and production checkpoint integration
tests passed. Lint passed with zero issues. The full local race suite passed
all other packages, including the integration package (99.59 s); only nomic's
two Unix-socket bind tests failed with `socket: operation not permitted` on
this host. All four Python regressions passed separately (the Go failure stops
the combined task before Python). GitHub CI results are recorded in the pull
request; earlier model validation remains in
[cloud validation](2026-11-06-cloud-validation.md).

Current Work Mode has no native rollout JSONL available for this thread. Session
recording uses an explicitly labelled export of visible conversation text; it
does not claim automatic capture of the hosted client or hidden context.
