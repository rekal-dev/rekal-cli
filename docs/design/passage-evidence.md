# Passage retrieval and cited evidence

Session retrieval and evidence localization are separate operations. A useful
session score does not prove its opening turn supports the query. Recall now
reports located ranges and leaves an unlocated session match explicit.

## Derived passage index

`session_passages` and `passage_embeddings` live only in `index.db`. They never
change `data.db` or the wire format. Documents contain original text with role,
assertion timestamp, turn index and byte bounds. Every speaker is admitted;
the older whole-session document's assistant allowance does not apply here.

Documents contain up to eight turn pieces within an 8 KiB byte budget and
overlap by one piece. A long turn is split at UTF-8 boundaries, retaining its
original turn index and byte range. These are context budgets, not tuned
relevance thresholds. Passage IDs are content-addressed per session. On append,
only changed tail documents lose vectors; unchanged documents retain them.

`rekal embed` fills session, passage and knowledge vectors in bounded bites,
up to 16 uncached documents per layer. The same configured embedding backend
and model ID apply to session and passage vectors. The content-hash cache
reuses exact document/model inputs across index rebuilds. New index structures
are additive; older indexes use their whole-session vectors until passage
backfill. An upgrade can run `rekal index` followed by `rekal embed`; the former
also reclaims old index bloat, and team sessions return on the next sync.
Passage backfill loads at most 512 missing documents per bite, including cache
hits. Recall streams vectors and retains only the best ranges per session.

## Located recall results

Passage semantic scores aggregate to session candidates while retaining up to
three semantic ranges. Lexical retrieval retains several distinct matching
turns. Exact lexical matches in a very long turn retain a nearby byte window,
so a bounded fetch can reach late evidence. Unbounded long-turn matches follow
located passages. Recall exposes up to four distinct anchors per result, preserving useful
locations across query-framing fusion. Each anchor names its source session,
turn range, optional byte range and retrieval kind. Scores from different
retrieval kinds are diagnostics, not interchangeable confidence values.

Without a located range, `match_scope` is `session`, `snippet_turn_index` is
`-1`, and no first-turn snippet is invented. Semantic passage previews have
`match_scope=passage`; their anchor contains the complete range. Filter-only
recall labels its opening preview `preview`. A grouped conversation that borrows
a child's preview carries `snippet_session_id`, so its turn number cannot be
misread as a turn in the parent transcript. Default digest headers stay intact;
additional `anchors=` lines expose compact source pointers.

## Mechanical context assembly

`rekal evidence` accepts selected anchors or recall JSON. It admits matched
ranges before neighboring turns, removes overlap, preserves source roles/time,
and returns exact source citations and original text. The default limits are
16,000 text bytes and 80 slices. Metadata has separate structural limits; the
byte count is not represented as an exact model-token count.

Truncation and gaps are visible. A continuation retains canonical session IDs,
radius, cursor and byte position, with a source-layout hash to detect changed
windows. The agent decides whether to continue, choose different anchors or
reformulate a query. No fixed answer workflow or automatic summary is imposed.

## Evaluation

`scripts/passage-eval` compares whole-session/primary windows, passage/primary
windows, and passage/multiple anchors using the same LoCoMo questions, query
vectors, reader-free retrieval settings, and context limits. It records exact
delivered turn IDs, full-support coverage, UTF-8 bytes, latency and context
hashes without redistributing raw conversation text. This measures evidence
delivery; it runs no answering model or judge and claims no QA accuracy gain.
An agent-follow-up experiment requires its own logged calls and matched reader.
