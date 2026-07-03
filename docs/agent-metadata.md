# Agent-harness metadata: internal, not a CLI surface

**Status:** decided (2026-07). Applies to `team_name`, `workflow_name`, `agent_id`, `parent_session_id`.

## Context

Claude Code sessions can now spawn subagents, teams, and dynamic workflows.
Their transcripts are captured as their own sessions, linked to the trunk
conversation, and the harness metadata (team name, workflow name, agent id,
parent session) is stored first-class in `sessions` and `session_facets`.

Other agent harnesses have none of these concepts. A Codex rollout has no
team. An OpenCode session has no workflow. Future adapters will bring their
own vocabulary (threads, tabs, projects, …).

The question: do we expose `--team`, `--workflow`, `--agent` filters on
recall/query, or keep the metadata internal?

## Decision

**No new CLI flags.** The metadata is optional, nullable everywhere, carried
in output for grouping and drill-down, and reachable via raw SQL. It is not a
filter surface.

## Reasoning

- **Filters are universal concepts.** The existing filters — `--actor`,
  `--author`, `--commit`, `--file` — mean the same thing for every session
  regardless of which agent produced it. A `--workflow` filter is meaningful
  only for Claude Code sessions: against a repo full of Codex sessions it
  silently returns nothing. A filter that only means something for one agent
  type is suspect.
- **Per-agent flag sprawl.** Accepting one harness's taxonomy as CLI surface
  invites the next one's (Codex threads, OpenCode projects, …). "We don't add
  options where a good default exists. Simple." (SOUL.md)
- **Agent first.** The consumer is an agent doing progressive loading:
  recall → drill-down. It is better served by results that *carry* the
  metadata (so it can group workflow transcripts under their trunk and decide
  what to drill into) than by having to know harness-specific flags up front.
- **The escape hatch already exists.** `rekal query --index
  "SELECT ... FROM session_facets WHERE workflow_name = '...'"` covers every
  power-user filtering need with zero new surface. That is what SQL mode is
  for.

## What the metadata does instead

- **Output enrichment.** Recall results (`session` detail) and
  `query --session` output include `agent_id`, `team_name`, `workflow_name`,
  and `parent_session_id` when present, omitted when absent (`omitempty`).
  A Codex session's output is byte-identical to what it was before these
  fields existed.
- **Grouping / drill-down.** `parent_session_id` lets the agent (or any
  consumer) roll subagent and workflow transcripts up under their trunk
  conversation. `session_facets` has indexes on the new columns for cheap
  grouping queries. Implemented in `recall.go` — see below.
- **Ranking.** The columns are ranking signals (demoting subagent-internal
  turns below their trunk session) — implemented in `recall.go`, see below.

## Compatibility rules

- All four columns are nullable; no code path may assume presence.
- Adapters that lack a concept simply never set the field — no
  per-agent special-casing anywhere downstream.
- Schema migrations are additive and idempotent (`MigrateDataSchema`,
  `MigrateIndexSchema`); data/index DBs written by older rekal versions are
  migrated in place on open. data.db is append-only and shared via git —
  teammates on older versions must keep working.

## Using the metadata in recall: grouping, weighting, budget

**Status:** decided (2026-07). Implemented in `cmd/rekal/cli/recall.go`.

The decision above kept `team_name`/`workflow_name`/`agent_id`/
`parent_session_id` out of the CLI surface but promised they'd shape output
and ranking without one. This section is that follow-through — four small,
additive changes to `hybridSearch`, each traced back to a specific belief in
`SOUL.md` rather than to "better search" in the abstract.

### 1. Group-and-collapse (`groupByConversation`)

A dynamic workflow with ten steps used to show up as ten separate recall
results, crowding out everything else in the top-k. `parent_session_id` is
walked to each hit's trunk conversation (`resolveRoot`); hits sharing a root
fold into one result, with the non-winning hits nested under `children`. A
session with no parent and no matching descendants renders exactly as before
— the field is `omitempty`, so ungrouped output is byte-identical to
pre-grouping recall.

- **Traces to:** *agent first* — "the agent is the consumer... every
  decision... favors the agent." Ten near-duplicate rows for one
  conversation is a tax on the agent's context budget, not a richer result.
