# Rekal CLI

Rekal gives your agent precise memory — the exact context it needs for what it's working on.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/rekal-dev/cli/main/scripts/install.sh | bash
```

Or with a specific version:

```bash
REKAL_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/rekal-dev/cli/main/scripts/install.sh | bash
```

Install location: `~/.local/bin` (override with `REKAL_INSTALL_DIR`).

## Usage

```bash
rekal version
```

*(M0: only version is implemented. More commands coming in later milestones.)*

## Development

Uses [mise](https://mise.jdx.dev/) for tools and tasks (same pattern as [Entire](https://github.com/entireio/cli)).

```bash
mise install          # install Go, golangci-lint
mise run fmt          # format code
mise run lint         # run linters
mise run test         # run tests
mise run test:ci      # run tests with -race
mise run build        # build rekal binary
```

**Run test and lint before push:** install the pre-push hook so CI-style checks run locally before each `git push`:

```bash
./scripts/install-hooks.sh
```

CI runs `mise run test:ci` on push/PR; lint and license-check run as separate workflows.

## License

Apache-2.0 — see [LICENSE](LICENSE).
