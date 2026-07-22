//go:build darwin && amd64

package db

import _ "embed"

// The darwin/amd64 (Intel Mac) FTS extension is not committed to the repo; it is
// downloaded during the build by scripts/fetch-fts-extension.sh (see
// LoadFTSExtension).
//
//go:embed extensions/fts_osx_amd64.duckdb_extension.gz
var ftsExtensionGZ []byte
