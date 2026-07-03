package cli

import (
	"database/sql"
	"fmt"
	"io"
	"strconv"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
	"github.com/spf13/cobra"
)

func newIndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index",
		Short: "Rebuild the index DB from the data DB",
		Long: `Drop and rebuild the index DB (.rekal/index.db) from the data DB.

The index is local-only and never synced. It contains:
  - Full-text search index (BM25) over conversation turns
  - LSA vector embeddings for lightweight semantic similarity
  - Nomic deep semantic embeddings (on supported platforms)
  - Session facets (author, branch, actor, counts) for fast filtering
  - File co-occurrence graph
  - Tool call indexes

Rebuild when the index is out of date or after importing new data.
'rekal sync' rebuilds the index automatically.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}
			if err := EnsureInitDone(gitRoot); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			return runIndex(cmd, gitRoot)
		},
	}
}

func runIndex(cmd *cobra.Command, gitRoot string) error {
	w := cmd.ErrOrStderr()

	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		return fmt.Errorf("open index db: %w", err)
	}
	defer indexDB.Close()

	// Load FTS extension.
	if err := db.LoadFTSExtension(indexDB); err != nil {
		return fmt.Errorf("load fts extension: %w", err)
	}

	// Clean slate.
	fmt.Fprintln(w, "dropping existing index tables...")
	if err := db.DropIndexTables(indexDB); err != nil {
		return fmt.Errorf("drop index tables: %w", err)
	}

	// Create schema.
	if err := db.InitIndexSchema(indexDB); err != nil {
		return fmt.Errorf("create index schema: %w", err)
	}

	// Populate from data DB.
	fmt.Fprintln(w, "populating index from data db...")
	if err := db.PopulateIndex(indexDB, gitRoot); err != nil {
		return fmt.Errorf("populate index: %w", err)
	}

	// Count what we indexed.
	var sessionCount, turnCount int
	if err := indexDB.QueryRow("SELECT count(*) FROM session_facets").Scan(&sessionCount); err != nil {
		return fmt.Errorf("count sessions: %w", err)
	}
	if err := indexDB.QueryRow("SELECT count(*) FROM turns_ft").Scan(&turnCount); err != nil {
		return fmt.Errorf("count turns: %w", err)
	}

	// Create FTS index (only if there are turns).
	if turnCount > 0 {
		fmt.Fprintln(w, "creating full-text search index...")
		if err := db.CreateFTSIndex(indexDB); err != nil {
			return fmt.Errorf("create fts index: %w", err)
		}
	}

	// LSA pass.
	embeddingDim := 0
	if sessionCount >= 2 {
		fmt.Fprintln(w, "building LSA embeddings...")
		sessionContent, err := db.QuerySessionContent(indexDB)
		if err != nil {
			return fmt.Errorf("query session content: %w", err)
		}

		model, err := lsa.Build(sessionContent, lsa.DefaultDimension)
		if err != nil {
			fmt.Fprintf(w, "warning: LSA build failed: %v\n", err)
		} else if model != nil {
			vectors := model.Vectors()
			if err := db.StoreEmbeddings(indexDB, vectors, "lsa-v1"); err != nil {
				return fmt.Errorf("store embeddings: %w", err)
			}
			embeddingDim = model.Dim
			fmt.Fprintf(w, "stored %d LSA embeddings (%d dimensions)\n", len(vectors), embeddingDim)

			// Cache the query-projection state so recall doesn't refit the
			// whole model on every call — non-fatal, recall falls back to
			// rebuilding from raw content if this is missing.
			if err := storeLSAProjection(indexDB, model); err != nil {
				fmt.Fprintf(w, "warning: caching LSA projection failed: %v\n", err)
			}
		}

		// Nomic pass (non-fatal).
		if err := buildNomicEmbeddings(indexDB, sessionContent, w, gitRoot); err != nil {
			fmt.Fprintf(w, "warning: nomic embeddings skipped: %v\n", err)
		}
	}

	// Write index state.
	if err := db.WriteIndexState(indexDB, "session_count", strconv.Itoa(sessionCount)); err != nil {
		return err
	}
	if err := db.WriteIndexState(indexDB, "turn_count", strconv.Itoa(turnCount)); err != nil {
		return err
	}
	if err := db.WriteIndexState(indexDB, "embedding_dim", strconv.Itoa(embeddingDim)); err != nil {
		return err
	}
	if err := db.WriteIndexState(indexDB, "last_indexed_at", "now"); err != nil {
		return err
	}

	fmt.Fprintf(w, "index rebuilt: %d sessions, %d turns\n", sessionCount, turnCount)
	return nil
}

// buildNomicEmbeddings generates nomic-embed-text embeddings for all sessions
// and stores them in the index DB. Non-fatal: returns error on any failure.
func buildNomicEmbeddings(indexDB *sql.DB, sessionContent map[string]string, w io.Writer, gitRoot string) error {
	if !nomic.Supported() {
		return nil
	}

	fmt.Fprintln(w, "building nomic deep semantic embeddings...")
	client, err := nomic.NewClient(gitRoot)
	if err != nil {
		return err
	}
	defer client.Close()

	vectors, err := client.EmbedSessions(sessionContent)
	if err != nil {
		return err
	}

	if err := db.StoreEmbeddings(indexDB, vectors, nomic.ModelName); err != nil {
		return err
	}
	fmt.Fprintf(w, "stored %d nomic embeddings (%d dimensions)\n", len(vectors), nomic.EmbedDim)
	return nil
}
