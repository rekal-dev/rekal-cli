# Memory-frame scenario and acceptance matrix

This matrix enumerates the proposed feature's scenarios. It is not a claim that every scenario has been implemented. All executed cases use synthetic data.

Status legend:

- **Codec/Git passed:** tested through actual upstream codec/Git writer, with a prototype memory caller. Not the full CLI/database pipeline.
- **Model passed:** executed consumer/contract test with a fixture scorer, supplied identity/scope and synthetic session data.
- **Risk reproduced:** an undesirable behavior was reproduced by a model and supported by source inspection.
- **Earlier proof:** covered by the companion-file experiment, not counted as a new body test.
- **Integration pending:** requires real adapters, database, hooks, Git lifecycle or CLI work.
- **Evaluation pending:** requires real tasks/corpora, performance measurements or agent evaluation.

## A. Onboarding and capture

| ID | Scenario and required behavior | Evidence/status |
| --- | --- | --- |
| A01 | Existing native notes predate Rekal installation. Capture an observed baseline, retain source dates and exact bytes; do not claim they were created by the current commit. | Model passed: timestamp and onboarding eligibility cases |
| A02 | Install Rekal where no native memory exists. Ordinary session capture/search continues quietly. | Integration pending: native discovery absent-source path |
| A03 | Re-run onboarding with unchanged files. No duplicate content or artificial recency increase. | Earlier proof: idempotent capture; model passed: recency uses native time |
| A04 | Upgrade an already-initialized repo. New memory discovery/backfill runs even though current init refreshes managed assets and returns early. | Integration pending |
| A05 | Native notes have no creation or modification metadata. Preserve unknown rather than assigning installation time as fact time. | Model passed: unknown native time |
| A06 | Only memory changes, with no new conversation turns or code changes. Capture a memory-only checkpoint with no fabricated session. | Codec/Git passed for frame; actual checkpoint early-return integration pending |
| A07 | Nothing changes. Avoid repeated compression, embeddings and Git churn. | Earlier proof for content no-op; performance integration pending |
| A08 | Configured native memory directory differs from default paths, including profile overrides. Resolve actual provider configuration. | Integration pending |
| A09 | Native source is a pending proposal/inbox patch rather than adopted memory. Preserve status; do not silently present it as an adopted note. | Integration pending, particularly Gemini-like stores |
| A10 | A note is already tracked in Git. Reference its committed blob/path instead of duplicating its content in the ledger. | Integration pending; SOUL requirement |

## B. Scope, ownership and multiple agents

| ID | Scenario and required behavior | Evidence/status |
| --- | --- | --- |
| B01 | Shared project note written by Alice is read by Bob using another agent. Preserve origin; the receiving agent need not support the producer's native directory layout. | Model passed; fresh-clone codec/Git journey passed |
| B02 | Private project note concerns the same repo. Repository relevance does not make it shareable. | Model passed for visibility/direct fetch; real capture/export policy pending |
| B03 | Personal/account memory applies across repos. Do not automatically include it in a repository's team transport. | Earlier proof for serialization exclusion; native mapping pending |
| B04 | Same OS user has different work/personal provider profiles. Keep private sources distinct. | Model passed for supplied profile labels; actual account resolution pending |
| B05 | Two agents use the same relative filename. Namespace source identity; do not overwrite each other. | Earlier proof for source identities; provider mapping pending |
| B06 | Main-agent and persistent subagent memory use different native scopes. Preserve user/project/local distinction. | Integration pending |
| B07 | Worktrees share a native repo memory directory. Discover once and preserve shared-source semantics rather than attributing all notes to the committing worktree. | Integration pending |
| B08 | Main is committed while a feature worktree has contributed notes to the same native directory. Do not infer that all note content is merged. | Integration pending; merge gate alone is insufficient |
| B09 | A shared note cites a private or other-repo session. The citation cannot grant access or trigger automatic export of that session. | Model passed: cross-repo evidence filter; export-side reference policy pending |
| B10 | Teammate memory includes instruction-like text. Mark it as retrieved context; native adoption remains the agent's choice. | Model passed for labels/no write-back; real-agent behavior evaluation pending |

## C. Body transport and version compatibility

| ID | Scenario and required behavior | Evidence/status |
| --- | --- | --- |
| C01 | A memory record is encoded as its own standalone frame. Decode original record and retain original body prefix. | Codec/Git passed: standalone round trip |
| C02 | Memory is batched with a checkpoint using existing zstd compression. Session/checkpoint payload formats stay unchanged. | Codec/Git passed: batch round trip |
| C03 | A memory-only checkpoint has zero session refs. | Codec/Git passed |
| C04 | Frame carries an explicit checkpoint ID; payload commit and supplied checkpoint commit disagree. Reject the mismatch. | Codec/Git passed |
| C05 | An old reader sees a new memory member and known sessions. Known-type dispatch still decodes sessions and skips memory. | Model of legacy dispatch using actual decoders passed |
| C06 | The exact current Git writer writes a body containing memory, then a second update. Both memory revisions remain. | Codec/Git passed through unmodified writer |
| C07 | Team pushes to a remote and another developer clones. Body, dictionary, checkpoint commit and reconstructed memory are available. | Codec/Git passed using local bare remote |
| C08 | An old client performs full re-export from understood records. Demonstrate that memory is omitted; do not claim complete old-writer compatibility. | Risk reproduced: reconstruction model + ExportAllFrames source inspection |
| C09 | Updated repair/re-export knows memory records. Retain their payload and refuse unsupported frame/schema versions. | Model passed using actual codec; production DB re-export pending |
| C10 | A newer memory schema or unrelated future frame appears. Make partial support visible and prevent destructive repair. | Codec/model passed for refusal; user-facing diagnostics pending |

