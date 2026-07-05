# Cross-repo local session import

**Status:** design (2026-07). Not yet implemented. This note is the spec to
sign off before code.

## Problem

Rekal captures sessions per repo: `checkpoint` only discovers the session
directory that matches the current repo (`session.FindSessionDir`). An agent
working here therefore cannot recall how you solved a similar problem in
*another* repo last month, even though that session is sitting on the same
machine. That is squarely the second reason Rekal exists (SOUL.md: "Agents
can't remember") — just scoped one repo too narrowly.

This feature lets a developer, on demand, fold **all** of their local agent
sessions — every other repo, and non-repo/shell sessions too — into this
repo's local recall, so the agent's memory spans the developer's whole
history instead of one project.

It is powerful and sensitive, so the design is built around one rule.

## The hard rule

> Cross-repo (and shell) sessions must be **structurally incapable** of
> reaching the team wire.

Rekal's push path exports `data.db` checkpoints to the shared orphan branch.
If cross-repo content ever became an exportable checkpoint, `rekal push` would
ship every other project's conversations — other clients, other employers,
secrets, proprietary architecture — to this repo's teammates. That violates
two soul beliefs directly ("the data never leaves git and the local machine";
"intent lives next to the code"). Scrubbing does not save this: it strips
secrets, not semantic leakage (client names, architecture in prose). The
guarantee must be structural, like immutability is structural — not a policy
in one `WHERE` clause.

Framing that reconciles the soul: **local agent memory for you — yes; part of
this repo's shared immutable ledger — never.**

## Storage: index-only (never `data.db`)

Imported sessions go into `index.db` only — the same isolation model teammate
data already uses (`transport.ImportBranchToIndex`), which writes teammate
sessions into `turns_ft` / `session_facets` / embeddings and never into
`data.db`. Because `data.db` is the only thing `export`/`push` reads, content
that never enters it is structurally unexportable. No new flag on the export
path, no local-only column to remember to filter — the safety comes from
where the bytes live, not from discipline.

### Re-derived, not stored — and that is the point

`index.db` is derived and disposable (SOUL.md). It is rebuilt from its sources
of truth on every `rekal index` and `rekal sync`:

- `data.db` (this repo's own captured sessions), and
- the synced remote branches (teammate sessions).

Cross-repo import adds a third source of truth, already present on the machine:
the local session tree at `~/.claude/projects/*`. It is re-read on the rebuild
that requests it, exactly as teammate data is re-read from remote branches.

So a **plain** `rekal index` or `rekal sync` (without the flags below) drops
the cross-repo view. That is intended and soul-aligned: the index re-derives
exactly what you explicitly asked for this time. There is no persisted
preference and no config file — nothing hidden to reason about or remove. The
only persistent copy of the data stays in `~/.claude`, right where it already
was; drop `index.db` and the cross-repo content is simply gone.

## Interface

Two per-invocation flags on the existing `index` command — the correct home,
because the data is index-layer, not ledger-layer, and the flag name states
exactly what happens:

```
rekal index --include-all           # this rebuild: every local project + shell session
rekal index --include <repo> [...]  # this rebuild: only the named repo(s)
rekal index                         # plain rebuild: no cross-repo (local view dropped)
```

- `--include <repo>` takes a **repo path** — the working directory the other
  sessions were launched in — resolved to its session directory the same way
  the current repo is (`session.SanitizeRepoPath` →
  `~/.claude/projects/<sanitized>`). Repeatable for several repos.
- `--include-all` supersedes `--include` and covers everything, including
  non-repo/shell working directories.
- **Cover all, no tiers.** Shell-only sessions are included by
  `--include-all`; they are not gated or classified out. This is acceptable
  *because* of the hard rule: nothing imported can leave the machine, and the
  data already lives in `~/.claude` — the feature widens the local *recall*
  surface, not the *exposure* surface.

### Confirmation and voice

`--include-all` prompts once (unless `--yes`), because it reaches personal and
shell sessions:

```
rekal: this imports every local agent session on this machine — 412 sessions
       across 37 projects, including non-repo/shell sessions — into this repo's
       local recall. They stay local and are never pushed. Continue? [y/N]
rekal: imported 412 sessions from 37 local projects (local only, never pushed)
```

`--include <repo>` is narrow and self-evident; it imports without a prompt and
reports what it did:

```
rekal: imported 23 sessions from /Users/frank/work/api (local only, never pushed)
```

Optional narrowing (`--since <date>`) can land later; not required for v1.

## The walk

Reuse the existing discovery. For each requested project directory under
`~/.claude/projects/` (all of them for `--include-all`, the resolved ones for
`--include`), run `session.discoverSessionRefs`, which already yields the
trunk `.jsonl` plus its nested `subagents/…` tree. The only new discovery code
is enumerating the project directories; the per-project recursion is unchanged.

`CLAUDE_CONFIG_DIR` is honored as it is today. v1 scope is Claude Code's store;
other adapters (codex/gemini/opencode) are a follow-up — their per-tool roots
plug into the same "enumerate roots → discover → import" shape.

## Dedup

Because the rebuild starts from an empty index, dedup needs no persistent
state: during the cross-repo import pass, skip any session whose content hash
already exists among this repo's own `data.db` sessions
(`db.SessionExistsByHash`), so a session that belongs to this repo is not
double-counted. Everything else is imported fresh each rebuild.

## Provenance / labeling

Cross-repo hits must be visibly not-from-here, for the agent and the human:

- `session_facets` gains an `origin` column. For imported sessions it records
  the source read from the transcript's `cwd`: `repo:/Users/frank/work/api` or
  `shell:/Users/frank/scratch`. This repo's own sessions leave it empty/`local`.
- Recall output surfaces `origin` on each result, so a cross-context hit is
  always labeled — an agent can tell a match came from another project or a
  shell session, and can down-weight or ignore it.

Reading `cwd` from the transcript (ground truth) rather than reversing the
lossy sanitized directory name is how origin is classified. It is used only
for the label — never to gate what is imported.

## The residual caveat: recall echo

Model A makes the imported *data* structurally unexportable, but there is an
indirect path worth stating plainly so the choice is informed: content that
**recall surfaces** can be quoted by the agent into the current (work) session,
and that work session *is* captured and pushed. So a snippet from a personal
shell session could, in principle, be echoed into a work conversation and reach
the team that way.

This is not a storage leak (the imported data itself never touches `data.db`
or the wire); it is inherent to any recall that surfaces cross-context data,
and `--include-all` widens what can surface. Mitigation is the origin label
(the agent sees the hit is from elsewhere) — and, ultimately, it is the
developer's own data, on their own machine, echoed by their own choice. It is
documented here so `--include-all` is chosen with eyes open, not hidden.

## Cost, and a future embedding cache

Each rebuild that requests cross-repo import re-parses, re-scrubs, and
re-embeds (nomic, CGO) the requested sessions, because `index.db` — where
embeddings live — is wiped and rebuilt. Teammate data already pays this on
every `sync`; cross-repo just adds volume. For a large local history,
`--include-all` rebuilds will be noticeably slower.

If that becomes painful, the mitigation is a **content-hash-keyed embedding
cache** — persist only the vectors (never the raw session text), so a rebuild
re-embeds only content it has not seen. That is the one place the design would
drift slightly toward a persistent artifact, and it is for derived embeddings
only. Ship without it first; add it only if measured rebuild time demands it.

## Soul check

| Question (SOUL.md) | This design |
|---|---|
| Preserves immutability? | N/A — index is derived, not the ledger. The ledger (`data.db`) is untouched. |
| Intent stays next to the code? | The shared ledger stays this-repo-only. Cross-repo intent is local recall, never merged into the shared record. |
| Thin on the wire? | Nothing imported ever reaches the wire — structural. |
| Data stays within git and the local machine? | Yes. The data already lives in `~/.claude`; this only makes it locally searchable. Never exported. |
| Simple — zero config? | Two flags on an existing command. No config file, no persisted preference, no new DB. |
| Transparent — see and remove? | Origin-labeled in output; nothing persisted beyond the disposable index; a plain reindex or `clean` removes it entirely. |
| Agent gets what it needs? | Wider memory, explicitly labeled by origin so the agent can judge relevance. |

## Implementation sketch

1. `session`: add `EnumerateProjectDirs()` (list `~/.claude/projects/*`) and a
   resolver from a repo path to its project dir (reuse `SanitizeRepoPath`).
2. `db`: add `origin` column to `session_facets` (index-only migration); a
   local-import populate function mirroring `PopulateIndex`'s session/turn/
   facet/embedding steps but sourced from parsed local session files, hash-
   deduped against `data.db`.
3. `index_cmd`: `--include-all` / `--include` flags; after the normal
   `PopulateIndex`, run the local-import pass for the requested roots, then the
   embedding pass over the combined content. Confirmation for `--include-all`.
4. `search`: thread `origin` into `SessionDetail` / recall JSON.
5. Docs: update `docs/spec/command/index.md`; note the feature in `CLAUDE.md`.
