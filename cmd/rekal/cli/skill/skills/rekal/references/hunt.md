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

### Reformulation is one lever; depth is the other — but going deep is a judgment

The bigger the history, the deeper the right memory can sit: in a large corpus
the evidence often lands past rank 20. But depth is not free. Every extra
candidate is more tokens and, worse, more plausible-but-wrong distractors. On a
question whose answer is *not* in the ledger, digging deeper manufactures false
positives and breaks SILENCE. So decide *whether* to go deep before you do —
that judgment is the skill, not the widening itself.

**Go deeper only when a signal says the evidence exists but is mis-ranked:**

- the gate was `INJECT` / a candidate looked confident, yet nothing you read
  actually answers the question — the right memory is likely just lower;
- scores are flat with small gaps (no decisive winner) — the top is not the last
  word;
- the question shape hides its anchor: **temporal** ("how long between X and Y",
  "when did I first…") or a **single-mention preference / fact** stated once long
  ago and easily outranked by topical chatter;
- the history is large (many sessions) — more room for evidence to sit deep.

```bash
rekal -n 50  "<q>"   # first widening, when a signal above is present
rekal -n 100 "<q>"   # temporal / single-mention, still not surfaced
```

**Stay shallow — and prefer SILENCE — when nothing points to buried evidence:**

- reformulation *and* the first window both came back weak and flat — that is
  absence, not depth; do not dig for a positive;
- a strong, clear top hit already answers — stop.

Depth recovers evidence that is mis-ranked; it does not create evidence that
isn't there. Evidence for the payoff when the signal *is* present: on a
large-haystack corpus 11 of 15 top-20 misses were already sitting at rank
21-100 — recoverable by depth alone (evidence-recall 0.97 → 0.99, window 20 →
100), all of them temporal or single-mention questions.

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
