---
name: rekal
description: |
  Use this skill when working in a repo with Rekal initialized (.rekal/
  exists). Rekal is memory of prior AI sessions — who changed what, why, and
  when. Decide WHERE the answer lives before you spend a token: a question
  about a coding project is answered by one of three substrates — the TREE
  (current code, via grep/read), the LEDGER (recorded session intent, via
  rekal), or the MAP (comprehended structure, the bridge). This skill is the
  triage and the workflows: classify the question, dispatch to the one
  substrate that can answer it, and stay silent where memory is not the
  right tool. Route, do not stack.
---

# Rekal — which substrate answers this?

Git holds two different things, and most wasted effort comes from asking the
wrong one. Before recalling, reading, or grepping, decide which substrate
the question belongs to.

| Substrate | Holds | Tense | Primitive | Answers |
|---|---|---|---|---|
| **Tree** (code @ HEAD) | current state | present | `grep` / read / AST | "what does X do / where is it / what's the value now" |
| **Ledger** (sessions) | recorded intent | past | `rekal` recall + modes | "why is it like this / what was tried / what was rejected" |
| **Map** (structure) | comprehended shape | — | MAP workflow | "how is the whole thing built / what connects to what" |

The map is derived from the tree but serves memory: it is the tree's shape,
cached and git-watermarked, so breadth questions don't re-grep the world.

**The rule:** grep the tree for present-tense facts. Recall the ledger for
past-tense intent. The map tells you which — and stay silent when it's
neither.

If `rekal` is not on PATH, run `export PATH="$HOME/.local/bin:$PATH"` first.

## Triage — classify the question, then dispatch

