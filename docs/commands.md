# Command reference

Every command, every flag, and which of the four substrates each one reaches.
Per-command behavioural specs live in [`spec/command/`](spec/command/); this
page is the flat reference you scan when you already know what you want.

`rekal --help` is always authoritative. If this page and `--help` disagree,
`--help` is right and this page is a bug.

---

## The four routes

Rekal answers from four different places. Picking the wrong one is the most
expensive mistake available, because three of them will still return something.

| Substrate | Question it answers | Reach it with |
|---|---|---|
| **Tree** | What does the code do *now*? | `grep` / `rg` — not Rekal |
| **Knowledge** | What does the prose say *now*, at HEAD? | `rekal "<q>"` → `KNOWLEDGE` line |
| **Ledger** | What did we decide, try, reject — and why? | `rekal "<q>"`, `rekal find`, `rekal query` |
| **Map** | How is the repo structured? | `rekal query --sql` over `files_index` |

Only the **ledger** holds intent. The tree holds present-tense fact, and it is
always cheaper to grep than to recall. Rekal's own skill (`SKILL.md`, shipped
inside the binary) routes this automatically for agents.

A session's **commit message** — subject and body — is folded into its facet
document, so wording that only ever appeared in a commit still points at the
conversation that produced it. The body is the point: a subject is a label,
the body is where the reasoning is written down. Derived locally from your own
clone at index time — never stored in `data.db`, never on the wire, because
every clone already has it.

---

## Retrieval

### `rekal "<query>"` — recall

Recall is the **default action**, not a subcommand — the usage line is
`rekal [filters...] [query]`. There is no `rekal recall`. It returns the **seed
digest** (a compact ranked verdict), not raw rows.

| Flag | Short | Default | Meaning |
|---|---|---|---|
| `--limit` | `-n` | 20 | Results per framing. `0` = none, negative rejected. Soft: the RRF-fused union across framings may exceed it |
| `--json` | `-j` | off | Raw structured JSON instead of the digest |
| `--explain` | `-e` | off | Per-layer scores + related-session joins |
| `--file` | `-p` | — | Filter by file path (regex) |
| `--commit` | `-c` | — | Filter by git commit SHA |
| `--author` | `-a` | — | Filter by author email |
| `--actor` | `-A` | — | Filter by actor type (`human`\|`agent`) |

**Exit code 1 means SILENCE** — Rekal found nothing it can stand behind. That
is a real answer, not an error. Treat it as "the ledger does not know", and go
look at the tree.

Digest header: `INJECT top=0.71 gap=0.21 15 seeds`

- `top` — highest **absolute** confidence in the result set. Corpus-independent,
  unlike the max-normalized `score` used for ranking.
- `gap` — `top` minus the runner-up's confidence. Only load-bearing in the
  middle band: `top ≥ 0.25` passes on its own; `0.20–0.25` needs `gap ≥ 0.02`;
  below `0.20` is silence. One clearly-best answer is signal, several bunched
  together is noise.
- `[reached N× drilled M×· "query"]` — usage history. `reached` is how often
  search surfaced this memory (the engine quoting itself — high on any small
  store); `drilled` is how often an agent opened it, and is the half that feeds
  `reach_boost`. The query is the one that surfaced it most often. Ranking hint
  only; deliberately excluded from the silence gate, because popular is not the
  same as relevant.

`SEMANTIC warming` means the embedding model is still loading and you got
keyword+LSA only. Re-run with backoff for full quality.

### `rekal find "<term>" [role]` — enumeration

Complete, time-ordered sweep over every turn. No ranking, no silence gate, no
cutoff. Use it when you need *all* occurrences and recall's top-N would lie by
omission — counting, auditing, "did we ever mention X". `role` is optional:
`human`, `assistant`, `human_steering`, `summary`.

### `rekal query` — drill-down and SQL

Two mutually exclusive modes. `--sql`, a bare positional statement, and
`--session` cannot be combined.

| Flag | Short | Default | Meaning |
|---|---|---|---|
| `--session` | `-s` | — | Drill into one session by short handle (`s3`) or ULID |
| `--sql` | `-q` | — | SQL SELECT to run (a bare positional is accepted as shorthand) |
| `--index` | `-i` | off | Run SQL against `index.db` instead of `data.db` |
| `--full` | `-F` | off | Include tool calls and files in session output |
| `--offset` | `-o` | 0 | Skip first N turns (with `--session`) |
| `--limit` | `-n` | 0 | Max turns, `0` = no limit (with `--session`) |
| `--role` | `-r` | — | Filter turns by role (with `--session`) |
| `--json` | `-j` | off | JSON instead of text/TSV |

