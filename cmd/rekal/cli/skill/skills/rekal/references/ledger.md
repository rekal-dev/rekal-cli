# LEDGER — reasoning over past intent

The ledger holds what *was*: the sessions where problems were explored,
alternatives rejected, decisions made. You reach it two ways, chosen by the
shape of the question:

- **Pointed episode** ("why did we…", "what did X say about…") → `rekal` recall,
  printed as a seed digest. Recall finds the loudest match.
- **Analytical / complete-set / temporal** ("how many…", "all of…", "in what
  order…") → `rekal query` SQL. SQL enumerates the whole set.

Recall answers "which episode matches this phrase?"; SQL answers "what does the
whole record say?" Pick before you spend a token. Cite session / turn / commit
with every claim.

## Runbook — shape → first move → watch for

Pick the row before improvising. Each shape has one first move; the sections
below carry the depth when the first move isn't enough.

| Question shape | First move | Watch for |
|---|---|---|
| Pointed episode — why / what did X say | `rekal "<q>"`; drill top seed at `t<n>`; weak? `rekal "<q>" --also` several framings | premise mismatch; near-miss entity in the answer |
| Complete-set — all / every / how many / which of N | `rekal find "<term>"` sweep, drill each mention | stopping early; instance vs class; the other speaker's uptake |
| Temporal — when / before / after / how long | `rekal when <anchor> "<phrase>"` for the date, SQL window on `ts` (`BETWEEN`, never `LIKE` — `ts` is TIMESTAMP) | event time ≠ mention time; the record's edge is not "now" |
| Whose-fact — my / I / their | drill to the assertion turn ("my X is", "I bought") | discussed / suggested ≠ owned |
| False-premise suspicion | drill the premise subject before answering | fabricating the asserted fact; answering a corrected question |
| Decision arc — why did this evolve | steering/`because` SQL gather (below) | thin trail synthesized into fiction |
| Reflection / pattern / census | decompose to SQL (below); never plain recall | ranking when the ask is exhaustive |

**Stopping rule.** Stop when the answer is grounded in drilled turns you can
cite. Before concluding SILENCE, `rekal "<q>" --also` one widening across real alternative
framings — a partial seed is not absence. But if that fuse and two further
moves add no new evidence, report what you have — or the gap — instead of
searching on: extra moves past that point manufacture plausible-but-wrong
distractors. And grep of the tree never answers a ledger question; if you catch
yourself grepping code for a past-tense fact, come back to the table.

## The recall pipeline

```bash
rekal "JWT expiry"                      # recall → seed digest (text by default)
rekal --file src/auth/ "token refresh"  # optional filters
rekal --commit <sha>
rekal -n 5 --explain "error handling"
```

| Route stdout | Action |
|---|---|
| `INJECT top=… gap=… N seeds` + rows (+ optional `KNOWLEDGE`) | Confident episode(s). Rows are seed context — `sid conf=… t<n> "snippet"`, then `(+N more)`. Weigh `conf` if useful; drill `sid` at `t<n>`. A trailing `KNOWLEDGE` line means HEAD prose also matched — inclusive, not if/else. |
| `KNOWLEDGE path=score …` | Knowledge half of a mixed report, or the only substrate when episodes are empty/near-zero. Judge the per-file score *distribution*: clear leader that falls off → Read at `lines` (`anchor`); flat near the noise floor → stay silent on prose. |
| `SILENCE reason=…` | No confident episode and no knowledge at all. Say so. Don't pad with near-misses. |

On `INJECT` the route prints a **seed digest** — the top-20 candidates each as
`sid conf=… t<turn> "snippet"`, in rank order (`sid` is the short handle;
`session_id` ULID stays in the raw JSON). `confidence` =
`max(saturate(bm25), cosine) + 0.15·saturate(facet)`; only the BM25 term is
corpus-invariant, the cosine term drifts — which is why the floor is super-low
and you weigh `conf=` yourself. Mass stays inside recall (never a veto).
**Work from the seed.** It carries
enough to synthesize a multi-hop answer without drilling each; drill `sid` at
`t<turn>` for a full turn (or `--offset/--limit` to zoom), and re-read raw
recall only for a field the seed omits (`files`). If the top-20 isn't enough,
**reformulate and multi-search**. The digest is **cost-bounded**: a `-n 100`
read costs about the same through the route as a `-n 20` one.

A seed may carry `[reached N×· "past query"]` before its snippet — the L1 recall
graph: this session was reached (recalled or drilled) N times by past work, and
last for that query. It is a **usage** signal, not relevance — a high `reached`
count marks well-trodden, load-bearing memory and makes a good *first* drill,
and the echoed past query hints how others framed the same need. It never
changes `conf=`; weigh relevance from `conf=` + content exactly as before. A
fresh seed with no `[reached]` is not worse — just newly surfaced.

Recall labels (`INJECT`/`SILENCE`) are **recommendations**. It is biased toward more data than
decision: a **super-low** episode floor on absolute `confidence` (≥ 0.25; soft
≥ 0.20 with gap ≥ 0.02) so only empty / near-zero is machine-silenced, then
`conf=` on the header and each seed for **you** to weigh. Knowledge is reported
only above a matching super-low score floor (≥ 0.25); junk markers are omitted.
Both substrates can appear together when the question is mixed. Neither →
SILENCE. Drills and SQL print readable text by default (add `--json` for raw).

## One query is a guess — widen before you conclude

The ledger indexes the words the *past* session used, not the words you asked
with. Evidence routinely lands at rank 5–9, not rank 1 — `rekal` returns 20
candidates (`-n` to change), so read past the first before you judge. A single
phrasing is one lookup; a confident answer survives more than one.

**Widening is a command — supply the framings, let recall fuse.**

```bash
rekal "token refresh expiry" --also "JWT session timeout" --also "logout invalidate"
```

`--also` runs recall once per framing and RRF-fuses the candidate lists into
one ranked seed (the recall digest, `conf=` per session is that session's
strongest framing). A session that surfaces under two framings rises to the
top — that convergence is the signal. **You** pick the framings; the fuse is
mechanical. Good framings to hand it: the keyword-only form (drop the question
words), each clause of a compound/multi-hop question, an entity or path anchor
you already know, and a synonym set for the same idea.

Only conclude SILENCE after a `seek` across real alternatives also comes back
weak — **a partial seed is not absence**, and one blank phrasing never was.

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
rekal query "SELECT COUNT(*) FROM turns WHERE role='human' AND content ILIKE '%<term>%'"
# then page: ORDER BY ts LIMIT 50 OFFSET 0, 50, 100 … until every row
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
rekal query "SELECT MIN(ts), MAX(ts), COUNT(DISTINCT session_id) FROM turns"
rekal query "SELECT ts, session_id, content FROM turns WHERE ts BETWEEN '<from>' AND '<to>' AND role='human' ORDER BY ts"
```

- **Anchor first.** "A month ago", "the day before X" are relative — resolve the
  anchor, then scan the window.
- **"Now" is the asker's present, not the record's edge.** `MAX(ts)` is where the
  ledger *stops*, not today. Prefer the real current date. If a relative
  reference computed from the true "now" lands at or past the edge, the latest
  events are the candidates — check the edge, not just the interior.
- **Event time ≠ mention time.** A turn's `ts` is when it was *said*. "Last
  month I…" shifts the event. Date the event, not the mention — and expect the
  report of a trip to land in the *next* session, after it happened. Resolve
  the shift with the calendar, not mental math:

  ```bash
  rekal when 2023-05-25 "last Saturday"    # -> 2023-05-20 (Saturday)
  rekal when 2023-08-14 "last night"       # -> 2023-08-13 (Sunday)
  rekal when 2023-11-22 "a few days before" # -> 2023-11-17..2023-11-21 (approx)
  ```

  `rekal when` takes the mention's date as the anchor and returns the absolute
  date — or an honest window for a vague phrase. Deterministic; you pick the
  anchor and judge the result.
- **Answer in event time, honest precision.** "Yesterday" said Oct 21 → *Oct 20*.
  "A few days ago" said Aug 19 → *a few days before Aug 19* (`rekal when` returns
  the window). Don't flatten a relative phrase to the mention date, don't fake
  precision the record lacks, and don't round a relative anchor into a vaguer gloss.
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
**Don't** answer these with plain recall — decompose, gather, then reason.

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
  ORDER BY session_id, turn_index"
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
  ORDER BY session_id, turn_index"
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
rekal query --session <id> --role human_steering
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
# Drills print readable turns by default (add --json for raw).
# Prefer the short handle from the digest (s3); ULID still works.
rekal query --session <sid|ulid> --offset <snippet_turn_index - 2> --limit 5
rekal query --session <sid|ulid> --role summary
rekal query --session <sid|ulid> --role human
rekal query --session <sid|ulid> --role human_steering
rekal query --session <sid|ulid> --full  # last resort
```

Fields: `sid` (short handle — prefer for digests/drills), `session_id` (ULID),
`score`, `confidence`, `mass`, `snippet`,
`snippet_turn_index`, `snippet_role` (`human_steering` = high intent),
`summary_turn_index`, `children`, `origin` (cross-repo — not this repo's
conventions).
