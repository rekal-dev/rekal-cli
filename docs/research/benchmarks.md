# Benchmark results and scope

These are **previously reported results**, not a fresh evaluation of v2.0.0 or a measured old-versus-new release comparison. The historical answering
agent was GPT-5 Sol. Different reader models, judges, samples and context budgets
prevent a direct ranking against other projects' published scores.

| Historical benchmark | Reported answer accuracy | Reported Recall@20 | Context tokens/query | Agent turns/query |
|---|---:|---:|---:|---:|
| LoCoMo | 90.57% | 98.61% | ~7.5K | 5.9 |
| LongMemEval | 86.60% | 99% | ~10.5K | 6.6 |

LoCoMo also reported Recall@10 of 93.60%, Recall@5 of 86.44%, and answer F1 of
63.0. Answer scores, retrieval recall and evidence coverage are separate metrics.
See the [run procedure](RUN.md) and [benchmark harness](../../scripts/bench/).
These historical runs did not evaluate native memory integration.

**2.0 development evidence diagnostic (14 November 2026):** on a 400-question LoCoMo
sample, multiple passages delivered every annotated supporting turn for 54.50%
of questions; passages plus extractive notes reached 56.25%, using 17.9% more
source text. This is **complete evidence coverage, not answer accuracy**. It used
no answering model, judge or Claude distillation. The session-primary control
was a simulation, not a v1.0.2 binary execution. [Results, tested commit and limits](runs/2026-11-14-locomo-memory/README.md).

**Local store, wire & recall latency** (Rekal's own coding sessions, Apple M4 —
6 Claude transcripts / 14 sessions, 473 turns; pure `rekal`, no answering model):

| Metric | Result |
|---|---|
| Raw JSONL | 8.5 MB |
| Wire | 54 KB (**~158×**) |
| `data.db` | 5.8 MB |
| `index.db` | 10.8 MB |
| Local store (`data` + `index`) | 16.5 MB |
| Recall latency | median **~150 ms** (p95 ~210 ms) |
| LoCoMo synthetic recall | median ~350 ms (p95 ~400 ms) |
| In-process search | ~76 ms |

The footprint and latency rows above are historical measurements on the named
corpus and machine, not guarantees for every repository or the current release
candidate. Wire reduction compares encoded transport with raw transcript files;
local databases retain richer search structures.

Checkpoint currently performs some embedding work synchronously. Large native
transcripts can make a commit hook slow. Background `rekal embed` fills missing
vectors after index/sync; it does not make checkpoint latency disappear.