## D. Revision history and temporal meaning

| ID | Scenario and required behavior | Evidence/status |
| --- | --- | --- |
| D01 | Native note is edited. Append a revision and retain old bytes. | Earlier proof; codec/Git second-update test passed |
| D02 | Note is removed from a complete observed source. Remove it from the current view, retaining history. | Earlier proof; real adapter scan completeness pending |
| D03 | A directory read fails halfway. Do not interpret missing files as deletions. | Earlier proof for explicit incomplete-scan rejection; actual detector pending |
| D04 | Search finds a specific revision, then native memory changes. Opening the hit returns that saved revision. | Model passed |
| D05 | Query targets a commit predating onboarding. An old native timestamp cannot backdate a later observation into that commit's history. | Model passed with supplied ancestry set |
| D06 | Raw session continues after a checkpoint. Historical evidence must stop at its captured turn/byte extent. | Model passed with supplied extents; persistence/integration pending |
| D07 | A file is renamed or two paths contain identical content. Preserve content identity while recording path membership changes. | Earlier proof for content deduplication; rename semantics integration pending |
| D08 | Rebase, squash, cherry-pick or fork changes commit ancestry/repository identity. Define explicit mapping rather than relying on timestamps or similar remote URLs. | Integration pending |

## E. Consumer handshake and navigation

| ID | Scenario and required behavior | Evidence/status |
| --- | --- | --- |
| E01 | Agent starts or Rekal upgrades. Thin guidance advertises available capabilities and schema; rich context stays on demand. | Model passed for capability response; existing skill/CLI integration pending |
| E02 | Consumer understands sessions but not memory schema. Keep raw session search working and report unsupported memory. | Model passed |
| E03 | Search returns a memory hit. Include kind, stable ref, revision/snapshot, origin and both time meanings with a bounded snippet. | Model passed |
| E04 | Agent opens the memory hit. Fetch exact archived content, not today's file at the same native path. | Model passed |
| E05 | Memory explicitly cites a session turn. Translate emitted citation syntax to accepted evidence-anchor syntax mechanically. | Model passed; production citation bridge pending |
| E06 | Memory has no citations. Report that fact; keep ordinary raw-session search available. | Model passed |
| E07 | Citation references unavailable/deleted/unshared raw material. Return unavailable rather than manufacture support. | Model passed |
| E08 | Different memories cite several turns or sessions. Fetch them under one shared text-byte and slice budget. | Model passed for multiple sessions and ranges |
| E09 | Citation windows overlap. Deliver overlapping source bytes once per page. | Model passed |
| E10 | User begins with a raw session hit. A reverse hop identifies visible memory revisions explicitly citing that passage. | Model passed |
| E11 | Reverse hop encounters duplicate citations in one note or private notes. Return each visible note once, with a bounded result count. | Model passed |
| E12 | Index is deleted/rebuilt, changing short local handles. Canonical references still resolve. | Model passed; actual CLI short-handle integration pending |
| E13 | Evidence is truncated. Return canonical continuation; a changed referenced source invalidates stale continuation. | Model passed |
| E14 | Agent asks about a code file/commit, then wants memory and supporting history. Join structural file/commit associations without claiming semantic support. | Integration pending |
| E15 | Agent elects to adopt useful information into its own native memory. It can retain a canonical source reference; Rekal does not overwrite native notes automatically. | No write-back in prototype; native adoption flow is agent-owned and untested |
| E16 | Links form a cycle or very large neighborhood. Every tool call stays one bounded hop; no implicit recursive expansion. | Model passed for bounded reverse hop; agent-loop evaluation pending |

## F. Ranking, weights and configuration

