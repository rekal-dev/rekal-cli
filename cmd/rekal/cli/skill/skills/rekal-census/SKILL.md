---
name: rekal-census
description: |
  Use this skill in a repo with Rekal (.rekal/ exists) to read and summarise
  the WHOLE corpus (or a bounded slice of it), not just the top matches for a
  query. Where the base rekal skill ranks and returns the best few sessions,
  this one scans exhaustively: every session in scope is read exactly once and
  folded into one faithful summary. Reach for it for "summarise everything we
  know", onboarding digests, retrospectives, or an inventory of a project's
  whole session history. It runs on raw `rekal query` SQL, because coverage —
  not relevance — is the goal.
---

# Rekal — Corpus Census

Search answers "what are the *best* sessions for X?" — ranked, top-N, lossy by
design. A census answers "what is *all* of it, summarised?" — exhaustive,
ordered, complete. These are opposite goals, so the census does not touch the
ranking engine at all. It runs on raw `rekal query` SQL: enumerate every
session deterministically, read each once, and fold the parts into a whole.

The method is **map-reduce over the ledger**: inventory the scope, walk a fixed
spine of sessions, summarise each in isolation, then reduce the summaries into
one digest. Done right it is faithful (nothing in scope is skipped), bounded
(it never tries to hold the whole corpus in context at once), and resumable
(the spine is deterministic, so a stopped scan restarts where it left off).

## Goal

Produce **one faithful summary of an entire defined scope** — a coverage
guarantee, not a relevance sample. Success is: every session in scope was read
exactly once, and the digest is traceable back to session ids. If you find
yourself ranking or cherry-picking, you want the base **rekal** skill, not this
one.

## Boundaries — define the scope first (do not skip this)

"Full scan" is meaningless until you bound it. **A whole developer's corpus can
be tens of thousands of turns; you cannot and should not read it raw.** Pick the
narrowest scope that answers the question, and state it explicitly before
scanning. The scope is a `WHERE` clause you will reuse at every step:

| Scope | `WHERE` clause | Use for |
|---|---|---|
| Everything | *(none)* | small repos only; verify the size first |
| A time window | `captured_at >= '2026-06-01'` | "what happened this quarter" |
| One branch/feature | `git_branch = 'feat/webhooks'` | feature retrospective |
| Human vs agent | `actor_type = 'agent'` | "what did the agents do" |
| A subsystem | `session_id IN (SELECT session_id FROM files_index WHERE file_path LIKE '%billing%')` | subsystem digest |
| Trunk only | `parent_session_id IS NULL` | exclude subagent chatter |

Combine them with `AND`. **Always size the scope before scanning** (step 1) —
if it's larger than a few hundred sessions, narrow it or scan in time-window
batches. An unbounded scan on a large corpus is the one failure mode of this
skill; the boundary is the safeguard.

## Systematic workflow

### 1. Size the scope — decide if a full scan is even feasible

```bash
rekal query --index "SELECT count(*) sessions, sum(turn_count) turns,
  min(captured_at) first, max(captured_at) last
  FROM session_facets WHERE <scope>"
```

Read the totals. Rough budget: summarising is ~1 cheap read per session. If
`sessions` is in the low hundreds, scan straight through. If it's thousands,
**batch by time window** (loop step 2–3 per month) or narrow the scope. Never
attempt to pull all turns of a large corpus into context at once.

### 2. Enumerate the spine — every session, deterministically ordered

This ordered list is the scan's backbone. `captured_at` ordering makes the scan
idempotent and resumable: the same scope always yields the same spine, so a
stopped scan restarts at the last session it summarised.

```bash
rekal query --index "SELECT session_id, captured_at, git_branch, actor_type,
  turn_count, description
  FROM session_facets WHERE <scope> ORDER BY captured_at"
```

### 3. Map — summarise each session in isolation, cheapest read first

Walk the spine. For each session, read only as deeply as it warrants, and emit a
**one- to three-line summary keyed by session id**. Then discard the raw turns
before moving on — this is what keeps the scan bounded.

```bash
rekal query --session <id> --role human        # intent — often enough alone
rekal query --session <id> --role human_steering # corrections, if any
rekal query --session <id>                      # full turns, only if it's pivotal
```

Keep each map output tiny and uniform, e.g.:

```
01JNQ… (agent, feat/webhooks, 210t): added retry+backoff to delivery;
  steered off fixed-delay toward exponential.
```

The running list of these lines — not the transcripts — is the scan's working
set. It stays small no matter how big the corpus.

### 4. Reduce — fold the summaries into one digest

With every session reduced to a line, fold the lines into themes and a timeline.
This step reads only your own map output, so it is cheap regardless of corpus
size:

- **Themes** — cluster the lines into a handful of topics; each theme cites its
  session ids.
- **Timeline** — order the themes by first/last `captured_at` to show how the
  work evolved.
- **Open threads & dead-ends** — call out unfinished work and abandoned
  branches (see the **rekal-distill** boundary library).

### 5. Verify coverage — the census's one hard check

A census is only trustworthy if it is complete. Confirm you summarised exactly
the spine, no more, no fewer:

```bash
rekal query --index "SELECT count(*) FROM session_facets WHERE <scope>"
```

The count must equal the number of session lines in your map. If it doesn't, you
skipped or double-counted — reconcile before publishing the digest. State the
coverage explicitly in the output ("summarised 148/148 sessions in scope").

## Output

A digest with the scope stated up front, then themes and a timeline, closing
with the coverage check:

```
Census — scope: actor=agent, branch=feat/webhooks (2026-05..2026-07)
Coverage: 41/41 sessions read.

Themes
  Delivery reliability (12 sessions, 01J…, 01K…): retries, backoff, DLQ
  Test scaffolding (7 sessions, …): fixture helpers, flaky-test triage
  …

Timeline
  May: initial delivery path …
  Jun: hardening under load …
  Jul: batching spike — dropped (ordering broke), see 01K…

Open threads: dead-letter queue undecided (01L…)
```

The stated scope + coverage count is what separates a census from a search: the
reader knows nothing in that scope was left out.

## Guidelines

- **Bound before you scan.** State the scope; size it; batch if large. An
  unbounded scan on a big corpus is the only way this skill goes wrong.
- **Summarise, then discard.** Keep per-session lines, not transcripts. The
  working set stays small; that is what makes a full scan feasible.
- **Coverage over relevance.** If you're picking the "interesting" sessions,
  use base rekal instead — a census that skips sessions isn't a census.
- **Cite session ids** in every theme so the digest is drillable, and always
  report the coverage count.
