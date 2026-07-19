# LEDGER — reasoning over past intent

The ledger holds what *was*: the sessions where problems were explored,
alternatives rejected, decisions made. You reach it two ways, chosen by the
shape of the question:

- **Pointed episode** ("why did we…", "what did X say about…") → `rekal` recall,
  gated by `route.py`. Recall finds the loudest match.
- **Analytical / complete-set / temporal** ("how many…", "all of…", "in what
  order…") → `rekal query` SQL. SQL enumerates the whole set.

Recall answers "which episode matches this phrase?"; SQL answers "what does the
whole record say?" Pick before you spend a token. Cite session / turn / commit
with every claim.

## The recall pipeline

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
rekal "JWT expiry" | python3 "$ROOT/scripts/route.py"
# optional filters:
rekal --file src/auth/ "token refresh" | python3 "$ROOT/scripts/route.py"
rekal --commit <sha>                   | python3 "$ROOT/scripts/route.py"
rekal -n 5 --explain "error handling"  | python3 "$ROOT/scripts/route.py"
```

| Route stdout | Action |
|---|---|
| `INJECT top=… gap=… N seed…` + rows (+ optional `KNOWLEDGE`) | Confident episode(s). Rows are seed context — `sid conf=… t<n> "snippet"`. Weigh `conf` if useful; drill `sid` at `t<n>`. A trailing `KNOWLEDGE` line means HEAD prose also matched — inclusive, not if/else. |
| `KNOWLEDGE path=score …` | Knowledge half of a mixed report, or the only substrate when episodes are empty/near-zero. Judge the per-file score *distribution*: clear leader that falls off → Read at `lines` (`anchor`); flat near the noise floor → stay silent on prose. |
| `SILENCE reason=…` | No confident episode and no knowledge at all. Say so. Don't pad with near-misses. |

On `INJECT` the route prints a **seed digest** — the top-20 candidates each as
`session_id conf=… t<turn> "snippet"`, in rank order. Confidence is the
corpus-invariant signal (saturating BM25) so you can fetch/weigh it; mass
stayed inside the script (never a veto). **Work from the seed.** It carries
enough to synthesize a multi-hop answer without drilling each; drill `sid` at
`t<turn>` for a full turn (or `--offset/--limit` to zoom), and re-read raw
recall only for a field the seed omits (`files`). If the top-20 isn't enough,
**reformulate and multi-search**. The digest is **cost-bounded**: a `-n 100`
read costs about the same through the route as a `-n 20` one.

`route.py` labels are **recommendations**. It is biased toward more data than
decision: a **super-low** episode floor on absolute `confidence` (≥ 0.25; soft
≥ 0.20 with gap ≥ 0.02) so only empty / near-zero is machine-silenced, then
`conf=` on the header and each seed for **you** to weigh. Knowledge is reported
only above a matching super-low score floor (≥ 0.25); junk markers are omitted.
Both substrates can appear together when the question is mixed. Neither →
SILENCE. Drills and SQL always pipe through `view.py`.

## One query is a guess — widen before you conclude

The ledger indexes the words the *past* session used, not the words you asked
with. Evidence routinely lands at rank 5–9, not rank 1 — `rekal` returns 20
candidates (`-n` to change), so read past the first before you judge. A single
phrasing is one lookup; a confident answer survives more than one.

| Reformulation | When | Example |
|---|---|---|
| keyword-only (drop question words) | always — cheap second look | `rekal "token refresh expiry"` |
| split a compound question | "X and Y", multi-hop | query each clause, then join |
| temporal emphasis | "when / before / after / first / last" | add the date/era, or `--file` for that period |
| entity / path anchor | you know a file or a name | `rekal --file src/auth/ "<q>"` |

Take the union across phrasings: a session that surfaces under two different
queries is almost always the answer. Only conclude SILENCE after a
reformulation also comes back empty — one blank query is not absence.

### Depth is a judgment, not a reflex

The bigger the history, the deeper the right memory can sit — but depth is not
free. Every extra candidate is more tokens and more plausible-but-wrong
distractors. On a question whose answer is *not* in the ledger, digging deeper
manufactures false positives and breaks SILENCE. Decide *whether* to go deep
before you do.

**Go deeper only when a signal says the evidence exists but is mis-ranked:**
- the gate was `INJECT` / a candidate looked confident, yet nothing you read
  answers the question — the right memory is likely just lower;
- scores are flat with small gaps — no decisive winner;
- the shape hides its anchor: **temporal** ("how long between X and Y") or a
  **single-mention fact** stated once long ago and outranked by topical chatter;
- the history is large — more room for evidence to sit deep.

```bash
rekal -n 50  "<q>"   # first widening, when a signal above is present
rekal -n 100 "<q>"   # temporal / single-mention, still not surfaced
```

**Stay shallow — and prefer SILENCE — when nothing points to buried evidence:**
reformulation *and* the first window both came back weak and flat is absence,
not depth. A strong, clear top hit already answers — stop. Depth recovers
evidence that is mis-ranked; it does not create evidence that isn't there.

## Complete-set questions: enumerate, don't rank

"All", "every", "in what order", "which of the N" — and quieter shapes whose
answer is still a list. Ranked recall returns the loudest matches, not the full
set; a **partial list is a wrong answer**. Switch to SQL across *all* sessions.
Page until empty — do not stop because the first hits sound complete.

```bash
rekal query "SELECT COUNT(*) FROM turns WHERE role='human' AND content ILIKE '%<term>%'" \
  | python3 "$ROOT/scripts/view.py"
# then page: ORDER BY ts LIMIT 50 OFFSET 0, 50, 100 … | view.py until every row
```

- **Page until empty.** Stop only when a further OFFSET returns nothing new.
- **Count before you LIMIT.** An `ORDER BY ts LIMIT 20` you never paged is a
  silent truncation — the answer is often in the rows you cut.
- **LIKE is stricter than search.** Reformulate SQL patterns too: synonyms and
  adjacent vocabulary, not just the words your first hits used.
- **Enumerate by the entity, not the verbs.** Pattern on the thing named in
  the question and read *every* row; items you'd never guess only surface from
  the entity's own mention list.
- **Check the set size.** If the question fixes N, don't answer until you can
  point at N distinct events, each verified in context.
- **Report what the record names.** Omit nothing that is there; invent nothing
  that isn't. Prefer the ledger's own labels over a tidied paraphrase.
- **Non-verbal evidence counts.** Shared-media captions (`[shares a photo: …]`),
  links, quoted lists are premises — read them as part of the turn.
- **Scan the uptake, not just the utterance.** In two-party questions the
  cleanest evidence is often the reaction while the original line is
  keyword-sparse. Enumerate both sides.
- **Instances vs classes.** Turns often name instances; the question may ask
  for the class. Enumerate instances from the ledger first, then map — searching
  the class word alone misses every instance that never says it.

## Time questions navigate by time

The ledger has a clock. Use it before you rank.

```bash
rekal query "SELECT MIN(ts), MAX(ts), COUNT(DISTINCT session_id) FROM turns" \
  | python3 "$ROOT/scripts/view.py"
rekal query "SELECT ts, session_id, content FROM turns WHERE ts BETWEEN '<from>' AND '<to>' AND role='human' ORDER BY ts" \
  | python3 "$ROOT/scripts/view.py"
```

- **Anchor first.** "A month ago", "the day before X" are relative — resolve the
  anchor, then scan the window.
- **"Now" is the asker's present, not the record's edge.** `MAX(ts)` is where the
  ledger *stops*, not today. Prefer the real current date. If a relative
  reference computed from the true "now" lands at or past the edge, the latest
  events are the candidates — check the edge, not just the interior.
- **Event time ≠ mention time.** A turn's `ts` is when it was *said*. "Last
  month I…" shifts the event. Date the event, not the mention — and expect the
  report of a trip to land in the *next* session, after it happened.
- **Answer in event time, honest precision.** "Yesterday" said Oct 21 → *Oct 20*.
  "A few days ago" said Aug 19 → *a few days before Aug 19*. Don't flatten a
  relative phrase to the mention date, and don't fake precision the record lacks.
  Prefer "last week before &lt;session-date&gt;" over a rounded gloss like
  "early October" when that is all the speaker gave.
- **Routine ≠ episode.** "I usually / around 10pm" is a habit; a question about
  one occasion needs the past-tense report of that occasion.
- **One event in the window is not the answer** until you've scanned the whole
  window and no competing event sits in it.

## Whose fact is it? — and false premises

Before answering a "my / I" question — a preference, possession, experience —
drill to the turn where the user *asserts* it ("my X is", "I bought", "it turned
out"). A thing only discussed, suggested, or compared is not the user's. When
two candidates compete, drill both to the ownership/outcome statement; answer
from the one the user owns and keep the other out — a near-miss entity in the
output is a wrong answer, not color.

**A false premise is a SILENCE, not a correction.** Questions arrive loaded:
"what did A design…" when the ledger shows *B* designed it; "the campaign in May"
when there was none. Verify the premise — subject, event, time — against the
turn. If it fails, the honest answer is that the ledger doesn't hold it. Do
**not** swap in the right person or nearest event and answer anyway: a corrected
answer to a question nobody asked is a fabrication with citations.

## Premises from the ledger, inference from you

Judgment questions — "would X enjoy…", "does X's shop employ many people",
"what nearby would suit X" — are rarely answered verbatim. The ledger's job is
the *premises* (what X loves, owns, plans, said); yours is the *inference*.
Gather the premises with the pipeline above and cite them, then reason. Don't
demand the conclusion appear in a session, and don't go silent when the premises
are on the record. SILENCE is for missing premises, not missing conclusions.

## Analytical asks — decompose to SQL first, don't route

Reflection and pattern-mining need signal clusters, not one pointed episode.
**Don't** pipe these through `route.py` — decompose, gather, then reason.

1. **Classify.** Reflection / pattern / open question — not a single episode, not
   a tree fact.
2. **Decompose** into SQL with three filters: **scope** (subsystem/topic via
   `files_index`, or whole-repo), **signal** (words people type when the
   phenomenon happens), **role** (`human_steering` / `assistant` / any).
3. **Gather** turns with session/turn pointers. Widen signals before giving up.

| Ask | Signal words | Role |
|---|---|---|
| my mistakes / corrections | `no,`, `don't`, `instead`, `always`, `never` | `human_steering` |
| agent errors from missing context | `already exist`, `assumed`, `turns out`, `i missed` | `assistant` |
| still undecided | `TODO`, `undecided`, `open question`, `not sure` | any |
| why X | choice + `because`, `instead of`, `rejected` | any |

```bash
rekal query --index "SELECT session_id, turn_index, role, substr(content,1,400) FROM turns_ft \
  WHERE role = 'human_steering' \
  AND (content LIKE '%auth%' OR content LIKE '%token%') \
  ORDER BY session_id, turn_index" | python3 "$ROOT/scripts/view.py"
```

Modes: **reflect** (cluster recurring corrections into one durable rule),
**distill** (context / decision / rules / boundary), **census** (exhaustive
coverage of a bounded scope — state the scope in the first sentence or don't
start; walk a deterministic spine, summarize each once, reduce with
traceability). Ranking or cherry-picking means you want a pointed recall, not a
census.

## Decision arcs (why) — gather the trail, then synthesize

The rationale for an evolved decision is distributed across sessions. Gather the
trail, then synthesize — under-gathering invents fiction.

```bash
rekal query --index "SELECT session_id, turn_index, role, substr(content,1,300) FROM turns_ft \
  WHERE (role = 'human_steering' OR content LIKE '%because%' OR content LIKE '%instead of%' \
         OR content LIKE '%constraint%' OR content LIKE '%rejected%' OR content LIKE '%decided%') \
  AND (content LIKE '%<topic-1>%' OR content LIKE '%<topic-2>%') \
  ORDER BY session_id, turn_index" | python3 "$ROOT/scripts/view.py"
```

A real arc needs a real trail: a couple of rows is not a rationale. When the
gather is thin, widen the terms; if it stays thin, the honest answer is that the
ledger doesn't hold the why — a gap beats a fabricated rationale. Once you have
a real trail (aim ~30 turns), pull code on demand (`git show <sha>`) and
synthesize the arc with pointers: original design → alternatives rejected →
constraint → final rationale, each `(session <id> turn <n>, commit <sha>)`.

## Provenance — artifact → commit → session → intent

When the anchor is a specific file, function, line, or commit:

```bash
git log --oneline -15 -- path/to/file.go
git log --oneline -15 -L :FuncName:path.go       # follow a function
rekal --commit <sha>                              # commit → sessions
rekal query --session <id> --role human_steering | python3 "$ROOT/scripts/view.py"
```

Emit the chain: artifact → commit `<sha>` → session `<id>` → human intent
(quoted / steering), with turn pointers. If no session links to the commit, say
so — don't invent.

## Drilling — strongest, cheapest first

When top candidates' `confidence` are within the gap band, drill the top 2–3,
not only #1. A snippet is a keyhole: read the room around it — 5 turns costs
almost nothing and catches misreads (a routine mistaken for an episode, a
discussed item mistaken for an owned one). Never `--full` by default:

```bash
# Always pipe drills through view.py — raw turns, not JSON chrome.
rekal query --session <id> --offset <snippet_turn_index - 2> --limit 5 | python3 "$ROOT/scripts/view.py"
rekal query --session <id> --role summary                               | python3 "$ROOT/scripts/view.py"
rekal query --session <id> --role human                                 | python3 "$ROOT/scripts/view.py"
rekal query --session <id> --role human_steering                        | python3 "$ROOT/scripts/view.py"
rekal query --session <id> --full                                       | python3 "$ROOT/scripts/view.py"  # last resort
```

Fields: `session_id`, `score`, `confidence`, `mass`, `snippet`,
`snippet_turn_index`, `snippet_role` (`human_steering` = high intent),
`summary_turn_index`, `children`, `origin` (cross-repo — not this repo's
conventions).
