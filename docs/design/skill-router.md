# Skill router — one thin route, three homes

The Claude Code surface is a **single** skill (`skills/rekal/`), redesigned from
`SOUL.md`'s "The skill" tenets. It is thin on the route, rich on arrival, and
organized around three homes:

- **Function → a script** — deterministic data for the agent's judgment.
- **Knowledge → rich prose, on demand** — informs judgment, never makes it.
- **Judgment → the agent's reasoning** — never frozen into a script or a rule.

The agent classifies the question, loads one module (or runs one script), and
stops. Install copies the whole tree; `clean` / refresh purge legacy companion
dirs. No corpus profiles ship — the route is general.

## Layers

```mermaid
flowchart TB
  tip["SKILL.md route<br/>always loaded, thin"]
  tip --> triage{"Substrate?"}
  triage -->|Tree| grep["grep / read HEAD"]
  triage -->|Knowledge / ledger| route["scripts/route.py"]
  triage -->|Map| mapf["scripts/map.sh fresh"]
  triage -->|past reasoning| ref["Read references/ledger.md"]
  route -->|KNOWLEDGE| readk["Read pointer at HEAD"]
  route -->|INJECT| drill["references/ledger.md → drill"]
  route -->|SILENCE| quiet["Stay silent on memory"]
  mapf --> mapr["references/map.md"]
```

| Layer | Path | Loads when |
|-------|------|------------|
| Route | `SKILL.md` | Always (triage + dispatch only; trusts reasoning) |
| Scripts | `scripts/*` | Route or reference names them — deterministic data |
| References | `references/*.md` | One `Read` after triage — then stop |

## Substrate triage

```mermaid
flowchart TD
  q["Question"] --> tense{"True now, or was?"}
  tense -->|was / only record is a conversation| ledger["Ledger<br/>route.py recall / SQL"]
  tense -->|now| kind{"Code or prose?"}
  kind -->|code| tree["Tree — grep / read<br/>do not recall"]
  kind -->|prose| know["Knowledge — rekal → route.py<br/>Read pointer, stop"]
  q --> shape{"Breadth / structure?"}
  shape -->|yes| map["Map — map.sh fresh first"]
```

Boundary line (route): *grep for code that is · knowledge for prose that is ·
ledger for the why that was.* A fact whose only record is a past conversation is
ledger, not knowledge — so a **pure-dialogue corpus** (no code, no HEAD prose, no
structure) routes to the ledger by degeneration, with no chat profile or
separate build.

## Recall route (knowledge vs episode vs silence)

Bars live in `route.py` — not route prose. Ranking still uses max-normalized
`score`; the gate uses absolute `confidence`. **Mass is a signal, not a veto.**

```mermaid
flowchart LR
  r["rekal JSON"] --> rt["route.py"]
  rt -->|confidence≥0.70 (soft 0.68, gap≥0.04)| i["INJECT top=/gap= + top-20<br/>sid conf=· t·n· snippet<br/>+ KNOWLEDGE line if present"]
  rt -->|else + knowledge present| k["KNOWLEDGE path=score<br/>agent judges the distribution"]
  rt -->|else| s["SILENCE"]
```

`confidence` = `max(saturate(bm25), cosine) + 0.15·saturate(facet)` — never
divided by the candidate-set max (junk queries also normalize `score` ≈ 1.0).
Hard floor 0.70; soft path 0.68 with gap ≥ 0.04 (above offtopic ~0.55–0.63).
This floor is permitted because it gates on saturating BM25 — a bounded
transform whose junk baseline is corpus-invariant by construction, not a number
read off one dataset (SOUL.md: no *tuned* constant decides).

Substrates are **inclusive**: a confident episode and a knowledge hit can both
report (mixed convention + why questions). Line 1 stays the primary verdict;
a trailing `KNOWLEDGE` line accompanies `INJECT` when prose also matched.
Knowledge alone is the report when the episode gate fails. SILENCE is
machine-only when neither substrate has signal.

The knowledge `score` has no corpus-invariant floor — it blends semantic cosine,
whose junk baseline drifts — so route.py reports the per-file distribution and
the agent judges. `mass` stays inside the script (never a veto, never emitted).
`confidence` is emitted on the INJECT header (`top=`/`gap=`) and each seed row
so the agent can weigh or drill selectively.

On INJECT the digest is seed coverage: top-20 as `sid conf=… t<turn> "snippet"`,
roughly constant in `-n`. Beyond 20: reformulate / multi-search.

## Other gates

| Script | Machine event |
|--------|----------------|
| `map.sh fresh` | `FRESH` / `STALE` / `MISSING` vs HEAD watermark |
| `map.sh watermark` | Write line-1 watermark (+ stub if missing) |
| `wiki-gate.sh` | Refuse wiki writes on the default branch |

## Dispatch map (route → module)

```mermaid
flowchart LR
  subgraph tip_dispatch ["Route dispatch"]
    A["present prose"] --> R["route.py"]
    B["pointed past episode"] --> R
    C["temporal / analytical / why / provenance"] --> L["references/ledger.md"]
    F["breadth"] --> MF["map.sh fresh → map.md"]
    H["docs/wiki PR"] --> WG["wiki-gate.sh → wiki.md"]
    I["flags / SQL / schema"] --> RF["references/reference.md"]
  end
```

`ledger.md` is the one rich page on reasoning over the past — recall, widen,
depth-as-judgment, time-axis, enumeration, whose-fact/premise, analytical SQL,
decision arcs, provenance. One question, one substrate. The route returns data;
the agent decides the move. Cite session / turn / commit with every memory claim.
