# rekal find

**Role:** Advanced unpaged, time-ordered literal matching-turn enumeration.
Ordinary search now includes paginated literal coverage. Find remains useful for
chronological text output and role filtering. The sweep is complete over its
available matching rows; counts are turns, not facts or all related concepts.

**Invocation:** `rekal find "<term>" [role]`

Optional `role` ∈ `human` | `assistant` | `human_steering` | `summary`.

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done.

---

## What find does

1. **Run shared preconditions** — Git root, init done.
2. **Open the index DB** — ILIKE sweep over `turns_ft.content` (and role filter
   when set). No BM25 ranking, no limit — every matching turn. Coverage is the
   available index: local capture plus every synced teammate session (which never
   reaches `data.db`), minus the duplicate captures supersession collapsed. When
   there is no index to read — never built, or built empty — the sweep falls
   back to `data.db.turns` so a fresh store still answers.
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
| Exact literal sweep, paginated | `rekal "<term>"`, then returned `--cursor` |
| Unpaged chronological sweep or role filter | `rekal find "<term>" [role]` |
| Analytical / temporal SQL | `rekal query --sql "…"` |
