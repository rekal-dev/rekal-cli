package codec

import _ "embed"

// presetDict is a zstd dictionary trained on programming conversation patterns.
// It improves compression of small frames (~200 bytes) by ~2x.
// If empty, frames are compressed without a preset dictionary.
//
//go:embed preset.dict
var presetDict []byte
