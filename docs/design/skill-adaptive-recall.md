# Skill-adaptive recall — gates + weights as an agent feature

**Status:** design + MVP scripts (2026-07-17). Driven by industry-bench
miss logs: chat distributions differ from coding; skill ≈ stock on
aggregate LoCoMo @5; confidence ordering rescues specific rank-0 cases.

## The interesting claim

Once we have labeled miss data (evidence@k, miss_reason, scoring lineage),
the **agent** — not a human grid search — can:

1. Diagnose *why* recall failed (gate vs rank vs true miss).
2. Propose a **local** weight/gate profile for this question class.
3. Apply it **per query** via `rekal --weights '{...}'` (preferred) or
   sticky `.rekal/config.json` / env (gitignored) and re-query.
4. Keep the ledger immutable — only query-time knobs move.

That is a product story distinct from “we tuned defaults on a paper”:
**memory that adapts under the agent’s hand**, with a transparent audit
trail of which profile was active.

## Soul check

| Question | Answer |
|---|---|
| Immutable? | Yes — `data.db` untouched; weights are query-time only. |
| Thin on wire? | Profiles live in local config / `.rekal/calibration/` (gitignored). |
| Secure / local? | No phone-home; agent runs scripts on-machine. |
| Simple? | One skill path (`calibrate.md`); `--weights` per query; opt-in `--apply`. |
| Agent first? | Skill decides when to calibrate; human sees the JSON / CLI it used. |

**Not:** silent auto-tuning on every query (that would fight Simple and
make recall non-reproducible). Calibration is an explicit skill move,
like MAP refresh.

## Two layers to tune

| Layer | Knobs | Where | Rebuild? |
|---|---|---|---|
| **Hunt gate** | `CONF_MIN`, `CONF_SOFT`, `GAP_MIN`, `MASS_MIN`, `KNOWLEDGE_MIN` | `hunt-gate.py` via `REKAL_HUNT_*` (env) today | No |
| **Hybrid weights** | `bm25`, `lsa`, `semantic`, boosts, `facet_boost` | `rekal --weights '{...}'` (CLI wins) or `.rekal/config.json` → `weights` | No (query-time) |

Industry-bench already proved env gate overrides. Weights already merge
local over global. CLI `--weights` makes the skill purely flexible without
writing config. Sticky `--apply` remains optional.

## Closed loop (lineage)

```
diagnose (miss / SILENCE)
   → calibrate-recall.py propose profile
   → rekal --weights '{...}' "<q>"     # no config write
   → `.rekal/scoring-lineage.ndjson`   # query.weights + candidate.contrib
   → next diagnose reads lineage
```

With `scoring_lineage.enabled` (local `.rekal/config.json` or global),
every hybrid recall already logs into **`.rekal/`** when `path` is
relative:

1. **`query`** — effective `weights` / `weights_normalized`, and
   `weights_source: "cli"` when `--weights` was set this turn.
2. **`candidate`** — per-layer raw/norm and weighted **`contrib`**
   (`bm25`/`lsa`/`nomic`/`facet`), confidence, mass.
3. **`result`** — returned set + timings.

That is the audit trail for “which profile was active and how layers
paid.” No separate telemetry. Calibrate consumes smoke manifests and/or
lineage (`--from-lineage`); it never rewrites `data.db`.

## Profiles (dynamic, not one global)

From miss_reason distributions we expect at least:

| Profile | When | Hint |
|---|---|---|
| `coding` (default) | ADLC sessions, high BM25 mass | ship defaults: CONF 0.70, MASS 3.5, semantic-heavy |
| `chat` | pure dialogue / industry-bench | lower CONF, MASS≈0, confidence-first inject |
| `multi-hop` | WHY / synthesis questions | deeper `--limit`, slightly higher LSA, WHY trail before HUNT |

Dynamic = **select profile per question class** (skill classifier already
triages pointed / why / map). Static = freeze one profile per repo after a
calibration session.

## Data inputs (once full-tier lands)

1. `runs/full/**/per_question.jsonl` — `miss_reason`, `evidence_rank`, `route_gate`
2. `.rekal/scoring-lineage.ndjson` — per-layer scores when enabled (relative path)
3. Optional: official judge labels (later) as the objective instead of evidence@k

Objective for MVP: maximize evidence@5 on a **held-out** slice; never tune
on the test split you report.

## MVP shipped in this change

| Artifact | Role |
|---|---|
| `scripts/calibrate-recall.py` | Propose gate/weight JSON from smoke manifests or lineage |
| `references/calibrate.md` | Skill workflow: diagnose → propose → apply → re-hunt |
| `SKILL.md` dispatch row | Analytical “recall feels wrong / calibrate” → calibrate.md |
| `scripts/industry-bench/calibration/propose_from_smokes.py` | Thin wrapper over the same logic for bench runs |

`--print-cli` / `--apply` emit:

- stdout compact JSON for `rekal --weights '...'` (preferred; no file write)
- optional sticky `.rekal/config.json` `weights` + `.rekal/calibration/`
- Gate env vars printed for the agent to export in-process

Gate bars are not yet in config.json (optional follow-up).

## What we will not do

- Hardcode chat bars into shipped `hunt-gate.py` defaults (breaks coding).
- Auto-apply on every `rekal` invocation.
- Tune on paper test splits after freeze (pre-registration discipline).
- Claim industry-bench gains as ADLC SOTA.

## Next after full-tier data

1. Fit `chat` / `multi-hop` profiles on LME-S **dev** only; commit proposals
   under `scripts/industry-bench/calibration/` (not into hunt-gate defaults).
2. Optional: `rekal config set-weights` CLI for humans.
3. Optional: config-backed hunt bars (`gates` section) so apply is one file.
4. Skill tip: on repeated SILENCE for non-abstention chat asks, suggest
   `Read references/calibrate.md` once — not a loop.
