---
name: rekal-reflect
description: |
  Use this skill in a repo with Rekal (.rekal/ exists) to learn from your own
  past work before repeating it. It mines prior agent sessions — especially the
  human_steering turns where a human corrected the agent mid-task — for
  recurring mistakes and unstated preferences, then distills them into explicit
  rules. Reach for it at the start of a task ("have I done this before, and how
  was I corrected?") or at the end ("what should I write down so I don't repeat
  this?").
---

# Rekal — Self-Reflection

Agents forget between sessions. The same correction gets made again and again
because the lesson never left the transcript it happened in. Rekal keeps those
transcripts. This skill reads them so a correction happens *once* and becomes a
rule, not a recurring tax.

The core signal is the **`human_steering` turn**: a moment the human
interrupted to redirect the agent — "no, use X not Y", "always run the linter
first", "we don't do it that way here". These are unstated preferences made
briefly visible. Harvested across many sessions, the repeated ones are the
rules the project actually runs on.

## When to Use

- **Before a task** — "how was I steered last time I worked on this?" Load the
  corrections so you start already past them.
- **After a task** — "what did I get corrected on today that I should write
  down?" Turn the friction into a durable rule.
- **Periodically** — distill the top recurring corrections into CLAUDE.md so the
  whole team's agents inherit them.

## Workflow

### 1. Find where you were corrected most

Steering turns concentrate in the sessions where the agent most misjudged the
task. Rank sessions by how much steering they drew:

```bash
rekal query --index "SELECT session_id, count(*) AS corrections \
  FROM turns_ft WHERE role = 'human_steering' \
  GROUP BY session_id ORDER BY corrections DESC LIMIT 10"
```

Or scope to a topic you're about to work on:

```bash
rekal --actor agent "database migration"     # prior agent work on this topic
```

### 2. Read the corrections themselves

```bash
rekal query --session <id> --role human_steering
```

Each steering turn is a compressed lesson. Read them across the top sessions
and cluster by theme: testing habits, naming, tools to prefer/avoid, review
etiquette, "don't touch X."

### 3. Distill — recurring correction → explicit rule

A correction seen *once* is a one-off. A correction seen **three or more times
across different sessions** is a rule the project has but never wrote down.
Promote those:

```
Observed (5 sessions): human re-ran `mise run lint` after I said "done".
Rule: Run `mise run fmt && lint && test:ci` before claiming a task is done.

Observed (3 sessions): human rejected new top-level packages.
Rule: New code goes under cmd/rekal/cli/<pkg>/; discuss before adding a
top-level package.
```

Write the distilled rules where agents will actually load them — `CLAUDE.md`,
or a `docs/` note the base rekal skill can later recall. Keep each rule to one
imperative sentence in the project's voice.

### 4. Close the loop

Reflection only pays off if the rule is reachable next time. After writing
rules, the very next session recalls them via the base **rekal** skill
(`rekal "linting before done"`), so the lesson is in-context before the mistake
can recur. Reflection writes; recall reads.

## Distinguishing steering from ordinary turns

- `role = 'human'` — the original ask (intent). Belongs to *what* was wanted.
- `role = 'human_steering'` — a mid-course correction (preference). Belongs to
  *how* it must be done. **This is the reflection signal.**
- `role = 'assistant'` — the agent's reasoning and actions. Read to see what
  triggered the correction.

Down-weight subagent chatter: sessions with a `parent_session_id` are spawned
helpers, and their corrections are often about the sub-task, not the project.
Reflect primarily on trunk sessions.

```bash
rekal query --index "SELECT session_id, count(*) AS c FROM turns_ft \
  WHERE role = 'human_steering' AND session_id IN \
  (SELECT session_id FROM session_facets WHERE parent_session_id IS NULL) \
  GROUP BY session_id ORDER BY c DESC LIMIT 10"
```

## Guidelines

- Distill *patterns*, not incidents. One correction is noise; three is a rule.
- Write rules in the imperative, in the project's voice — they will be read by
  the next agent as instructions, not as history.
- Prefer promoting a rule to CLAUDE.md over keeping it in your head; the point
  is that it survives you.
- Reflection is read-then-write: read `human_steering`, write rules. Don't
  edit code from this skill — its output is knowledge, not a diff.
