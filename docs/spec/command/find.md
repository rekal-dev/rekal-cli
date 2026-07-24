# rekal find

**Role:** Complete, time-ordered enumeration of every ledger mention of a term —
the command for "all / every / how many / which of the N" questions where a
partial list is a wrong answer.

**Invocation:** `rekal find "<term>" [role]`

Optional `role` ∈ `human` | `assistant` | `human_steering` | `summary`.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done.

---

## What find does

1. **Run shared preconditions** — Git root, init done.
2. **Open the index DB** — ILIKE sweep over `turns_ft.content` (and role filter
   when set). No BM25 ranking, no limit — every matching turn.
3. **Output** — One compact text line per mention:

   ```text
   <session_id> t<turn> <ts> <role>: …context around the match…
   ```

   then a total count. Drill a mention with:

   ```bash
   rekal query --session <session_id> --offset <turn-2> --limit 5
   ```

There is no `--json` flag; the text lines are the stable agent surface. Use
`rekal query --sql` when a program needs structured rows.

---

## When to use find vs recall

| Ask | Command |
|-----|---------|
| "What do we know about X?" / best seeds | `rekal "<q>"` (hybrid + digest) |
| "List every mention of X" / complete set | `rekal find "<term>"` |
| Analytical / temporal SQL | `rekal query --sql "…"` |
