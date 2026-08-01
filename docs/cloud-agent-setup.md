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
rekal "<question>" | python3 .claude/skills/rekal/scripts/route.py   # recall, gated
rekal query --session <sid> | python3 .claude/skills/rekal/scripts/view.py  # drill
```

`rekal init`/`sync` run under your configured git identity (`git config
user.email`) and push to `rekal/<email>`; set the identity you intend before
running if the environment defaults to a bot user. After a **dev** binary
rebuild, re-run `init` (safe; data untouched) so the skill/hooks refresh
(trap 3) — released binaries self-heal the skill on the next recall.

The first recall after `init`/`sync` may print `SEMANTIC warming` — the nomic
daemon is still loading; results are keyword + LSA. Re-run with backoff
(2s/4s/8s) for full-quality ranking.

## 3. Verify the semantic layer is actually on

```bash
rekal index                                   # rebuild; spawns background embed
until [ ! -f .rekal/embed.lock ]; do sleep 3; done   # wait for embeddings
rekal query --index "SELECT COUNT(*) FROM knowledge_embeddings"   # > 0 = nomic on
```

If the count is 0 or you see `gzip: invalid header`, the model wasn't pulled —
go back to build step 1.

## Dev loop without mise

Cloud containers often lack `mise`. The task equivalents (see `mise.toml`):

```bash
gofmt -s -w .                          # mise run fmt
gofmt -l -s . && golangci-lint run --timeout=5m ./...   # mise run lint (golangci-lint v2)
go test ./...                          # mise run test
go test -tags=integration ./cmd/rekal/cli/integration_test/...   # test:integration
go test -tags=integration -race ./...  # mise run test:ci
```
