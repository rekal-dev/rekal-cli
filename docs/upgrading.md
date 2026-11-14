# Upgrading from v1.0.2

This guide describes **unreleased main**, proposed for v2.0. It is not a release
announcement. Published installation commands remain pinned to v1.0.2 until an
actual release is verified. See [CHANGELOG.md](../CHANGELOG.md).

## What changes

| Area | Unreleased behavior |
|---|---|
| Ordinary search | Ranked conversations, separate memory/prose results and paginated literal conversation matches |
| Reading | `rekal query --ref` opens a returned source; `query --cursor` continues it |
| Native memory | Claude project notes captured by default at checkpoints; explicit personal sources remain index-only |
| Compatibility | Existing `find`, session queries, SQL and evidence commands remain available; default search output changes |

## Forward migration and reindexing

The intended upgrade direction is forward: newer binaries migrate local schemas
and read retained history; derived indexes can be rebuilt. A CLI major version
does not require a new wire format or rewritten remote history.

Reindexing repairs derived state, not missing source records. It cannot recover
unknown frames omitted by an older writer's full export rebuild, deleted native
local notes, or history that has not been imported. The current requirement to
upgrade all memory-bearing branch writers therefore remains necessary. Fully
automatic migration must preserve the remote record before it can rely on a
local rebuild. The steps below describe current behavior, not a claim that every
upgrade/backfill already happens automatically.

## Before opening an existing store

Back up `.rekal/data.db` and configuration while no Rekal process is writing.
Keep any original local-note files you need. `index.db` is disposable; your
unpublished ledger and native notes are not. Do not use `rekal clean` as an
upgrade or reindex operation: it deletes the local store.

Review `.rekal/config.json`. To stop new native note capture and local note
indexing before the first checkpoint, merge `"memory": false` into that object.
This does not erase existing repository history. A nonempty `memory_sources`
list replaces automatic discovery; explicit `scope: repo` allows publication
through the merged-commit gate. Omitted scope on custom sources means local.
See [configuration](configuration.md) and [native memory](design/native-memory.md).

Upgrade **every writer of a memory-bearing Git branch** before sharing notes.
Unknown-frame skipping does not make an old writer's full rebuild lossless.
An older binary can omit memory records it does not understand. Local schema
version 2 also prevents an unsafe downgrade; switching back is not a supported
migration strategy.

## After installing the candidate binary

```bash
rekal version
rekal init
rekal index
rekal embed
rekal "a decision you remember"
```

Run inside the initialized repository. `init` refreshes hooks and agent setup
without resetting stored history. `index` is recommended once for an existing
store to reclaim old index churn and build current projections. It is not
required solely for the new search/read syntax. Missing semantic vectors fill
through `embed`; index/sync also start background embedding. If another embed
process is active, let it finish. Inspect the completion/log output before
judging semantic retrieval quality.

Follow a returned `rekal query --ref ...` command and compare the source with
what you expect. `rekal sync` fetches/imports available teammate history; a plain
Git fetch is not a substitute for import and indexing. Do not publish just to
test an upgrade.

The first successful native-note scan captures a baseline. Historical revisions
are append-only; local personal snapshots retain only their current native file
contents. Rebuilding cannot restore deleted local files.

## References and consumers

Persist canonical `session:`, `passage:` and repository `memory:` references,
not `s1`/`s100` aliases, which can change with the index. Local-memory snapshots
expire after changes/removal. Search and read cursors are distinct and may expire
when their source changes; repeat the original request.

Consumers should use `--json` and handle optional/additional fields. Literal totals
count matching indexed conversation turns, not every fact, synonym or note.
Memory is a separate keyword result set; cross-type ranking and semantic note
embeddings are not implemented. A ranked result's default reference opens its
first anchor; `rekal evidence` supports combining multiple anchors explicitly.

## Release-readiness limits

Native checkpoint embedding can still be synchronous and slow on long transcripts.
A passing read-contract test is not a checkpoint-latency guarantee. The latest
LoCoMo diagnostic measures evidence coverage, not answer accuracy or Claude
summary quality. See [the recorded test boundaries](research/runs/2026-11-14-locomo-memory/README.md).

This upgrade changes neither ordinary code commit bodies nor the Git transport
branch arrangement. No force push or rewritten code history is required.
