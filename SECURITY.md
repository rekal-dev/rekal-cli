# Security Policy

## Reporting a vulnerability

Report privately through GitHub's private vulnerability reporting:

**[Report a vulnerability](https://github.com/rekal-dev/rekal-cli/security/advisories/new)**
— or the *Security* tab → *Report a vulnerability*.

Do not open a public issue for a security report. Do not post it in Discord.

Include what you have: affected version (`rekal version`), platform, the
commands or config that trigger it, and what an attacker gains. A reproduction
is welcome but not required to file.

**What to expect:** acknowledgement within 3 business days, an assessment within
10, and a fix released before public disclosure. We will credit you in the
advisory unless you ask us not to.

## Supported versions

Rekal is pre-1.0. Only the **latest release** receives security fixes. Upgrade
before reporting against an older tag.

## The security model

Rekal has no server, no API, no account, and no telemetry. It does not phone
home — not even crash reports. The data touches two places: the local
filesystem and the git remote you already trust with your source code.

There is nothing to breach because there is nothing to connect to.

That shape means most classes of vulnerability common to memory systems do not
apply here. The classes that *do* apply are specific, and they mostly concern
one thing: **material reaching the wire that should have stayed local.**

## In scope

Report these. Severity is roughly in this order.

### Data escaping to the shared branch

The orphan branch is pushed to a remote your whole team can read, and the wire
format is append-only. Anything that reaches it is durable and distributed.

- **Redaction failure.** `scrub/` misses a secret, credential, or token in a
  transcript and it lands in `data.db` and then on the branch. Highest
  severity — a leaked secret here is both shared and hard to retract.
- **Path anonymization failure.** Local filesystem paths that should be
  anonymized survive into the wire frame.
- **Merged-only gate bypass.** `transport.filterMerged` exports a checkpoint
  whose commit never merged to the default branch — unmerged or abandoned work
  reaching teammates. See [docs/design/merged-only-sharing.md](docs/design/merged-only-sharing.md).
- **Cross-repo leakage.** Sessions folded in by `local_import.go` are
  index-only and structurally unpushable. A path that lets imported material
  reach `data.db` would put *other repositories'* conversations on this repo's
  shared branch.
- **Local-only records reaching the wire.** `recall_edges`,
  `checkpoint_state` and `merge_gate_cache` are deliberately never wired. `recall_edges` holds query
  text and access patterns; exporting it would publish what each developer
  searched for.
- **Config on the wire.** `.rekal/config.json` and
  `~/.config/rekal/config.json` are gitignored and local-only. They may hold an
  API key. Anything that commits, pushes, or syncs them is a vulnerability.

### Credential handling

- An API key from the `embedding` config (including `$VAR` expansion and
  `api_key_env`) appearing in `data.db`, a wire frame, a log, `--explain`
  output, or scoring-lineage NDJSON.
- The opt-in HTTP embedding backend sending text anywhere when it has not been
  explicitly configured. Sending to a *configured* endpoint is the documented
  behavior of that opt-in; sending without configuration is a bug and a leak.

### Untrusted input

- **Malicious wire frames.** `rekal sync` decodes frames authored by whoever
  can push to the remote. Decompression bombs, allocation blowups, or memory
  unsafety in `codec/` decode paths are in scope.
- **Malicious transcripts.** Session adapters parse JSON and JSONL written by
  agent tooling. Crashes are low severity; anything reaching outside the parse
  (file writes, path traversal on discovery, command execution) is not.
- **Path traversal** anywhere a stored path is used to read or write.

### Local integrity

- Anything that writes to `data.db` outside the append-only path, or mutates a
  row already written. The ledger's immutability is a correctness property, and
  breaking it breaks the basis for trusting the record.
- Insecure permissions or predictable paths on `.rekal/` state — `daemon.lock`,
  `embed.lock`, `recall-log.ndjson`, `embed-cache.db` — that let another local
  user read or poison a store.
- Anything that causes `rekal init` to write outside the repository and
  `~/.config/rekal/`, or `rekal clean` to leave credentials behind.

## Out of scope

- **Recall quality.** Missing, irrelevant, or wrongly ranked results are bugs,
  not vulnerabilities. Open an issue.
- **An agent acting on recalled content.** Rekal returns your team's own past
  conversations. Deciding what to do with them belongs to the agent and its
  operator.
- **Secrets already in your git history.** Rekal captures conversations, not
  your repository's existing contents.
- **`rekal query --sql`.** It executes SQL you supply against your own local
  index by design. That is the feature.
- **Requiring local machine access.** An attacker who can already read your
  filesystem can read `.rekal/` the same as your source.

## If a secret reaches the ledger

Be direct about the sequence, because immutability makes the usual instinct the
wrong one:

1. **Rotate the credential first.** Assume it is compromised. Everyone who
   fetched the branch has a copy, and no history rewrite changes that.
2. **Then clean the branch.** The `rekal/<email>` orphan branch is an ordinary
   git branch — it can be rewritten or deleted and force-pushed. Append-only is
   a guarantee of the *wire format*, not a technical bar to deleting a branch.
   Rewriting it is an exceptional remedy, not a routine operation.
3. **Report the redaction gap** through the process above, so the miss is fixed
   for everyone rather than for one store.

## Scope of this policy

This policy covers the `rekal` binary and this repository, including the
setup-only Claude Code plugin under `plugin/`. Vulnerabilities in dependencies
(DuckDB, llama.cpp, the Go toolchain) should be reported upstream; tell us too
if Rekal's usage makes them exploitable in a way the upstream advisory does not
cover.
