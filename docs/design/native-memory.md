# Native memory integration

Memory uses ordinary checkpoint, push, sync and recall paths. Rekal never writes
back to native agent files or controls the agent's distillation process.

## Default: repository memory

Claude project memory is discovered automatically in the primary worktree's
conventional Claude project memory directory. No configuration is needed. The
next checkpoint captures a baseline; later checkpoints append changed notes or
deletion markers. Unchanged text is a no-op. Memory-only changes can create a
checkpoint with zero conversation sessions.

These repository notes follow the same redaction, merged-commit export gate and
ordinary push/sync flow as conversations. They use the existing user branch and
`rekal.body`; first export appends typed compressed frames. No force push or extra
branch is required. Association with a merged commit records when a note was
observed, not proof that every sentence concerns merged work.

Missing automatically discovered directories are quiet no-ops, even if a directory
was previously observed. Only a successful scan of an existing directory can
establish that a note was deleted. Invalid or unreadable sources produce a warning
and leave memory history unchanged; transcript capture can continue.

## Local sources: index only

Personal/account-wide sources are not automatically scanned. Explicit local
sources are read during ordinary recall and stored only in
`index.db.local_memory_documents`. They never enter data.db, checkpoint revisions,
wire frames, or the historical memory archive. Their native files are the source
of truth. Removing a note, disabling its source, or a failed refresh removes its
searchable cached snapshot. Refresh retries on the next recall.

The local index retains only the current snapshot. Rebuilding it re-reads native
files; it cannot recover deleted personal notes or provide historical time travel.
Unchanged snapshots retain the same reference across refreshes and rebuilds.
Changed or removed snapshots invalidate their earlier references.

## Configuration

All memory settings are local to the repository's gitignored `.rekal/config.json`.
They are not inherited from global configuration or received from teammates.
Merge settings with any existing configuration.

Set `"memory": false` to stop native capture and local-source indexing. It does
not remove previously captured repository history or previously published notes.
Removing the switch or setting it true restores configured/default behavior.

A nonempty `memory_sources` list replaces automatic discovery. This example keeps
default repository capture and adds one local source:

```json
{
  "memory_sources": [
    {"name": "claude-project", "adapter": "claude", "scope": "repo"},
    {"name": "personal-notes", "adapter": "markdown", "scope": "local",
     "path": "/absolute/path/to/personal/notes"}
  ]
}
```

Custom sources default to `scope: local` when scope is omitted. Explicit
`scope: repo` includes that source in repository capture and ordinary publication.
There is no separate sharing switch. `profile` defaults to `default`; use distinct
names/profiles for distinct accounts. Do not configure the same source twice.

The Claude adapter respects `CLAUDE_CONFIG_DIR` and, when set alongside it,
`CLAUDE_CODE_PROJECT_DIR_NAME`. Custom `autoMemoryDirectory` and managed settings
require an explicit path; Rekal does not interpret Claude's trust hierarchy.
Paths are absolute or relative to the primary worktree. JSON paths do not expand
shell `~`. Local source paths are not put on the wire.

Adapters accept UTF-8 Markdown recursively, bounded to 1 MiB/note, 16 MiB/source
and 1,024 notes. Symlinks, non-regular notes and files changing during read fail
the scan. Existing transcript redaction/anonymization applies before storage.
Claude's simple frontmatter `modified` and `type` metadata is retained when
recognized; unknown metadata remains text rather than acquiring invented meaning.

## Identity, history and retrieval

Stable source identity uses name, adapter, Git user email and profile. Scope and
publication do not participate in that identity. Changing source scope does not
promote earlier private history: entering repository capture takes a new observed
baseline when there is no prior repository revision. Changing to local scope
stops future ledger capture but cannot retract already captured/published history.

Repository revision IDs address immutable records associated with checkpoints and
code commits. The index contains `memory_archive` (historical revisions) and
`memory_documents` (latest nondeleted repository revisions, plus retained legacy
records). Current selection orders observations by timestamp, then revision ID;
it is deterministic but is not causal conflict resolution across machine clocks.

`modified_at` is filesystem time; `source_modified_at` is recognized native time;
`observed_at` is capture/index observation time. None establishes when a fact
became true. Local snapshot hashes exclude observation time so an unchanged file
does not acquire a new reference at every search.

Ordinary recall returns bounded keyword memory hits with source/scope labels,
revision references opened by `rekal query --ref <reference>`. Reads are bounded
and return a continuation for long notes. Advanced SQL can still target
`local_memory_documents` or `memory_archive`. Sessions retain their existing ranking. At most five memory
notes and bounded UTF-8 snippets are returned. Author, commit, path and limit
filters apply; local notes have no invented checkpoint/commit association.
Actor-filtered queries exclude memory.

Explicit `SESSION:tTURN:bytesSTART-END` citations expose pointers to existing
evidence reads. They are unverified citations, not proof of support. No session
links are inferred from a memory checkpoint. The common `query --ref` reader is implemented for memory, conversations and
passages. Semantic memory embeddings and cross-type ranking are not implemented.

## Compatibility

Schema-1 memory revisions remain readable without rewriting their payloads or
revision hashes. Earlier `share: false` configurations map to local sources;
`share: true` maps to repository sources. Legacy private ledger history stays
private and is not promoted into exports or ordinary recall. New observations
use explicit scope in revision schema 2. Legacy source IDs are recognized when
finding prior repository revisions; index grouping handles both identities.

Data schema version 2 prevents unsafe local downgrades. New binaries migrate old
stores, but older binaries on another machine can still omit unknown memory
records during full wire rebuilds. Reindexing does not fix that wire limitation.
Upgrade every writer of a memory-bearing branch before use. Ordinary append
preserves the existing body prefix; local reindex does not modify Git wire data.

All integration fixtures are synthetic. Tests exercise repository capture/export,
local-only indexing and rebuild, deletion and failed scans, stable snapshot/source
identities, legacy configuration, forbidden ledger/wire insertion, and revision
validation. The broader experimental scenario matrix is not a claim that every
prototype behavior ships here.

Claude storage reference: <https://code.claude.com/docs/en/memory#storage-location>.