SQL is **read-only** — one SELECT per call, on a connection opened in DuckDB's
read-only mode. A non-SELECT statement is rejected, and so is a second statement
after a `;` (a semicolon inside a literal or comment is data and stays legal).
The append-only ledger is protected by the handle, not by the parse. The full
queryable schema is in `rekal query --help`.

```bash
rekal query -s s3                     # readable turns
rekal query -s s3 -r summary          # just the compaction summaries
rekal query -s s3 -o 200 -n 50        # a window deep into a long session
rekal query -i -q "SELECT count(*) FROM turns_ft"
```

---

## Lifecycle

| Command | What it does |
|---|---|
| `rekal init` | Bootstrap: store, hooks, orphan branch, skill, one CLAUDE.md line. Re-running refreshes managed assets. |
| `rekal checkpoint` | Capture the session against the current commit. Runs automatically via post-commit; no-ops during a rebase. |
| `rekal index` | Rebuild `index.db` from `data.db`. Structural only — vectors fill via background `rekal embed`. |
| `rekal embed` | Fill missing semantic vectors in budgeted bites. Resumable, safe to interrupt. |
| `rekal log` | Recent checkpoints. `--limit` / `-n`, default 20. |
| `rekal clean` | Remove Rekal setup completely, no residue. Asks first: prompts at a terminal, refuses without `-y` / `--yes` anywhere else. |

`rekal index` also carries the cross-repo import preference, which persists:
`--include-all`, `--include <path>`, `--no-local`. Imported sessions are
**index-only** and can never be pushed.

---

## Sharing

| Command | Flag | Short | What it does |
|---|---|---|---|
| `rekal push` | | | Export merged checkpoints to your `rekal/<email>` branch |
| | `--remote` | | Remote to publish to (default `origin`; the hook forwards git's) |
| | `--strict` | | Exit non-zero when publication fails (default: warn, exit 0) |
| | `--progress` | | Print timed stages |
| | `--timeout` | | Deadline per git network call (default 2m) |
| | `--rebuild` | | Re-encode the branch's wire data from `data.db`. Refuses unless the branch's current body is already contained in what it would write. (`--re-export` is a deprecated alias) |
| `rekal sync` | | | Fetch and import your teammates' branches |
| | `--self` | | Fetch only your own branch — across your own machines |

**Only merged work is shared.** A checkpoint reaches the wire when its commit is
an ancestor of the default branch, or its branch landed as a patch-equivalent
squash. Unmerged work stays local and is re-checked on every push, so it ships
automatically once the branch merges — and never if it is abandoned.

**There is no force flag, and nothing in rekal force-pushes.** The wire format is append-only — no byte is modified
after it is written — and that is a structural guarantee, not a policy with an
override. `rekal push` appends checkpoints and can do nothing else.

Discarding what a branch holds is a git operation, not a memory operation. If
you truly mean it, say so in git:

```bash
git push --force origin rekal/<email>
```

That asymmetry is deliberate. A flag inside `rekal push` would make overwriting
part of the memory protocol; leaving it in git keeps it what it is — an
operation on a ref, performed by a person who went looking for it.

**When a push is rejected**, the branch genuinely diverged: another machine
under the same identity pushed checkpoints this one has never seen. Push from
*that* machine — it appends to the branch, and this one fast-forwards onto it
next time. `sync` imports other branches into the index and never into
`data.db`, so no local re-export can reproduce another machine's checkpoints.

`--rebuild` is the one path that rewrites the body, and it is bounded: it
refuses unless the branch's current body is already contained in what it would
write, so it can only ever produce a superset. It repairs *derived* bytes from
`data.db`, which is the source of truth — it never edits the ledger.

---

## Conventions

**One letter, one meaning.** A shorthand means the same thing in every command
that has it, which is why `--self` has none: `-s` belongs to `--session`, and
`-f` to `--force`. A test pins the whole table (`shorthand_test.go`) — a
shorthand is a contract the moment it ships, and silently moving one breaks
callers in the worst way, by still running with a different meaning.

**Text by default, JSON on request.** Every retrieval command returns compact
agent-readable text; `--json` / `-j` gives the raw structure. The default is the
useful form, not the machine form.

**Exit codes.** `0` success (including a recall that found nothing to say but
had knowledge hits), `1` SILENCE or error.

**Two databases.** `data.db` is the append-only ledger and the source of truth —
it only ever gains rows. `index.db` is derived and disposable; anything
corrective (duplicate collapse, role reclassification) happens there, never in
the ledger. Delete `index.db` and `rekal index` rebuilds it.

**Store location.** `.rekal/` lives in the repository's main worktree. Every
linked worktree resolves to that one shared store.

## See also

- [`usage.md`](usage.md) — the operational guide
- [`configuration.md`](configuration.md) — ranking weights, embedding backends
- [`spec/command/`](spec/command/) — per-command behavioural specs
- [`design/skill-router.md`](design/skill-router.md) — how the skill routes
