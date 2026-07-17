# HUNT — pointed, gated episodic recall

Use after `recall-route.py` printed `INJECT`. Not for analytical mining
(mine.md) or present-tense code (grep).

## 1. Pipeline (required)

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
rekal "JWT expiry" | python3 "$ROOT/scripts/recall-route.py"
# optional filters:
rekal --file src/auth/ "token refresh" | python3 "$ROOT/scripts/recall-route.py"
rekal --commit <sha> | python3 "$ROOT/scripts/recall-route.py"
rekal -n 5 --explain "error handling" | python3 "$ROOT/scripts/recall-route.py"
```

| Route stdout | Action |
|---|---|
| `INJECT …` | Confident episode — drill below. Knowledge docs may also exist; do not let them block. |
| `KNOWLEDGE …` | Episode below bars; from `knowledge[0]` Read `path` at `lines` (`anchor`). **Do not** drill weak sessions. |
| `SILENCE …` | No confident episode and no knowledge. Say so. Do not pad with near-misses. |

Episode bars live only in `scripts/hunt-gate.py` (invoked by recall-route).
Confident episode outranks a non-empty knowledge block. No gate output → SILENCE.

## 2. Drill, cheapest first

Never `--full` by default:

```bash
rekal query --session <id> --role summary
rekal query --session <id> --offset <snippet_turn_index - 2> --limit 5
rekal query --session <id> --role human
rekal query --session <id> --role human_steering
rekal query --session <id> --full   # last resort
```

Fields: `session_id`, `score`, `snippet`, `snippet_turn_index`,
`snippet_role` (`human_steering` = high intent), `summary_turn_index`,
`children`, `origin` (cross-repo — not this repo's conventions).

Cite session/turn/commit with every claim.
