# Soul

## Two problems. Everything flows from them.

### 1. Intent has no ledger

Code has git. Every line, every change, every author — recorded forever.

But the reasoning behind the code has nothing. The conversations where a developer and an AI explored a problem, debated approaches, rejected alternatives, and arrived at a decision — those vanish the moment the session ends.

The code says *what*. The intent says *why*. And the *why* has no permanent, shared, immutable record.

Rekal is that record.

An append-only intent ledger. Distributed through git. Shared across the team. Immutable once written. Every decision, every rejected alternative, every dead end — captured at the moment it happens and preserved exactly as it was. No one can edit it. No one can delete it. The record is the record.

This is not a feature of Rekal. This is the first reason Rekal exists.

### 2. Agents can't remember

An AI agent starts every session blank. It reads the code. It does not know why the code looks the way it does. It does not know what was tried and rejected last week. It does not know that the team already explored and abandoned the approach it's about to suggest.

Humans have institutional memory — imperfect, fading, but real. Agents have none.

Rekal gives agents memory. Not general knowledge. Not search results. Recall — the precise prior context for the problem the agent is working on right now, delivered in structured form, bounded to a token budget, scored by relevance.

The agent is the reader. Not the human. Every decision about output format, query interface, and context loading puts the agent first.

This is not a feature of Rekal. This is the second reason Rekal exists.

---

## Beliefs

- Immutable. Immutability is the foundation of a ledger and the basis of trust. If anyone can edit the record, no one can trust it. If no one can trust it, there is no basis for sharing. Append-only is not a technical constraint — it is what makes the whole system work.
- Intent lives next to the code. The code says what. The intent says why. They belong together — not in separate systems, not behind someone else's service.
- Thin on the wire, rich on the machine. Git is the transport and every byte costs. Strip what git already has, compress what remains. Indexes, embeddings, search — all computed locally. When there's a trade-off between wire size and local compute, local compute wins every time.
- Secure by design. The data never leaves git and the local machine. No servers. No APIs. No telemetry. The security model is simple: there is nothing to breach because there is nothing to connect to.
- Simple. Zero dependencies. Single binary, everything embedded. Nothing to install, nothing to configure, nothing to break.
- Transparent. The user sees everything that was created and can remove all of it. One command to install, one command to uninstall. No sticky tape allowed.
- Agent first. The agent is the consumer. Every decision — output format, query interface, context loading, invisibility, ease of setup — favors the agent. Humans benefit as a side effect.

---

## Character

Rekal is quiet. It runs in the background and never interrupts. The agent doesn't need to think about it.

Rekal is precise. It captures exactly what matters and discards everything else. Thin on the wire.

Rekal is honest. The record is the record. Immutable. No silent updates. No rewriting history.

Rekal is opinionated. Fewer choices, fewer things that break. Simple.

Rekal is patient. It captures context today that might not matter for months. It trusts that the agent will know what to ask for when the time comes.

---

## Voice

When Rekal speaks — in CLI output, error messages, docs — it sounds like a competent colleague. Short sentences. Plain words. Say what happened, say what to do, stop.

```
rekal: not a git repository (run this inside a project)
rekal: captured 3 sessions, 847 turns
rekal: pushed to rekal/alice@example.com (291 bytes added)
rekal: no sessions match "JWT expiry" in src/auth/
```

No exclamation marks. No emoji. No "oops." Just the facts, clearly stated, with enough context to act on.

---

## Decisions

When facing a choice, we ask:

1. Does it preserve immutability?
2. Does the intent stay next to the code?
3. Is it thin on the wire?
4. Does the data stay within git and the local machine?
5. Is it simple — zero dependencies, zero config?
6. Is it transparent — can the user see and remove everything?
7. Does the agent get what it needs?

If the answer to any of these is no, we find another way.

---

## Refusals

We don't rewrite history. The ledger is immutable.

We don't store what git already has. Thin on the wire.

We don't depend on external infrastructure. Secure by design.

We don't phone home. Not even crash reports. Secure by design.

We don't add options where a good default exists. Simple.

We don't leave residue. Transparent.

We don't optimize for human reading at the expense of agent consumption. Agent first.

---

## The soul in practice

Real decisions where the beliefs broke the tie.

| Belief | Decision | Why |
|--------|----------|-----|
| Immutable | Append-only wire format | No byte is ever modified after written. Immutability is a structural guarantee, not a policy. |
| Intent next to code | Git orphan branch transport | Share context through standard git push/fetch. No sync server. Works with any remote. |
| Thin on the wire | 99% payload reduction | A 2–10 MB session becomes ~300 bytes. Strip everything git already has. Indexes and embeddings computed locally. A year of team context fits in 200 KB. |
| Secure by design | No external calls | Embedding model ships inside the binary. No API keys. No accounts. Data touches only git and the local filesystem. |
| Simple | Single binary, embedded everything | Database engine, embedding model, compression dictionary — all inside one file. Download it and you're done. |
| Transparent | Clean install/uninstall | `init` creates `.rekal/` and a hook. `clean` removes both completely. No hidden state anywhere else on the system. |
| Agent first | Hybrid ranked search, structured output | Three-signal ranking. Agent controls the token budget. JSON output, not human formatting. |
