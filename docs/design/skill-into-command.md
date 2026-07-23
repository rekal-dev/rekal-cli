# Skill → command: agent-first defaults, boundaries, and the skill/command seam

Status: design (interface lock before migration). No performance change is
permitted — see §6.

> **Superseded in part (later refinements):** the `seek` verb is no longer a
> `--also` flag — multi-framing RRF is now folded into the base `rekal "<q>"`,
> which auto-derives deterministic reformulations and fuses them (see
> `deriveFramings` in `recall.go` and the recall spec). And the `when` command
> was removed: relative-date resolution moved back to the agent's own reasoning
> (skill time-axis guidance + SQL date math), since a pure calendar utility
> doesn't earn a peer command and can't ship as a script in the scriptless
> skill. The rest of this document stands.

## 1. Why — the two souls compose, they don't conflict

The skill soul says *function belongs in a script … move capability down to
function … the skill shrinks as it sharpens.* The CLI soul says *agent first:
output format, query interface, context loading favor the agent — single
binary, zero dependencies.* Read together they point one way: the deterministic
logic that today lives in Python skill scripts belongs **in the command**,
because the command is the better function-home — it is already the agent-first,
single-binary, zero-dependency surface. Folding the scripts in removes a
dependency (`python3`), removes installed files and PATH shims, and lets the
skill shrink to what only it can hold: judgment.

The scripts were scaffolding. The mature shape: **command owns function +
default presentation; skill owns judgment + orientation.**

## 2. The default-uplift principle

### 2.0 Output doctrine — text by default, JSON on opt-in

The governing rule for every read command:

> **Default output is compact, agent-readable text. Structured JSON is opt-in
> (`--json`).**

This inverts today's default (recall/query emit JSON) and it is the truest
reading of "agent first" *for an LLM consumer*. The soul's intent is "agent
consumption over human reading"; when it was written, "agent consumption" was
assumed to be JSON (agent = a program that parses). But our agent is an LLM,
and the measured evidence is the opposite: the route/view/find/seek **text**
digests ran ~13–25× cheaper in tokens than raw recall JSON, and the LoCoMo
89.3% accuracy lives in the digest form — JSON braces/keys are tokens the model
spends parsing structure instead of reasoning over content. So the agent-first
output is the form the LLM reasons over cheapest: **compact structured text.**

