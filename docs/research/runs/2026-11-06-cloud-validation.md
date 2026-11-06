# Cloud setup and Codex capture validation — 6 November 2026

Source baseline: `4550e602eaa347d7afe28cf7f38f25f8c43f9afc`.
Platform: Linux amd64, Go 1.25.6, golangci-lint 2.8.0, llama.cpp b8157
with `GGML_NATIVE=OFF`; the LFS model was checked against its SHA-256.

## Local results

- Built and installed Rekal; initialized its store and composed Rekal capture
  with the existing Git LFS hook. Team sync imported 133 remote sessions.
- Codex parser/discovery tests passed, including custom `CODEX_HOME`, active
  and archived sessions, sibling-repo exclusion, nested metadata IDs,
  response-only messages, mirrored-stream deduplication and oversized records.
- Lint passed with zero issues. The race suite passed the integration package
  and all packages other than nomic's two Unix-socket bind tests. This host
  denies Unix sockets (`socket: operation not permitted`); those two failures
  are environment restrictions and are not reported as a green full gate.
- Four Python regressions passed: foreign-owner release archive installation,
  explicit JSON plus the shipped digest, legitimate SILENCE exits, and command
  errors kept distinct from SILENCE. The live shim also returned INJECT with
  two diagnostic calls and no removed Python route script.
- The real local model produced finite 768-dimensional embeddings in a direct
  smoke check. Production daemon-based semantic recall is unavailable on this
  host because of the socket restriction. No answer-quality score is claimed.
- A manually labelled export of this conversation's visible user requests and
  full technical review was captured through the production Codex adapter.
  Five turns were checked for exact text round-trip and the session surfaced
  through the public recall CLI. A subsequent visible progress message exercises
  append-on-recapture in the commit hook. Original unavailable timestamps are
  omitted; export time is not presented as original message time. No system,
  developer, private reasoning or unrelated personal context was exported.

## Release scope

This patch repairs capture, installation and benchmark execution compatibility.
Passage embeddings, multiple retrieval anchors and batched evidence assembly
remain experiments requiring a matched evaluation; they are not shipped here.
The complete GitHub CI/lint/license gates must pass before merge and release.
