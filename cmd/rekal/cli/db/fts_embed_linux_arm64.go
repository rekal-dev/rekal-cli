//go:build linux && arm64

package db

import _ "embed"

// The linux/arm64 FTS extension is not committed to the repo; it is downloaded
// during the build by scripts/fetch-fts-extension.sh (see LoadFTSExtension).
//
//go:embed extensions/fts_linux_arm64.duckdb_extension.gz
var ftsExtensionGZ []byte
