# Rekal CLI — Spec

Experience and behaviour per command. One file per command under `command/`.

**Shared:** [preconditions.md](preconditions.md) — how all commands (except init and clean) check git repo and init.

| Command | Spec |
|---------|------|
| `rekal init` | [command/init.md](command/init.md) |
| `rekal clean` | [command/clean.md](command/clean.md) |
| `rekal checkpoint` | [command/checkpoint.md](command/checkpoint.md) |
| `rekal push` | [command/push.md](command/push.md) |
| `rekal sync` | [command/sync.md](command/sync.md) |
| `rekal index` | [command/index.md](command/index.md) |
| `rekal embed` | [command/embed.md](command/embed.md) |
| `rekal log` | [command/log.md](command/log.md) |
| `rekal find` | [command/find.md](command/find.md) |
| `rekal query` | [command/query.md](command/query.md) |
| `rekal` (root recall) | [command/recall.md](command/recall.md) |

**Soul:** Minimum touch. Root = recall only. Everything else is explicit subcommands. We keep the command set small — no extra subcommands unless necessary.

**Output defaults (keep help/specs in sync):** recall and `query --session` print agent-readable text by default; `query` SQL prints TSV; `--json` opts into structured JSON/NDJSON. `find` is text-only.

**Evidence:** `cmd/rekal/cli/help_sync_test.go` asserts (1) cobra help phrases for defaults/rebuild, (2) every public command is registered, (3) every command above has a matching `command/*.md` (root → `recall.md`), (4) key spec/README phrases still match those defaults, (5) embed budgets are 16. Update the test in the same change when you change help or specs.
