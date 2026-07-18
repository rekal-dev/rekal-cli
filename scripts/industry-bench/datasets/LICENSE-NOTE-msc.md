# MSC (Multi-Session Chat) — local-use license note

Source mirror: https://huggingface.co/datasets/gonced8/multi-session_chat
(cleaned ParlAI MSC; original paper arXiv 2107.07567 / Facebook Research).

We download at run time via `get_msc.sh` and do **not** commit raw JSONL.
Normalized conversations are also gitignored. MSC has no official QA
labels in this release — `normalize_msc.py` synthesizes persona-fact
recall questions for adapter regression / full-corpus ingest only.
