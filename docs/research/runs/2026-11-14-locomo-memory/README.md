# LoCoMo passage and extractive-memory diagnostic — 14 November 2026

Tested product code: `67ae9e39f5c06e0aea8926de718d0c2152344834`. This is **retrieval/evidence delivery, not QA accuracy**. No answering model, judge or Claude native distillation was run.

## Matched sample

40 supported non-adversarial questions per conversation, all ten conversations: 400 questions and 2,000 variant records. The corpus contains 272 sessions / 5,882 turns. Selection is the first 40 eligible questions in each conversation, not a random sample or the full 1,986-question benchmark. Category counts: `{'temporal': 181, 'open-domain': 54, 'multi-hop': 158, 'single-hop': 7}`.

Every arm shares query vectors, Nomic `nomic-v1.5-c8k-d1`, default hybrid weights, five session results, one query framing, radius two, and a total ceiling of 16,000 source-text bytes / 80 slices. This does not simulate the full answering agent or the root command's multi-framing policy. Session-primary is a reference simulation, not an execution of the v1.0.2 binary.

## Results

| Variant | All support delivered | Supporting turns delivered | Mean text bytes |
| --- | ---: | ---: | ---: |
| Session primary (reference simulation) | 177/400 (44.25%) | 348/823 (42.3%) | 3,585 |
| Passage primary | 174/400 (43.50%) | 344/823 (41.8%) | 3,610 |
| Passage multiple | 218/400 (54.50%) | 452/823 (54.9%) | 11,594 |
| Passage + extractive notes | 225/400 (56.25%) | 471/823 (57.2%) | 13,669 |
| Extractive notes only | 67/400 (16.75%) | 142/823 (17.3%) | 3,607 |

Under the fixed note-first policy, adding extractive notes gained full-support coverage on **10** questions and lost it on **3** compared with passage-multiple. Question IDs are in `summary.json`; all variant measurements are in `per_question.jsonl.gz`. These changes establish evidence delivery under this policy, not answer correctness or the value of Claude-generated summaries.

Average delivered text grew by **17.9%** with notes. Matched maximum budgets
do not imply equal actual context consumption. The 1.75 percentage-point coverage
gain therefore does not isolate distillation quality, demonstrate token savings,
or establish a better default weighting policy. Notes alone supplied all support
for only 67/400 questions, so this control supports preserving raw evidence access.

The sample is heavily temporal/multi-hop (339/400 questions), with only seven
single-hop questions. It should not be treated as a representative overall QA score.

## Memory control

The generator is question-independent: its function receives only source sessions and source-ID mappings. It selects two complete turns per role/session, preferring longer eligible turns of at most 800 bytes, then restores source order. Notes retain names, dates and explicit source references. This deliberately simple extractive baseline does not resolve contradictions or distill facts.

The shipped memory index projection and keyword lookup select up to five notes. Only whole notes fitting 4,000 bytes are admitted. Their text bytes and copied turn counts are charged before raw evidence against the shared 16,000-byte / 80-slice ceiling. Duplicate raw/note text still consumes budget. Memory-only keeps the same 4,000-byte note allowance. This note-first consumer policy is an experiment; the product returns source types for the agent to judge and does not adopt this fixed policy.

All 2,000 records passed the byte/slice checks. Existing `lookup_ms` and `evidence_fetch_ms` measure raw lookup/fetch stages; they exclude memory lookup/read. Do not interpret them as end-to-end latency comparisons.

## Compiled CLI smoke

An isolated repository captured an excerpt note from the first LoCoMo session through the default native-memory path. Search found it; `query --ref` recovered exact text; reindexing retained its reference; deleting the native file preserved the archived revision. `binary_smoke.json` records booleans and hashes only. This checks the executable's integration path; the aggregate comparison uses isolated index APIs for controlled ablation.

## CI validation boundary

The first result commit's full CI exposed a 60-second timeout while checkpointing
the binary-contract suite's artificial 300-turn / long-Unicode transcript with
native embeddings. Checkpoint currently performs native embedding work
synchronously. The read-contract fixture now uses a local deterministic
embedding endpoint; its large source text and exact pagination assertions are
unchanged. The real Nomic LoCoMo experiment above is separate and unchanged.

This resolves the test's coupling to model throughput, not the latency of native
checkpoint work. Long native-embedding checkpoint latency remains a performance
limitation to profile; passing the text contract must not be read as a latency
guarantee. The original failed run is
[43541841899](https://github.com/rekal-dev/rekal-cli/actions/runs/43541841899).

## Limits

- Supporting-turn annotations can be incorrect. There are 5 unresolved annotation references across 5 sampled questions.
- 32 sampled questions appear in the repository's known-bad answer list. This diagnostic retains them and scores source support rather than the answer key.
- Complete source-turn delivery is appropriate for this extractive control, but cannot assess a paraphrased/distilled note's factual sufficiency.
- The experiment gives no official LoCoMo QA score and does not reproduce the reported 88–94% answer scores. Actual Claude memory and end-to-end answer evaluation require a separately recorded generator/reader/judge protocol.
- No user conversation memory, notes, source dialogue, questions or answers are included in these artifacts. Only public benchmark IDs, measurements, metadata and hashes are saved.

## Reproduce

Download/normalize the public data with `scripts/industry-bench/datasets/`, install native build prerequisites, and fetch the pinned LFS model. From the repository root:

```sh
go run ./scripts/passage-eval --conversations 0 --questions 40 \
  --extractive-memory --out /tmp/rekal-memory-eval
```

The raw dataset SHA-256 is `79fa87e90f04081343b8c8debecb80a9a6842b76a7aa537dc9fdf651ea698ff4`; model and evaluator hashes are in `summary.json`. Raw source data is fetched from <https://github.com/snap-research/locomo> and stays outside the result commit.
