# Industry-Bench: Rekal on the Standard Memory Benchmarks

**Program goal:** the second paper. Paper #1 (arXiv:2607.14390) argued and
measured git-bound memory on Rekal's home turf (RekalBench, self-labeled from
commit–session links). Paper #2 takes the same structure onto the field's
shared ground: run Rekal — unchanged core, adapters only — against the
industry-standard memory benchmarks and report where the git-bound design
transfers, where it doesn't, and what it costs per question.

**Thesis (one sentence):** a memory system built for the agentic development
lifecycle transfers to general conversational-memory benchmarks by mapping
"session → commit" onto synthetic history, and answers at a token budget the
tiered-store systems don't report.

This directory is a **playbook for distributed agent work**: every workstream
is specified with inputs, outputs, interface contracts, and a definition of
done, so independent agents (or people) can build in parallel without
coordinating beyond these documents.

## Read in this order

| Doc | Question it answers |
|---|---|
| [00-landscape.md](00-landscape.md) | What benchmarks exist, where they live, and which are known-flawed? (compiled reference) |
| [01-selection.md](01-selection.md) | Which benchmarks we run, in what order, and why — with the rejection list. |
| [02-adapter-architecture.md](02-adapter-architecture.md) | How Rekal plugs in: the synthetic-history ingestion contract, mode mapping, gate recalibration, metric alignment. |
| [03-workstreams.md](03-workstreams.md) | The distributable work: seven workstreams with contracts, dependencies, and definitions of done. |
| [04-procedures.md](04-procedures.md) | Runbooks: environment, dataset acquisition, smoke test, full-run protocol, pre-registration discipline, honesty rules. |
| [05-paper-plan.md](05-paper-plan.md) | Paper #2's claims ladder and evidence-gap table. |

## The five decisions that shape everything (made; do not relitigate in a workstream)

1. **Ingest through the production pipeline, not around it.** Benchmark
   conversations become session files in the Claude Code JSONL format inside
   a synthetic git repo; `git commit` + the post-commit hook run the real
   `rekal checkpoint` path (parse → dedup → `sessions`/`turns`/`tool_calls`
   → `checkpoints` → `checkpoint_sessions`). No raw writes to `data.db`.
   Rationale and the fallback for 10M-token scale: [02](02-adapter-architecture.md#ingestion).
2. **Rekal core is frozen.** The adapter package lives in
   `scripts/industry-bench/` and may read anything, but the only Rekal
   surface it touches is the CLI (`rekal init/checkpoint/index/query`, recall
   JSON) and `.rekal/config.json`. A change to core to make a benchmark pass
   is a finding, not a fix — file it in the run notes and stop.
3. **LongMemEval-S first, LoCoMo second, never headline LoCoMo alone.**
   LoCoMo's answer key is ~6.4% wrong and its judge is gameable
   ([00-landscape §6](00-landscape.md)); it runs for comparability with
   vendor numbers, always with the caveat block from
   [04-procedures](04-procedures.md#honesty).
4. **Official metrics are the headline; our judge is secondary.** Each
   benchmark is scored by its own published eval script and judge first.
   Rekal's blind answer-sufficiency judge runs alongside for continuity with
   paper #1, clearly labeled.
5. **Tokens-per-question is reported everywhere.** It is the differentiator
   paper #1 established (382–980 tok/q) and the column the vendor comparison
   pages all omit. Every run logs retrieved-context tokens and total
   answer-path tokens per question.

## Status board

Workstream status lives in [03-workstreams.md](03-workstreams.md#board) —
update the board in the same commit as the work it describes.

## Relationship to existing research docs

`docs/research/` (00–07, RUN.md) is paper #1's chain and stays frozen as its
provenance. This directory is paper #2's chain. Shared machinery
(`scripts/bench/` scoring conventions, the one-snapshot rule from
`06-eval-strategy.md`, the `runs/` record format) is referenced, not copied.
