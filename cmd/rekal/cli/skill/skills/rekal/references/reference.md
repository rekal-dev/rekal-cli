# Reference — flags, PATH, SQL, schema

## PATH

If `rekal` is missing: `export PATH="$HOME/.local/bin:$PATH"`.

## Skill root

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
```

Scripts under `$ROOT/scripts/` compress every skill-facing rekal. Prefer
`python3` / `bash` so mode bits never matter. **Never read raw JSON** — pipe
recall through `route.py`, query/SQL/session through `view.py`, term sweeps
through `find.py`, multi-framing widening through `seek.py`, relative dates
through `when.py`. `rekal init` also installs `rekal-route` / `rekal-view` /
`rekal-find` / `rekal-seek` / `rekal-when` into `~/.local/bin` — the same
scripts without the `$ROOT` boilerplate (`rekal clean` removes them).

## Root recall flags

| Flag | Description |
|------|-------------|
| `--file <regex>` | Filter by file path (regex, git-root-relative) |
| `--commit <sha>` | Filter by git commit SHA |
| `--author <email>` | Filter by author email |
| `--actor <human\|agent>` | Filter by actor type |
| `-n`, `--limit <n>` | Max results (default 20; `0` = empty set; negative rejected) |
| `--explain` | Adds `layers` + `related` (file-sharing sessions) |

## Cross-repo local import (index-only, never pushed)

```bash
rekal index --include-all
rekal index --include /path/to/repo
rekal index --no-local
```

Hits carry `origin`. Suggest widening when a search misses but the problem
smells solved elsewhere — don't widen unprompted.

## Raw SQL

```bash
rekal query "SELECT id, user_email, branch FROM sessions ORDER BY captured_at DESC LIMIT 5" \
  | python3 "$ROOT/scripts/view.py"
rekal query --index "SELECT * FROM file_cooccurrence WHERE file_a LIKE '%auth%' ORDER BY count DESC" \
  | python3 "$ROOT/scripts/view.py"
```

`rekal query --help` documents both DB schemas. `tool_calls.path` is the most
complete "files this session touched" source.

- Pipe SQL with `2>&1` so engine errors flow through `view.py` — it forwards
  them verbatim; an error is not an empty set.
- `turns.ts` is a **TIMESTAMP**: `ts LIKE '2023-05%'` throws a Binder error
  (wrongful absence on temporal questions). Use
  `ts BETWEEN TIMESTAMP '2023-05-01' AND TIMESTAMP '2023-06-01'` or
  `CAST(ts AS VARCHAR) LIKE '2023-05%'`.

## Semantic embeddings

Structural index finishes first; deep vectors continue via `rekal embed`
(background after `index`/`sync`). Keyword/LSA/knowledge-FTS work immediately.

## Gate scripts (quick map)

| Script | Role |
|---|---|
| `route.py` | Recall → recommendation digest (INJECT / KNOWLEDGE / SILENCE). Super-low floors; agent judges `conf=` / `path=score` |
| `view.py` | Query/SQL/session → raw turns or TSV rows (no JSON chrome); forwards engine errors verbatim |
| `find.py "<term>" [role]` | Every ledger mention of a term, time order, complete — enumeration without hand-SQL |
| `seek.py "<f1>" "<f2>" …` | Multi-framing recall, RRF-fused into one route.py digest — widen a weak single-phrasing recall |
| `when.py <anchor> "<phrase>"` | Relative phrase → absolute date (or honest window); pure calendar, no store |
| `map.sh fresh` / `map.sh watermark` | Map watermark vs HEAD / write-refresh (+ stub) |
| `wiki-gate.sh` | Refuse wiki on default branch |

```mermaid
flowchart LR
  r["rekal recall"] --> rr["route.py"]
  q["rekal query"] --> v["view.py"]
  rr -->|INJECT ± KNOWLEDGE| i["view.py drill + optional Read HEAD"]
  rr -->|KNOWLEDGE| k["Read HEAD"]
  rr -->|SILENCE| s["stop"]
  v --> a["agent reads compact text"]
```
