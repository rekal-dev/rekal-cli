---
name: rekal
description: >
  Use in a repo with Rekal initialized (.rekal/ exists). Rekal is memory of
  prior AI sessions — who changed what, why, and when. Before spending a token,
  decide WHERE the answer lives: TREE (current code — grep/read, present),
  KNOWLEDGE (current prose at HEAD — rekal knowledge block, present), MAP
  (structure), or LEDGER (session intent — rekal recall, past). Classify,
  dispatch to one substrate, stay silent when memory is not the tool. On
  analytical asks, decompose to SQL before hybrid search. Route, do not stack.
---

# Rekal — which substrate answers this?

Most wasted effort is the wrong substrate. Decide before you grep or recall.

```bash
ROOT="${CLAUDE_SKILL_DIR:-$(git rev-parse --show-toplevel)/.claude/skills/rekal}"
```

| Substrate | Holds | Tense | Primitive | Answers |
|---|---|---|---|---|
| Tree | current code | now | grep / read | what does X do, where is it |
| Knowledge | current prose | now | `rekal` → route script → Read | convention / what we know |
| Map | structure | — | map scripts + workflow | how is it built |
| Ledger | session intent | past | gated `rekal` + drill | why, tried, rejected |

## Boundary

1. True **now**, or something that **was**?
   - **Was** (reason, rejected, past correction) → Ledger.
2. If now — **code** or **prose**?
   - **Code** (path/symbol, present tense) → Tree. Do not recall.
   - **Prose** → Knowledge. Never invent episodes when HEAD prose answers.

grep for code that is · knowledge for prose that is · ledger for the why that was.

## Silence is a machine event

Do not invent thresholds. Pipe recall through the route script. No `INJECT` /
`KNOWLEDGE` line → stay silent on memory (or re-route). Near-misses are noise.

## Dispatch — one Read or one script, then stop

| The question is… | Do |
|---|---|
| Present prose / convention | `rekal "<q>" \| python3 "$ROOT/scripts/recall-route.py"` → on `KNOWLEDGE`, Read `path`@`lines`, **stop** |
| Pointed past episode | same pipeline → on `INJECT`, `Read references/hunt.md` and drill; on `SILENCE`, stop |
| Why / decision arc | `Read references/why.md` · gate gather with `scripts/why-trail-gate.py` |
| Analytical patterns | `Read references/mine.md` |
| File / line / commit archaeology | `Read references/provenance.md` |
| Breadth / shape | `bash "$ROOT/scripts/map-fresh.sh"` then `Read references/map.md` |
| Rules / libraries / census | `Read references/analytics.md` |
| Publish `docs/wiki/` | `bash "$ROOT/scripts/wiki-branch-gate.sh"` then `Read references/wiki.md` |
| Flags, SQL, PATH, schema | `Read references/reference.md` |

One question, one substrate. Cite session / turn / commit with every memory claim.
