# rekal checkpoint

**Role:** Capture the current session after a commit. Invoked by the post-commit hook; can also be run manually. Does not update the index.

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
6. **Write to data DB:**
   - Insert session row (`sessions` table) with ULID, content hash, actor type, email, branch, timestamp.
   - Insert turn rows (`turns` table) with role, content, timestamp.
   - Insert tool call rows (`tool_calls` table) with tool name, path, command prefix.
   - Update `checkpoint_state` cache.
7. **Create checkpoint** — Insert a `checkpoints` row linking to the HEAD commit SHA, branch, email.
8. **Link sessions** — Insert `checkpoint_sessions` junction rows and `files_touched` rows (from `git diff --name-status HEAD~1 HEAD`).
9. **Print summary** — `rekal: N session(s) captured` (silent if nothing new).

---

## No flags

No user-facing flags. Same behaviour when invoked by the hook or manually.

---

## Idempotent

If nothing changed since the last checkpoint (same file size + hash, or session already exists by content hash), no rows are written.