| ID | Scenario and required behavior | Evidence/status |
| --- | --- | --- |
| F01 | Memory is enabled. Existing session ranking remains unchanged in its own result block. | Model passed; production additive-block integration pending |
| F02 | Keyword versus semantic weights change. Reorder eligible candidates without rebuilding stored records/index. | Model passed using injected semantic scores; production ranking pending |
| F03 | Semantic embeddings are absent, warming or incompatible. Preserve keyword retrieval and expose degradation where useful. | Model passed for absent-score fallback; real embedding lifecycle pending |
| F04 | Old notes are imported today. Use source modification semantics for any freshness signal, not import time. | Model passed |
| F05 | Source time is unknown. Do not invent freshness. | Model passed |
| F06 | Identical memory is copied between teammates/agents. Group the hit, retain origins and avoid accumulating votes. | Model passed for identical content |
| F07 | Distilled memory and raw evidence derive from the same source; paraphrases differ in bytes. Do not treat them as independent corroboration. | Evaluation/integration pending beyond exact-copy dedup |
| F08 | Memory is short and polished but outdated; raw evidence contradicts it. Preserve both and let the agent reason from evidence. | Evaluation pending; no automatic truth winner |
| F09 | Machine defaults, repo preferences and request overrides differ. Apply documented precedence; explicit zero must disable the intended signal. | Model passed; prototype keys are not shipped configuration |
| F10 | Config contains negative/non-finite weights, unknown fields, invalid budgets or no active retrieval layer. Validate and surface a useful diagnostic/fallback. | Model passed for rejection; match current CLI fallback behavior during integration |
| F11 | A ranking config requests a very large boost for a private or irrelevant hit. It cannot widen visibility or create candidate eligibility. | Model passed |
| F12 | User disables memory results or sets their limit to zero. Ordinary raw retrieval is unaffected. | Model passed |

## G. Failure, concurrency and operations

| ID | Scenario and required behavior | Evidence/status |
| --- | --- | --- |
| G01 | Tail contains a partial frame envelope or compressed payload. Reject/report partial data rather than accepting silent truncation. | Codec/Git passed via stricter prototype wrapper |
| G02 | Memory payload exceeds the envelope's advisory uncompressed length. Large valid note still round-trips. | Codec/Git passed; hard compressed limit also covered by upstream codec tests |
| G03 | Corruption affects a member or an entire compressed batch. Report what is unavailable; understand that a corrupt shared zstd batch can affect all its members. | Upstream codec corruption tests pass; recovery/diagnostic integration pending |
| G04 | Two processes capture the same source simultaneously. Serialize local writes and avoid duplicate observations or mismatched checkpoint associations. | Integration pending |
| G05 | Two machines advance the same author's branch independently. Reject stale updates without overwriting the winner; define reconciliation. | Earlier proof for compare-and-swap; actual reconciliation pending |
| G06 | Checkpoint succeeds but push fails/offline. Retain local data and retry without duplicates or marking export complete prematurely. | Integration pending |
| G07 | Crash occurs during capture/database commit/body write/index rebuild. Recover a consistent durable state and rebuild derived state. | Integration pending; no crash injection performed |
| G08 | Native agent writes while capture reads. Detect instability or label the observation honestly; no false atomic-directory guarantee. | Integration pending |
| G09 | Symlinks, relative traversal, huge files or provider path changes appear. Follow supported source rules, bound reads and avoid unexpected scope expansion. | Earlier proof for normalized archive paths; real filesystem hardening pending |
| G10 | User disables memory capture or uninstalls Rekal. Remove derived local integration transparently; explain that earlier Git history is retained. | Integration pending |

## H. Quality, cost and product acceptance

| ID | Scenario and required behavior | Evidence/status |
| --- | --- | --- |
| H01 | New teammate's first useful question after cloning. Shared memory reduces orientation work while supporting raw evidence drill-down. | Traversal mechanism passed; task success/time evaluation pending |
| H02 | Existing user's first install over a large native store. Bound onboarding time and keep keyword retrieval available while embeddings catch up. | Performance evaluation pending |
| H03 | Query asks current convention versus historical why versus a precise date. Evaluate routing and evidence use by question type. | Evaluation pending |
| H04 | Raw-only, memory-only, memory-routes-to-raw, and memory-plus-raw answerers are compared under controlled answering model and evidence budgets. | Evaluation pending |
| H05 | Memory generation precedes benchmark questions and cannot read answers/evidence labels. Freeze generated notes and record their cost. | Evaluation protocol only; no LoCoMo run performed |
| H06 | Correctly grounded answer needs several notes and sessions. Measure total tokens, tool calls, evidence coverage and answer accuracy, including failed/repeated hops. | Evaluation pending |
| H07 | Git history, body size and local index grow over months with unchanged notes, revisions and many teammates. Measure bytes per changed note and latency. | Performance evaluation pending |
| H08 | Production release candidates run the full adapter/storage/export/import/rebuild/CLI matrix on supported platforms and mixed versions. | Release gate; not satisfied by this prototype |

## Release acceptance order

1. Real adapter baseline capture, explicit source scope and memory-only checkpoint integration.
2. Durable storage, frame import/export and full rebuild with immutable revision/checkpoint identities.
3. Search/open/evidence/backlink operations with stable canonical references, bounded output and no native write-back.
4. Real re-export compatibility policy, failure recovery and concurrent-writer behavior.
5. Representative task evaluations before changing ranking defaults or claiming benchmark gains.

The architectural idea is demonstrated. Full production integration and measured retrieval improvement remain separate acceptance gates.
