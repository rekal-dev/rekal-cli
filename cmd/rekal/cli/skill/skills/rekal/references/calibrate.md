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

# 1. Propose from smoke / miss logs (or --profile chat|coding|multi-hop)
python3 "$ROOT/scripts/calibrate-recall.py" \
  --from-smoke path/to/runs/smoke/locomo-conv-26-skill-20260717 \
  --print-env

# 2. Review JSON on stdout. If it matches the failure mode, apply LOCAL only:
python3 "$ROOT/scripts/calibrate-recall.py" \
  --from-smoke path/to/run \
  --apply --print-env

# 3. Export gates for this shell, then re-hunt:
eval "$(python3 "$ROOT/scripts/calibrate-recall.py" --profile chat --print-env 2>&1 >/dev/null | grep '^export')"
rekal "<q>" | python3 "$ROOT/scripts/recall-route.py"
```

`--apply` writes:

- `.rekal/config.json` → `weights` (query-time; no reindex)
- `.rekal/calibration/active-profile.json` (audit: profile + sha)

Both are local / typically gitignored. Do not commit calibration into the
shared ledger path.

## Profiles

| Profile | Use |
|---|---|
| `coding` | Default ADLC — high mass / conf bars, facet on |
| `chat` | Pure dialogue — MASS_MIN=0, lower CONF, less facet |
| `multi-hop` | Synthesis / WHY — more LSA, chat-like gates |
| `auto` | Infer from `miss_reason` / category histograms |

## After apply

Re-run the same queries. If still `true_miss`, calibration will not help —
need WHY route or better retrieval, not softer gates. Cite
`active-profile.json` `proposal_sha256` in any run note.
