//go:build (darwin && arm64) || (linux && amd64)

package nomic

/*
#cgo CFLAGS: -I${SRCDIR}/../../../../.deps/llama.cpp/include -I${SRCDIR}/../../../../.deps/llama.cpp/ggml/include
#cgo LDFLAGS: -L${SRCDIR}/../../../../.deps/llama.cpp/build/src -lllama
#cgo LDFLAGS: -L${SRCDIR}/../../../../.deps/llama.cpp/build/ggml/src -lggml -lggml-base -lggml-cpu
#cgo LDFLAGS: -L${SRCDIR}/../../../../.deps/llama.cpp/build/common -lcommon
#cgo LDFLAGS: -lstdc++ -lm
#cgo darwin LDFLAGS: -L${SRCDIR}/../../../../.deps/llama.cpp/build/ggml/src/ggml-metal -lggml-metal
#cgo darwin LDFLAGS: -L${SRCDIR}/../../../../.deps/llama.cpp/build/ggml/src/ggml-blas -lggml-blas
#cgo darwin LDFLAGS: -framework Foundation -framework Metal -framework MetalKit -framework Accelerate
#include "embed.h"
#include <stdlib.h>
*/
import "C"

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"runtime"
	"unsafe"
)

const (
	// ModelName identifies the embedding model for the session_embeddings table.
	ModelName = "nomic-v1.5"
	// EmbedDim is the output dimensionality of nomic-embed-text-v1.5.
	EmbedDim = 768
)

// Supported reports whether nomic embeddings are available on this platform.
func Supported() bool {
	return len(modelGZ) > 0
}

// Embedder wraps a loaded nomic-embed-text model.
type Embedder struct {
	handle   *C.nomic_embedder
	nEmbd    int
	tempPath string
}

// NewEmbedder decompresses the embedded model to a temp file and loads it.
func NewEmbedder() (*Embedder, error) {
	if !Supported() {
		return nil, ErrNotSupported
	}

	// Decompress GGUF to temp file.
	gz, err := gzip.NewReader(bytes.NewReader(modelGZ))
	if err != nil {
		return nil, fmt.Errorf("nomic: decompress model: %w", err)
	}
	defer gz.Close() //nolint:errcheck

	tmp, err := os.CreateTemp("", "nomic-embed-*.gguf")
	if err != nil {
		return nil, fmt.Errorf("nomic: create temp file: %w", err)
	}

	if _, err := io.Copy(tmp, gz); err != nil {
		tmp.Close()
		os.Remove(tmp.Name()) //nolint:errcheck
		return nil, fmt.Errorf("nomic: write temp model: %w", err)
	}
	tmp.Close()

	threads := runtime.NumCPU()
	if threads > 8 {
		threads = 8
	}

	cPath := C.CString(tmp.Name())
	defer C.free(unsafe.Pointer(cPath))

	handle := C.nomic_embedder_load(cPath, C.int(threads))
	if handle == nil {
		os.Remove(tmp.Name()) //nolint:errcheck
		return nil, fmt.Errorf("nomic: failed to load model")
	}

	nEmbd := int(C.nomic_embedder_n_embd(handle))

	return &Embedder{handle: handle, nEmbd: nEmbd, tempPath: tmp.Name()}, nil
}

// Close frees the model and removes the temp file.
func (e *Embedder) Close() {
	if e.handle != nil {
		C.nomic_embedder_free(e.handle)
		e.handle = nil
	}
	if e.tempPath != "" {
		os.Remove(e.tempPath) //nolint:errcheck
	}
}

// EmbedDocument embeds text with the "search_document: " prefix.
func (e *Embedder) EmbedDocument(text string) ([]float64, error) {
	return e.embed("search_document: " + text)
}

// EmbedQuery embeds text with the "search_query: " prefix.
func (e *Embedder) EmbedQuery(text string) ([]float64, error) {
	return e.embed("search_query: " + text)
}

func (e *Embedder) embed(text string) ([]float64, error) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	buf := make([]float32, e.nEmbd)
	rc := C.nomic_embedder_embed(e.handle, cText, (*C.float)(unsafe.Pointer(&buf[0])))
	if rc != 0 {
		return nil, fmt.Errorf("nomic: embed failed (rc=%d)", int(rc))
	}

	// Convert float32 → float64 for consistency with the rest of the codebase.
	result := make([]float64, e.nEmbd)
	for i, v := range buf {
		result[i] = float64(v)
	}
	return result, nil
}

// EmbedSessions embeds multiple sessions in batch. Returns session_id → vector.
// Individual session failures are skipped — only successfully embedded sessions
// are included in the result.
func (e *Embedder) EmbedSessions(sessions map[string]string) (map[string][]float64, error) {
	result := make(map[string][]float64, len(sessions))
	for id, content := range sessions {
		vec, err := e.EmbedDocument(content)
		if err != nil {
			// Skip this session — non-fatal.
			continue
		}
		result[id] = vec
	}
	return result, nil
}
