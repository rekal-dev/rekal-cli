# industry-bench handover — run on your machine

**Date:** 2026-07-17  
**Goal:** finish LongMemEval-S full ingest (500 conversations / 23,867 sessions), then run stock + skill smokes and official scoring.

Cloud-agent `/tmp` workdirs are **not portable**. On your machine you re-download data and re-ingest from scratch (or copy `datasets/data/` + workdirs if you saved them).

### Copy-paste start (after clone + build)

```bash
git checkout research/industry-bench
export REKAL="$PWD/rekal"   # or path to built binary

bash scripts/industry-bench/datasets/get_longmemeval.sh s
python3 scripts/industry-bench/datasets/normalize_longmemeval.py --variant s
python3 scripts/industry-bench/datasets/verify_dataset.py longmemeval-s

chmod +x scripts/industry-bench/run_ingest_shards.sh
./scripts/industry-bench/run_ingest_shards.sh ~/imb-lme-s 8 25
```

Then bulk-index (ingest does not run `rekal index`):

```bash
find ~/imb-lme-s -path '*/repo/.rekal' -printf '%h\n' | while read -r repo; do
  (cd "$repo" && "$REKAL" index)
done
```

---

## 1. What is done (in git)

| Item | Status |
|---|---|
| WS-A: LME-S + LoCoMo getters, normalizers, `verify_dataset.py` | done |
| LoCoMo known-bad list (99 score-corrupting) | committed |
| WS-B: `sh_gen` round-trip + `--offset` for sharding | done |
| WS-E: `shim/shim.py` + `scoring/smoke_score.py` | done |
| Core fix: `hunt-gate.py` ignores marker-only knowledge | in PR #49 |
| Stock + skill smoke manifests (toy-scale) | under `runs/smoke/` |
| Explain logs for top-5 misses | `runs/notes/2026-07-17-topk-misses-explain.md` |

**Branches / PRs**

