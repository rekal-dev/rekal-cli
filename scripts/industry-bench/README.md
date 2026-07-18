# industry-bench — Rekal on the standard memory benchmarks

Implementation package for the program in
[`docs/research/industry-bench/`](../../docs/research/industry-bench/README.md).
Read the playbook first; this README is only the quickstart.

## Layout (target; built incrementally per workstream)

```
datasets/    # WS-A: acquisition scripts + SCHEMA.md (+ toy/ committed corpus)
sh_gen/      # WS-B: synthetic-history generator (this exists)
modes/       # WS-C: persona map + question classifier
calibration/ # WS-D: frozen per-dataset gates/weights
shim/        # WS-E: add/search/answer harness shim
baselines/   # WS-F: mem0 / zep / full-context / naive-RAG runners
scoring/     # WS-E: official-script wrappers + tables
runs/        # committed manifests + aggregates
```

## Quickstart: ingest the toy corpus and verify (WS-B round trip)

Requirements: a built `rekal` binary on PATH (see `docs/DEVELOPMENT.md` —
llama.cpp under `.deps/` first; **CI pins llama.cpp `b8157`** and
`--target common` — see `.github/workflows/ci.yml`), `git`, Python 3.11+.

```bash
cd rekal-cli
python3 scripts/industry-bench/sh_gen/gen.py \
  --input scripts/industry-bench/datasets/toy/conversations.jsonl \
  --out /tmp/imb-toy --verify
```

`--verify` runs the ingest-verification SQL from
[04-procedures §3](../../docs/research/industry-bench/04-procedures.md) per
conversation and exits non-zero on any mismatch.

## Real datasets (WS-A)

```bash
# LongMemEval-S (cleaned, Hugging Face) → conversations.jsonl
scripts/industry-bench/datasets/get_longmemeval.sh s
python3 scripts/industry-bench/datasets/normalize_longmemeval.py --variant s
python3 scripts/industry-bench/datasets/verify_dataset.py longmemeval-s

# LoCoMo + Penfield known-bad (99 score-corrupting)
scripts/industry-bench/datasets/get_locomo.sh
python3 scripts/industry-bench/datasets/normalize_locomo.py
scripts/industry-bench/datasets/get_locomo_known_bad.sh
python3 scripts/industry-bench/datasets/verify_dataset.py locomo
```

Raw files stay under `datasets/data/` (gitignored).

## Stock smoke (WS-E)

After `sh_gen --index` on a conversation:

```bash
python3 scripts/industry-bench/shim/shim.py smoke \
  --workdir /tmp/imb-lme-s-limit1 \
  --conversation-id e47becba \
  --input scripts/industry-bench/datasets/data/longmemeval-s-limit1.jsonl \
  --out scripts/industry-bench/runs/smoke/lme-s-e47becba \
  --rekal ./rekal
```

## Environment hazards (learned the hard way; encoded in sh_gen)

- **Never set `REKAL_BENCH` / `REKAL_SKIP_CHECKPOINT` in the ingest
  environment** — they make `SkipCapture` true for *every* payload
  (`cmd/rekal/cli/session/bench.go`), so checkpoints silently capture
  nothing. Those variables exist to protect a developer's *real* repos while
  a coding agent runs harness scripts. `sh_gen` strips them from the child
  environment it uses for `git commit` / `rekal` calls, and the same guard's
  cwd fingerprints mean a synthetic workdir path must not contain
  `/scripts/bench`, `rekal-bench`, or `/.rekal-bench` — `sh_gen` refuses
  such `--out` paths.
- Session discovery is isolated per workdir via `CLAUDE_CONFIG_DIR`
  (`<out>/<conversation>/claude-config/`), so synthetic sessions never mix
  with the machine's real `~/.claude` history — and vice versa.
- `sessions.captured_at` is ingestion time by design; the benchmark's
  temporal axis lives in `turns.ts` and the backdated commit dates. Verify
  against those, not `captured_at`.
