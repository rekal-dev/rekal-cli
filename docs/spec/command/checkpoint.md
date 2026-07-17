# rekal checkpoint

**Role:** Capture the current session after a commit. Invoked by the post-commit hook; can also be run manually. Incrementally updates the index for newly captured sessions.

**Invocation:** `rekal checkpoint`.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run.

---

## What checkpoint does

1. **Run shared preconditions** — Git root, init done.
2. **Find session directory** — Locate Claude Code session files under `~/.claude/projects/` matching the current git repo.
3. **Check for changes** — For each session file, compare size + SHA-256 hash against `checkpoint_state` cache. Skip unchanged files.
4. **Dedup by content hash** — Check `sessions.session_hash` to skip already-imported sessions.
5. **Parse transcript** — Extract conversation turns and tool calls from session JSON. Skip sessions with no turns and no tool calls.
5b. **Skip harness / bench pollution** — If `session.SkipCapture` is true, do not write the session (BUG 6). Triggers: `REKAL_BENCH` or `REKAL_SKIP_CHECKPOINT` set; transcript cwd under a bench tree (`scripts/bench`, `rekal-bench`); human turns matching RekalBench `gen_queries.py` prompt fingerprints. Same guard applies to cross-repo local import. See [scripts/bench/README.md](../../scripts/bench/README.md).
6. **Write to data DB:**
   - Insert session row (`sessions` table) with ULID, content hash, actor type, email, branch, timestamp.
   - Insert turn rows (`turns` table) with role, content, timestamp.
   - Insert tool call rows (`tool_calls` table) with tool name, path, command prefix.
   - Update `checkpoint_state` cache.
7. **Create checkpoint** — Insert a `checkpoints` row linking to the HEAD commit SHA, branch, email.
8. **Link sessions** — Insert `checkpoint_sessions` junction rows and `files_touched` rows (from `git diff --name-status HEAD~1 HEAD`).
9. **Incremental index update** — If index.db exists, incrementally add new sessions to the index:
   - Insert turns into `turns_ft` (auto-indexed by DuckDB FTS).
   - Insert tool calls into `tool_calls_index`.
   - Insert session facets into `session_facets`, then build their `facet_text` documents (incremental `PopulateFacetText` — the facet FTS index itself is only rebuilt by full `index`/`sync`, mirroring how turns FTS is handled).
   - Insert file entries into `files_index`.
   - Generate nomic-embed-text embeddings for new **trunk** sessions only (on supported platforms) — subagent/workflow transcripts are still FTS-indexed immediately, but their embeddings are deferred to the next full `rekal index`/`rekal sync` rebuild, so checkpoint time doesn't scale with subagent fan-out (see [agent-metadata.md](../../agent-metadata.md)).
   - LSA embeddings are skipped (require full corpus rebuild via `rekal index`).
   - Refresh the knowledge layer for the commit that fired the hook — watermark-gated, re-chunking only prose files whose git blob SHAs changed (usually zero or a few) — then embed up to 256 chunks still missing vectors (budgeted bite; full backfills run via `rekal embed` after index/sync). Best-effort: never fails the commit; recall re-runs the same refresh if skipped. See [knowledge-layer design](../../design/knowledge-layer.md) and [embed.md](embed.md).
   - Non-fatal: if incremental update fails, a warning is printed and the index can be rebuilt later with `rekal index`.
10. **Print summary** — `rekal: N session(s) captured` (silent if nothing new).

---

## No flags

No user-facing flags. Same behaviour when invoked by the hook or manually.

---

## Idempotent

If nothing changed since the last checkpoint (same file size + hash, or session already exists by content hash), no rows are written.
