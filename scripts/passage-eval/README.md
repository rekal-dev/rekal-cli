# Passage evidence diagnostic

Run from the repository root after the native build prerequisites in
[DEVELOPMENT.md](../../docs/DEVELOPMENT.md). The executable uses the real embedded
Nomic model in process and creates an isolated temporary index per conversation.
It never opens your Rekal ledger or sends model API requests.

```sh
./scripts/industry-bench/datasets/get_locomo.sh
python3 scripts/industry-bench/datasets/normalize_locomo.py
go run ./scripts/passage-eval --conversations 0 --out /tmp/rekal-passage-eval
```

The default is one conversation; `--conversations 0` selects all ten. Optional
`--questions N` limits supported questions per conversation for a smoke run.
Download scripts keep the dataset outside git. Check the dataset's research
license before redistributing any source conversations.

| Variant | Semantic documents | Evidence anchors |
| --- | --- | --- |
| `session-primary` | Existing whole-session document | One primary turn per result; legacy turn-zero fallback for unlocated matches |
| `passage-primary` | Role/time-preserving passages | First located anchor per result |
| `passage-multiple` | Same passages | All returned anchors, up to four per result |

All variants share exact questions and query vectors, default hybrid weights,
five session results, one query framing, radius two, 16,000 original text bytes
and 80 slices. The same evidence assembler admits matched ranges before context
and deduplicates overlap. No answering model, agent follow-up or judge is run.
This is a controlled evidence-delivery diagnostic, not an end-to-end LoCoMo QA
score or a reproduction of the paper's accuracy.

`per_question.jsonl` records annotated support IDs, fully delivered turn IDs,
exact delivered-context hashes, text bytes, slices, truncation, unresolved
support annotations and lookup/fetch latency. Full support means every byte of
each annotated supporting turn was delivered. Lookup timing excludes the query
embedding, which is computed once and shared. Raw questions, answers and source
conversations are omitted. `aggregate.json` records the source dataset hash,
model, limits and aggregate counts; QA accuracy and reader are explicitly null.

The dataset's original support annotations are used without known-bad-answer
filtering. Adversarial questions and questions with no support annotations are
excluded. Coverage is therefore subject to annotation errors and is not a
measure of factual answer correctness. Compare actual delivered bytes as well
as coverage: a common maximum budget does not mean equal context consumption.

## Source-only memory control

Add `--extractive-memory` to include `passage-extractive-memory` and
`extractive-memory-only`. This tests the shipped keyword memory lookup alongside
passage retrieval. It is an extractive control, **not Claude native distillation,
an answering-agent run, or a QA accuracy measurement**.

The note generator receives only sessions and source-ID mappings, never
questions, gold answers or support annotations. Per session it selects the two
longest eligible complete turns per role (at most 800 bytes each), restores
source order, and preserves speaker text, session date and passage reference.
It archives these excerpts through the same memory index revision projection.
There are no inferred fact corrections or synthetic summaries.

Each question uses the same query vector and session candidates across arms.
The augmented arm reads the top five keyword note matches, admitting only whole
notes that fit a 4,000-byte reservation. Their bytes and source-turn counts are
deducted from the total 16,000-byte / 80-slice budget before assembling raw
evidence. Copied complete source turns count toward support delivery. Duplicate
raw/note text still consumes budget. The memory-only control delivers notes
without raw evidence. No retrieval policy is tuned using the question's answer.

Memory totals disclose note count and bytes; per-question records also disclose
source-turn counts. Only IDs, hashes and measurements are written, never raw
notes or conversations. Keep source data and model files outside result commits.

The raw-stage `lookup_ms` / `evidence_fetch_ms` fields do not include memory
lookup/read time; do not use them as an end-to-end latency comparison. Likewise,
`truncated` describes the raw evidence fetch. The note admission policy is
bounded independently by the declared 4,000-byte ceiling.
