# Analytics — pick one mode, then scan

Shared spine: SQL over `turns_ft` / `session_facets`, **bound the scope
before you scan**, cite session ids. Prefer a MINE gather when you already
have signal turns.

| Mode | When | Bound |
|---|---|---|
| Reflect | recurring human corrections → rules | topic or path |
| Distill | orient: context / decision / rules / boundary | task |
| Census | exhaustive coverage, not relevance | time / path / author / topic — required |

## Reflect — steering → rules

```bash
rekal query --index "SELECT session_id, turn_index, substr(content,1,500) FROM turns_ft \
  WHERE role = 'human_steering' ORDER BY session_id, turn_index"
```

Cluster repeats ("always X", "never Y", "use Z not W"). One durable rule beats
re-learning the same correction.

## Distill — four libraries

| Library | Question | Where |
|---|---|---|
| Context | what is established | summaries, knowledge prose |
| Decision | what is still open | undecided / TODO language |
| Rules | what we prefer | `human_steering` repeats |
| Boundary | what was abandoned | rejected / don't / instead-of |

Zoom via `rekal --explain` and `file_cooccurrence`.

## Census — exhaustive, bounded map-reduce

**State the scope in the first sentence of your plan.** No scope → do not start.

1. Inventory sessions in scope (`session_facets` filters).
2. Walk a deterministic spine (ordered ids).
3. Summarise each once (summary role / short window).
4. Reduce into one digest with session-id traceability.

Ranking or cherry-picking → you want hunt, not census.
