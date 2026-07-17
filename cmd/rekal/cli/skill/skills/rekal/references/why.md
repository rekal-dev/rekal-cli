# WHY — rationale, reconstructed by synthesis

The rationale for an evolved decision is distributed across sessions. Gather
the trail, then synthesize. Under-gathering invents fiction — refuse that.

If the topic is fuzzy, run **mine.md** first and feed its turns here.

1. **Seed:** 2–3 phrasings of the decision and alternatives, or a MINE gather.
2. **Gather the decision trail** across *all* matching sessions:

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
rekal query --index "SELECT session_id, turn_index, role, substr(content,1,300) FROM turns_ft \
  WHERE (role = 'human_steering' OR content LIKE '%because%' OR content LIKE '%instead of%' \
         OR content LIKE '%constraint%' OR content LIKE '%rejected%' OR content LIKE '%decided%') \
  AND (content LIKE '%<topic-term-1>%' OR content LIKE '%<topic-term-2>%') \
  ORDER BY session_id, turn_index" \
  | python3 "$ROOT/scripts/why-trail-gate.py"
```

   Exit 0 (`PASS`) → synthesize. Exit 1 (`under_gathered`) → widen terms;
   do **not** invent a rationale. Aim ~30 turns once the gate clears.

3. **Pull code on demand:** `git show <commit-sha>` for claims that reference code.
4. **Synthesize the arc** with pointers: original design → alternatives rejected →
   constraint → final rationale, each `(session <id> turn <n>, commit <sha>)`.
5. **If the trail is absent**, say so — a gap beats a fabricated why.
