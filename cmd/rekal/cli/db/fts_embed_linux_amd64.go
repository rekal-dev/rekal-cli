//go:build linux && amd64

package db

import _ "embed"

//go:embed extensions/fts_linux_amd64.duckdb_extension.gz
var ftsExtensionGZ []byte
