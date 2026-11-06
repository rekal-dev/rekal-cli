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
