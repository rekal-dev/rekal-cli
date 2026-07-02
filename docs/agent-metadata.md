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
  grouping queries.
- **Ranking (future).** The columns are available as ranking signals (e.g.
  demoting raw workflow-step transcripts below their trunk session) without
  any interface change.

## Compatibility rules

- All four columns are nullable; no code path may assume presence.
- Adapters that lack a concept simply never set the field — no
  per-agent special-casing anywhere downstream.
- Schema migrations are additive and idempotent (`MigrateDataSchema`,
  `MigrateIndexSchema`); data/index DBs written by older rekal versions are
  migrated in place on open. data.db is append-only and shared via git —
  teammates on older versions must keep working.
