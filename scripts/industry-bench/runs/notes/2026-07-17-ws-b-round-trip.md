# WS-B round-trip: first green run

Date: 2026-07-17 · Machine: linux/amd64 container, 4 cores ·
Rekal: built from `research/industry-bench` (dev build, llama.cpp embedded)

## What ran

`sh_gen/gen.py --input datasets/toy/conversations.jsonl --verify`
(toy corpus: 1 conversation, 3 sessions, 12 turns, dates 2023-05-20 →
2023-06-03), then `rekal index` and two recall queries in the synthetic repo.

## Results

- **Round trip green on first run**: sessions=3, turns=12,
  checkpoint_sessions=3 (distinct=3), `turns.ts` spans the benchmark dates,
  all three commits backdated correctly. Wall time: seconds.
- **Recall works on chat history**: `rekal "what is the name of the user's
  dog?"` → top hit is the exact turn ("…We named him Biscuit"), score 0.35,
  confidence 0.33, correctly linked to its synthetic commit and marker file.
- **Abstention signal exists**: the unanswerable "user's sister" query
  returns the dog turn at confidence 0.21 (vs 0.33 for the true hit) — the
  gate has separation to work with; thresholding it on chat distributions is
  exactly WS-D.

This is the first evidence for paper claim #1 (the commit anchor is a
format, not a domain restriction): the unmodified production pipeline
ingested pure dialogue and recall ranked it.

## Findings / gotchas encoded back into the playbook

1. `REKAL_BENCH=1` kills ALL capture (`session.SkipCapture`) — the original
   04-procedures env setup would have silently broken ingestion. Fixed:
   sh-gen strips it from the child env; workdir paths must avoid the guard's
   cwd fingerprints. (04-procedures §1, sh_gen/gen.py.)
2. `sessions.captured_at` is ingestion time (`ParseTranscript` sets
   `time.Now()`); the benchmark temporal axis is `turns.ts` + backdated
   commits. Verification SQL corrected. (04-procedures §3.)
3. Build gotcha: upstream llama.cpp renamed `common` → `llama-common`
   behind `LLAMA_BUILD_COMMON=ON`; docs' cmake line no longer produces
   `libcommon.a`. Workaround in 04-procedures §1; DEVELOPMENT.md should be
   updated separately (out of scope for this branch).
4. Post-`rekal index`, a background embed process briefly holds the
   `index.db` DuckDB lock; recall retries succeed within seconds. Harness
   code (WS-E shim) should retry on "another rekal process" rather than
   fail.
