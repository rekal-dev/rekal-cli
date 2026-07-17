# WS-A/B/E: LoCoMo real ingest + known-bad list

Date: 2026-07-17 · Machine: linux/amd64 cloud agent

## Ingest

`sh_gen` on LoCoMo `conv-26` (19 sessions, 419 turns, 199 questions in
haystack) — verify green.

## Known-bad labels

Fetched `dial481/locomo-audit` `errors.json` (156 issues). Emitted
`datasets/locomo-known-bad.jsonl` with the **99 score-corrupting** rows
(excludes 57 `WRONG_CITATION`). Mapped `locomo_<i>_qa<n>` → our
`conversation_id`/`question_id` (`q<n>`). Getter:
`datasets/get_locomo_known_bad.sh`.

## WS-E smoke (`runs/smoke/locomo-conv-26-q10/`)

First 10 questions, stock recall + extractive answer:

- evidence_in_top rate: **0.90** (9/10)
- retrieved_context_tokens mean: 209.9
- answer_path_tokens mean: 239.1

Miss: q8 (multi-hop). Token columns non-zero.
