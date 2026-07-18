# 06 — LoCoMo Skill Closure: the iterative subagent process

**Goal:** close the gap between Rekal's ~98% retrieval and Mem0's ~92.5%
end-to-end LoCoMo accuracy **at the skill layer only**. The core is frozen;
the data is already retrievable; every remaining point of accuracy lives in
`SKILL.md` + `references/hunt.md` + the route/gate/calibrate scripts. This
document is the handover: the current state, the loop that has already
produced six merged skill PRs, and the process to keep turning it.

**Scope:** LoCoMo (10 conversations, 1,986 questions, 1,536 answerable).
The same loop applies to LME-M/S, but this page is written for LoCoMo.

---

## 0. Setup from scratch (new environment)

Nothing from the original machine is portable — the ingested repos, the
`/tmp` task files, and the run artifacts under `~/imb-*` all get rebuilt.
Everything needed is in git; setup is deterministic (session/turn UUIDs are
uuid5 of stable keys, commit dates come from the dataset), so a re-ingest
reproduces the same history.

```bash
git clone <repo> && cd rekal-cli
git checkout research/industry-bench
mise run build
export REKAL="$PWD/rekal"
IB=scripts/industry-bench

# 1. Download + normalize + verify (10 conv / 1,986 questions expected)
bash    $IB/datasets/get_locomo.sh
python3 $IB/datasets/normalize_locomo.py
python3 $IB/datasets/verify_dataset.py locomo

# 2. Ingest through the production pipeline: one synthetic git repo per
#    conversation, commits backdated to session dates. --fast defaults to
#    1 (contract mode), correct for LoCoMo. Minutes, not hours.
python3 $IB/sh_gen/gen.py \
  --input $IB/datasets/data/locomo-conversations.jsonl \
  --out ~/imb-locomo --rekal "$REKAL" --index --verify
```

Sanity check: `cd ~/imb-locomo/conv-26/repo && "$REKAL" "LGBTQ support group"`
should return session hits.

**3. Build tasks/gold files** from the interchange format (each
conversation's `questions[]` carries `question_id`, `category`, `question`,
`answer`, `evidence_session_ids`). Exclude the 99 known-bad questions
(`$IB/datasets/locomo-known-bad.jsonl`, keyed by `conversation_id` +
`question_id`):

```bash
mkdir -p /tmp/locomo-local && python3 - <<'EOF'
import json
IB = "scripts/industry-bench"
bad = {(r["conversation_id"], r["question_id"])
       for r in map(json.loads, open(f"{IB}/datasets/locomo-known-bad.jsonl"))}
tasks, gold = [], []
for c in map(json.loads, open(f"{IB}/datasets/data/locomo-conversations.jsonl")):
    for q in c["questions"]:
        if (c["conversation_id"], q["question_id"]) in bad:
            continue
        qid = f'{c["conversation_id"]}:{q["question_id"]}'
        tasks.append({"id": qid, "conv": c["conversation_id"],
                      "category": q["category"], "question": q["question"]})
        gold.append({"id": qid, "gold": q["answer"],
                     "evidence": q["evidence_session_ids"]})
with open("/tmp/locomo-local/tasks.jsonl", "w") as f:
    f.writelines(json.dumps(t) + "\n" for t in tasks)
with open("/tmp/locomo-local/gold.jsonl", "w") as f:
    f.writelines(json.dumps(g) + "\n" for g in gold)
print(len(tasks), "tasks")
EOF
```

For a stratified dev sample, filter `tasks` by category before writing (§3.1).

**4. Run** (local Claude must be installed and authed — `claude -p 'hi'`
must answer):

```bash
python3 $IB/run_local_e2e.py \
  --tasks /tmp/locomo-local/tasks.jsonl \
  --gold  /tmp/locomo-local/gold.jsonl \
  --repos-root ~/imb-locomo --out $IB/runs/<run-name> \
  --model haiku --judge-model sonnet --prompt-style card
```

The runner checkpoints per question and pauses/resumes on token-window
exhaustion; re-invoking with the same `--out` continues where it stopped.
Category 5 (adversarial) gold is abstention — the normalized answers encode
this, so the runner and judge handle it without special-casing.

