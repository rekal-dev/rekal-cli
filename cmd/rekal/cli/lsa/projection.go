package lsa

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// projectionData is the gob-serializable subset of Model needed to embed a
// new query into an already-fitted LSA space. Embed only ever reads
// Vocabulary, IDF, Uk, Sk, and Dim — Vk and SessionIDs exist purely to
// produce the per-session vectors Build's caller stores via Vectors(), and
// those vectors are already persisted separately (session_embeddings,
// model "lsa-v1"). Re-deriving Vk/SessionIDs here would duplicate storage
// for data Embed never uses.
type projectionData struct {
	Vocabulary map[string]int
	IDF        []float64
	UkRows     int
	UkCols     int
	UkData     []float64 // row-major, UkRows*UkCols entries
	Sk         []float64
	Dim        int
}

// EncodeProjection serializes the query-projection state of a fitted model
// so it can be persisted (e.g. in index.db) and reloaded by DecodeProjection
// without re-running Build against the full corpus. This is the fix for
// recall re-fitting the entire LSA model — tokenizing every session and
// re-running SVD — on every single query; only the model actually needs to
// be rebuilt when the corpus changes (at `rekal index`/`sync` time), not on
// every recall.
func EncodeProjection(m *Model) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("lsa: cannot encode nil model")
	}
	rows, cols := m.Uk.Dims()
	ukData := make([]float64, rows*cols)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			ukData[i*cols+j] = m.Uk.At(i, j)
		}
	}
	pd := projectionData{
		Vocabulary: m.Vocabulary,
		IDF:        append([]float64(nil), m.IDF...),
		UkRows:     rows,
		UkCols:     cols,
		UkData:     ukData,
		Sk:         append([]float64(nil), m.Sk...),
		Dim:        m.Dim,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&pd); err != nil {
		return nil, fmt.Errorf("lsa: encode projection: %w", err)
	}
	return buf.Bytes(), nil
}

// DecodeProjection reconstructs a Model from EncodeProjection's output. The
// returned Model is projection-only: Vk and SessionIDs are nil, so calling
// Vectors() on it will panic — it exists solely to call Embed(query) against
// an already-fitted space, matching how recall.go uses it.
func DecodeProjection(data []byte) (*Model, error) {
	var pd projectionData
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&pd); err != nil {
		return nil, fmt.Errorf("lsa: decode projection: %w", err)
	}
	return &Model{
		Vocabulary: pd.Vocabulary,
		IDF:        pd.IDF,
		Uk:         mat.NewDense(pd.UkRows, pd.UkCols, pd.UkData),
		Sk:         pd.Sk,
		Dim:        pd.Dim,
	}, nil
}
