# Passage evidence validation — 7 November 2026

A retrieval/evidence diagnostic on all ten public LoCoMo conversations, using
the real embedded `nomic-v1.5-c8k-d1` model. This runs no answering model or
judge. The percentages below are supporting-evidence coverage, not QA accuracy
and not the paper’s published scores.

## Controls and scope

- 1,536 questions with supporting annotations; adversarial/no-support questions excluded.
- Exact questions, query vectors and default hybrid weights shared across variants.
- Five session results, one query framing, radius two, 16,000 original UTF-8 text bytes and 80 slices per page.
- The same evidence assembler admits matches before context and removes overlap.
- Whole-session baseline uses its primary lexical hit, with the former turn-zero fallback when unlocated.
- Passage-primary uses the first located anchor. Passage-multiple uses up to four anchors per result.
- Original support annotations are retained, including nine unresolved support IDs; no answer-key corrections or tuning were applied.

A supporting turn counts as delivered only when every byte of its normalized
source text was delivered. A question counts as complete only when every
annotated supporting turn was fully delivered. Speaker labels remain in the
normalized source text; the original session date is retained, not a fabricated
per-turn timestamp.

## Results

| Variant | Complete supporting evidence | Individual support turns | Mean text bytes | Mean slices | Truncated pages |
| --- | ---: | ---: | ---: | ---: | ---: |
| `session-primary` | 993/1,536 (64.65%) | 1,373/2,355 | 3,655.1 | 23.40 | 0 |
| `passage-primary` | 992/1,536 (64.58%) | 1,369/2,355 | 3,665.4 | 23.56 | 0 |
| `passage-multiple` | 1,179/1,536 (76.76%) | 1,681/2,355 | 11,728.3 | 79.36 | 1,276 |

Multiple anchors add **186 complete questions (+12.11 percentage points)**: 196
paired wins and 10 paired losses. They consume **3.21×** as much text on
average despite the identical maximum budget. Most multiple-anchor pages reach
the slice or byte limit and offer a continuation. A wider seed is not free.

Passage-primary alone is effectively flat (one fewer complete question). This
run supports retaining distinct evidence locations and fetching them together;
it does not establish an independent passage-ranking gain or a QA gain.

| Category | Questions | Session-primary complete | Passage-primary complete | Passage-multiple complete |
| --- | ---: | ---: | ---: | ---: |
| multi-hop | 282 | 45 | 45 | 82 |
| open-domain | 92 | 28 | 28 | 42 |
| single-hop | 841 | 690 | 693 | 791 |
| temporal | 321 | 230 | 226 | 264 |

## Observed latency and limits

| Variant | Lookup p50 / p95 (ms) | Fetch p50 / p95 (ms) |
| --- | ---: | ---: |
| `session-primary` | 41.4 / 75.9 | 34.7 / 66.6 |
| `passage-primary` | 49.3 / 104.9 | 38.7 / 89.3 |
| `passage-multiple` | 48.5 / 86.3 | 174.8 / 315.7 |

Query embeddings are precomputed once and excluded from lookup time. Native
model loading, corpus embedding, index construction and all queries together
took 40.3 minutes. This host also ran validation tests; timings
are descriptive, not a service latency claim.

The run was compiled from the feature worktree based on `9f6f83c4`, before
the final bounded vector streaming, framing-fusion and long-turn UTF-8 fixes.
Those refinements leave this diagnostic’s cosine calculations and evidence
inputs unchanged: it uses one framing, and its longest normalized turn is
496 bytes. The timing samples describe the earlier vector scan, not a final
release performance baseline. Final code snapshot `ff194f7c` has tree
`69da4d3f29954d040260d6b2ba14e7bf88534818`; the public CLI round-trip and
modified-package race tests passed on that implementation.

The production embedding daemon cannot bind a Unix socket in this box. The
diagnostic therefore uses the same native model in process. Daemon socket
integration remains a GitHub CI check. No API reader, agent follow-up or QA
judge has been exercised here.

## Reproduction and audit files

- [Runner and method](../../../scripts/passage-eval/README.md)
- [Aggregate counts](2026-11-07-passage-evidence.aggregate.json)
- [Per-question records, gzip](2026-11-07-passage-evidence.per-question.jsonl.gz)

The records contain question IDs, annotations, fully delivered source turn IDs,
exact rendered-context hashes, text bytes, slices, truncation and timings.
They contain no raw questions, answers or source conversations.

- Dataset SHA-256: `79fa87e90f04081343b8c8debecb80a9a6842b76a7aa537dc9fdf651ea698ff4`
- Uncompressed records SHA-256: `dd7f1c661fe6d5b6c6d480eed4c596b0274b843e683ee4280161201293bc327c`
- Gzip records SHA-256: `acd19f104658d9ab1490273ac6c1ed38b5e2822dd46b088faee4c5aa6977c2e8`

Decode the records with `gzip -dc` and verify aggregate counts before comparing
variants. See the runner README for exact commands and dataset limitations.
