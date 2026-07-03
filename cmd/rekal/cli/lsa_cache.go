package cli

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
)

// lsaProjectionStateKey is the index_state key holding the persisted LSA
// query-projection state (see lsa.EncodeProjection). Reusing index_state
// (already used for session_count/schema_version/...) avoids a schema
// migration for what is, functionally, just another piece of index
// metadata.
const lsaProjectionStateKey = "lsa_projection"

// storeLSAProjection persists model's query-projection state to index.db so
// a later recall doesn't need to re-tokenize the whole corpus and re-run SVD
// just to embed the query string — only the corpus-dependent part (fitting
// the model) needs redoing when the corpus changes, at `rekal index`/`sync`
// time, not on every single recall call. Non-fatal by design at call
// sites: a query still works without this cache, just slower.
func storeLSAProjection(indexDB *sql.DB, model *lsa.Model) error {
	if model == nil {
		return nil
	}
	encoded, err := lsa.EncodeProjection(model)
	if err != nil {
		return fmt.Errorf("encode lsa projection: %w", err)
	}
	return db.WriteIndexState(indexDB, lsaProjectionStateKey, base64.StdEncoding.EncodeToString(encoded))
}

// loadLSAProjection loads the persisted LSA query-projection state, if any.
// Returns (nil, nil) when absent — an index.db built before this cache
// existed, or one whose corpus was too small for LSA to fit a model at all
// — so callers fall back to rebuilding from raw content.
func loadLSAProjection(indexDB *sql.DB) (*lsa.Model, error) {
	var encoded string
	err := indexDB.QueryRow(
		`SELECT value FROM index_state WHERE key = $1`, lsaProjectionStateKey,
	).Scan(&encoded)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("read lsa projection state: %w", err)
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode lsa projection state: %w", err)
	}
	model, err := lsa.DecodeProjection(raw)
	if err != nil {
		return nil, fmt.Errorf("decode lsa projection: %w", err)
	}
	return model, nil
}
