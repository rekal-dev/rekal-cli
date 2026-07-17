# The knowledge layer (files at HEAD as memory)

**Status:** implemented, v1 (2026-07). Index the repo's tracked prose files
(markdown and plain text) into recall as a **knowledge layer**: file hits at
HEAD returned alongside session hits, as pointers, cross-linked through the
sessions that touched them. Derived entirely in `index.db` — zero bytes on
the wire, no new commands, no new config.

v1 shipped the keyword-fresh layer (steps 1–5 of the sketch): chunker
(`cmd/rekal/cli/knowledge/`), `knowledge_chunks` + guarded FTS in `index.db`
(`db/knowledge.go`), blob-SHA incremental refresh at recall and full build at
`rekal index` (`cli/knowledge_index.go`), the `knowledge` output block with
provenance edges (`search/knowledge.go`), and the router-skill knowledge
stratum.

v1.1 ships the semantic half: chunk vectors in `knowledge_embeddings`
(content-hash + model keyed), built through the same `.rekal/embed-cache.db`
the session layer uses — a moved section or reverted edit re-fills from
cache, never from the model. Structural rebuilds (`rekal index`/`sync`)
chunk + FTS synchronously, then spawn background `rekal embed` for vectors
(budgeted bites, DuckDB lock released between passes — see
`docs/spec/command/embed.md`). The post-commit hook still embeds up to 256
chunks per commit. Recall never embeds — latency stays pure read. At query
time the knowledge score blends *absolute* saturating BM25 with cosine
similarity using the session ranking's keyword/semantic split
(`weights.layers2`) — never max-normalized within the candidate set (that
made every top hit ≈1.0 and hid weak matches from the router). The query
vector is shared from the session semantic pass so one recall embeds its query
once. Cosine runs over BM25 candidates (≤100 hashes); when BM25 returns
nothing, a capped semantic-only fallback (≤2000 embeddings) can still surface
prose — larger corpora wait for ANN. Semantic-only extras *alongside* BM25
hits stay deferred. Every rung fails soft: no vectors,
model mismatch, or an old index.db degrades to keyword-only. Orphaned vectors are pruned when their content leaves every chunk.
The RekalBench knowledge task set (step 6) remains the follow-up.

## Problem

Recall answers *"what did we discuss and decide?"* from sessions. It cannot
answer *"what do we currently know?"* — the question agents ask most.

Today that question falls between substrates. The router (the `rekal` skill)
triages tree (grep, present tense) vs ledger (rekal, past tense) vs map
(structure). But *"what is our convention for token refresh?"* is none of
these: grep finds symbols, not knowledge; the ledger finds the debate, not
the conclusion; the map finds the structure, not the content. The conclusion
lives in prose files — `docs/`, `CLAUDE.md`, wiki pages — and Rekal cannot
rank or return them.

The gap matters twice over:

1. **In code repos**, the semantic memory (docs, conventions, design notes)
   is invisible to recall even though the episodic memory (sessions) behind
   it is fully indexed.
2. **In knowledge repos** — an Obsidian vault, a team handbook, a docs-only
   repo — the files *are* the primary memory and Rekal today offers almost
   nothing. Yet everything else already works there: `init` installs, the
   hook captures sessions that edit notes, sync shares the ledger. Only
   recall is blind to the content.

## The insight: the repo is the brain

Fleet-memory systems (e.g. MemClaw) build a database of "current truths"
next to the work, then need machinery to maintain it: superseding,
contradiction detection, decay, trust tiers, audit. Git provides every one
of those structurally:

| They build | Git already is |
|---|---|
| Superseding + contradiction resolution | Editing the file; HEAD is current truth by construction |
| Memory history / audit trail | `git log` |
| Trust tiers, governance | PR review, CODEOWNERS, branch protection |
| Fleet sharing under governance | Remotes + the review gate |
| Decay / freshness | The HEAD-vs-history split — reading HEAD *is* the filter |

So Rekal's memory model is already two-layer; only one layer is indexed:

- **Episodic memory** — the ledger (`data.db`). What happened and why.
  Immutable, captured, indexed. Done.
- **Semantic memory** — the repo's prose files at HEAD. What we currently
  believe. Versioned, reviewed, agent-readable. **Not indexed.**

