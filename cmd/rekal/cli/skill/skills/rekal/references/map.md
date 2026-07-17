# MAP — breadth from structure

Map file: `.rekal/map.md` (local, gitignored). Structured markdown agents can
grep — not a diagram.

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
```

## Freshness first

```bash
bash "$ROOT/scripts/map-fresh.sh"
```

| Result | Action |
|---|---|
| FRESH | Answer from the map. |
| STALE | Diff listed paths; rewrite only touched subsystem sections; then watermark. |
| MISSING | Build (below), then watermark. |

```bash
bash "$ROOT/scripts/map-write-watermark.sh"   # after any edit / stub create
```

Watermark line 1: `<!-- rekal-map <branch> <HEAD-sha> -->` — written only by
the script. If diff > ~50 files or boundaries moved → full rebuild.

## Build (when MISSING or full rebuild)

1. Skeleton: top two dir levels, manifests, CI — no file contents yet.
2. README / docs / CLAUDE.md = claims to verify, not truth.
3. Cut 5–12 subsystems by responsibility.
4. Per subsystem: 1–3 load-bearing files; purpose; what breaks if deleted.
5. Trace edges that cross subsystems.
6. Optional: `rekal --explain "<subsystem>"` for memory hooks.
7. Emit ≤12 lines/subsystem, ≤150 lines total; run `map-write-watermark.sh`.

```markdown
<!-- rekal-map <branch> <HEAD-sha> -->
# Map — <repo name>

## System in one paragraph
…

## Subsystems
### <name> — `<primary path>`
- purpose: …
- key files: `<a>`, `<b>`
- depends on: <subsystem> (<what crosses>)
- invariant: …   [optional]

## Flows
- <flow>: A → B → C (<what moves>)

## Pointers   [optional]
- <topic>: session <id> (<why>)
```
