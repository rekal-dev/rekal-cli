# Configuration

Rekal is zero-config by default. When you do want to tune it, there is exactly
one file: `.rekal/config.json` — gitignored, local to the machine, never
committed, pushed, or synced.

```json
{
  "local_import": { "all": true },
  "weights": {
    "bm25": 0.35,
    "lsa": 0.10,
    "nomic": 0.55,
    "steering_boost": 1.3,
    "subagent_downweight": 0.7,
    "facet_boost": 0.3,
    "recency_boost": 0,
    "reach_boost": 0
  },
  "embedding": {
    "endpoint": "$EMBED_ENDPOINT",
    "model": "nomic-embed-text-v1.5",
    "api_key_env": "EMBED_API_KEY",
    "timeout_seconds": 10
  }
}
```

A local `.rekal/config.json` deep-merges over an optional global
`~/.config/rekal/config.json` (honoring `$REKAL_CONFIG_HOME` then
`$XDG_CONFIG_HOME`); precedence is local → global → built-in defaults.

## `weights`

Tunes recall ranking (layer mix, steering-turn boost, subagent discount, and
`facet_boost` — the facet layer over each session's tool paths/commands/steering
text, on by default at 0.3; set 0 to disable). Applied at query time — changing
them takes effect on the next search, no reindex, at any corpus size.

Two additive ranking layers ship **off** (0) and are opt-in:

- **`recency_boost`** — nudges more recently captured sessions up the ranking
  (min-max over the candidate set: newest → +boost, oldest → +0).
- **`reach_boost`** — nudges sessions the L1 recall graph has reached before up
  the ranking (max-normalized `session_reach.reach_count`), turning the
  `[reached N×]` usage hint from display-only into ranking. Fails soft on an
  index with no reach data.

Both are additive terms applied before the subagent discount, exactly like
`facet_boost`; both reorder *within* a result set and **never** feed the silence
gate (a newer or oft-reached session is not inherently more relevant). Start
small (e.g. `0.1`–`0.2`) and tune per corpus; `0` is byte-identical to the
engine without them.

## `embedding`

Switches deep semantic embeddings from the embedded nomic model to any
OpenAI-compatible endpoint (vLLM, Ollama, LM Studio, TEI). Requests are batched
and hard-timeboxed so a slow server can never stall a commit (embedding is
always non-fatal). Pointed at localhost, your data still never leaves the
machine; pointed at a cloud API, session text leaves — your call, made
explicitly.

Switching embedding model/endpoint requires one `rekal index` to regenerate
vectors. A content-hash-keyed cache (`.rekal/embed-cache.db`, vectors only,
never text) makes routine rebuilds embed only new sessions — and makes a model
switch cost exactly one full pass.

## API key: three ways, pick one

| Form | Example | Where the secret lives |
|---|---|---|
| Real string | `"api_key": "sk-abc123"` | In the file (gitignored, this machine only) |
| Env reference | `"api_key": "$MY_KEY"` | In the environment, expanded at run time |
| Env var name | `"api_key_env": "EMBED_API_KEY"` | In the environment, read directly |

Precedence: `api_key_env` wins when set and the variable is non-empty; otherwise
`api_key` (after `$VAR` expansion) is used; no key at all just omits the
`Authorization` header — the normal case for a localhost server. `endpoint`
expands `$VAR` the same way. One edge: a *hardcoded* `api_key` containing a
literal `$` would be treated as an env reference — real provider keys never
contain `$`, and `api_key_env` is the unambiguous form for anything sensitive.
