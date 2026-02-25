# rekal (root) — recall

**Role:** The main search. Root invocation is recall — no `search` subcommand. When `rekal` is called with a query and no recognised subcommand, it runs recall against the index DB. **Primary consumer is the agent.** Output is for machine consumption (tool result); we do not intend for users to read it directly.

**Invocation:** Root only — `rekal [filters...] [query]`. Subcommands (init, clean, checkpoint, push, etc.) take precedence when present.

**Soul:** Simplify. One search path (fulltext + semantic combined inside the implementation; the user does not choose). Filters are structured, one flag per dimension, like `grep` / `git log` / `gh search`: combine with AND. Minimal surface: no `search` subcommand, few flags.

**Reference: openclaw** — We align with openclaw memory search (ref-projects/openclaw): one search command, no "fulltext vs semantic" mode exposed; implementation uses semantic/vector + fulltext as appropriate. Openclaw CLI: `openclaw memory search "<query>"` with `--agent`, `--max-results`, `--min-score`, `--json`. Rekal will do similar semantic/vector + fulltext under the hood; this spec focuses on the interface (filters + result shaping + output format).

---

## Preconditions

See [preconditions.md](../preconditions.md): git repo, init done. Index DB existence is part of preconditions (init creates it; if missing or empty we run index). No separate "run rekal index" in recall.

---

## What recall does

1. **Run shared preconditions** — Git root, init done, index exists (run index if missing/empty).
2. **Resolve query** — Single positional argument: keywords or short phrase for semantic/fulltext search. (The agent translates user questions into this form; that is documented in the skill.)
3. **Apply filters** — All given filter flags are ANDed. See table below.
4. **Run recall** — Read-only against `.rekal/index.db`. Never modifies data DB or index DB.
5. **Output** — Structured for machine consumption (e.g. JSON). Intended as tool result for the agent; not designed for human reading. Compact JSON is token-efficient compared to prose (no "Here are the results:" etc.); Claude receives this as Bash tool stdout and parses it.

---

## Filters (structured, grep / gh style)

One flag per dimension. Multiple filters = AND. Same idea as `git log --author=... --since=...` or `gh search issues --author ... --label ...`.

| Flag | Meaning |
|------|--------|
| `--file <regex>` | Sessions that touched a file whose path matches the regex. Paths are git-root-relative. See path semantics below. |
| `--commit <sha>` | Sessions linked to this git commit |
| `--checkpoint <ref>` | Query as of this historical snapshot (git ref on rekal orphan branch: SHA, HEAD~n, @{date}). See [push.md](push.md#checkpoint-id--orphan-branch-commit-sha). |
| `--author <email>` | Sessions by this committer / author (e.g. session facet `user_email`) |
| `--actor <human\|agent>` | Actor type (default: both; use `human` or `agent` to restrict) |

**`--file` path semantics:** The value is a **regex** applied to stored file paths (all paths are **git-root-relative**). E.g. `--file '^src/auth/'` = folder; `--file 'src/auth/middleware\.go'` = exact file (escape `.`); `--file '.*_test\.go'` = all test files. No separate `-r` flag — regex only.

**Result shaping:**

| Flag | Meaning |
|------|--------|
| `-n`, `--limit <n>` | Cap at N results. **Default: no limit** — return all matching results when omitted. Use `-n` only when the agent wants to bound result size (e.g. token budget). |

Output format is structured (e.g. JSON) by default — for tool result, not for human reading. No separate human-readable mode.

**How the agent knows what to set:** The Rekal skill (`.claude/rekal.md`, installed by `rekal init`) documents when to pass `-n`. Default is all; limit is opt-in.

Optional later: `--min-score <n>`. Omit from first version.

---

## Read-only

Recall never modifies `.rekal/data.db` or `.rekal/index.db`.

---

## Examples

Query is keywords or short phrase (semantic/fulltext). Translation from user questions is a skill concern.

```bash
rekal "JWT"
rekal "JWT expiry"
rekal --file src/auth/middleware.go "JWT"
rekal --file '^src/auth/' "JWT"
rekal --commit a3f9b12 "JWT"
rekal --author alice@example.com "refactor"
rekal --file src/auth.go --actor human "auth"
rekal "JWT" --checkpoint HEAD~5
rekal "JWT" -n 10
```
