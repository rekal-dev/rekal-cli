package cli

import (
	"encoding/json"
	"fmt"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/embedhttp"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/spf13/cobra"
)

// runRecall opens (and, if empty, rebuilds) the index DB, runs the search, and
// prints the result as JSON. The ranking/grouping engine lives in the search
// package; this function is the command-side orchestration around it.
func runRecall(cmd *cobra.Command, gitRoot string, filters search.Filters) error {
	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		return fmt.Errorf("open index db: %w", err)
	}
	defer indexDB.Close()

	// The index may predate optional harness-metadata columns; migrations
	// are additive and idempotent.
	if err := db.MigrateIndexSchema(indexDB); err != nil {
		return fmt.Errorf("migrate index db: %w", err)
	}

	if err := db.LoadFTSExtension(indexDB); err != nil {
		return fmt.Errorf("load fts extension: %w", err)
	}

	// Auto-rebuild if the index is empty.
	if !db.IsIndexPopulated(indexDB) {
		fmt.Fprintln(cmd.ErrOrStderr(), "index not built, rebuilding...")
		indexDB.Close()
		if err := runIndex(cmd, gitRoot); err != nil {
			return err
		}
		indexDB, err = db.OpenIndex(gitRoot)
		if err != nil {
			return fmt.Errorf("reopen index db: %w", err)
		}
		defer indexDB.Close()
		if err := db.LoadFTSExtension(indexDB); err != nil {
			return fmt.Errorf("reload fts extension: %w", err)
		}
	}

	// Recall tuning + embedding backend come from .rekal/config.json. A bad
	// config falls back to defaults with a warning — recall must keep working.
	cfg, err := readMergedConfig(gitRoot)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: config unreadable, using defaults: %v\n", err)
		cfg = Config{}
	}
	weights, err := cfg.Weights.resolve()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: %v — using default weights\n", err)
		weights = search.DefaultWeights()
	}
	var qe search.QueryEmbedder
	if cfg.Embedding != nil {
		if ec, eerr := cfg.Embedding.resolve(); eerr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: %v — semantic layer uses the embedded model\n", eerr)
		} else {
			qe = embedhttp.New(ec)
		}
	}

	// Scoring lineage is global-only and off by default. Failures opening the
	// sink warn and continue — diagnostics must never break recall.
	if lin, closer, lerr := cfg.ScoringLineage.openLineage(cmd.ErrOrStderr()); lerr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: scoring_lineage: %v — lineage disabled\n", lerr)
	} else {
		if closer != nil {
			defer closer.Close() //nolint:errcheck
		}
		filters.Lineage = lin
	}

	out, err := search.Run(indexDB, filters, gitRoot, weights, qe)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	// Flush the staged lineage "result" event with the agent-facing payload
	// size. No-op when lineage is off or nothing was staged (filter mode).
	if filters.Lineage != nil {
		filters.Lineage.FlushResult(len(data))
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
