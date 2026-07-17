# MINE — analytical questions, decompose then route

Hybrid search answers "which episode matches this phrase?" Analytical asks
need signal clusters. **Do not** pipe these through `recall-route.py` first —
decompose to SQL, gather, then route.

## Pipeline

1. **Classify.** Reflection / pattern-mining / open questions — not a single
   pointed episode (hunt) and not a tree fact.
2. **Decompose** into SQL with three filters:
   - **scope** — subsystem/topic via `files_index` (or whole-repo)
   - **signal** — words people type when the phenomenon happens
   - **role** — `human_steering` / `assistant` / any
3. **Gather turns** with session/turn pointers. Widen signals before giving up.
4. **Route the set:**
   - mistakes / steering → analytics.md (reflect)
   - decision arc → why.md (then `why-trail-gate.py`)
   - coverage of a scope → analytics.md (census)
   - one concrete episode revealed → hunt drill via recall-route

## Signal examples

| Ask | Signal words | Role | Then |
|---|---|---|---|
| my mistakes / corrections | `no,`, `don't`, `instead`, `always`, `never` | `human_steering` | reflect |
| agent mistakes from missing context | `already exist`, `assumed`, `turns out`, `i missed` | `assistant` | why-shaped |
| still undecided | `TODO`, `undecided`, `open question`, `not sure` | any | census / distill |
| why X | choice + `because`, `instead of`, `rejected` | any | why.md |

## Example gather

```bash
rekal query --index "SELECT session_id, turn_index, role, substr(content,1,400) FROM turns_ft \
  WHERE role = 'human_steering' \
  AND (content LIKE '%auth%' OR content LIKE '%token%') \
  ORDER BY session_id, turn_index"
```

## When not to MINE

- Clear episode noun phrase → recall-route → hunt.md
- Named alternatives already in mind → why.md
- Present-tense code → Tree. Structure → map.md
