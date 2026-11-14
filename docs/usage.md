# Using Rekal

> This page describes the current main branch. Native memory integration,
> literal search pagination and `query --ref` are unreleased changes; the
> v1.0.2 installation does not include them. See the [changelog](../CHANGELOG.md) for release status.

This is the operational guide: how the two databases fit together, what
travels over git and what stays local, how agents query, and how the skill
routes. For the marketing overview and quick start, see the
[README](../README.md); for tuning, see [configuration.md](configuration.md).

## Setup and teardown

```bash
cd your-project
rekal init
```

`rekal init` creates the following on your system:

- `.rekal/` directory containing `data.db` (shared truth) and `index.db` (local
  search index)
- A `post-commit` and `pre-push` git hook (marked `# managed by rekal`)
- The Claude Code skill under `.claude/skills/rekal/` (see
  [the agent skill](#the-agent-skill))
- One marker-tagged sentence in `CLAUDE.md` pointing agents at the skill
  (created if missing; your own content is never touched)
- For each **other** AI agent detected on your machine, one marker-tagged line
  in the file it reads: `AGENTS.md` (Codex / Cursor / OpenCode), `GEMINI.md`
  (Gemini), `.github/copilot-instructions.md` (Copilot), or
  `.kiro/steering/rekal.md` (Kiro). Detection is per
  machine — a teammate who clones the repo and uses a different agent re-runs
  `rekal init` to add theirs. A file Rekal **newly creates** is gitignored
  (it's machine-specific, so it stays local); a file you already track keeps its
  tracked status and just gains the one line.
- An orphan branch `rekal/<your-email>` for transport
- Appends `.rekal/` to your `.gitignore`

Skill files also self-heal: the next recall after a binary upgrade compares the
pinned `.rekal-version` in `.claude/skills/` to the running binary and
re-installs the skill when it lags (hooks and instruction files are not
touched). Running `rekal init` again in an already-initialized repo does
**not** rebuild your store — it refreshes the skill, hooks, CLAUDE.md marker,
and detected-agent rules, and leaves your data untouched. A full reinitialize
still requires `rekal clean` first.

```bash
rekal clean
```

`rekal clean` removes everything `init` created:

- Deletes the `.rekal/` directory and all its contents
- Removes the git hooks (only the ones marked `# managed by rekal`)
- Removes the installed skill (`.claude/skills/rekal/` plus any legacy
  `rekal-*` companion dirs), pruning `.claude/skills/` and `.claude/` only if
  they are left empty — your own `.claude` content is never touched
- Removes the marker-tagged `CLAUDE.md` sentence (deleting the file only if
  nothing else remains)

No residue. If you want to start over, run `clean` then `init`.

```bash
rekal version
```

When a newer release is available, the CLI prints an update notice after each
command.

## Two databases

Rekal keeps two local DuckDB databases in `.rekal/`. The split is deliberate —
thin on the wire, rich on the machine.

- **`data.db`** — the shared truth. Append-only. Sessions, turns, tool calls,
  checkpoints, files touched — every branch, merged or not. This is the only
  source `rekal push` encodes from (filtered to merged work — see below), and
  what `rekal query` reads.
- **`index.db`** — local intelligence. Full-text indexes, vector embeddings,
  file co-occurrence graphs, knowledge chunks. Never synced. Rebuilt anytime
  with `rekal index`. This is what powers `rekal "<query>"` search.

## Orphan branches and what gets shared

Rekal data lives on git orphan branches named `rekal/<email>`. These branches
have no common ancestor with your code branches — they never appear in your
project history, never affect merges, and never clutter your working tree.
Standard `git push`/`git fetch` move the data.

Your local databases keep **every** branch — full fidelity, nothing gated. The
wire is different: `rekal push` shares a session only when its code **landed on
the default branch**, detected two ways, both exact:

- its commit is an ancestor of `main` (merge-commit and rebase workflows), or
- its branch's changes landed as a **squash merge** (patch-equivalence
  detection — no heuristics)

Unmerged work simply waits: it stays local, is re-checked on every push, and
ships automatically the moment its branch merges. Abandoned branches never
qualify, so a dead-end spike never reaches your teammates. Commit everything for
yourself; share only what merged.

## Worktrees

Linked git worktrees (`git worktree add`) share **one** `.rekal/` store — the
one in the main checkout. Init once in the main repo; every worktree then reads
and writes the same data, index, and config, so there's no per-worktree
`rekal sync` or reindex. Checkpoints still record the branch and commit of
whichever worktree you committed in. A repo that never uses worktrees is
unaffected — the store is just its own `.rekal/`.

## How your agent uses it

Search, read the returned source, then follow a continuation when needed.

| Agent does | Rekal returns |
| --- | --- |
| `rekal "auth middleware"` | Ranked sessions, memory notes, current prose pointers, plus literal matching conversation turns |
| `rekal query --ref <reference>` | Bounded original conversation/passage text or a memory revision |
| `rekal --cursor <cursor>` | Next page of literal conversation matches |
| `rekal query --cursor <cursor>` | Next page of source text |
| Add `--json` | Structured output instead of readable text |
| Add search `--file`, `--commit`, `--author`, `--actor` | Narrow conversation search; memory filter rules are documented below |

Literal totals count indexed turns containing the exact case-insensitive query,
not every related fact, mention occurrence or synonym. They exclude memory notes,
current files and unsynced history. Follow all pages for an exhaustive literal
sweep, then judge and deduplicate the evidence. A changed matching set invalidates
its cursor. Filter-only searches omit the literal block.

Returned references use permanent source IDs. Short `sN` aliases can move when
the index changes. Repository memory revisions remain historical sources;
local-memory snapshots expire when edited, removed or disabled. Current prose
paths should be read from the working repository; a historical note is not proof
of current code behavior.

`--limit` is a per-framing soft bound for ranked sessions and a literal page
size capped at 100. `--limit 0` returns no literal hits/continuation but still
reports the total. Native memory returns at most five keyword matches (also
bounded by an explicit limit); author, path and commit filters apply, while an
actor filter excludes memory. Knowledge pointers keep their existing independent
filter behavior. See [recall](spec/command/recall.md) and
[source reads](spec/command/query.md) for exact output contracts.

Advanced tools remain available: `find` for an unpaged chronological literal
sweep with a role filter; `query --session` for session metadata, tool calls,
role filters and explicit turn offsets; SQL for analysis; `evidence` for
multiple anchors and adjustable windows. They are optional, not extra steps
required before reading a search result.

## Native memory

Claude project memory is captured by default at checkpoints and shared through
ordinary push/sync. Explicit personal sources use `scope: local` and remain
only in the disposable index. `"memory": false` stops new capture/local refresh,
without erasing repository history. Configuration, observation timestamps,
deletion behavior and upgrade constraints are in
[native memory](design/native-memory.md) and [configuration](configuration.md).

## The agent skill

`rekal init` installs one Claude Code skill under `.claude/skills/rekal/`.
It teaches search → read → continue, source attribution and evidence checks.
The agent decides whether the result answers the question. Specialist reasoning
references, SQL, map and wiki helpers remain available on demand; there is no
mandatory substrate classifier or answer-type routing gate.

Skills are versioned with the binary. After upgrading, run `rekal init` to
refresh the installed skill without changing captured history. Older router
designs remain documented as historical designs, not the current default flow.

### Starting from Claude Code — the plugin

If you'd rather not start at a shell, this repository is also a plugin
marketplace:

```
/plugin marketplace add rekal-dev/rekal-cli
/plugin install rekal@rekal-dev
```

The plugin is **setup only** — two commands:

| Command | Scope |
|---|---|
| `/rekal:install` | once per machine — installs the binary |
| `/rekal:init` | once per repository — runs `rekal init` |

Both confirm before touching anything, and both are model-invoked too: a `rekal`
command reporting `command not found` routes to the first, `not initialized` to
the second. The plugin also ships the installer itself on `PATH`
(`bin/rekal-install`), so setup runs code that came with the plugin rather than a
live `curl | bash`.

The recall skill above is not in the plugin; `rekal init` installs it, versioned
with the binary whose commands it describes. Shipping it in both places would put
two copies in your context and let the plugin's copy drift ahead of your
installed binary. Detail:
[`design/plugin-distribution.md`](design/plugin-distribution.md).

## Cross-repo recall (optional)

Your agent's memory can span your whole machine, not just this repo:

```bash
rekal index --include-all            # recall every local agent session (all agents, repos + shell)
rekal index --include /path/to/repo  # just that repo
rekal index --no-local               # back to this repo only
```

Imported sessions live in the **index only** — never in `data.db`, which is the
only thing `push` reads — so they are structurally impossible to share. Results
are labeled with their origin (`repo:/path`, `shell:/path`). The setting
persists across rebuilds.

## Ad-hoc usage

```bash
# Raw SQL for edge cases
rekal query "SELECT id, user_email, branch FROM sessions ORDER BY captured_at DESC LIMIT 5"

# Rebuild the search index after manual DB changes
rekal index

# View recent checkpoints
rekal log
```
