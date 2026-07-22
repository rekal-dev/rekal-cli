#!/usr/bin/env bash
# Download the DuckDB FTS extension for the target platform and place it where
# the Go build embeds it: cmd/rekal/cli/db/extensions/fts_<platform>.duckdb_extension.gz
#
# The linux/amd64 and osx/arm64 blobs are committed to the repo, so this is a
# no-op for those. The linux/arm64 and osx/amd64 blobs are not committed and are
# fetched here at build time (invoked by `mise run build` and the release
# workflow). Target platform comes from GOOS/GOARCH (defaulting to the host).
#
# The DuckDB version is read from go-duckdb's own pin so the extension can never
# drift from the linked library.
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
ext_dir="$repo_root/cmd/rekal/cli/db/extensions"

goos=${GOOS:-$(go env GOOS)}
goarch=${GOARCH:-$(go env GOARCH)}

case "$goos" in
  linux) os=linux ;;
  darwin) os=osx ;;
  *) echo "fetch-fts-extension: no embedded FTS extension for GOOS=$goos (uses remote fallback), skipping" >&2; exit 0 ;;
esac
case "$goarch" in
  amd64) arch=amd64 ;;
  arm64) arch=arm64 ;;
  *) echo "fetch-fts-extension: no embedded FTS extension for GOARCH=$goarch (uses remote fallback), skipping" >&2; exit 0 ;;
esac
platform="${os}_${arch}"

dest="$ext_dir/fts_${platform}.duckdb_extension.gz"
if [ -f "$dest" ]; then
  echo "fetch-fts-extension: $dest already present, skipping"
  exit 0
fi

# DuckDB version linked by go-duckdb — keeps the extension in lockstep with the
# engine. Falls back to a known-good version if the pin can't be read.
version=""
if mod_dir=$(go list -m -f '{{.Dir}}' github.com/marcboeker/go-duckdb 2>/dev/null) && [ -f "$mod_dir/Makefile" ]; then
  version=$(grep -E '^DUCKDB_BRANCH=' "$mod_dir/Makefile" | head -1 | cut -d= -f2 || true)
fi
version=${version:-v1.1.3}

url="https://extensions.duckdb.org/${version}/${platform}/fts.duckdb_extension.gz"
echo "fetch-fts-extension: downloading $url"
mkdir -p "$ext_dir"
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
curl -fsSL -o "$tmp" "$url"

# Validate it is a real gzip stream before trusting it (a proxy/error page is
# not an extension).
if ! gzip -t "$tmp" 2>/dev/null; then
  echo "fetch-fts-extension: downloaded file is not valid gzip" >&2
  exit 1
fi

mv "$tmp" "$dest"
trap - EXIT
echo "fetch-fts-extension: wrote $dest"
