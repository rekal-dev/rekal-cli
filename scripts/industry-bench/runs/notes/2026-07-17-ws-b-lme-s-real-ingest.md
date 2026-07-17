# WS-B: LongMemEval-S real-data ingest (first instances)

Date: 2026-07-17 · Machine: linux/amd64 cloud agent (4 cores) ·
Rekal: `/workspace/rekal` built on this branch (`e19b20d`, CGO + nomic;
llama.cpp tip with `llama-common` compat — CI pins `b8157` / `--target
common`; rebuild against that pin when matching CI exactly)

## What ran

1. Normalized LongMemEval-S already verified (WS-A note).
2. `sh_gen/gen.py --input longmemeval-s-limit1.jsonl --verify --index`
   → conversation `e47becba` (53 sessions, 550 turns).
3. `sh_gen/gen.py --input longmemeval-s-limit3.jsonl --verify`
   → 3 conversations (incl. `e47becba`, plus two more).

## Results

| Run | conversations | sessions (sum) | wall | verify |
|---|---:|---:|---:|---|
| limit-1 + index | 1 | 53 | ~100 s | green |
| limit-3 | 3 | ~156 | ~280 s | green |

All ingest-verification checks from 04-procedures §3 passed: sessions,
turns, checkpoint_sessions links, `turns.ts` date span, backdated commits
per session id.

**Extrapolated full LongMemEval-S ingest** (500 instances, ~23867 sessions)
at ~1.8 s/session ≈ **12 hours** serial on this machine class. Record as
laptop-class estimate for WS-B DoD #3; full run should be kicked overnight
or parallelized across machines (one workdir shard per agent).

## Recall spot-check (`e47becba`)

Question: "What degree did I graduate with?" → gold "Business Administration"
(evidence session `answer_280352e9`). Run recall **from inside** the
synthetic repo after `--index` (cwd must be the repo; CLAUDE_CONFIG_DIR is
only for ingest discovery).

Result: top `results[0]` is the evidence session (snippet contains
"I graduated with a degree in Business Administration…"), score **0.87**,
confidence **0.83**, mass 7.11, linked commit for
`sessions/2023-05-30-answer_280352e9.md`. Knowledge layer also surfaces the
marker file for that session. First real-data evidence that stock hybrid
recall transfers on LongMemEval-S single-session-user.

## Findings

1. sh-gen + production checkpoint path works on real LongMemEval-S haystacks
   (claim #1 evidence beyond toy).
2. Per-conversation cost dominated by one `git commit` + `rekal checkpoint`
   per session (`--fast` batches sessions/commit for BEAM only).
3. Local build used tip llama.cpp; prefer CI pin `b8157` for reproducibility
   (see `.github/workflows/ci.yml`).
