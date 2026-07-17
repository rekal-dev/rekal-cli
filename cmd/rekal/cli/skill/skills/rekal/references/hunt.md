# HUNT — pointed, gated episodic recall

Use after `recall-route.py` printed `INJECT`. Not for analytical mining
(mine.md) or present-tense code (grep).

## 1. Pipeline (required)

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
rekal "JWT expiry" | python3 "$ROOT/scripts/recall-route.py"
# optional filters:
rekal --file src/auth/ "token refresh" | python3 "$ROOT/scripts/recall-route.py"
rekal --commit <sha> | python3 "$ROOT/scripts/recall-route.py"
rekal -n 5 --explain "error handling" | python3 "$ROOT/scripts/recall-route.py"
```

| Route stdout | Action |
|---|---|
| `INJECT …` | Confident episode — drill below. Knowledge docs may also exist; do not let them block. |
| `KNOWLEDGE …` | Episode below bars; from `knowledge[0]` Read `path` at `lines` (`anchor`). **Do not** drill weak sessions. |
| `SILENCE …` | No confident episode and no knowledge. Say so. Do not pad with near-misses. |

Bars live only in `scripts/hunt-gate.py` (via recall-route): absolute
`confidence` ≥ 0.70 (soft ≥ 0.68 with gap ≥ 0.04; raw `mass` floor when
present). Knowledge fallback requires absolute `knowledge[0].score` ≥ 0.40
(max-norm knowledge scores are gone — weak prose must not route).
Max-normalized session `score` alone is not enough. Confident episode
outranks a non-empty knowledge block. No gate output → SILENCE.

## 2. One query is a guess — widen before you conclude

The ledger indexes the words the *past* session used, not the words you asked
with. Evidence routinely lands at rank 5-9, not rank 1 — `rekal` already returns
20 candidates (`-n` to change), so read past the first before you judge. A single
phrasing is one lookup; a confident answer survives more than one. Re-query and
fuse whenever the top episode does not answer, or before you conclude SILENCE:

| Reformulation | When | Example |
|---|---|---|
| keyword-only (drop question words) | always — cheap second look | `rekal "token refresh expiry"` |
| split a compound question | "X and Y", multi-hop | query each clause, then join |
| temporal emphasis | "when / before / after / first / last" | add the date/era, or `--file` for that period |
| entity / path anchor | you know a file or a name | `rekal --file src/auth/ "<q>"` |

Take the union of top candidates across phrasings: a session that surfaces under
two different queries is almost always the answer. Only SILENCE after a
reformulation also comes back empty — one blank query is not absence. This
multi-lookup loop is where recall moves from "top result" to "right result"
(LoCoMo: single-shot 0.88 → multi-lookup 0.98 evidence-recall).

## 3. Drill the strongest, cheapest first

When the top candidates' `confidence` values are within the gate's gap band,
drill the top 2-3 — not only #1. Never `--full` by default:

```bash
rekal query --session <id> --role summary
rekal query --session <id> --offset <snippet_turn_index - 2> --limit 5
rekal query --session <id> --role human
rekal query --session <id> --role human_steering
rekal query --session <id> --full   # last resort
```

Fields: `session_id`, `score`, `snippet`, `snippet_turn_index`,
`snippet_role` (`human_steering` = high intent), `summary_turn_index`,
`children`, `origin` (cross-repo — not this repo's conventions).

Cite session/turn/commit with every claim.