The knowledge layer closes that gap. One sentence: **files at HEAD are what
the team knows; the ledger is why they know it; recall returns both, joined.**

Deliberate memory already has a write API — editing a file and committing.
No `rekal remember` command, ever: the repo write *is* remember, the
checkpoint automatically attaches the why-trail, and review is the admission
gate.

## Principle

- **Index prose, not code.** Tracked `.md` and plain-text files only. Code
  belongs to the tree substrate — the agent greps and reads it directly;
  hybrid search tuned for prose would be a worse grep. Tracked-only means
  gitignored/vendored/generated files are excluded for free, and the layer
  adds zero wire cost — the files are already in git.
- **Pointers, not payloads.** The `summary_turn_index` rule verbatim: a
  knowledge hit is a path + anchor + line range + short snippet, never file
  content. The index *finds*; the agent's own Read tool *serves* — live from
  HEAD, so served content is never stale even when the index is a commit
  behind.
- **Separate blocks, cross-linked.** Knowledge hits and session hits have
  different epistemic status (current truth vs history) and different next
  actions (read the file vs drill the session). They are never interleaved
  into one ranked list; they are joined by edges.
- **Additive output.** The `sessions` block stays byte-identical to today.
  The `knowledge` block is new and empty in repos with no prose files. Same
  discipline as `facet_boost=0` and `--explain`.
- **No new config, no new commands.** Default on. One good chunking default.
  The soul's refusal list applies: no options where a good default exists.

## Mechanism 1 — chunking

The unit of indexing is the **heading section**: everything under one
markdown heading until the next same-or-higher heading. Each chunk carries:

- **Breadcrumb trail** (`Auth Guide > Token handling > Refresh rotation`),
  prepended to the chunk text before embedding — a lone paragraph is
  semantically ambiguous; with its trail it embeds as what it means (the
  same trick as Nomic task prefixes).
- **Line range at HEAD**, so every pointer is directly Read-able.
- **Content hash** as identity — the key into the existing embedcache.

Oversized sections (heading-free walls of text) split at paragraph
boundaries under a size cap; trailing fragments merge into their parent.
Plain-text files without headings chunk by paragraph with the filename as
breadcrumb. Binary and oversized files are skipped. Chunk text is run through
`scrub.SanitizeText` before hashing and again at insert (DuckDB rejects
invalid-UTF-8 VARCHAR binds — one bad incident-log `.txt` must not abort the
whole knowledge-layer transaction).

Content-hash identity makes reindexing surgical: edit one section of a
500-line doc and only that chunk re-embeds; every other chunk is a cache
hit. Move a section between files: same hash, free.

## Mechanism 2 — indexing and freshness

Git content-addresses every tracked file (`git ls-tree -r HEAD` yields blob
SHAs), which gives an exact, free change detector:

- **Watermark, don't poll.** The knowledge index records the commit SHA it
  was built at — the same pattern as the MAP skill's `.rekal/map.md`
  watermark. On recall (and in the post-commit hook, hard-timeboxed like
  `embedhttp` so a commit can never stall), diff watermark..HEAD and reindex
  only touched files. Deletes and renames fall out of the same diff.
- **Two-speed layers.** BM25/FTS over chunk text is milliseconds — rebuilt
  eagerly, always fresh. Deep embeddings are the expensive part and they are
  content-hash cached (`embedcache.db`); if the embedding backend is absent
  or slow the semantic layer **fails soft**, exactly like the facet FTS
  index — a brand-new doc is findable by keyword before it is findable
  semantically, and recall never blocks on freshness.
- **Stale is bounded and harmless.** Worst case a stale watermark degrades
  ranking by one commit's worth of edits; it can never serve stale content,
  because content is never served — the agent reads HEAD from disk.

Everything lands in `index.db` (plus vectors in `embed-cache.db`). Nothing
touches `data.db`, nothing crosses the wire, `clean` already removes it all,
and a rebuild from scratch re-derives it — the layer is a pure view.

## Mechanism 3 — scoring: chunks scored, files returned