---

## 1. Where we are (measured, 2026-07-18)

**Retrieval is not the problem.** Stock `rekal` with `bm25-push` weights on
all 1,536 answerable questions:

| Metric | Value |
|---|---:|
| evidence@5 | 0.8919 |
| evidence@10 | 0.9538 |
| evidence@20 (skill's real operating point) | **0.9798** |
| evidence@20 with multi-lookup RRF fusion | 0.982 |

**End-to-end is the problem.** LLM answers a question by running the skill
against the ingested repo, judged against gold:

| Run | Executor | Prompt style | Score | Avg non-cached tok/q |
|---|---|---|---:|---:|
| `runs/locomo-local-e2e` | haiku | skill files | 18/40 (45%) | ~8k |
| `runs/locomo-local-e2e-card` | haiku | inlined card | 25/40 (62%) | ~2.7k |
| `runs/locomo-local-e2e-sonnet` | sonnet | inlined card | 22/40 (55%) | — |
| `runs/locomo-5q-sonnet-skill` | sonnet | full skill, coding gate | 1/5 | — |
| `runs/locomo-5q-sonnet-chatgate` | sonnet | full skill, **chat gate** | 2/5 | ~2.5k |
| Fresh 40q via strong subagents, post-#54/#55 skill | subagent | full skill | **36/40 (90%)**, 8/8 adversarial abstain | — |
| Mem0 (reference) | gpt-4o | single-pass RAG | ~92.5% | ~7k |

The 98%→~50-62% drop on weak executors decomposes into exactly three causes,
in this order of impact:

1. **Wrong gate profile.** `hunt-gate.py` shipped defaults
   (`MASS_MIN=3.5`, `CONF_MIN=0.70`) are calibrated for coding ledgers.
   Chat dialogue scores real hits at conf 0.5–0.6 with low BM25 mass →
   `SILENCE below_mass` → wrongful abstention. Fixed by profile routing
   (PR #56): the skill now selects `chat` gates before HUNT on dialogue
   ledgers. This alone took the sonnet 5q probe from 1/5 to 2/5, and the
   remaining 3 were reasoning errors, not silences.
2. **Executor capability.** Haiku cannot hold the multi-step procedure from
   skill files; the inlined card recovers most of it (45%→62%). Strong
   executors (subagent-class) running the same skill hit 90%. The skill text
   must therefore be written so the *weakest intended executor* can follow it
   — every fix should compress to a card-expressible rule.
3. **Category-specific reasoning gaps.** The long tail: false-premise
   adversarials, open-domain inference, complete-set enumeration, event-time
   phrasing, instance→class mapping, non-verbal evidence (photo captions).
   Each one so far has been fixable with a targeted `hunt.md` rule.

## 2. The invariant (do not relitigate)

Only **general skill-layer changes** ship to `main`: `SKILL.md`, the
`references/*.md` pages, and the `scripts/` under the skill
(`recall-route.py`, `hunt-gate.py`, `calibrate-recall.py`). No
LoCoMo-specific strings, no benchmark IDs, no per-dataset thresholds baked
into the skill. A fix qualifies only if it is stated as a general recall
principle ("enumerate instances, map to the question's class") that happens
to be exercised by LoCoMo. Benchmark plumbing (`run_local_e2e.py`, shim,
sweeps) stays on `research/industry-bench`.

## 3. The loop

One iteration = sample → run → judge → cluster → mine → patch → regress →
merge → fresh sample. Six iterations have already run (PRs #51–#56).
Each stage names who executes it: the orchestrator (you) or a subagent.

### 3.1 Sample (orchestrator)

Draw N questions (40 is the working size) from the **dev pool** —
conversations reserved for tuning. Never draw from the frozen test split you
intend to report. Stratify across the five LoCoMo categories
(single-hop / multi-hop / temporal / open-domain / adversarial) —
the card run's mix (12/8/8/4/8) is a reasonable template. Exclude the 99
known-bad questions (committed list under `scripts/industry-bench/datasets/`).

### 3.2 Run (one subagent per question — clean context is non-negotiable)

Each question runs in an **isolated subagent with no carryover context**:
the subagent gets the repo path, the question, and the instruction to follow
the installed skill (`.claude/skills/rekal/SKILL.md`). Nothing else. Prior
answers, prior transcripts, and the orchestrator's knowledge of gold labels
must not leak in — carryover context was a measured contaminant in early
runs.

Two harness options:

- **Subagent route** (strong executor, what produced 36/40): Task-tool
  subagents, one per question, full skill files.
- **Local runner** (`scripts/industry-bench/run_local_e2e.py`): local
  Claude (haiku/sonnet), checkpointed, token-window-aware pause/resume,
  per-question `usage` logging (input / output / cache_read /
  cache_creation / cost). Use `--prompt-style card` for weak executors.

Always record per-question token usage. Non-cached tokens
(`input + cache_creation`) is the number comparable to Mem0's ~7k.

### 3.3 Judge (one subagent, gold-blind runners)

A separate judge subagent scores `answers.jsonl` against gold →
`judged.jsonl` with `verdict` ∈ {CORRECT, INCORRECT, ABSTAIN}. The judge
never runs questions; runners never see gold. LoCoMo's official protocol is
LLM-as-judge, so a local strong model is acceptable for dev iterations;
the reported run should use the official judge model.

### 3.4 Cluster failures (one analyst subagent, broad context)

Feed the analyst **all failed questions' full transcripts** (including every
`rekal` command the runner issued) and ask for failure clusters, not
per-question fixes. This is the highest-leverage step and the one that found
every pattern so far. The Fable-powered review that produced PR #52
("the skill taught Rekal as a search box, not a navigation instrument") came
from exactly this: reading transcripts for *how the agent moved*, not just
what it answered.

Classify each failure as one of:

| Class | Meaning | Fix lives in |
|---|---|---|
| `gate` | wrongful SILENCE (below_mass / below_gate on a real hit) | profile routing / calibrate |
| `nav` | evidence retrievable but agent didn't reach it (no reformulation, no depth widening, no time-axis move) | `hunt.md` procedure |
| `reason` | evidence in hand, wrong conclusion (premise, class-mapping, enumeration, temporal phrasing) | `hunt.md` rules / card |
| `executor` | skill text correct, model failed to follow it | card compression, not skill content |
| `data` | evidence genuinely absent or question known-bad | exclude, no skill change |

### 3.5 Mine skill opportunities (orchestrator + Rekal itself)

Before writing a fix, run `rekal "<failure pattern>"` — the prior solution
may already exist in the ledger (this is the dogfood loop: reasoned answers
go back into memory). Then draft the fix as a **general rule** and check it
against the invariant in §2. If the rule can't be stated without mentioning
the benchmark, it's overfit — reject it.

### 3.6 Patch + regress (orchestrator)

Apply the change to the skill files, then re-run **the accumulated
regression set**: every question fixed in a prior iteration, plus the current
failures. The regression set grows monotonically; a patch that fixes 3 and
breaks 2 previously-fixed is a net regression and does not merge. Current
regression seeds: the LME-M 5q set (temporal anchoring), the 9-question
three-run-consistent LoCoMo failures (false premise, open-domain, enumeration,
event-time), the turtles-bath / photo-caption pair, the titles→authors
class-mapping case, and the sonnet 5q chat-gate set.

### 3.7 Merge to main (orchestrator)

Skill changes go to `main` via a worktree branch and PR, then get synced back
into `research/industry-bench`. This has been the standing rule ("skill goes
to main always once confirmed working") and the ledger below shows the
precedent.

### 3.8 Fresh sample (orchestrator)

After merge, draw a **new** stratified sample (never re-report the tuning
sample) and run §3.2–§3.3. This number is the iteration's headline. The
post-#54/#55 fresh sample scored 36/40 with perfect adversarial abstention.

## 4. Ledger of iterations so far

| Iter | Failure signal | Skill change | PR |
|---|---|---|---|
| 1 | ev@20 misses recoverable at depth 100 | depth-as-judgment guidance | #51 |
| 2 | ad-hoc SQL improvisation, no zoom moves | time-axis navigation, enumeration discipline, attribution, turn-window drill | #52 |
| 3 | `search -n 20` raw JSON = 57% of token cost | compact candidate digest on INJECT (92–97% fewer tokens) | #53 |
| 4 | 9 three-run-consistent failures: false premise / open-domain / enumeration / event-time | four card-level rules | #54 |
| 5 | turtles-bath enumeration, photo-caption evidence, titles→authors | entity-class enumeration + instance→class mapping | #55 |
| 6 | coding gate silences chat hits (`below_mass`) | coding-vs-chat **profile routing** before HUNT + `calibrate-recall.py` shipped | #56 |

## 5. Definition of done (vs Mem0)

Dev target, then the frozen run:

1. **Dev exit bar:** ≥90% judge-correct on a fresh stratified 200-question
   dev sample, ≥7/8 adversarial abstention rate, at ≤7k non-cached
   tokens/question median — i.e. Mem0-class accuracy at Mem0-class or better
   token cost, with an executor no stronger than sonnet.
2. **Frozen run:** full answerable set (1,536 questions, or the per-category
   official protocol) on the untouched test conversations, official judge,
   no skill edits between dev exit and report. Report accuracy per category,
   abstention, and the token distribution (non-cached and effective-context
   separately; Mem0 reports single-pass tokens, we report the loop's total).
3. **Honesty rules from `04-procedures.md` apply:** pre-register the config
   (skill commit SHA, gate profile, weights, executor model, judge model)
   before the frozen run.

Why this is reachable: retrieval already delivers 0.98 of the answers into
the top-20; strong-executor runs already hit 90% on 40q; the remaining work
is compressing that behavior into skill text a mid-tier executor follows.
The lazy-inference thesis (Rekal defers reasoning to query time instead of
paying LLM extraction at write time) survives only if the loop's token cost
stays in Mem0's band — which the digest (#53) and chat profile (#56) already
brought to ~2.5–2.7k non-cached tokens/q on card runs.

## 6. Open items (next iterations start here)

- **Chat-profile adoption by weak executors.** #56 routes profiles in skill
  text; verify haiku-card actually exports the gates (the card must carry the
  one-liner). If not, fold the profile selection into `recall-route.py`
  itself (auto-detect dialogue ledger) — still skill-layer, zero executor
  burden.
- **Multi-lookup as default for chat.** `skill-multi` (reformulate → RRF
  fuse) adds +0.02 ev@20 for one extra search round. Decide whether hunt.md
  makes it the default chat move or keeps it as the reformulation fallback.
  Measure the token delta before deciding.
- **The 4/40 residue** from the 36/40 fresh sample — transcripts not yet
  clustered. Run §3.4 on them.
- **Judge alignment.** Local judge vs official judge agreement has not been
  measured; sample 50 judged answers with both before trusting dev numbers.
- **Scale the dev sample** from 40 to 200 once the above land, then hold the
  exit bar from §5.

## 7. Pointers

| What | Where |
|---|---|
| Local resilient runner | `scripts/industry-bench/run_local_e2e.py` |
| Shim (retrieval routes: stock / skill / skill-multi) | `scripts/industry-bench/shim/shim.py` |
| Fast weight sweep | `scripts/industry-bench/eval_bench_sweep.py` |
| Multi-lookup ceiling eval | `scripts/industry-bench/eval_locomo_multilookup.py` |
| Winning weights (bm25-push, 3-benchmark consensus) | `scripts/industry-bench/calibration/` |
| Run artifacts (answers/judged, per-question usage) | `scripts/industry-bench/runs/locomo-*` |
| Skill under iteration | `cmd/rekal/cli/skill/skills/rekal/` |
| Profile gates / weights | `.../skills/rekal/scripts/calibrate-recall.py` |
| Ingested LoCoMo repos | `~/imb-locomo/` (rebuilt from §0 in a new env) |
| Known-bad question list (99) | `scripts/industry-bench/datasets/locomo-known-bad.jsonl` |