1. **Is it about what the code IS right now?** ("what does this function
   do", "where is X defined", "what's the current config") → **Tree.** Use
   grep / read / your code tools. Do NOT use rekal — the ledger does not
   store current code content; it would be slower and risk a stale answer.
   This is the most common misroute. Answer from the source.
2. **Is it about the SHAPE of the system?** ("how is the pipeline
   architected end-to-end", "what subsystems exist", "how do they connect")
   → **MAP workflow.** No single file or session holds this; structure
   does. The map also orients steps 1 and 3 (where to grep, which sessions
   to recall).
3. **Is it about WHY / how it evolved / a past decision?** ("why was X
   chosen over Y", "what did we reject", "who decided") → **Ledger.** Not
   in the code — the reasoning expired with the session.
   - pointed ("which session did X, and the detail") → **HUNT workflow**
     (seed recall + drill, confidence-gated).
   - rationale / evolved decision ("why X instead of Y") → **WHY workflow**
     (gather every decision-relevant turn, synthesize the arc; single-shot
     recall only returns one fragment).
4. **Hybrid — need the actual code behind a past decision?** Rekal owns the
   pointer, git owns the content: recall gives the commit SHA it recorded,
   then `git show <sha>` reconstructs the diff on demand. That is "grep on
   content, scoped by memory" — it keeps the ledger thin.

### The gate — when to stay silent

Answering from the wrong substrate is worse than not answering. Two
silences matter:

- **Don't inject memory into a tree question.** If step 1 fits, read the
  code. A recalled session about how X *used* to work can mislead on how it
  works now.
- **Don't inject low-confidence episodes.** When HUNT recall is weak (no
  clear top hit, flat score gap — see the gate below), the episode stays
  OUT of context. Ungated low-confidence episodes measurably degrade a good
  answer. Silence beats noise.

## Workflow MAP — breadth, answered from structure

The map lives at `.rekal/map.md` (derived, local — `.rekal/` is gitignored;
the committed browse surface is rekal-wiki, not this file). Its reader is
an agent: optimize for density, greppable anchors, and clean diffs — that
means **structured markdown sections, not a mermaid diagram** (a diagram is
human decoration here; text sections carry more per token, every line can
name a real path to grep next, and section-scoped rewrites keep the
watermark refresh cheap).

### Freshness protocol

Line 1 is the watermark: `<!-- rekal-map <branch> <HEAD-sha> -->`.

- **Fresh** (watermark SHA == `git rev-parse HEAD`): answer from the map.
- **Stale** (any other SHA — including after a branch switch; the map is a
  function of the tree at a SHA, not of the branch): run
  `git diff --name-only <watermark-sha> HEAD`, map the changed files to
  subsystem sections, rewrite only those sections and any flow that touches
  them, update the watermark. If the diff exceeds ~50 files or subsystem
  boundaries moved, rebuild in full.
- **Missing:** build it (below).

### Build procedure — prescriptive

1. **Inventory the skeleton** (no file contents yet): top two directory
   levels, the manifests (`go.mod` / `package.json` / `pyproject.toml`),
   build and CI files. This bounds what exists.
2. **Read the repo's own claims**: README, `docs/`, architecture notes,
   CLAUDE.md. Treat them as claims to verify against the tree, not truth.
3. **Cut subsystems by responsibility, not by directory.** Aim for 5–12;
   more means you're listing folders, fewer means you're hand-waving. Every
   system has: entry points, core domain, storage, transport/IO, external
   surfaces — find where this repo puts each.
4. **Comprehend each subsystem**: open its 1–3 load-bearing files (the
   entry point, the central type) and read enough to state *what it is
   for* and *what breaks if it's deleted*. If you cannot answer the second,
   you haven't comprehended it — read more. Never paste file contents into
   the map.
5. **Trace the load-bearing edges**: which subsystems import/call which,
   and what crosses the edge (data, events, files). List only edges that
   carry the system's main flows.
6. **Optionally add memory hooks**: `rekal --explain "<subsystem>"` — if
   one or two sessions clearly own a hot area, note their ids as drill
   pointers.
7. **Emit in the exact template below.** Deterministic order (subsystems
   by primary path), ≤12 lines per subsystem, ≤150 lines total.

### Template

```markdown
<!-- rekal-map <branch> <HEAD-sha> -->
# Map — <repo name>

## System in one paragraph
<what it is, for whom, the one main flow>

## Subsystems
### <name> — `<primary path>`
- purpose: <behavior, not a file list>
- key files: `<a>`, `<b>`
- depends on: <subsystem> (<what crosses the edge>)
- invariant: <constraint worth knowing before editing>   [optional]

## Flows
- <flow name>: <subsystem> → <subsystem> → <subsystem> (<what moves>)

## Pointers   [optional]
- <topic>: session <id> (<why it's the authority>)
```

**Quality bar:** every subsystem names real, greppable paths; purpose is
stated as behavior; every edge says what crosses it; anyone reading only
the map can decide where to grep and which sessions to recall next.

## Workflow HUNT — pointed, gated episodic recall

1. **Search:**

```bash
rekal "JWT expiry"                      # hybrid search (BM25+LSA+semantic+facet)
rekal --file src/auth/ "token refresh"  # scope by file path (regex)
rekal --commit <sha>                     # sessions that produced a commit
rekal -n 5 --explain "error handling"   # + per-layer scores and related sessions
```

2. **Gate.** From the result JSON take the top score and the gap to the
   second. Confident: top score ≥ 0.9, or gap ≥ 0.04. Below both → treat as
   a miss: say memory has no confident episode (and consider re-routing — a
   "pointed" question that misses is often a why or breadth question in
   disguise). Do not pad the answer with near-misses.

3. **Drill, cheapest first** — never `--full` by default:

```bash
rekal query --session <id> --role summary          # compaction summary — cheapest dense overview
rekal query --session <id> --offset <snippet_turn_index - 2> --limit 5   # window around the match
rekal query --session <id> --role human            # intent
rekal query --session <id> --role human_steering   # mid-task corrections — highest signal
rekal query --session <id> --full                  # last resort
```

Result fields that matter: `session_id`, `score`, `snippet`,
`snippet_turn_index` (jump target), `snippet_role` (`human_steering` = a
human correcting the agent — high intent; `summary` = harness distillation
— dense but machine text), `summary_turn_index` (present when a compaction
summary exists), `children` (grouped subagent/workflow transcripts),
`origin` (present only on cross-repo hits — prior art from *another*
project, not this repo's conventions).

## Workflow WHY — rationale, reconstructed by synthesis

The rationale for an evolved decision is distributed across sessions.
Gather the decision trail, then synthesize. **The gather bounds the
quality** — an under-gathered synthesis starves; gather generously before
concluding anything.

1. **Seed:** search 2–3 phrasings of the decision and its alternatives.
   Note candidate sessions and their commits.
2. **Gather the decision trail** — steering turns and reasoning-marked
   turns across *all* sessions, not the top hit:

```bash
rekal query --index "SELECT session_id, turn_index, role, substr(content,1,300) FROM turns_ft \
  WHERE (role = 'human_steering' OR content LIKE '%because%' OR content LIKE '%instead of%' \
         OR content LIKE '%constraint%' OR content LIKE '%rejected%' OR content LIKE '%decided%') \
  AND (content LIKE '%<topic-term-1>%' OR content LIKE '%<topic-term-2>%') \
  ORDER BY session_id, turn_index"
```

   Aim for the full trail (often ~30 turns). Fewer than ~10 → widen the
   topic terms before concluding.
3. **Pull code on demand:** `git show <commit-sha>` for any claim that
   references code (the `commit` field on results) — intent lives in the
   ledger, content lives in git.
4. **Synthesize the arc**, every claim carrying pointers: original design →
   alternatives rejected → the constraint that forced the change → final
   rationale, each step cited as `(session <id> turn <n>, commit <sha>)`.
   The chain is the deliverable, not the transcripts.
5. **If the trail is genuinely absent**, say so plainly — the answer was
   never verbalized in the ledger (a capture gap); inventing a rationale is
   worse than reporting the gap.

## Widen memory — cross-repo recall

```bash
rekal index --include-all           # all repos + shell sessions on this machine
rekal index --include /path/to/repo # just that repo's sessions
rekal index --no-local              # back to this repo only
```

Imported sessions are index-only — recallable here, structurally impossible
to push to the team — and carry the `origin` label. Suggest `--include-all`
when a search misses but the problem smells solved elsewhere; don't run it
unprompted, it's the developer's history to widen.

## Raw SQL — the escape hatch

```bash
rekal query "SELECT id, user_email, branch FROM sessions ORDER BY captured_at DESC LIMIT 5"
rekal query --index "SELECT * FROM file_cooccurrence WHERE file_a LIKE '%auth%' ORDER BY count DESC"
```

`rekal query --help` documents both DB schemas. `files_touched` includes
git-native change types (M/A/D/R) plus `T` for files Written/Edited during
the session; `tool_calls.path` is the most complete "what files did this
session interact with" source.

## Filters (root command)

| Flag | Description |
|------|-------------|
| `--file <regex>` | Filter by file path (regex, git-root-relative) |
| `--commit <sha>` | Filter by git commit SHA |
| `--author <email>` | Filter by author email |
| `--actor <human\|agent>` | Filter by actor type |
| `-n`, `--limit <n>` | Max results (default: 20, 0 = no limit) |
| `--explain` | Adds `layers` (normalized bm25/lsa/nomic/facet scores) and `related` (sessions sharing touched files — zoom edges) |

## Companion skills

Deep-dive skills build on this router; reach for them when the task *is*
the deep dive:

- **rekal-provenance** — artifact → commit → session → intent why-chain
- **rekal-reflect** — mine your own steering turns into explicit rules
- **rekal-distill** — survey memory as four libraries (context / decision / rules / boundary)
- **rekal-census** — exhaustive full-corpus scan (coverage, not relevance)
- **rekal-wiki** — materialize committed `docs/wiki/` topic pages via PR

## Why this boundary

You don't recall a memory to learn what a function does — you read it. You
recall memory to learn *why it's shaped that way*. Rekal is the intent
layer, not a grep replacement; it earns its place by owning the one thing
the tree cannot hold — the reasoning that expired with the terminal window.
The tree stays the state layer, grep stays its primitive, and the router
keeps the line clean.

## Guidelines

- Triage first; one question, one substrate — routed, not stacked
- Gate episodes: below the confidence bar, silence beats noise
- Breadth answers come from the map, why answers from synthesis — don't
  force either through top-k retrieval
- Start small when drilling (`summary` role, snippet window); `--full` is a
  last resort
- Human turns carry intent; `human_steering` carries the moments decisions
  actually got made
- Cross-repo hits (`origin` set) are prior art, not this repo's conventions
- Report pointers (session, turn, commit) with every claim from memory
