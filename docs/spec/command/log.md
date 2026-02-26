# rekal log

**Role:** Show recent checkpoints, like `git log`. Lists checkpoints from the data DB with session counts.

**Invocation:** `rekal log [--limit N]`.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done. Reads from data DB; no index required.

---

## What log does

1. **Run shared preconditions** — Git root, init done.
2. **Query checkpoints** — `SELECT` from `checkpoints` joined with `checkpoint_sessions` for session count, ordered by `ts DESC`.
3. **Apply limit** — Show at most `--limit` entries (default: 20).
4. **Output** — Git-log style, one block per checkpoint:
   ```
   checkpoint <ULID>
   Date:     2026-02-25T10:00:00Z
   Commit:   abc123...
   Branch:   main
   Author:   alice@example.com
   Sessions: 2
   ```

---

## Flag

| Flag | Meaning |
|------|--------|
| `--limit <n>` | Max entries to show (default: 20) |

---

## Examples

```bash
rekal log
rekal log --limit 10
```
