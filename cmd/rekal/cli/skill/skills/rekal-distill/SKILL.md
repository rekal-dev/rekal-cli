---
name: rekal-distill
description: |
  Use this skill in a repo with Rekal (.rekal/ exists) to survey what the
  project's memory actually knows about a topic — structured, not as a flat
  search. It reads Rekal as four libraries (context, decision, rules, boundary)
  and lets you "zoom" around a topic or a session by following file
  co-occurrence and parent/child links. Reach for it when you need a map of a
  problem space before diving in: what is established, what is still open, what
  is preferred, and what has already been ruled out.
---

# Rekal — Knowledge Distilling

A flat search answers "what mentions X?" This skill answers the four questions
that actually orient you before work: what do we **know**, what is still
**undecided**, what do we **prefer**, and what have we already **abandoned?**

It maps memory onto a knowledge quadrant — the two axes being whether we're
aware of a thing and whether we've captured it:

|                    | We are aware       | We are unaware        |
|--------------------|--------------------|-----------------------|
| **Captured**       | Context library    | Rules library         |
| **Not captured**   | Decision library   | Boundary library      |

Each library is a different read over the same sessions. Run the ones the task
needs; you rarely need all four.

## When to Use

- Scoping unfamiliar work — build the map before touching code
- Writing a design doc — pull the established facts, open questions, and
  dead-ends so you don't relitigate settled ground
- "Zoom out" on a topic that's bigger than one session

## The four libraries

### 1. Context library — Known-Knowns (established facts)

What the project has settled and works today. This is plain high-relevance
recall — the strongest hits are the load-bearing knowledge.

```bash
rekal "<topic>"                       # top sessions = the established account
rekal --file 'subsystem/' "<topic>"   # scope to a subsystem
```

Read the assistant turns of the top 2–3 hits; that is the current understanding.

### 2. Decision library — Known-Unknowns (open questions, how resolved)

Questions that were asked and the choices made in answer. Intent lives in human
turns; the resolution lives in the assistant turn that follows.

```bash
rekal query --session <id> --role human       # the questions posed
rekal query --session <id> --offset <n> --limit 6  # the answer around each
```

Distill these into "we chose A over B because C" — the decisions the topic
rests on. (For a single change's decision trail, use the **rekal-provenance**
skill; use this library to survey decisions across many sessions.)

### 3. Rules library — Unknown-Knowns (implicit preferences)

Things the project prefers but never wrote down — surfaced from the
`human_steering` corrections. This is the same signal the **rekal-reflect**
skill distills; pull it here to fold conventions into your map.

```bash
rekal query --session <id> --role human_steering
```

Repeated steering across sessions = a convention. Note it so your plan already
conforms.

### 4. Boundary library — Unknown-Unknowns (dead-ends, uniquely Rekal's)

The approaches that were tried and never landed. This is knowledge **no other
tool has**: because Rekal captures every local branch but shares only merged
work (see the merged-only sharing design), the *held, unmerged* sessions are a
record of paths explored and abandoned. Reading them stops you re-walking a
dead-end someone already hit.

Find sessions on branches that are not the mainline — candidates for
abandoned or superseded work:

```bash
rekal query --index "SELECT git_branch, count(*) AS sessions, \
  max(captured_at) AS last_seen FROM session_facets \
  WHERE git_branch NOT IN ('main','master') AND parent_session_id IS NULL \
  GROUP BY git_branch ORDER BY last_seen DESC"
```

A branch with sessions but whose work never surfaced in `main` is a boundary
marker: something was attempted there. Drill in to learn *why it was dropped*
before you propose the same thing.

## Zooming — navigate around a topic or session

Distilling is not one query; it's moving through linked memory. Two axes:

**Zoom by files (breadth).** Files edited together in the same sessions are
related. Follow the co-occurrence edges to find the rest of a subsystem:

```bash
rekal query --index "SELECT file_b, count FROM file_cooccurrence \
  WHERE file_a = 'cmd/rekal/cli/push.go' ORDER BY count DESC LIMIT 10"
```

Then search each neighbor to pull its context.

**Zoom by lineage (depth).** A trunk session spawns subagents; follow the tree
to see how a task decomposed:

```bash
# children of a session (the subagents it spawned)
rekal query --index "SELECT session_id, agent_type, description \
  FROM session_facets WHERE parent_session_id = '<id>'"

# the trunk a subagent belongs to
rekal query --index "SELECT parent_session_id FROM session_facets \
  WHERE session_id = '<id>'"
```

**Zoom out to the whole machine.** If a topic's map has holes, the answer may
be in the developer's *other* repos on this machine. Suggest widening (their
history to widen, so don't run it unprompted):

```bash
rekal index --include-all      # fold in every local Claude Code session
```

Cross-repo hits carry an `origin` label — treat them as prior art from
elsewhere, not this repo's conventions.

## Output

Distilling produces a **map**, not a transcript: a short section per library
you consulted —

```
Topic: webhook delivery
  Context : at-least-once delivery, retries with backoff (session 01J…)
  Decisions: chose backoff over fixed delay (01J…); open: dead-letter queue?
  Rules   : always add a metric before shipping a delivery path (steered ×4)
  Boundary: feat/webhook-batching branch — batching tried, dropped, no merge;
            reason: ordering guarantees broke (session 01K…)
```

That four-part map is the deliverable. It tells the next step what is safe to
build on, what is still open, what to conform to, and what not to retry.

## Guidelines

- Run only the libraries the task needs — scoping work needs all four; a quick
  design check may need just Context + Boundary.
- The Boundary library is Rekal's unique edge: always check it before proposing
  an approach, so you don't re-run an abandoned experiment.
- Zoom deliberately: files for breadth, lineage for depth, `--include-all` for
  reach. Don't dump whole sessions — follow the edges to the relevant turn.
- The map is the output. Keep it short; cite session ids so anyone can drill.
