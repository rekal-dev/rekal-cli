# Procedures — Runbooks and Discipline

Operational companion to [03-workstreams.md](03-workstreams.md). These are
the procedures every agent follows; deviations are recorded in
`scripts/industry-bench/runs/notes/`, not improvised silently.

## 1. Environment setup (once per machine)

```bash
# Rekal binary: build from THIS branch's parent main, record the SHA
cd rekal-cli && mise install && <build>   # see docs/DEVELOPMENT.md
rekal version                             # goes in every run manifest

# Python: 3.11+; stdlib + duckdb + requests only (03-workstreams global rules)
python3 -m venv .venv && . .venv/bin/activate && pip install duckdb requests
```

**`REKAL_BENCH` is a capture kill-switch — handle it precisely.** Setting it
makes `SkipCapture` true for *every* payload
(`cmd/rekal/cli/session/bench.go`), which protects your real repos while a
coding agent runs harness scripts — and silently breaks synthetic ingestion
if it leaks into `sh-gen`'s environment. The rule:

- In the shell where a coding agent drives this work: `export REKAL_BENCH=1`.
- `sh_gen/gen.py` strips `REKAL_BENCH`/`REKAL_SKIP_CHECKPOINT` from the
  child environment it uses for `git commit`/`rekal` — never bypass that.
