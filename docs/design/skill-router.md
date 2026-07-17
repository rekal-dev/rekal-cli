# Skill router — one tip, progressive disclosure, executable gates

The Claude Code surface is a **single** skill (`skills/rekal/`). The agent
never chooses among `rekal-*` companions. It classifies the question, loads
one module (or runs one script), and stops.

Install copies the whole tree; `clean` / refresh purge legacy companion dirs.

## Layers

```mermaid
flowchart TB
  tip["SKILL.md tip<br/>always loaded"]
  tip --> triage{"Substrate?"}
  triage -->|Tree| grep["grep / read HEAD"]
  triage -->|Knowledge / pointed ledger| route["scripts/recall-route.py"]
  triage -->|Map| mapf["scripts/map-fresh.sh"]
  triage -->|Why / mine / …| ref["Read references/*.md"]
  route -->|KNOWLEDGE| readk["Read pointer at HEAD"]
  route -->|INJECT| hunt["references/hunt.md → drill"]
  route -->|SILENCE| quiet["Stay silent on memory"]
  mapf --> mapr["references/map.md"]
  ref --> gates["Optional gate scripts"]
```

| Layer | Path | Loads when |
|-------|------|------------|
| Tip | `SKILL.md` | Always (triage + dispatch only) |
| Scripts | `scripts/*` | Tip or reference names them — deterministic |
| References | `references/*.md` | One `Read` after triage — then stop |

## Substrate triage

```mermaid
flowchart TD
  q["Question"] --> tense{"True now, or was?"}
  tense -->|was| ledger["Ledger<br/>gated recall / SQL"]
  tense -->|now| kind{"Code or prose?"}
  kind -->|code| tree["Tree — grep / read<br/>do not recall"]
  kind -->|prose| know["Knowledge — rekal → route script<br/>Read pointer, stop"]
  q --> shape{"Breadth / structure?"}
  shape -->|yes| map["Map — freshness script first"]
```

Boundary line (tip): *grep for code that is · knowledge for prose that is ·
ledger for the why that was.*

## Recall route (knowledge vs episode vs silence)

Bars and labels live in scripts — not in tip prose.

```mermaid
flowchart LR
  r["rekal JSON"] --> rr["recall-route.py"]
  rr --> hg["hunt-gate.py"]
  hg -->|knowledge non-empty| k["KNOWLEDGE<br/>exit 0 — no episode inject"]
  hg -->|top≥0.9 or gap≥0.04 n≥2| i["INJECT / PASS_EPISODE"]
  hg -->|else| s["SILENCE"]
```

Single result: absolute top ≥ 0.9 only (gap alone must not pass).

## Other gates

| Script | Machine event |
|--------|----------------|
| `why-trail-gate.py` | WHY synthesize only if gather rows ≥ 10 |
| `map-fresh.sh` | `FRESH` / `STALE` / `MISSING` vs HEAD watermark |
| `map-write-watermark.sh` | Write line-1 watermark (+ stub if missing) |
| `wiki-branch-gate.sh` | Refuse wiki writes on the default branch |

## Dispatch map (tip → module)

```mermaid
flowchart LR
  subgraph tip_dispatch ["Tip dispatch"]
    A["present prose"] --> R["recall-route"]
    B["pointed past"] --> R
    C["why arc"] --> W["why.md + why-trail-gate"]
    D["analytical"] --> M["mine.md"]
    E["file/line/commit"] --> P["provenance.md"]
    F["breadth"] --> MF["map-fresh → map.md"]
    G["rules / libraries / census"] --> AN["analytics.md"]
    H["docs/wiki PR"] --> WG["wiki-branch-gate → wiki.md"]
    I["flags / SQL"] --> RF["reference.md"]
  end
```

One question, one substrate. Cite session / turn / commit with every memory claim.
