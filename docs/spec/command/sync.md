# rekal sync

**Role:** Receive the team's Rekal context. Fetches all remote `rekal/*` branches from the same git remote (e.g. origin), merges only **core** data from other users into the local data DB (no tool-call payloads), then rebuilds the index so recall and log see the combined data.

**Invocation:** Subcommand only — `rekal sync`.

---

## Git remote and rekal branches

Rekal uses the **same git remote** as the repo (e.g. `origin`). On the remote (e.g. GitHub), it's just **one branch per user** — normal branches named `rekal/<user_email>` (e.g. `rekal/alice@example.com`). The branch you see as "yours" is the one you create and push to; everyone else has their own. Same repo, same remote; no special zone. When you **sync**, you fetch **all** `rekal/*` branches, merge each branch's dump into your **single** local `.rekal/data.db`, then **rebuild the index once**. So: many branches on the remote (one per user), one local data DB and one local index after sync. Config stores which remote to use (e.g. `config.json` → `git_remote: origin`).

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run. Remote must be configured (e.g. `origin`). No user prompts; use existing config.

---

## What sync does

1. **Run shared preconditions** — Git root, init done; exit with shared message/warning if not.
2. **Fetch rekal branches** — Fetch all `rekal/*` branches from the configured remote (e.g. `git fetch origin 'refs/heads/rekal/*:refs/remotes/origin/rekal/*'`). No code branches are updated.
3. **Merge core data only** — For each fetched rekal branch, apply its dump to `.rekal/data.db` with idempotent semantics (e.g. INSERT OR IGNORE). **Only sync the core from other users:** sessions (conversation turns, metadata), checkpoints, checkpoint_sessions, files_touched. **Ignore or strip tool-call detail** when ingesting from others — do not merge full tool-call payloads. That keeps sync lean and preserves context (who said what, when, which files) without copying heavy tool-call data. Index DB is never synced; it is rebuilt locally from the merged data DB.
4. **Rebuild index** — Run the index step (full rebuild) so the index reflects the merged data DB. See [index.md](index.md). After sync, recall and log see both local and team checkpoints.
5. **Exit** — Print short status (e.g. branches fetched, rows merged, index rebuilt). Success or clear error.

---

## Index rebuild after sync

Sync **requires** an index rebuild. We use the same data DB and same index model: the index is built from the data DB. After merging other users' data into the data DB, we run a full index rebuild so recall and log see the combined data. No incremental index merge from sync — one rebuild after the merge.

---

## Idempotent merge

Sync does not replace the data DB. It merges: remote rows are applied with idempotent insert (e.g. by session id / checkpoint id). Re-running sync is safe; already-present rows are skipped. Data DB remains source of truth; sync only adds or updates from remote.

---

## Flags

### `--self`

Only fetch your own rekal branch from the remote, not the whole team. Useful when working across multiple machines with distributed git — e.g. pulling your own context from your work laptop to your home machine without syncing the whole team's data.

When `--self` is set, sync fetches only `rekal/<your_email>` instead of `rekal/*`. The merge and index rebuild steps are the same; only the fetch scope changes.

---

## When to run

Run `rekal sync` when you want to pull teammates' context (e.g. after they've pushed their rekal branches). After sync, `rekal log` and recall can show their checkpoints too. The skill can tell the agent not to run sync mid-session unless the user asks — it's a deliberate "pull team context" step.
