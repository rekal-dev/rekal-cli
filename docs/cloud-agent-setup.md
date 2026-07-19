# Cloud-agent setup — building and running Rekal from scratch

For agents working in an ephemeral cloud environment (Claude Code on the web,
CI, a fresh container) where the repo is a clean clone and nothing is cached.
Follow this exactly — it encodes the two setup mistakes that cost real time:

1. **llama.cpp HEAD breaks the link.** HEAD relocated/renamed the `common`
   static lib, so `-lcommon` in `nomic_cgo.go` fails with
   `ld: cannot find -lcommon`. **Pin tag `b8157`** (what CI uses).
2. **The nomic model is a git-LFS pointer.** A fresh clone ships a 134-byte
   pointer, not the 140MB GGUF. Without `git lfs pull` the binary builds but
   nomic fails soft (`gzip: invalid header`), so recall runs **BM25 + LSA only**
   — the semantic layer is silently off and any quality testing is
   unrepresentative. Always pull the model before you judge recall quality.

## One-time build

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
  -DCMAKE_BUILD_TYPE=Release
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

## Use Rekal in this repo

```bash
rekal init          # store + hooks + skill; imports your rekal/<email> branch
rekal sync          # pull teammate ledgers (all rekal/* branches), rebuild index
rekal "<question>"  # recall; pipe through the skill route for gating:
rekal "<q>" | python3 .claude/skills/rekal/scripts/route.py
```

`rekal init`/`sync` run under your configured git identity (`git config
user.email`) and push to `rekal/<email>`; set the identity you intend before
running if the environment defaults to a bot user.

## Verify the semantic layer is actually on

```bash
rekal index                                   # rebuild; spawns background embed
until [ ! -f .rekal/embed.lock ]; do sleep 3; done   # wait for embeddings
rekal query --index "SELECT COUNT(*) FROM knowledge_embeddings"   # > 0 = nomic on
```

If the count is 0 or you see `gzip: invalid header`, the model wasn't pulled —
go back to step 1.
