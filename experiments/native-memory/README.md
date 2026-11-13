# Native memory experiments

Recovered executable research for optional native-agent memory capture in Rekal.
These are isolated prototypes, not an enabled CLI feature or a stable wire format.

## Contents

- `companion/`: append-only memory snapshots in a separate binary, scope and timestamp preservation, deterministic index rebuild, and Git writer compatibility experiments.
- `frames/`: typed snapshots inside `rekal.body`, consumer capability discovery, historical revision fetch, explicit citation links and backlinks, bounded evidence, configurable ranking experiments, and an 84-case scenario matrix.

All example notes and identities are synthetic. No captured conversations, native personal memory, credentials, or local indexes are included.

## Validation

Recovery validation reran both Go harnesses successfully, all 34 frame consumer/integration tests, and all 20 companion archive/Git tests. The copied upstream codec and new memory-frame tests also pass. Included result files are the original experiment evidence; rerunning the Python scripts regenerates their reports.

With Git, Python 3.11+ and Go 1.25.6+ installed, run:

```sh
sh experiments/native-memory/test.sh
```

The nested Go modules deliberately keep experiment code out of the production CLI build. Their copied codec and writer implementations are pinned to upstream `16cc899274af026f8a8d6cf8216d391be9aa8489`; provenance and upstream licensing are retained.

## Production work still outstanding

The earlier unfinished integration checkout was not recovered. These packages do not implement native-source discovery, production database migrations, checkpoint capture, import/export integration, memory results in existing retrieval paths, or end-to-end production rebuild. They do not establish retrieval-quality improvements.

Implementation should use an optional adapter interface for discovery and reading; preserve immutable revisions separately from the current searchable view; enforce sharing scope before serialization; keep native modification time distinct from observation time; and refuse unsupported formats during rebuild/re-export. Explicit citations may yield deterministic links, but notes without citations must not acquire invented provenance.

See `frames/SCENARIOS.md` for implemented proof cases versus remaining integration and evaluation work. Mixed old/new writers remain a compatibility decision: new guards cannot constrain an old executable.

## Integration boundary

Follow `SOUL.md`: integrate memory as another source behind existing Rekal workflows. This work does not require a new CLI command family, a mandatory consumer handshake, a service, or a scripted agent retrieval sequence.

- Capture: optional native-memory adapters discover and read existing notes during the existing capture/checkpoint lifecycle, including onboarding and memory-only changes.
- Storage and sync: append immutable revisions with source scope, timestamps and checkpoint associations; reuse Git transport. Sharing follows explicit source policy, never inferred from a note merely mentioning the repository.
- Local index: rebuild searchable memory and explicit citation/backlink projections from the archive. Indexes and embeddings stay local.
- Consumption: include labelled memory evidence and stable revision references through existing search, drill and bounded evidence paths. Resolve links mechanically where available; let the agent decide what to follow and whether the note applies.
- Ownership: native agents retain ownership of their memory. No automatic native-memory write-back or interception of agent distillation is required.

The Python consumer methods are executable test contracts, not proposed CLI commands. Their ranking knobs are experiment controls, not a requirement to expand shipped configuration. Production integration remains outstanding; this document defines its scope rather than claiming it is implemented.
