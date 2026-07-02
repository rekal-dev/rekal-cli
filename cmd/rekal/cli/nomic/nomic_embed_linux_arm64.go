//go:build linux && arm64

package nomic

import _ "embed"

//go:embed models/nomic-embed-text-v1.5.Q8_0.gguf.gz
var modelGZ []byte