This is **not human formatting** (the soul's actual refusal). The digest is
agent formatting — line-oriented, token-minimal, for the LLM; humans benefit as
a side effect, exactly as the soul promises. JSON remains the right shape for
*program* consumers (git hooks, the bench harness, external tooling) — hence
`--json`, not its removal.

Consequence for §4: every read command's default is its text form; `--json` is
one universal raw escape hatch. `view` is not a mode — it is this inversion
(compact text default, `--json` raw) applied to `query`.

### 2.1 Which commands

Agent-first does **not** mean "rewrite every command." It means: for a command
whose output the agent *reads and reasons over*, the **default** (no flags)
should already be the agent-optimal form — the digest, the compression, the
resolved value — with `--json` as the universal escape hatch that preserves the
raw contract for tooling.

The test for whether a command gets uplifted:

> Does the command **return information the agent reasons over**, or does it
> **perform an action and report status**?

- **Returns information** → uplift the default (agent form by default, `--json`
  raw). These are the read/retrieval/navigation commands.
- **Performs an action** → leave it. Its terse status line ("captured 3
  sessions, 847 turns") is *already* agent-first in the soul's voice: say what
  happened, say what to do, stop. Uplifting it would be noise.

So the migration is bounded: it touches ~5–6 read commands, not the ~7
lifecycle commands.

## 3. Command boundaries — one verb each

Three families. A command owns exactly one verb; overlap is a smell.

### Retrieval — "which/what in the record answers this?"

**Two primitives, not three.** `seek` folds into base recall; `find` cannot.

| Primitive | Verb | Shape | For |
|---|---|---|---|
| `rekal "<q…>"` (recall ∪ seek) | rank / fuse | hybrid BM25+LSA+nomic, top-N, semantic, ranked-partial; 1 framing ranks, N framings RRF-fuse | pointed questions; multi-framing widening |
| `rekal find <term>` | enumerate | literal `ILIKE` sweep, every mention, no ranking, time order, complete | complete-set (all / how many / every) |

**Why `seek` collapses into recall.** `seek` is recall *generalized*: recall run
over N framings, RRF-fused. One framing (n=1) reduces to recall exactly — RRF of
a single ranked list preserves its order, so the output is byte-identical and
performance is unchanged. There is no genuine recall-vs-seek: it is one
relevance command taking 1..N positional framings. Folding it kills a verb with
no semantic loss (`rekal "<q>"` ranks; `rekal "<q1>" "<q2>"` fuses).

**Why `find` cannot fold in.** It is a different *contract*, not a parameter, on
three axes:
1. **Complete-unranked vs bounded-ranked.** recall returns the loudest top-N —
   a partial list is the *right* answer. find returns *every* mention — a
   partial list is a *wrong* answer. Opposite completeness guarantees; a
   truncating ranking engine cannot also promise completeness.
2. **Literal vs semantic.** recall matches by hybrid meaning ("turtle" surfaces
   pet talk that never says the word); find is a literal substring sweep. A
   "how many times did X say turtle" question needs literal enumeration —
   semantic ranking would give a wrong count.
3. **Different family.** find is a named SQL sweep — it belongs to the
   navigation/`query` family, on top of raw SQL, not the ranking engine.
   Merging recall+seek stays in one family; folding find in crosses a boundary.

A `--all` flag on base would bury *two* semantics (completeness + literal↔
semantic) in a flag and erase the visible **rank-vs-enumerate** choice — the one
retrieval judgment that most often goes wrong ("a partial set is a wrong
answer"). The two names force that choice; that is a feature, not redundancy.

### Navigation — "transform / resolve / drill the record"

| Command | Verb | For |
|---|---|---|
| `query` | SQL + session drill | the general primitive: raw SQL, `--session` drill |
| `query --view` (mode) | compress | strip JSON chrome from drills/rows |
| `when` | resolve | relative phrase → absolute date (pure calendar) |

`query` is the substrate everything else *could* be expressed in. `find` and
`when` exist because each encodes a **recurring SQL/compute shape** the agent
would otherwise re-derive every time — the soul's rule-accretion stop, one layer
down: *a SQL shape the agent keeps re-deriving is a subcommand waiting to be
named.* The boundary: `query` is the primitive; the named verbs are the
specializations that remove toil, never thought. (Multi-framing fusion is *not*
a navigation verb — it's base recall over N framings; see the retrieval note.)

### Lifecycle — "manage the store" (not uplifted)

`init`, `clean`, `checkpoint`, `push`, `sync`, `index`, `embed`. Side-effecting;
output is a status line, already right. Left alone.

## 4. The uplift map

| Command | Default today | Uplifted default | Raw escape |
|---|---|---|---|
| `rekal "<q…>"` (recall ∪ seek; 1 framing ranks, N fuse) | raw JSON | route digest (INJECT/KNOWLEDGE/SILENCE + seeds + `conf=`) | `--json` |
| `rekal query "<sql>"` | NDJSON | compressed rows | `--json` |
| `rekal query --session` | JSON turns | compressed turns | `--json` |
| `rekal find <term>` | — (new) | enumeration lines | `--json` |
| `rekal when <d> "<p>"` | — (new) | resolved date / window | — (already terse) |
| `rekal log` | table | unchanged (already terse) | `--json` (optional) |
| init/clean/checkpoint/push/sync/index/embed | status line | unchanged | — |

`--json` becomes the **one universal raw flag** across read commands: the
uplifted default is the agent form; `--json` is the byte-stable contract for any
machine consumer (including the bench harness — see §6).

## 5. The skill ↔ command seam

| | Command | Skill |
|---|---|---|
| Owns | function + default presentation | judgment + orientation |
| Never | decides *for* the agent (route stays a recommendation) | re-implements function (no scripts) |
| Contract | its **output** (byte-pinned by tests) | reads the output, weighs it |

**The mixing rule:** the command makes the right thing the default; the skill
teaches *when* and *why*, never *how to invoke plumbing*. If the skill has to
teach a pipe or a flag incantation to get usable output, that is a smell — the
default should be uplifted so the skill doesn't have to.

**The skill-less test** (the soul-aligned end state): *could a capable agent
with no skill installed still get good output from `rekal "<q>"`?* If yes —
because the default is uplifted — then the skill is purely **additive
judgment**, not a crutch. That is "the skill enables; it does not prescribe"
made literal: if the skill vanished, the command is still agent-first.

After the migration the skill is `SKILL.md` (substrate triage, boundary,
silence, judgment) + `references/*.md` (rich prose on demand). No `scripts/`,
no PATH wrappers. It orients; the binary computes.

## 6. The one hard constraint — no performance change

Proven behavior (LoCoMo 89.3%, the 223 route invariants) is a function of **the
exact bytes the agent sees.** Therefore:

1. **Byte-identical output.** Each command default reproduces its script's
   output exactly; port the script's test vectors (skill_test.go, when.py
   cases, skill-permtest.py) to assert against the command.
2. **`--json` and pin consumers first.** Add `--json` and repoint the bench
   harness / any raw-JSON consumer to it **before** flipping any default.
   Flipping recall's default from JSON to digest is a breaking change for
   whatever parses raw JSON today; the escape hatch must exist and be adopted
   first.
3. **Route stays a recommendation.** Moving the digest into the command must not
   turn INJECT/SILENCE into a hard gate. Super-low env-overridable floor, emits
   `conf=`, agent overrides. The command presents; the agent judges.

## 7. Sequence — DONE (each step gated by ported test vectors)

1. ✅ Review the command surface; make raw SQL first-class + documented
   (knowledge tables, `ts` TIMESTAMP note). *(`f88112d1`)*
2. ✅ Add `--json` to recall + query. *(`b297b498`)*
3. ✅ Port the pure verbs to subcommands, byte-identical: `when`, `find`.
   *(`d465f03b`)*
4. ✅ Fold `seek` into base recall — repeatable `--also` framing, RRF-fuse
   (n=1 byte-identical). Chose `--also` (decision B) over positional framings so
   the proven recall path is unchanged. *(`c2c9eac5`)*
5. ✅ Fold `view` into `query` (text default, `--json` raw). *(`873c2bc0`)*
6. ✅ Flip recall default → digest (`--json` raw); `route` in-command
   (`digest.go`, golden-tested) as a recommendation. *(`873c2bc0`)*
7. ✅ Strip `scripts/*.py` + PATH wrappers; skill rewritten to pure commands;
   `init`/`clean` simplified. *(`873c2bc0`)*

**Remaining (outside this repo's test reach):** repoint the bench harness on
`research/industry-bench` to `--json` (recall's default is now the digest), then
re-run the LoCoMo verify to confirm the 89.3% headline is unmoved — the corpus
lives on the research machine, not in CI.

## 8. Open decisions

- **SOUL.md line** — **resolved & applied** (commit `17436118`): the practice
  row now reads "compact agent-readable text by default, structured JSON on
  opt-in." Belief + Refusal untouched.
- **`view` as `query --view` vs the `--json` inversion** — **resolved** (§2.0):
  the `--json` inversion. Compact text is the default; `--json` is raw.
- **`log --json`**: uplift needed, or is the table already fine? (leaning: leave)
- **Naming**: `--json` vs `--raw` for the escape hatch (leaning `--json`, it
  names the format, not a vibe).
