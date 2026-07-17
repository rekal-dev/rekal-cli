# Interchange format — `conversations.jsonl`

Every dataset normalizer (WS-A) emits this; every consumer (sh_gen, shim,
scoring) reads only this. One JSON object per line, one line per
benchmark conversation ("haystack").

```json
{
  "conversation_id": "toy-001",
  "sessions": [
    {
      "session_id": "s1",
      "date": "2023-05-20T10:00:00Z",
      "turns": [
        {"role": "user", "text": "..."},
        {"role": "assistant", "text": "..."}
      ]
    }
  ],
  "questions": [
    {
      "question_id": "q1",
      "category": "single-hop",
      "question": "...",
      "answer": "...",
      "evidence_session_ids": ["s1"]
    }
  ]
}
```

Rules:

- `sessions` are ordered chronologically; `date` is RFC3339 UTC and becomes
  the synthetic commit's author/committer date and each turn's timestamp
  base (turns within a session get `date + 60s * index` so ordering is
  stable and monotone).
- `role` is exactly `user` or `assistant`. Chat benchmarks have no tool
  calls, steering, or summaries — normalizers must not invent them.
- `category` uses the benchmark's own taxonomy verbatim (e.g. LongMemEval:
  `single-hop`, `multi-hop`, `temporal`, `knowledge-update`, `abstention`);
  scoring maps per-benchmark categories, not the normalizer.
- `evidence_session_ids` may be empty when the benchmark doesn't publish
  evidence pointers; it must be `[]` (present) for abstention-style
  questions whose correct answer is "unknown".
- Additional benchmark-specific fields go under an optional `"extra": {}`
  object at either level; consumers must ignore unknown keys inside
  `extra` and must not depend on them.
