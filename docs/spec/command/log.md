# rekal log

**Role:** Show recent checkpoints, like git log. List rows from the checkpoints table (data DB) — checkpoint id, git_sha, author, date (or equivalent). No filters. Nice and simple.

**Invocation:** Subcommand only — `rekal log [--limit <n>]`.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done. Log reads from the data DB (checkpoints table); no index required.

---

## What log does

1. **Run shared preconditions** — Git root, init done.
2. **Read checkpoints** — Query the `checkpoints` table in `.rekal/data.db` (e.g. order by ts desc). No filters.
3. **Apply limit** — Show at most `--limit` entries (default: implementation-defined, e.g. 20).
4. **Output** — Print the list (e.g. checkpoint id / git_sha, author, date). Git-log style. One line per checkpoint or implementation-defined.

---

## Flag

| Flag | Meaning |
|------|--------|
| `--limit <n>` | Max entries to show (default: e.g. 20). |

---

## Examples

```bash
rekal log
rekal log --limit 10
```
