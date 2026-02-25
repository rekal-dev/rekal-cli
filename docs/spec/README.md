# Rekal CLI — Spec

Experience and behaviour per command. One file per command under `command/`.

**Shared:** [preconditions.md](preconditions.md) — how all commands (except init and clean) check git repo and init.

| Command | Spec |
|---------|------|
| `rekal init` | [command/init.md](command/init.md) |
| `rekal clean` | [command/clean.md](command/clean.md) |
| `rekal checkpoint` | [command/checkpoint.md](command/checkpoint.md) |
| `rekal push` | [command/push.md](command/push.md) |
| `rekal index` | [command/index.md](command/index.md) |
| `rekal query "<sql>"` | [command/query.md](command/query.md) |
| `rekal log` | [command/log.md](command/log.md) |
| `rekal sync` | [command/sync.md](command/sync.md) |
| `rekal` (root recall) | [command/recall.md](command/recall.md) |

**Soul:** Minimum touch. Root = recall only. Everything else is explicit subcommands. We keep the command set small — no extra subcommands unless necessary.
