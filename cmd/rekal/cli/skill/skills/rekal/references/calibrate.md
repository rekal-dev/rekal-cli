# CALIBRATE — local recall profile (agent-driven)

Use when memory feels wrong on **this repo** and you have miss signals
(smoke manifests, scoring lineage, or repeated SILENCE on answerable asks).
Not for everyday HUNT. Not a silent auto-tune.

Design: `docs/design/skill-adaptive-recall.md`.

## When

- Industry-bench / chat haystack: deep ranks, low mass, skill ≈ stock.
- Coding repo: only after several clear false SILENCE / wrong-rank cases.
- Never on a frozen **test** split you will report — tune on **dev** only.

## Pipeline

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"

# 1. Propose from smoke and/or scoring lineage (or --profile chat|coding|multi-hop)
python3 "$ROOT/scripts/calibrate-recall.py" \
  --from-smoke path/to/runs/smoke/locomo-conv-26-skill-20260717 \
  --from-lineage .rekal/scoring-lineage.ndjson \
  --print-env --print-cli

# 2. Review proposal (stderr with --print-cli). Prefer per-query --weights:
W=$(python3 "$ROOT/scripts/calibrate-recall.py" --profile chat --print-cli 2>/dev/null)
eval "$(python3 "$ROOT/scripts/calibrate-recall.py" --profile chat --print-env 2>&1 >/dev/null | grep '^export')"
rekal --weights "$W" "<q>" | python3 "$ROOT/scripts/recall-route.py"

# 3. Optional sticky local defaults (gitignored) — only when you want every
#    bare `rekal` in this repo to keep the profile without passing flags:
python3 "$ROOT/scripts/calibrate-recall.py" --profile chat --apply --print-env
```

`--print-cli` prints one line: the weights JSON for `rekal --weights '...'`.

`--apply` (optional) writes:

- `.rekal/config.json` → `weights` (query-time; no reindex)
- `.rekal/calibration/active-profile.json` (audit: profile + sha)

Prefer `--weights` for skill flexibility. Use `--apply` only for a sticky
repo default. Do not commit calibration into the shared ledger path.

With `scoring_lineage.enabled`, each recall appends to
`.rekal/scoring-lineage.ndjson` (relative path): `query.weights`
(+ `weights_source: "cli"` when you passed `--weights`) and
`candidate.contrib`. That closes the loop next to calibration/.

## Profiles

| Profile | Use |
|---|---|
| `coding` | Default ADLC — high mass / conf bars, facet on |
| `chat` | Pure dialogue — MASS_MIN=0, lower CONF, less facet |
| `multi-hop` | Synthesis / WHY — more LSA, chat-like gates |
| `auto` | Infer from `miss_reason` / category histograms |

## After calibrate

Re-run the same queries with `--weights`. If still `true_miss`, calibration
will not help — need WHY route or better retrieval, not softer gates. Cite
`proposal_sha256` (stdout or `active-profile.json`) in any run note.
