# Rekal separate memory.bin: executable proof

This experiment tests whether native memory snapshots can live in a separate binary data file while Rekal's existing session wire format stays unchanged. All notes, identities, repositories and remotes used by the tests are synthetic. No user memory was read, captured or published.

## What is exercised

The harness compiles and executes the **exact `CommitWireFormat` function** from Rekal commit `16cc899274af026f8a8d6cf8216d391be9aa8489`, along with its three unchanged Git helpers. `source_provenance.json` records the source and function hash. The entire Rekal CLI, database pipeline, adapters and search engine are not built or tested here.

It also compiles a small proposed writer, `harness/transport/companion.go`. That writer reads the existing Git tree into a temporary index, updates only the supplied files, creates a commit and updates the ref with a compare-and-swap check. It leaves the user's working index untouched. This is isolated prototype code, not a patch applied to Rekal.

`memory_proof.py` implements an experimental append-only companion archive and a disposable JSON index. The archive uses Python's built-in zlib to keep the proof dependency-free; a production implementation can reuse Rekal's existing zstd codec. **This is not a proposed final byte-level wire standard.**

## Findings to inspect in the test results

1. Existing notes retain their exact bytes and native `modified` timestamp. Observation time is a separate field; unknown native timestamps remain unknown.
2. Repeating an identical snapshot is a byte-for-byte no-op.
3. A memory-only change appends a frame. Every previous archive byte remains an unchanged prefix, and old content remains available.
4. Unchanged file content is stored once; complete snapshot membership points to content hashes.
5. Deleting and rebuilding the derived index produces identical record IDs and citation backlinks.
6. Personal/private fixture scopes are rejected before team-archive serialization. This tests the archive boundary with explicit classifications; it does not prove real adapter scope discovery.
7. Identical relative paths from different owners, profiles or repositories remain distinct. A removal affects only its source namespace.
8. An incomplete directory scan cannot be interpreted as deleting its files.
9. Truncated records, changed checksums and unsupported format versions are rejected.
10. The current Rekal writer builds a new tree with only `dict.bin` and `rekal.body`. A `memory.bin` companion disappears from the **new tip's tree**, while remaining available in the earlier Git commit.
11. The proposed preserving writer retains `memory.bin` through a session-only update and retains both session files through a memory-only update.
12. A push to a local bare remote followed by a fresh teammate clone preserves the archive exactly; the derived index and observed code-commit reference rebuild successfully.
13. A later invocation of the old writer still omits the companion. Updating one client does not make a mixed-version deployment safe.
14. A memory file on a separately named branch is unaffected by the unchanged old session writer.
15. A stale proposed writer is rejected without replacing the winning writer's update. This is a stale-write test, not a stress test or automatic merge implementation.
16. Unrelated sidecar files and the user's staging index survive the proposed writer.

See `test-results.txt` for test names and `results.json` for machine-readable results. `example-memory.bin` and `example-index.json` contain one synthetic onboarding example.

## Small experimental contract

Each complete source snapshot has:

- A source namespace: adapter, profile, owner, repository, source root and sharing scope. These are fixture labels, not authenticated identities or an access-control implementation.
- Sorted relative paths, content hashes and optional source modification timestamps.
- A content-derived snapshot ID.
- An observation time and the code commit at which that version was observed.
- Only content blobs not already in this archive.

Each record is framed with a compressed length and SHA-256 checksum. The file has a magic/version header. A complete empty snapshot records that a source currently contains no files without deleting older snapshots. Missing/incomplete reads are rejected rather than becoming removals.

Explicit citations stay in the captured Markdown, using the current emitted shape `SESSION_ID:tTURN[:bytesSTART-END]`. Reverse links are extracted on index rebuild. The prototype labels them as explicit citations, not verified support, and does not invent links for notes lacking citations. It does not resolve them through the real Rekal CLI.

The sample is an **observed baseline**, not evidence that the note was created by the capture commit. A file timestamp does not establish its author, originating session, causal code change, accuracy or permission to share.

## Two viable physical layouts

| Layout | Required change | Compatibility consequence |
| --- | --- | --- |
| `memory.bin` beside `rekal.body` and `dict.bin` on the existing per-user branch | Make all writers preserve existing tree entries; add companion capture/import/index handling | An old writer on the same branch can still omit the companion from the next tree |
| `memory.bin` on `refs/heads/rekal-memory/<writer>` | Add a separate fetch/push path for that ref namespace | The tested old session writer does not touch this branch; the current session fetch wildcard does not fetch it either |

A separate branch has the same repository access boundary. It does not make personal content private from repository readers. Teammate sync should read shared snapshots as evidence; it need not overwrite the receiving agent's native memory files.

## Fit with SOUL.md

- Preserve observed bytes in an immutable ledger.
- Keep transport in Git and indexes local.
- Reuse existing session payloads unchanged.
- Deduplicate content and avoid storing files that Git already tracks; the latter is a production integration requirement, not implemented in this untracked-note fixture.
- Let adapters discover native files and native scope.
- Let agents judge applicability and adoption. No new distillation model, provider API, synchronization service or native-memory write-back is required by this experiment.

## Limits before production

- The fixture parser recognizes a simple YAML `modified` line; it is not a complete Claude or cross-agent adapter.
- The tests supply explicit sharing scope. Real discovery must resolve configured paths, profile boundaries and native user/project/local semantics.
- The tests reject incomplete scans but do not prove an atomic snapshot of a directory that another process is changing.
- Unchanged snapshots are no-ops, even at a new code commit. Recording every unchanged snapshot-to-checkpoint observation is a separate contract decision.
- The current-per-source projection does not implement historical `--commit` queries, conflict reconciliation between devices, embeddings, ranking or an improvement in LoCoMo scores.
- The prototype does not apply Rekal's merged-only gate, add source authentication, migrate an existing store, run the full CLI, or implement automatic retry after compare-and-swap rejection.
- Malformed frame tests do not constitute exhaustive fuzzing or hostile-input hardening.
- No version tag, remote branch or release was created or changed on GitHub.

## Reproduce

Requirements: Git, Python 3.11+ and Go 1.25.6 or later. The harness has no third-party Go module dependencies.

```sh
cd harness
go build -buildvcs=false -o ../probe ./cmd/probe
cd ..
python3 test_proof.py
```

The included harness is already extracted. To independently check it against a checkout of the recorded upstream commit:

```sh
python3 prepare_harness.py /path/to/rekal-checkout-at-16cc899
```

To inspect the saved example without compiling Go:

```sh
python3 - <<'PY'
import json
from pathlib import Path
from memory_proof import rebuild
print(json.dumps(rebuild(Path('example-memory.bin').read_bytes()), indent=2))
PY
```

Copied upstream code is covered by `UPSTREAM_LICENSE`. All newly written files here are an experimental proof, with no production compatibility promise.