- Synthetic workdir paths must not contain `/scripts/bench`, `rekal-bench`,
  or `/.rekal-bench` (the same guard's cwd fingerprints); `sh-gen` refuses
  such `--out` paths.

**Building `rekal` (2026-07 gotcha):** upstream llama.cpp renamed the
`common` target to `llama-common` and gates it behind
`LLAMA_BUILD_COMMON=ON`. Until DEVELOPMENT.md catches up: configure with
`-DLLAMA_BUILD_COMMON=ON`, build `--target llama-common`, copy
`libllama-common.a` to `build/common/libcommon.a` (plus
`libllama-common-base.a` alongside), and link with
`CGO_LDFLAGS="-lllama-common-base"`.

GPU/embedding server (full LongMemEval / BEAM only): serve
`nomic-embed-text-v1.5` on an OpenAI-compatible endpoint (vLLM/Ollama), then
per synthetic repo set `.rekal/config.json` →
`embedding.endpoint`/`model`. Same model as embedded; endpoint recorded in
the manifest.

## 2. Dataset acquisition (WS-A runbook)

1. `datasets/get_<name>.sh` — download from the canonical source in
   [00-landscape](00-landscape.md), verify checksum, write `LICENSE-NOTE.md`
   (may we redistribute? usually no → datasets stay `.gitignore`d).
2. Normalize to `conversations.jsonl` (interchange schema:
   [03 WS-A](03-workstreams.md)).
3. `python3 datasets/verify_dataset.py <name>` — counts must match the
   benchmark paper's reported sizes; paste the output into `runs/notes/`.
4. **LoCoMo only**: also fetch the Penfield audit list of bad labels
   ([00 §6](00-landscape.md)); store as `datasets/locomo-known-bad.jsonl`;
   scorer reports with-and-without-bad-labels columns.

## <a name="verify"></a>3. Ingest verification (after every `sh-gen` run)

`sh_gen/gen.py --verify` runs this automatically; the checks it makes:

```bash
cd <workdir>/<conversation-id>/repo
rekal query "SELECT count(*) FROM sessions"              # = session count
rekal query "SELECT count(*) FROM turns"                 # = sum of turns
rekal query "SELECT count(*) FROM checkpoint_sessions"   # = session count
rekal query "SELECT count(DISTINCT session_id) FROM checkpoint_sessions"
rekal query "SELECT min(ts), max(ts) FROM turns"  # spans the benchmark dates
git log --format='%aI %s' --reverse           # commit dates = session dates
```

**`sessions.captured_at` is ingestion time by design** (`ParseTranscript`
sets `CapturedAt = time.Now()`), so it must NOT be used to verify the
benchmark's temporal axis — that axis lives in `turns.ts` and the backdated
commit dates, which is what temporal-reasoning questions retrieve against.

Any mismatch: stop, fix `sh-gen`, re-ingest from scratch (`rekal clean` +
delete workdir). Never hand-patch a corpus.

## 4. Smoke test (before any full run; WS-E DoD)

10 dev-split questions, LongMemEval-S, stock gates:
ingest → classify → route → recall → answer → official scorer → table.
Committed to `runs/smoke/`. A full run may only start on a green smoke run
of the exact code that will do the full run.

## <a name="prereg"></a>5. Pre-registration discipline (WS-D)

Adopted unchanged from paper #1's RHO ship discipline and the one-snapshot
rule (`docs/research/06-eval-strategy.md`):

1. All tuning on the **dev split only**.
2. Frozen thresholds/weights committed to `calibration/<dataset>.json`
   **before** the first test-split run; the commit SHA of the calibration is
   recorded in the test run's manifest.
3. One test run per (system, dataset, calibration) triple counts. Reruns
   happen only for harness bugs, are labeled `rerun-of=<manifest-id>`, and
   the superseded run stays committed.
4. Negative results ship: a mode or gate that doesn't transfer is a paper
   section, not a deleted row (paper #1's mechanism graveyard is the model).

## <a name="pins"></a>6. Pins (one table, all systems)

Every comparison table holds these constant across Rekal and all baselines:

| Pin | Where recorded |
|---|---|
| Answer model + version + temperature | manifest `answer_model` |
| Judge model + prompt hash | manifest `judge` |
| Dataset snapshot checksum | manifest `dataset_sha` |
| Rekal binary SHA / baseline lib versions | manifest `system` |
| Calibration file SHA (or `stock`) | manifest `calibration` |
| Seed | manifest `seed` |

If an upstream harness hardcodes a different answer model for its own
system, that system's row is footnoted — never silently mixed.

## <a name="manifest"></a>7. Run manifest & record format

Every run writes `runs/<dataset>/<system>/<timestamp>/`:

- `manifest.json` — the pins table above + git SHA of
  `scripts/industry-bench/` + wall time + machine class.
- `per_question.jsonl` — question id, category, routed mode, gate verdict,
  retrieved session/turn ids, retrieved-context tokens, answer-path tokens,
  answer text, official-judge verdict, our-judge verdict.
- `table.md` — the generated summary table (official metric per category ×
  tokens mean/p95).

Aggregates and manifests are committed; raw retrieved text is not (dataset
license). `runs/consolidated/` follows paper #1's convention including its
`TRANSCRIBED_PENDING_VERIFICATION` status marker until numbers are
double-entered.

## <a name="scoring"></a>8. Scoring authority

1. Official benchmark script = headline. Wrap, don't fork; record upstream
   commit SHA of the eval script in the manifest.
2. Our blind sufficiency judge = secondary column, labeled.
3. Tokens-per-question = mandatory in every table (mean + p95, per
   category). No table ships without it.

## <a name="honesty"></a>9. Honesty rules (paper-blocking)

- **LoCoMo caveat block is mandatory** anywhere a LoCoMo number appears:
  ~6.4% wrong labels; judge accepts up to 63% of intentionally wrong
  answers; we report with/without known-bad labels ([00 §6](00-landscape.md)).
- **LongMemEval-S context-window disclosure**: report the naive
  full-context baseline in the same table so the reader sees what fits in a
  window; the full-size LongMemEval column is the answer to that objection.
- **No cross-benchmark averaging.** LoCoMo and LongMemEval measure different
  constructs.
- **Vendor numbers** appear only in a "self-reported" column with citations;
  our table's comparisons are only runs we executed under §6 pins.
- **Out-of-domain framing**: Rekal is an ADLC system running off its home
  turf. The paper claims *structure transfer at a disclosed token budget* —
  it does not need SOTA to succeed, and the writing must not quietly imply
  SOTA if the numbers don't show it.
- **Gate honesty**: abstention accuracy is reported with the gate's false
  -silence rate (questions it wrongly refused), not just its wins.

## 10. Escalation

Stop and surface to the operator (don't work around): dataset license
forbidding even local benchmark use; any need to modify Rekal core; a
baseline reproducing >10 points off its published number after a
harness-bug check; judge-model access problems that would force an unpinned
substitute.
