#!/usr/bin/env bash
# WS-A: fetch Multi-Session Chat (MSC) from the cleaned ParlAI mirror
# gonced8/multi-session_chat on Hugging Face.
#
# Original paper: Xu, Szlam, Weston — "Beyond Goldfish Memory" (arXiv 2107.07567).
# We download at run time; raw JSONL stays under datasets/data/ (gitignored).
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)/data"
mkdir -p "$DIR"
BASE="https://huggingface.co/datasets/gonced8/multi-session_chat/resolve/main"

for split in train valid test; do
  out="$DIR/msc-${split}.jsonl"
  echo "fetching msc ${split} ..."
  curl -fsSL "$BASE/${split}.jsonl" -o "$out"
  bytes=$(wc -c < "$out")
  lines=$(wc -l < "$out")
  echo "downloaded $out ($bytes bytes, $lines lines)"
done

NOTE_PATH="$(cd "$(dirname "$0")" && pwd)/LICENSE-NOTE-msc.md"
cat > "$NOTE_PATH" <<'NOTE'
# MSC (Multi-Session Chat) — local-use license note

Source mirror: https://huggingface.co/datasets/gonced8/multi-session_chat
(cleaned ParlAI MSC; original paper arXiv 2107.07567 / Facebook Research).

We download at run time via `get_msc.sh` and do **not** commit raw JSONL.
Normalized conversations are also gitignored. MSC has no official QA
labels in this release — `normalize_msc.py` synthesizes persona-fact
recall questions for adapter regression / full-corpus ingest only.
NOTE
echo "license note: $NOTE_PATH"
