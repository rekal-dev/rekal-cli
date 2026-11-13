# Native memory integration

Rekal can capture native Markdown memory alongside conversations, without owning
agent distillation or adding a command family. Capture is optional and off by
default. The existing checkpoint, push, sync, index, recall and read-only query
paths do the work. No agent file is written back or modified.

## Enable a source locally

Add a source to the repository's gitignored `.rekal/config.json` (merge it with
any existing settings):

```json
{
  "memory_sources": [
    {
      "name": "claude-project",
      "adapter": "claude",
      "profile": "work",
      "share": false
    }
  ]
}
```

The Claude adapter defaults to the primary worktree's Claude project memory
directory. It respects `CLAUDE_CONFIG_DIR` and, when set alongside it,
`CLAUDE_CODE_PROJECT_DIR_NAME`. For a custom `autoMemoryDirectory`, managed
settings, or another native agent's memory directory, give an explicit `path`.
Rekal does not interpret Claude's settings/trust hierarchy. Paths can be absolute
or relative to the repository's primary worktree; use an absolute path rather
than shell `~` expansion in JSON.

```json
{
  "memory_sources": [
    {
      "name": "native-project-notes",
      "adapter": "markdown",
      "path": "/absolute/path/to/native/memory",
      "profile": "work",
      "share": false
    }
  ]
}
```

`markdown` accepts UTF-8 `.md` files recursively. `claude` uses the same reader
and additionally extracts recognized simple `modified` and `type` frontmatter
scalars. Complex/unrecognized metadata remains in the archived text without
inventing a timestamp or type. Custom native formats need an adapter implementing
`memory.Adapter`; they are not treated as Markdown implicitly.

Source identity consists of name, adapter, git user email, profile and sharing
policy. Keep these stable across your machines. Use distinct source names or
profiles for different accounts. The local directory path is not sent to Git.
Global settings cannot enable capture in repositories: `memory_sources` is
per-repository only. An empty list disables future capture.

The next ordinary checkpoint captures existing notes as an observed baseline.
No onboarding command is required. A source configured before its directory
exists produces a warning and leaves memory history unchanged. Transcript
capture can continue. The same applies to unreadable, invalid, oversized or
symlinked sources. Scans are bounded to 1 MiB per note, 16 MiB per source and
1,024 notes. A failed scan never means all the notes were deleted.

## Local versus shared

`share` defaults to false. Private revisions stay in the local data store; the
export path cannot serialize them. Recall includes a private source only when
its owner/profile/source identity is enabled locally. Shared sources can be
retrieved by teammates after the existing push/sync flow. This retrieval policy
is not encryption or a security boundary against someone who owns the local
filesystem and can query its databases.

Setting `share: true` authorizes sharing **every Markdown note in that source**.
A note's `type: project`, directory name or association with a repository does
not independently grant permission to publish it. Existing secret redaction
and path anonymization apply before storage; archived text is the scrubbed
observation, not a promise of byte-for-byte native-file preservation.

The existing merged-checkpoint gate also applies to shared memory. The linked
code commit says when the note was observed, not that every sentence originated
in that commit or only describes merged work. Turning sharing off later cannot
retract history already published through Git.

## Ledger, wire and index

Each changed note creates a content-addressed immutable revision associated with
a checkpoint. Identical content is a no-op, even if filesystem mtime changes.
A removal appends a tombstone; previous revisions remain readable. Reappearance
creates another observation even if its text matches an older version.
Memory-only changes create checkpoints with zero sessions. Memory is never
encoded as a fabricated conversation or human turn.

`memory_revisions` in data.db is the local ledger. A typed `FrameMemory` member
(`0x07`, schema 1) travels inside the existing compressed `rekal.body` batches.
Only changed note revisions travel. The existing user branch and transport are
reused; there is no extra archive, service or memory branch.

The local index contains:

- `memory_archive`: exact scrubbed revisions from the local ledger and shared
  Git wire history, including revisions that are no longer current.
- `memory_documents`: the latest revision per source and relative path, excluding
  tombstones. It is a disposable projection, not editable source data.

`rekal index` rebuilds from the local ledger, following its existing behavior.
`rekal sync` rebuilds the team view from local data and fetched teammate branches.
Repeated imports are idempotent. Rebuild order uses observation time, then revision
ID as a deterministic tie-break. It does not attempt a semantic merge of concurrent
notes or claim remote clock ordering establishes truth.

Timestamps retain separate meanings: `modified_at` is filesystem modification
time; `source_modified_at` is a recognized native timestamp (otherwise absent);
`observed_at` is Rekal capture time. Frontmatter type is provenance, not a weight.

## Consumption uses existing paths

Ordinary `rekal "query"` adds a labelled MEMORY block; JSON adds `memories`.
Session ranking remains unchanged. Memory search currently uses bounded keyword
matching in a separate block, with at most five notes and bounded UTF-8 snippets.
There is no memory confidence gate, recency/type boost or mandatory handshake.
The existing author, commit, path and limit filters apply; actor-filtered searches
exclude memory because notes are not session actors. Memory path filters refer to
note paths, not inferred code-file associations.

Each hit returns `memory:<revision-id>` and a read-only SQL drill for the exact
archived revision. Use the existing query command:

```sh
rekal query --index --sql "SELECT payload FROM memory_archive WHERE id = '<revision-id>'"
```

Explicit `SESSION:tTURN:bytesSTART-END` citations are translated to the existing
evidence-anchor grammar and exposed as bounded pointers. They are unverified
citations, not proof of support. The agent can follow them through the existing
evidence command and decide what to adopt. Notes lacking citations acquire no
invented session links. No automatic native-memory write-back occurs.

Semantic memory embeddings, calibrated cross-type ranking, semantic conflict
resolution and a dedicated reverse-link projection are not part of this change.
Historical session extents are not inferred from a memory checkpoint. The
prototype scenario matrix remains a broader research/acceptance inventory, not a
claim that every experimental behavior ships here.

## Compatibility and verification

Data schema version 2 prevents older binaries from rewriting a local ledger that
contains these records. Readers validate revision hashes, schema, sharing scope
and checkpoint/commit associations. Updated writers refuse unknown wire members
and re-export refuses to drop memory missing from the local ledger.

Old binaries on another machine cannot be retroactively guarded: they may skip
memory members and lose them during a full re-export. Upgrade every writer of a
shared Rekal branch before enabling shared memory. The experimental ZIP wire
fixtures are not production data and are not a supported import format.

Tests cover memory-only checkpoints, unchanged scans, edits/deletions/reappearance,
private-source filtering before ranking, private export rejection, exact revision
drill, explicit citation translation, deterministic rebuild, idempotent imports,
unsupported schemas and source-scan failure. All fixtures are synthetic.

Claude storage reference: <https://code.claude.com/docs/en/memory#storage-location>.
