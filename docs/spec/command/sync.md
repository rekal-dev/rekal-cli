# rekal sync

**Role:** Sync team context from remote rekal branches. Two modes: team sync (default) and self sync (`--self`).

**Invocation:** `rekal sync` or `rekal sync --self`.

---

## Preconditions

See [preconditions.md](../preconditions.md): must be in a git repository and init must have been run.

---

## Two Modes

### Team sync (default): `rekal sync`

Captures local work, pushes it, fetches remote branches, and rebuilds the search index from local data plus decoded remote wire format.

1. **Checkpoint** (non-fatal) — Capture the current session via `doCheckpoint`. If it fails, print a warning and continue.
2. **Push** (non-fatal) — Push local data to remote via `doPush`. If it fails, print a warning and continue.
3. **Fetch remote refs** (non-fatal) — `git fetch origin 'refs/heads/rekal/*:refs/remotes/origin/rekal/*'`. If fetch fails (no remote, offline), continue with local data only.
4. **List remote branches** — `git for-each-ref` on `refs/remotes/origin/rekal/`, excluding the current user's branch.
5. **Rebuild index** — Drop and recreate all index tables, then:
   - Populate from local `data.db` (sessions, turns, tool calls, files, facets, co-occurrence)
   - For each remote branch: decode wire format (`rekal.body` + `dict.bin`), insert into `turns_ft`, `session_facets`, `files_index` — **skip tool calls** for remote data
   - Create FTS index (BM25)
   - LSA embedding pass
   - Write index state
6. **Print summary** — `rekal: synced — N local sessions, N remote sessions from M team member(s)`.

### Self sync: `rekal sync --self`

Fetches your own remote branch and imports into `data.db` — useful for syncing across machines.

1. **Fetch own remote branch** — `git fetch origin rekal/<email>`. Fatal if fetch fails (that's the whole point of `--self`).
2. **Import to data.db** — Decode wire format from `origin/rekal/<email>`, import sessions + checkpoints into `data.db` with dedup by session ID and checkpoint ID. Tool calls are included.
3. **Full index rebuild** — Same as `rekal index`.

---

## Key differences between modes

| Aspect | Team sync | Self sync |
|--------|-----------|-----------|
| Checkpoint + push first | Yes (non-fatal) | No |
| Fetch scope | All `rekal/*` branches | Own branch only |
| Remote data goes to | Index DB only | Data DB (permanent) |
| Tool calls from remote | Skipped | Included |
| Fetch failure | Non-fatal | Fatal |

---

## Flags

| Flag | Description |
|------|-------------|
| `--self` | Only fetch your own rekal branch (not the whole team) |

---

## Error handling

- Checkpoint/push failures in team sync: non-fatal warnings — sync still fetches and rebuilds.
- Fetch failure in team sync: non-fatal — rebuild with local data only.
- Individual remote branch decode failures: non-fatal — skip branch, log warning, continue.
- `--self` fetch failure: fatal.

---

## When to run

Run `rekal sync` when you want to pull teammates' context. After sync, `rekal` recall and `rekal log` see both local and team sessions. Run `rekal sync --self` to pull your own context from another machine.