A direct mirror of session ranking, where **turns** are the scoring unit and
**sessions** the result unit. Here **chunks** are scored (same hybrid
BM25 + LSA + deep-embedding mix, same weights machinery) and **files** are
returned: a file's score is driven by its best chunk with a coverage bonus
when multiple chunks hit (a doc matching in three sections is more likely
*the* doc than three docs matching once). The top chunk supplies the snippet
and anchor; runners-up become additional anchors.

Knowledge and session scores are never merged into one ordering — a 0.8
against file prose and a 0.8 against session turns are not the same 0.8, and
interleaving them erases the epistemic distinction the agent needs.

## Mechanism 4 — output: two blocks, one edge

```json
{
  "knowledge": [
    {
      "path": "docs/auth.md",
      "anchor": "## Refresh rotation",
      "lines": "41-78",
      "snippet": "Refresh tokens rotate on every use; the old token is...",
      "also": [{"anchor": "## Failure modes", "lines": "102-131"}],
      "last_modified": "a3f9c21",
      "sessions": ["01J8...", "01J9..."],
      "score": 0.81
    }
  ],
  "sessions": [ "...unchanged..." ]
}
```

The `sessions` array on each knowledge hit is the provenance edge: the
sessions that touched that file, via the existing `files_index` join already
built for `--explain`. The agent gets *what we know* and *why we believe it*
as a pair — two blocks, one edge between them.

## The derived entity graph

The knowledge layer completes a graph Rekal already stores the edges for —
**inferred from structure, never extracted by an LLM**:

- **Nodes:** files/chunks (knowledge), sessions (episodes), commits (time).
- **Edges:** session→file (`files_index`, "this session touched this file"),
  session→commit (`git_sha`), file↔file (co-occurrence through shared
  sessions — the same signal `rekal-wiki` clusters on), chunk→file→directory
  (containment, free from paths).

Every edge is deterministic, local, and re-derivable on index rebuild — no
entity extraction, no similarity-threshold merging, no frozen taxonomy. The
graph is a *view over the ledger and the tree*, and it powers, without new
machinery:

- **Related-knowledge joins** on recall (the `sessions` edge above, and its
  inverse: session hits pointing at the docs they shaped).
- **Consolidation debt** — the compound step made visible: files whose
  subject matter has heavy ledger activity *after* the file's last commit
  ("twelve sessions about token refresh since `docs/auth.md` last changed")
  are behind the ledger. Computable from watermarks and `files_index` alone;
  feeds `rekal-wiki`/`rekal-reflect` as proposed PRs, where review is the
  admission gate.
- **Wiki clustering** (`rekal-wiki`) gains file-node anchors instead of
  reconstructing co-occurrence from sessions alone.

## The zoom ladder

Zoom is navigation across pointers, not an API. Four rungs, each one
pointer-follow from its neighbors:

1. **Repo level** — the MAP (`.rekal/map.md`): structure, greppable anchors.
2. **Corpus level** — the `knowledge` block: ranked file hits with anchors.
3. **Section level** — the agent Reads `docs/auth.md:41-78` (live HEAD;
   whole file if it wants context; the breadcrumb names the parent).
4. **Provenance level** — the `sessions` edge into the ledger, drillable
   with the existing tools (`--role summary`, `query`).

Rekal never ships content that the next rung would deliver better: the index
finds, the filesystem serves, the ledger explains.

## Router change

The ledger substrate becomes a **memory substrate with two strata**. Triage
gains one row:

- present-tense code fact → tree (unchanged)
- **current knowledge / convention / decision-state → knowledge stratum**
  (file hits at HEAD, provenance edges into sessions)
- past-tense why / history / what-was-tried → ledger stratum (unchanged)

A non-empty knowledge block **routes to HEAD prose** (`KNOWLEDGE`) — it does
not clear the episode inject gate. HEAD outranks sessions by sending the
agent to Read the pointer, not by injecting weak episodes. Full skill
topology (tip → scripts → references): [`skill-router.md`](skill-router.md).
MINE gains a knowledge dimension in its `scope × signal × role` decomposition.

## Onboarding cost (measured)

Synthetic 2,000-file markdown vault (~7,000 chunks), linux/amd64 container:

- **First recall after init** (cold: full session index + full knowledge
  build + FTS): ~6.3s, one-time — the existing "index not built,
  rebuilding..." moment, slightly longer.
- **Steady-state recall** (watermark hit): ~0.25s.
- **Post-commit hook, one doc changed** (incremental blob-diff refresh):
  ~0.1s — imperceptible; the hook stays quiet.
- **Post-commit hook with no sessions to capture** exits before the index
  update entirely, so a giant initial import commit costs the hook ~0.2s;
  the full build lands on the first recall instead.

The two mechanisms that keep these numbers flat as corpora grow: one
`git cat-file --batch` process for all blobs (never per-file `git show`),
and chunk inserts wrapped in a single transaction (never per-row
autocommit). A hook that does carry sessions *and* a huge prose delta pays
the full build once (~seconds); if that ever bites in practice, the refresh
can adopt `embedhttp`'s hard-timebox pattern and defer to recall.

## Init and hooks: no surface change

`init` is untouched — store, hooks, orphan branch, skills, one CLAUDE.md
line. The first knowledge index build rides the existing
auto-rebuild-on-recall path (or `rekal index`); the post-commit hook gains a
timeboxed incremental refresh of touched files. No detection, no modes:
`rekal init` in an Obsidian vault does exactly what it does in a code repo —
and simply works better there, because the whole repo is prose. That is the
product expansion in one line: *point Rekal at any git repo and it is agent
memory; files at HEAD are what you know, the ledger is why.*

## Soul check

| Question (SOUL.md) | This design |
|---|---|
| Preserves immutability? | Yes — `data.db` untouched; the layer is a rebuildable view in `index.db`. |
| Intent stays next to the code? | Strengthened — knowledge and its why-trail are joined in one answer. |
| Thin on the wire? | Zero bytes — indexed files are already in git; the index never ships. |
| Data stays within git and the local machine? | Yes — all compute local; embeddings via the existing (local or opt-in configured) backend and cache. |
| Simple — zero config? | No new commands, no new flags, no config keys; one chunking default; default on. |
| Transparent — see and remove? | `clean` already removes `index.db`/`embed-cache.db`; nothing new persists anywhere else. |
| Agent gets what it needs? | Pointers + snippets + provenance edges, token budget stays with the agent, live-HEAD reads. |

## Non-goals

- **No code indexing.** Code is the tree substrate; grep wins there.
- **No content in output.** Pointers only; the filesystem serves.
- **No merged ranking.** Knowledge and sessions stay separate blocks.
- **No LLM entity extraction.** The graph is derived from stored edges only.
- **No `rekal remember`.** The file write is the write API; review is the
  admission gate.
- **No new config surface** in v1 (a `weights.knowledge_*` key may follow
  once RekalBench says what it should default to).

## Implementation sketch

1. `session`/new `knowledge` package: markdown/plain-text chunker (heading
   sections, breadcrumbs, size caps, content hashes). Pure functions, unit
   tests.
2. `db`: knowledge tables in `index.db` (chunk text, path, anchor, lines,
   blob SHA, watermark commit SHA); FTS index over chunk text, guarded like
   the facet index; embeddings through `embedcache.db` keyed by chunk
   content hash.
3. `index`/recall auto-rebuild: full build at first index; incremental via
   watermark..HEAD diff. Post-commit hook: timeboxed incremental refresh.
4. `search/`: chunk scoring through the existing hybrid pipeline;
   file-level aggregation (best chunk + coverage bonus); `knowledge` block
   in recall JSON with `files_index` session joins. `sessions` block
   byte-identical without the layer.
5. Skills: router (`rekal`) triage row for the knowledge stratum; HUNT
   confidence rule; MINE knowledge dimension. `rekal-wiki`/consolidation
   debt as a follow-up.
6. Bench: RekalBench task set for knowledge queries ("what is our convention
   for X") before tuning any weights; the layer's contribution measured, not
   assumed.
7. Docs: `spec/command/recall.md` (output schema), `spec/command/index.md`,
   `CLAUDE.md`, `docs/agent-metadata.md` (knowledge block).

Steps 1–4 are independently shippable; the layer is useful from step 3
(keyword-fresh, embedding-cached) before the router even changes.
