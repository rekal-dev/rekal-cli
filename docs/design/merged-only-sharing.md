# Merged-only sharing (and worktree-shared state)

**Status:** design (2026-07). Not yet implemented. This note is the spec to
sign off before code. It captures two features that turn out to be one idea.

## Problem

Two open threads, one root.

1. **Sharing is indiscriminate.** `rekal push` exports *every* unexported
   checkpoint in `data.db` (`db.QueryUnexportedCheckpoints`), regardless of
   whether the code that session produced ever landed in `main`. So an
   abandoned `wip/spike` branch — a dead end you'll never merge — is pushed to
   your `rekal/<email>` branch and pulled by every teammate on `rekal sync`.
   That quietly breaks *intent-lives-next-to-the-code* (SOUL.md): you are
   shipping intent for code that isn't in the shared tree and may never be.

2. **Worktrees each need their own sync** (issue #4). Because `.rekal/` lives
   in the per-worktree checkout, every `git worktree` has its own index and
   must be `rekal sync`'d separately. The reporter wants one shared state
   across all worktrees of a repo.

## The insight: one predicate

Both reduce to a single question Rekal can already almost answer from what a
checkpoint stores (`git_sha` + `git_branch`):

> **Is this session's commit reachable from the default branch?**

- For **sharing**: yes → the code merged, the intent belongs in the shared
  ledger. No → keep it local until (if ever) it merges.
- For **worktrees**: the set of "reachable from main" sessions is a property of
  the *repository's object store*, which every worktree shares. So the shared
  ledger is intrinsically worktree-independent — it was only ever per-worktree
  by accident of *where the file sat*.

Stating the rule the soul way: **commit everything for yourself; share only
what merged.**

## Principle

- **Own history — capture all branches, always.** `data.db` keeps every
  session on every branch, merged or not. Full local fidelity; nothing is lost
  or gated locally. This is the developer's own memory of their own work.
- **Shared history — merged-to-main only.** The wire mirrors `main`. Intent for
  merged code flows to the team; intent for unmerged/abandoned work never
  leaves the machine.

This is the mirror image of the cross-repo import feature
(`cross-repo-import.md`): that keeps *other repos'* sessions index-only
(structural); this keeps *your own unmerged branches* off the wire (a filter).
The asymmetry is deliberate — cross-repo carried a cross-tenant-leak risk that
demanded a structural boundary; here it is all your own data in `data.db`, so
the lighter export-time filter is the right weight. The risk being managed is
teammate noise and intent fidelity, not secret leakage.

## Mechanism 1 — merged-gated export

The change is at the **export boundary only**, not a new store.

- `export` selects unexported checkpoints as today, then keeps only those whose
  `git_sha` is reachable from the repo's default branch
  (`git merge-base --is-ancestor <git_sha> <default>`).
- **Self-releasing.** Unmerged work simply doesn't match yet. When the branch
  lands in `main`, its commits *become* ancestors, so the next `push` ships
  them automatically — no on-merge hook, no "share now" button. Abandoned work
  never becomes an ancestor, so it never leaks.
- `data.db` is untouched: it still holds every branch. Only the bytes that
  cross the wire are filtered.

### Resolving "the default branch"

New `gitx` helper: resolve the default branch once per invocation, preferring
`origin/HEAD` (`git symbolic-ref refs/remotes/origin/HEAD`), falling back to
`origin/main`, then `main`. The ancestor test runs against that ref.

### The squash-merge wrinkle (the one real fork)

`is-ancestor` is exact for **merge-commit** and **rebase-merge** workflows.
Under **squash-merge**, the branch is rewritten into a single new commit on
`main`, so the original `git_sha`s never become ancestors — merged work would
never qualify to share. Most GitHub teams squash by default, so this decides
the design, not an edge case.

**Chosen default: ancestor + squash fallback.** Primary signal is the ancestor
test. Fallback for squash: a session's branch is treated as merged when that
branch no longer exists on the remote (`git ls-remote --heads origin <branch>`
is empty) **and** the default branch has advanced past the point the branch
forked from. This releases squash-merged branches by branch-name association
without a perfect commit match. It accepts a small heuristic risk (a
force-deleted-but-unmerged branch could be treated as merged); acceptable
because the data is the developer's own and the alternative — dropping squash
teams' entire history — is worse.

This is the one assumption to confirm. Alternatives if it's wrong:

- **Pure ancestor** — exact, zero heuristics, but squash teams share nearly
  nothing.
- **Explicit promote** — nothing auto-shares; `rekal push --promote <branch>`
  releases a branch's sessions. Deterministic and workflow-agnostic, at the
  cost of a manual step.
- **`share_policy` config** — expose all of the above as a `.rekal/config.json`
  knob (`merged-ancestor | merged-or-squash | manual | all`), safest default.
  `config.json` already exists as the home for exactly this kind of durable
  local setting.

## Mechanism 2 — worktree-shared state

Because the shared ledger is defined by "reachable from main," it is the same
for every worktree of a repo. So it should live in the repo's **common git
dir**, not the per-worktree checkout.

- Resolve the store root with `git rev-parse --git-common-dir` (returns the
  shared `.git` even from a linked worktree) instead of the working-tree root.
  Put `index.db` there — one index, read/written by every worktree.
- `data.db` can follow the same rule, so a checkpoint taken in any worktree
  lands in the one shared ledger. One `rekal sync` then serves all worktrees.
- The only genuinely per-worktree fact is *which branch is checked out right
  now*, which recall already reads live (`gitx.CurrentBranch`) for ranking. The
  store is one; the branch is a lens over it, not a partition of it.

Migration: existing repos have `.rekal/` in the working root (which for a
non-worktree repo *is* the common dir's parent). Detect the legacy location and
either move it or read it in place; a repo that never uses worktrees sees no
change.

## How the two link

They are the same idea seen from two sides:

- Merged-gating asks "which sessions belong to the shared, main-reachable
  history?" — a repo-level, branch-aware question.
- Worktrees ask "where does that repo-level shared history live?" — the answer
  falls out of the same framing: in the shared git dir, because
  main-reachability is a repo property, not a checkout property.

Build them together: add the default-branch resolver and branch-awareness once,
use it for both the export filter and the store location.

## Soul check

| Question (SOUL.md) | This design |
|---|---|
| Preserves immutability? | Yes — `data.db` still append-only; the wire is a filtered *view* of it. |
| Intent stays next to the code? | Strengthened — shared intent now corresponds exactly to code in `main`. |
| Thin on the wire? | Thinner — only merged sessions are pushed. |
| Data stays within git and the local machine? | Yes — unmerged work never leaves; merged work rides the existing orphan branch. |
| Simple — zero config? | The ancestor filter is automatic and invisible; worktree support is a path change, no new command. (A `share_policy` knob is optional, not required.) |
| Transparent — see and remove? | `log`/`recall` can mark sessions `shared` vs `held (unmerged)`; nothing hidden. |
| Agent gets what it needs? | The agent still recalls all your local work on any branch; teammates see only landed decisions. |

## Implementation sketch

1. `gitx`: add `DefaultBranch(gitRoot)` (origin/HEAD → origin/main → main) and
   `IsAncestor(gitRoot, sha, ref)`; add `CommonGitDir(gitRoot)`.
2. `db`: `QueryUnexportedCheckpoints` gains a shareability filter, or `export`
   filters the returned rows by `IsAncestor(cp.GitSHA, default)` plus the
   squash fallback. Keep the raw query for local use.
3. `transport/export.go`: apply the filter before encoding frames; never mark a
   held-back checkpoint exported (so it re-evaluates next push and releases on
   merge).
4. Store location: resolve `.rekal/` (at least `index.db`) under
   `CommonGitDir`; migrate/read legacy path.
5. `log`/`recall`: surface shared-vs-held state for transparency.
6. Optional: `share_policy` in `config.json` if teams need to override the
   squash default.
7. Docs: `spec/command/push.md`, `spec/command/sync.md`, `CLAUDE.md`.
