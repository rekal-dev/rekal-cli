//go:build darwin && arm64

package db

import _ "embed"

//go:embed extensions/fts_osx_arm64.duckdb_extension.gz
var ftsExtensionGZ []byte
