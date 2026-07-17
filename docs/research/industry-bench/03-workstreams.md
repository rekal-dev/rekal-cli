# Workstreams — Distributable Plan

Seven workstreams, designed so separate agents can execute them with no
coordination beyond this file and the contracts in
[02-adapter-architecture.md](02-adapter-architecture.md). Each has: inputs,
outputs, interface, definition of done (DoD). An agent claims a workstream by
setting the [board](#board) row to `in-progress (<who>)` in its first commit.

**Global rules for every workstream**
- Branch: work lands on `research/industry-bench` via small PRs (or direct
  commits if solo), one workstream per commit series.
- Rekal core is frozen (README decision #2). Needing a core change = stop,
  write it up in `runs/notes/`, surface to the operator.
- Everything under `scripts/industry-bench/` is Python 3 stdlib + `duckdb`
  + `requests` only, unless the workstream doc says otherwise (keeps agents
  from dependency-fighting each other).
- Every script: `--help`, deterministic given `--seed`, exits non-zero on
  contract violation.
- Set `REKAL_BENCH=1` in any environment where a coding agent runs the
  harness (see `cmd/rekal/cli/session/bench.go`).

---

## WS-A — Dataset acquisition & verification

- **Inputs**: [00-landscape](00-landscape.md) links; [01-selection](01-selection.md) order.
- **Outputs**: `scripts/industry-bench/datasets/get_<name>.sh` per dataset
  (download, checksum, license note, normalize to the *common interchange
  format* below); `datasets/SCHEMA.md` documenting that format.
- **Common interchange format** (the contract every other WS consumes):
  `conversations.jsonl` — one line per conversation:
  `{"conversation_id", "sessions": [{"session_id", "date", "turns": [{"role": "user"|"assistant", "text"}]}], "questions": [{"question_id", "category", "question", "answer", "evidence_session_ids"}]}`
- **DoD**: LongMemEval-S, LongMemEval, LoCoMo, MSC all normalize; a
  `verify_dataset.py` prints counts matching the papers' reported sizes;
  license/redistribution note per dataset (datasets are `.gitignore`d,
  scripts are committed).
- **Depends on**: nothing. **Feeds**: B, E, F.

## <a name="ws-b"></a>WS-B — Synthetic-history generator (`sh-gen`)

- **Inputs**: interchange format (WS-A); ingestion contract
  ([02 §1](02-adapter-architecture.md#ingestion)).
- **Outputs**: `sh_gen/` implementing the contract: repo-per-conversation,
  commit-per-session (backdated), Claude-JSONL session files at the
  sanitized discovery path, hook-driven `rekal checkpoint`, post-ingest
  `rekal index`, `--fast` batching mode for BEAM.
- **DoD**:
  1. Round-trip test: 3-session toy conversation → `sh-gen` → SQL
     verification (sessions=3, checkpoints=3, `checkpoint_sessions` links
     correct, turn counts exact, `captured_at`/commit dates match benchmark
     dates).
  2. Golden-file test: generated JSONL parses via `rekal checkpoint`
     (not via a reimplementation of the parser).
  3. LongMemEval-S full ingest completes on a laptop-class machine; wall
     time recorded in `runs/notes/`.
- **Depends on**: A (format only — can start against a hand-written toy
  `conversations.jsonl`). **Feeds**: C, D, E.

## WS-C — Mode mapping: persona map + question classifier

- **Inputs**: mode table ([02 §2](02-adapter-architecture.md)); a WS-B
  ingested corpus.
- **Outputs**: `modes/persona_map.py` (digest prompt + incremental,
  SHA-watermarked refresh; optional `docs/persona.md` write-back into the
  synthetic repo so the knowledge layer indexes it); `modes/classify.py`
  (question → pointed / why / persona / abstain-candidate; prompt derived
  from the shipped skill tip's triage language).
- **DoD**: on the LongMemEval-S dev split, classifier confusion matrix
  committed; persona map refresh is incremental (touching one new session
  re-digests only the delta) and reproducible.
- **Depends on**: B. **Feeds**: E.

## WS-D — Gate & weight calibration

- **Inputs**: [02 §3](02-adapter-architecture.md); dev split (WS-A); an
  ingested dev corpus (WS-B).
- **Outputs**: `calibration/<dataset>.json` (gate thresholds + `weights`),
  produced by a grid script following `scripts/bench/tune_weights.py`
  conventions; calibration report in `runs/notes/`.
- **DoD**: calibration files committed and tagged **before** any test-split
  run exists in `runs/` (pre-registration,
  [04 §prereg](04-procedures.md#prereg)); the grid script reruns
  deterministically.
- **Depends on**: B, and C for the persona/abstain routes. **Feeds**: E.

## WS-E — Harness shim, scoring & token accounting

- **Inputs**: shim contract ([02 §4](02-adapter-architecture.md#shim));
  harness choice ([01 §harness](01-selection.md)).
- **Outputs**: `shim/` (add/search/answer with token logging);
  `scoring/` (wrappers over each benchmark's official eval script; our
  blind judge as secondary; table generator emitting the mandatory
  tokens-per-question columns); run-manifest writer
  ([04 §manifest](04-procedures.md#manifest)).
- **DoD**: end-to-end smoke run — 10 LongMemEval-S dev questions through
  ingest → route → answer → official scorer → table — committed under
  `runs/smoke/`; token numbers non-zero and plausible; a deliberately wrong
  answer scores 0 (scorer sanity).
- **Depends on**: B (C, D pluggable later — shim must run with stock gates
  first). **Feeds**: G.

## WS-F — Baselines

- **Inputs**: baseline list ([01 §baselines](01-selection.md)); same
  pinned answer model/judge as WS-E.
- **Outputs**: `baselines/` runners for Mem0 OSS, Zep/Graphiti OSS,
  full-context stuffing, naive RAG — all through the same shim verbs and the
  same scorer, all logging tokens identically.
- **DoD**: each baseline completes the same 10-question smoke set; Mem0's
  LongMemEval-S dev number is within shouting distance of published values
  (else investigate harness bug before blaming Mem0); token columns present.
- **Depends on**: E. **Feeds**: G. **Stretch**: BEAM tier run per
  [01 #5](01-selection.md).

## WS-G — Analysis & paper #2

- **Inputs**: everything in `runs/`; [05-paper-plan](05-paper-plan.md).
- **Outputs**: consolidated results doc in `runs/consolidated/`; paper
  draft under `docs/research/industry-bench/paper/` (Typst, following
  paper #1's `paper/` conventions).
- **DoD**: every number in the draft traces to a committed run manifest;
  the claims ladder in [05](05-paper-plan.md) has evidence rows filled or
  explicitly marked unmet.
- **Depends on**: E + F (D, C improve it).

---

## Dependency DAG

```
A ──▶ B ──▶ C ──▶ D ──▶ E' (calibrated rerun)
      │           ▲
      └──────▶ E (stock-gate smoke) ──▶ F ──▶ G
```

Parallelization: A and B start immediately (B against the toy corpus);
C and E in parallel once B lands; D after C; F after E; G last. Maximum
useful simultaneous agents: 3–4.

## <a name="board"></a>Board

| WS | Title | Status | Owner | Notes |
|---|---|---|---|---|
| A | Datasets | in-progress (cursor/industry-bench-longmemeval-real-ff4f) | LME-S + LoCoMo normalize/verify green; Penfield known-bad 99 rows committed (`locomo-known-bad.jsonl`); MSC + LME-M still open |
| B | sh-gen | in-progress (cursor/industry-bench-longmemeval-real-ff4f) | LME-S limit-1/3 + LoCoMo conv-26 verify green; LME-S limit-20 ingest running; recall gold on e47becba (0.87) |
| C | Modes | open | — | |
| D | Calibration | open | — | pre-registration gate; abstention separation observed on toy (conf 0.21 vs 0.33) |
| E | Shim + scoring | in-progress (cursor/industry-bench-longmemeval-real-ff4f) | shim now supports `--route skill` (`recall-route.py`), marker-only knowledge suppression, and non-abstention SILENCE→stock fallback; LME/LoCoMo skill smokes committed |
| F | Baselines | open | — | |
| G | Analysis + paper | open | — | claim #1 has first toy-scale evidence |

Update this table in the same commit as the work. Statuses:
`open → in-progress (<owner>) → review → done`.