| Branch | Base | PR |
|---|---|---|
| `cursor/industry-bench-longmemeval-real-ff4f` | `research/industry-bench` | [#48](https://github.com/rekal-dev/rekal-cli/pull/48) |
| `cursor/industry-bench-skill-route-main-ff4f` | `main` | [#49](https://github.com/rekal-dev/rekal-cli/pull/49) |

Playbook (read first): `docs/research/industry-bench/README.md`

---

## 2. Measured Rekal results so far (real data, small N)

**Retrieval (stock `rekal`, evidence session in results)**

| Corpus | n | evidence@5 | evidence@10 |
|---|---:|---:|---:|
| LME-S smoke `e47becba` | 1 | 100% | 100% |
| LME-S indexed sample | 8 | 75% | **100%** |
| LoCoMo `conv-26` q1–q10 | 10 | 90% | **100%** |

**Skill layer:** `--route skill` fixes rank-0 artifacts when confidence ranks evidence lower than score (see `runs/notes/2026-07-17-failed-cases-skill-route.md`). LoCoMo q8 is still a true miss at rank 0 but evidence appears by rank 8.

**Not measured yet:** official LongMemEval judge accuracy, abstention category, full 500, baselines.

**Tokens (whitespace proxy, extractive smoke):** ~200–315 answer-path tokens/question vs full haystack.

---

## 3. Machine setup (do once)

### 3.1 Clone and branch

```bash
git clone https://github.com/rekal-dev/rekal-cli.git
cd rekal-cli
git checkout research/industry-bench
```

### 3.2 Build `rekal` (match CI — important)

CI pins **llama.cpp tag `b8157`** and builds target **`common`** (not `llama-common`).

```bash
# deps (Linux)
sudo apt-get install -y cmake build-essential g++ libstdc++-14-dev git git-lfs

git lfs install
git lfs pull   # nomic model embed: cmd/rekal/cli/nomic/models/*.gguf.gz

rm -rf .deps/llama.cpp
git clone --depth 1 --branch b8157 https://github.com/ggml-org/llama.cpp .deps/llama.cpp
cd .deps/llama.cpp
cmake -B build \
  -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF \
  -DLLAMA_BUILD_SERVER=OFF -DBUILD_SHARED_LIBS=OFF \
  -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release -j"$(nproc)" \
  --target llama --target ggml --target common
cd ../..

export CGO_ENABLED=1
go build -o rekal ./cmd/rekal
./rekal version
export PATH="$PWD:$PATH"
```

### 3.3 Python

Stdlib only for industry-bench scripts. No venv required unless you want one.

```bash
python3 --version   # 3.11+
```

### 3.4 Environment rules

| Variable | Ingest? | Why |
|---|---|---|
| `REKAL_BENCH=1` | OK in your shell | Protects real repos; `sh_gen` strips it before `git commit` / `rekal` |
| `REKAL_SKIP_CHECKPOINT` | **Never** for ingest | Silently captures nothing |

Workdir paths must **not** contain `/scripts/bench`, `rekal-bench`, or `/.rekal-bench`.

---

## 4. Datasets (WS-A)

All raw JSON stays in `scripts/industry-bench/datasets/data/` (gitignored).

```bash
cd rekal-cli

# LongMemEval-S cleaned (~277 MB download)
bash scripts/industry-bench/datasets/get_longmemeval.sh s
python3 scripts/industry-bench/datasets/normalize_longmemeval.py --variant s
python3 scripts/industry-bench/datasets/verify_dataset.py longmemeval-s
# expect: conversations=500 sessions=23867 turns=246750 abstention=30

# LoCoMo (optional, for second benchmark)
bash scripts/industry-bench/datasets/get_locomo.sh
python3 scripts/industry-bench/datasets/normalize_locomo.py
bash scripts/industry-bench/datasets/get_locomo_known_bad.sh
python3 scripts/industry-bench/datasets/verify_dataset.py locomo
```

Full interchange file:

`scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl`

---

## 5. Full ingest (WS-B) — parallel recipe

**Bottleneck:** ~**5 seconds per session** (`git commit` + `rekal checkpoint` per session).  
**Full set:** 23,867 sessions.

| Workers | Approx throughput | ETA (full 500) |
|---:|---:|---:|
| 1 | ~12 sess/min | ~33 h |
| 3 | ~30 sess/min | ~11 h |
| 8 | ~80 sess/min* | ~4–5 h* |

\*8-worker rate is extrapolated; measure your first shard and adjust.

### 5.1 Pick a durable workdir

```bash
export IMB_ROOT="$HOME/imb-lme-s"    # NOT /tmp if you care about survival
mkdir -p "$IMB_ROOT"
```

### 5.2 Shard with `--offset` + `--limit`

`sh_gen/gen.py` supports sharding on the full jsonl:

```bash
FULL=scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl
SHARD_SIZE=25          # 500/25 = 20 shards
WORKERS=8              # tune to your cores

run_shard() {
  local offset=$1
  local out="$IMB_ROOT/shard-$(printf '%04d' "$offset")"
  PYTHONUNBUFFERED=1 python3 scripts/industry-bench/sh_gen/gen.py \
    --input "$FULL" \
    --offset "$offset" \
    --limit "$SHARD_SIZE" \
    --out "$out" \
    --rekal "$(command -v rekal)" \
    --verify \
    > "$out.log" 2>&1
  echo "done offset=$offset"
}

export -f run_shard
export FULL IMB_ROOT SHARD_SIZE

# Example: offsets 0,25,50,...475
seq 0 $SHARD_SIZE 475 | parallel -j "$WORKERS" run_shard {}
```

Or use the wrapper (same logic, auto-detects GNU parallel):

```bash
chmod +x scripts/industry-bench/run_ingest_shards.sh
export REKAL=./rekal
./scripts/industry-bench/run_ingest_shards.sh "$IMB_ROOT" "$WORKERS" "$SHARD_SIZE"
```

### 5.3 After ingest: index (per conversation repo)

Ingest does **not** run `--index` by default. For recall/smoke:

```bash
# example: one conversation
cd "$IMB_ROOT/shard-0000/<conversation_id>/repo"
rekal index
```

For bulk indexing, loop over `*/repo` dirs (expect background `rekal embed` — wait for lock to clear).

### 5.4 Progress check

```bash
# count conversation dirs across shards
find "$IMB_ROOT" -mindepth 2 -maxdepth 2 -type d | wc -l   # expect 500

# verify SQL spot-check (one repo)
cd "$IMB_ROOT/shard-0000/<conversation_id>/repo"
rekal query "SELECT count(*) FROM sessions"
rekal query "SELECT count(*) FROM turns"
```

Record wall time in `scripts/industry-bench/runs/notes/` when a full run completes (WS-B DoD).

---

## 6. Smoke / eval (WS-E)

### Stock recall smoke

```bash
python3 scripts/industry-bench/shim/shim.py smoke \
  --workdir "$IMB_ROOT/shard-0000" \
  --conversation-id e47becba \
  --input scripts/industry-bench/datasets/data/longmemeval-s-conversations.jsonl \
  --out scripts/industry-bench/runs/smoke/lme-s-e47becba \
  --rekal ./rekal \
  --route stock
```

### Skill-routed smoke

```bash
python3 scripts/industry-bench/shim/shim.py smoke \
  ...same args... \
  --route skill
```

Skill route uses shipped `recall-route.py` + marker knowledge suppression in core `hunt-gate.py`. Non-abstention questions fall back to stock if skill says SILENCE.

### Debug a miss with layers

```bash
cd <repo>
rekal "your question" --limit 10 --explain
```

See `runs/notes/2026-07-17-topk-misses-explain.md` for LoCoMo rank-depth examples.

---

## 7. What to do next (ordered)

1. **Finish full LME-S ingest** (500 conv) using parallel shards on your machine.
2. **Bulk `rekal index`** on ingested repos (or add `--index` to sh_gen for smoke subset only).
3. **Run smokes** at scale: 10 dev questions → expand (WS-E DoD).
4. **WS-C:** `modes/classify.py` + persona map for preference / why routes.
5. **WS-D:** calibrate gates on dev split only; commit `calibration/longmemeval-s.json` before any test run.
6. **Official scorer** wrapper for LongMemEval (not just `smoke_score.py`).
7. **Baselines** (WS-F): Mem0, full-context, naive RAG.

---

## 8. Key files

| Path | Purpose |
|---|---|
| `docs/research/industry-bench/03-workstreams.md` | Board + DoD |
| `docs/research/industry-bench/04-procedures.md` | Runbooks, honesty, manifest format |
| `scripts/industry-bench/sh_gen/gen.py` | Synthetic ingest |
| `scripts/industry-bench/shim/shim.py` | Harness (`--route stock\|skill`) |
| `scripts/industry-bench/datasets/verify_dataset.py` | Count verification |
| `cmd/rekal/cli/skill/skills/rekal/scripts/hunt-gate.py` | Core gate (marker knowledge fix) |
| `runs/smoke/` | Committed smoke manifests |
| `runs/notes/` | Run notes + explain dumps |

---

## 9. Cloud run snapshot (ephemeral — do not rely on)

At handover time the cloud agent had ~**50/500** conversations ingested in `/tmp/imb-lme-s-*` (not on your machine). Treat as timing reference only.

---

## 10. Contact / continuity

- Research PR: **#48**
- Main PR (core gate + shim): **#49**
- Update `03-workstreams.md` board in the same commit as completed work.
