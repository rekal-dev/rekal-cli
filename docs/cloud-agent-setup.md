# Cloud-agent setup — cold environment to working Rekal

For agents working in an ephemeral cloud environment (Claude Code on the web,
CI, a fresh container) where the repo is a clean clone and nothing is cached.
The full cold-start is: **build → init → sync → verify**. Follow it exactly —
it encodes the setup mistakes that cost real time:

1. **llama.cpp HEAD breaks the link.** HEAD relocated/renamed the `common`
   static lib, so `-lcommon` in `nomic_cgo.go` fails with
   `ld: cannot find -lcommon`. **Pin tag `b8157`** (what CI uses).
2. **The nomic model is a git-LFS pointer.** A fresh clone ships a 134-byte
   pointer, not the 140MB GGUF. Without `git lfs pull` the binary builds but
   nomic fails soft (`gzip: invalid header`), so recall runs **BM25 + LSA only**
   — the semantic layer is silently off and any quality testing is
   unrepresentative. Always pull the model before you judge recall quality.
3. **`GGML_NATIVE` defaults to ON, and it is `-march=native`.** ggml optimizes
   for the CPU it is *compiled* on unless told otherwise, so a binary built on
   a runner with AVX512-VBMI takes **SIGILL** loading the model on a CPU
   without it. The daemon dies, recall degrades to keyword+LSA, and the agent
   is told `SEMANTIC warming — retry with backoff` forever. Always pass
   **`-DGGML_NATIVE=OFF`** (what CI and the release workflow now use).
   A native build is safe on the machine that built it, so this only bites
   when the artifact travels — a published release, or **a `.deps/llama.cpp`
   left over from an earlier container**. The second is the common one here:
   the cache outlives the container, the next session lands on a different
   CPU model, and the same tree that worked yesterday takes SIGILL today.
   If you inherit a `.deps` you did not build, rebuild it rather than trust it.
4. **A deeply nested repo path breaks the daemon socket.** `sockaddr_un` bounds
   the whole path, so `<gitroot>/.rekal/nomic/daemon.sock` over ~103 bytes
   fails `bind: invalid argument`. Rekal falls back to a short runtime path
   automatically; if you see this on an old binary, move the checkout shallower.
5. **The installed skill goes stale.** `.claude/skills/rekal/` is a copy
   written by `rekal init`, not a symlink to source. A **released** binary
   self-heals on the next recall when `.rekal-version` lags. A **dev** rebuild
   often keeps `Version="dev"`, so the marker still matches — **re-run
   `rekal init`** (data untouched) to refresh the skill/hooks. Otherwise every
   skill test and route call
   exercises an old router.

## 1. Build (one-time)

```bash
# 0. Deps
apt-get install -y cmake build-essential git git-lfs   # Linux
git lfs install

# 1. Real nomic model (140MB) — NOT the LFS pointer
git lfs pull   # fills cmd/rekal/cli/nomic/models/*.gguf.gz

# 2. llama.cpp at the PINNED tag (HEAD will not link)
git clone --depth 1 --branch b8157 https://github.com/ggml-org/llama.cpp .deps/llama.cpp
cd .deps/llama.cpp
cmake -B build \
  -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF \
  -DLLAMA_BUILD_SERVER=OFF -DBUILD_SHARED_LIBS=OFF \
  -DCMAKE_BUILD_TYPE=Release -DGGML_NATIVE=OFF
cmake --build build --config Release -j"$(nproc)" \
  --target llama --target ggml --target common     # 'common' -> libcommon.a
cd ../..

# 3. Build the binary (CGO required — DuckDB + llama.cpp)
CGO_ENABLED=1 go build -o rekal ./cmd/rekal
export PATH="$PWD:$PATH"
rekal version
```

`CGO_ENABLED=0` cannot work: DuckDB and nomic both need CGO. If you only need
recall to *run* (not representative semantic quality), you may skip step 1 —
recall degrades to BM25 + LSA and prints `nomic: … gzip: invalid header`
warnings. Never report recall-quality numbers from that state.

## 2. Init + sync the data

```bash
rekal init          # store + hooks + skill; imports your rekal/<email> branch
rekal sync          # pull teammate ledgers (all rekal/* branches), rebuild index
rekal "<question>"                       # shipped seed digest and verdict
rekal query --session <sid>              # readable source turns
rekal --json "<question>"                # structured results for programs
```

`rekal init`/`sync` run under your configured git identity (`git config
user.email`) and push to `rekal/<email>`; set the identity you intend before
running if the environment defaults to a bot user. After a **dev** binary
rebuild, re-run `init` (safe; data untouched) so the skill/hooks refresh
(trap 3) — released binaries self-heal the skill on the next recall.

The first recall after `init`/`sync` may print `SEMANTIC warming` — the nomic
daemon is still loading; results are keyword + LSA. Re-run with backoff
(2s/4s/8s) for full-quality ranking.

If Git LFS already installed `post-commit` or `pre-push` hooks, `rekal init`
preserves them and prints a notice. Add `rekal checkpoint` to the existing
post-commit hook and `rekal push` to the existing pre-push hook, preserving
their LFS commands. Otherwise automatic capture/sharing is not enabled.

### Codex session capture

The Codex adapter discovers active and archived JSONL rollouts under
`$CODEX_HOME/sessions/` and `$CODEX_HOME/archived_sessions/`; the default root
is `~/.codex`. Both `session_meta.payload.id` and the older top-level
`session_id` are accepted. Repo matching includes subdirectories and excludes
similarly named sibling repos.

Conversation turns come from `event_msg` user/agent messages, with
`response_item` messages used for response-only transcripts. Mirrored streams
are not counted twice. A host that exposes no local JSONL rollout, including
some Work Mode sessions, cannot be captured automatically by this adapter.
An explicitly exported conversation can be checkpointed from a separate
`CODEX_HOME`; label its export provenance and preserve the original visible
message text and available timestamps.

## 3. Verify the semantic layer is actually on

```bash
rekal index                                   # rebuild; spawns background embed
rekal embed                                   # finish missing vectors; retry if another embed holds the lock
rekal query --index "SELECT model, count(*) FROM knowledge_embeddings GROUP BY model"
```

Check that the intended model has vectors, not merely that old vectors exist.
The lock file can remain after its lock is released, so its existence is not a
completion signal. If embeddings remain missing, inspect `.rekal/embed.log`;
`gzip: invalid header` means the model wasn't pulled — return to build step 1.

## Dev loop without mise

Cloud containers often lack `mise`. The task equivalents (see `mise.toml`):

```bash
gofmt -s -w .                          # mise run fmt
gofmt -l -s . && golangci-lint run --timeout=5m ./...   # mise run lint (golangci-lint v2)
go test ./...                          # mise run test
go test -tags=integration ./cmd/rekal/cli/integration_test/...   # test:integration
go test -tags=integration -race ./...  # mise run test:ci
python3 -m unittest discover -s scripts/tests -v  # installer + benchmark CLI regressions
```
