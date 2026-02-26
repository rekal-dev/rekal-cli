//go:build !(darwin && arm64) && !(linux && amd64)

package db

var ftsExtensionGZ []byte