- **Traces to:** *simple* — the grouping key is `parent_session_id`, already
  indexed on `session_facets`. No new ranking framework, no
  cross-encoder, no second pass over the corpus.
- **Generic across agent types** (per the decision above): a session with a
  null parent groups with nothing and passes through unchanged. Codex,
  Gemini, and OpenCode sessions — which never set the column — are
  unaffected.

### 2. Signal weighting: `steeringBoost` and `subagentDownweight`

Two constants, applied where the hybrid score is already assembled — not a
re-ranking pass:

- `steeringBoost` (1.3×) raises the effective BM25 score of turns captured
  from `queue-operation`/`enqueue` (role `human_steering`, see below) when
  picking a session's best-matching turn. A steering message is text a human
  typed *while an agent was already working on something else* — it exists
  because the human judged it important enough to interrupt for. That is
  the highest-intent signal available in the corpus.
- `subagentDownweight` (0.7×) discounts a session's hybrid score whenever
  `parent_session_id` is non-null — a subagent's internal exploration
  (files it happened to read, dead ends it walked back) matters less than a
  trunk turn of equal textual relevance, because the trunk is where
  decisions actually get made and reported back.

Both are ordinary float multipliers on values the existing pipeline already
computes; there is no second scoring model, no learned weight, no config
flag to tune them.

- **Traces to:** *agent first* / *intent has no ledger* (the two problems in
  SOUL.md's opening section) — steering messages are the clearest first-hand
  record of *why*, which is the entire reason the ledger exists. Ranking
  should reflect that, not treat all human text as equal.
- **Traces to:** *simple* — two constants, not a model.

### 3. Steering turns are tagged at capture time, not inferred at query time

Rather than re-deriving "was this a steering message" from content at recall
time, the Claude parser now emits queue-operation/enqueue turns with role
`human_steering` instead of `human` (`cmd/rekal/cli/session/parse.go`). The
wire codec gained one more `Role` byte value (`RoleHumanSteering = 0x02`)
alongside the existing `RoleHuman`/`RoleAssistant` — the same one-byte field,
one more value, zero additional bytes on the wire for every turn that isn't
a steering message.

- **Traces to:** *thin on the wire.* A dedicated boolean column (in either
  `turns` or the wire `TurnRecord`) would have worked too, but every turn
  ever encoded would carry it forever. Reusing the existing `role` byte's
  unused values costs literally nothing extra on the wire or in storage.
- **Traces to:** *simple.* No schema migration needed at all — `turns.role`
  and `turns_ft.role` are already `VARCHAR`; the new value just flows
  through unchanged. Only the codec (three bytes' worth of const) and the
  three call sites that map string role ↔ byte role needed updating.
- Turns captured before this change keep role `human` — this is forward-only
  like every other migration in this codebase; there is no backfill, and
  none is attempted.

### 4. Per-conversation result budget (`conversationChildBudget`)

Folding hits under their trunk (§1) means one giant workflow could still
crowd out other conversations' *children* even after collapsing to one
top-level slot each — imagine fifty workflow-step transcripts all matching,
all nested under one entry. `conversationChildBudget` (3) caps how many
transcripts get folded into one group; the rest are simply not shown.

- **Traces to:** *agent first* — the budget exists for the same reason
  grouping does: protect the agent's limited result window from being
  dominated by one conversation's internal fan-out.
- **Traces to:** *simple* — a constant cap during grouping, not a
  diversity-aware re-ranking algorithm.

### 5. Drill-down surfaces the grouped structure

`query --session <id>` always includes `child_session_ids` — the sessions
whose `parent_session_id` points at the one being drilled into — alongside
the existing `agent_id`/`team_name`/`workflow_name`/`parent_session_id`
fields. Combined with the `children` array in grouped recall results (§1),
an agent can go: collapsed recall result → `children[i].session_id` (or
`child_session_ids` from a drill-down) → `query --session` on the exact
transcript that matched, without ever needing a `--workflow` flag or a raw
SQL query.

- **Traces to:** *agent first* — "output format, query interface, and
  context loading puts the agent first"; this is the last hop of the
  progressive-loading path the original decision promised.
- **Generic across agent types:** `child_session_ids` is populated from
  `parent_session_id` alone, so it works (and is usually empty) for every
  adapter, not just Claude Code's.
