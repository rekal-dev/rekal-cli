# rekal push

**Role:** Push local Rekal data to the remote branch. Exports unexported checkpoints from DuckDB to wire format, commits to the orphan branch, and pushes to origin.

**Invocation:** `rekal push`, `rekal push --force`, or `rekal push --re-export`.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run.

---

## What push does

1. **Run shared preconditions** — Git root, init done.
2. **Check local branch** — Verify the orphan branch (`rekal/<email>`) exists. If not, print "no data to push" and exit.
3. **Check remote** — Verify `origin` is configured. If not, print "no remote configured" and exit.
4. **Export wire format** — Query `data.db` for unexported checkpoints, then apply the **merged-only gate**: keep only checkpoints whose `git_sha` is an ancestor of the default branch (`gitx.DefaultBranch` → `origin/HEAD` → `origin/main`/`master` → local `main`/`master`). Unmerged/abandoned work is held back — it stays `exported = FALSE` and is re-evaluated on the next push, so it releases automatically once its branch merges. `data.db` keeps every branch locally; only the wire is filtered. See [merged-only-sharing design](../../design/merged-only-sharing.md). For each shared checkpoint:
   - Encode linked sessions as `SessionFrame` (turns + tool calls, zstd compressed).
   - Encode checkpoint as `CheckpointFrame` (git SHA, files touched, session refs).
   - Append a `MetaFrame` with summary counts.
   - Update string dictionary (`dict.bin`) with session IDs, emails, branches, paths.
   - Mark checkpoints as `exported = TRUE`.

   **Note on merge workflow:** the ancestor test is exact for merge-commit and rebase workflows. Under squash-merge the branch's original commits never become ancestors of the default branch, so squash-merged sessions are held back (fail-closed) rather than risk sharing unmerged work; squash support is a follow-up (see the design doc).
5. **Commit to orphan branch** — Write `rekal.body` and `dict.bin` via `git hash-object` + `git mktree` + `git commit-tree`. Uses the HEAD commit message from the main branch.
6. **Compare with remote** — Skip push if local and remote SHAs match.
7. **Push** — `git push --no-verify origin rekal/<email>`. Handle non-fast-forward with a warning suggesting `--force`.

---

## Flags

| Flag | Description |
|------|-------------|
| `--force`, `-f` | Force push, overwriting the remote branch with local data |
| `--re-export` | Rebuild the branch's wire data from scratch out of the local data DB, then force push. Implies `--force`. |

When a normal push is rejected (non-fast-forward), push prints a warning and suggests `rekal push --force`. Force push is safe because each user owns their branch and the local DuckDB is the source of truth.

`--re-export` re-encodes every **merged** checkpoint in data.db into a fresh rekal.body and dict.bin, ignoring exported flags and the branch's current contents. Use it to repair a branch whose wire bytes were written by a rekal version with the frame-count bug (sessions with more than 255 turns or tool calls were corrupted on the wire), or to drop stale meta frames accumulated by past pushes. The merged-only gate applies here too, so a repair regenerates the branch as merged-only and never re-leaks unmerged work. The branch is derived data; data.db is the source of truth.

---

## Hooked to git push

`rekal init` installs a pre-push hook that runs `rekal push` on `git push`. When invoked by the hook, `--force` is not passed — conflicts are reported and resolved on the next manual push.
