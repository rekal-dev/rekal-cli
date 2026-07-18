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

### Complete-set questions: enumerate, don't rank

"All", "every", "in what order", "which of the N" — ranked recall returns the
loudest matches, not the full set. Switch to SQL enumeration, and make it
exhaustive:

```bash
rekal query "SELECT COUNT(*) FROM turns WHERE role='human' AND content ILIKE '%<term>%'"
# then page: ORDER BY ts LIMIT 50 OFFSET 0, 50, 100 … until you've seen every row
```

- **Count before you LIMIT.** An `ORDER BY ts LIMIT 20` you never paged is a
  silent truncation — the answer is often in the rows you cut.
- **LIKE is stricter than search.** Reformulation applies to SQL patterns too:
  synonyms and adjacent vocabulary, not just the words your first hits used.
- **Check the set size.** If the question fixes N, do not answer until you can
  point at N distinct events, each verified in context.

## 2b. Time questions navigate by time, not by keyword

The ledger has a clock. Use it before you rank.

```bash
rekal query "SELECT MIN(ts), MAX(ts), COUNT(DISTINCT session_id) FROM turns"   # coverage window of the record
rekal query "SELECT ts, session_id, content FROM turns WHERE ts BETWEEN '<from>' AND '<to>' AND role='human' ORDER BY ts"
```

- **Anchor first.** "A month ago", "the day before X" are relative — resolve
  the anchor before hunting, then scan the window.
- **"Now" is the asker's present, not the record's edge.** `MAX(ts)` is where
  the ledger *stops*, not today — time has usually passed since the last entry.
  Prefer the real current date when you have it. If a relative reference
  ("a month ago") computed from the true "now" lands at or past the record's
  edge, the latest events in the record are the candidates — check the edge,
  not just the interior window.
- **Event time ≠ mention time.** A turn's `ts` is when it was *said*. "Last
  month I…", "back in February…" shift the event. Date the event, not the
  mention — and expect the report of a night or a trip to land in the *next*
  session, after it happened.
- **Routine ≠ episode.** "I usually / around 10pm" describes a habit. A
  question about one specific occasion needs the past-tense report of that
  occasion. Do not let a routine stand in for the episode.
- **One event in the window is not the answer** until you have scanned the
  whole window and no competing event sits in it.

## 2c. Whose fact is it?

Before answering a "my / I" question — a preference, a possession, an
experience — drill to the turn where the user *asserts* it: "my X is", "I
bought", "I made", "it turned out". A thing that was only discussed, suggested,
or compared is not the user's. When two candidates compete (two brands, two
recipes, two events), drill both to the ownership or outcome statement; answer
from the one the user owns, and keep the other out of the answer — a near-miss
entity in the output is a wrong answer, not extra color.

## 3. Drill the strongest, cheapest first

When the top candidates' `confidence` values are within the gate's gap band,
drill the top 2-3 — not only #1. A snippet is a keyhole: before trusting a hit,
read the room around it — 5 turns costs almost nothing and catches misreads
that a snippet invites (a routine mistaken for an episode, a discussed item
mistaken for an owned one). Never `--full` by default:

```bash
rekal query --session <id> --offset <snippet_turn_index - 2> --limit 5   # first move
rekal query --session <id> --role summary        # if summary_turn_index present
rekal query --session <id> --role human          # the user's own words, whole session
rekal query --session <id> --role human_steering
rekal query --session <id> --full   # last resort — whole transcript in context
```

Fields: `session_id`, `score`, `snippet`, `snippet_turn_index`,
`snippet_role` (`human_steering` = high intent), `summary_turn_index`,
`children`, `origin` (cross-repo — not this repo's conventions).

Cite session/turn/commit with every claim.
