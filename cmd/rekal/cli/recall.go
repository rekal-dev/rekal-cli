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
//
// jsonCompact selects compact single-line JSON (for machine consumers) over the
// default pretty output. It is the stable structured-output opt-in for when the
// default recall output becomes a compact text digest
// (docs/design/skill-into-command.md §2.0) — machine consumers adopt --json now
// and are unaffected by that flip.
func runRecall(cmd *cobra.Command, gitRoot string, filters search.Filters, jsonCompact, digestMode bool) error {
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

	// Keep the knowledge layer fresh with HEAD. Watermark-gated: the steady
	// state (no new commits since the last refresh) costs one rev-parse; a
	// moved HEAD re-chunks only prose files whose blobs changed. Best-effort —
	// recall proceeds on a stale or absent layer.
	if err := refreshKnowledge(nil, indexDB, gitRoot); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: knowledge refresh failed: %v\n", err)
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
	// Query embedder must speak the same model the index was built with.
	// When embedding is configured, use that HTTP backend — never fall back
	// to embedded nomic on resolve failure (that silently mismatches a
	// Cohere/HTTP-built index and disables the neural layer with no reason).
	var qe search.QueryEmbedder
	if cfg.Embedding != nil {
		if ec, eerr := cfg.Embedding.resolve(); eerr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: %v — semantic query embedder unavailable\n", eerr)
		} else {
			qe = embedhttp.New(ec)
		}
	}
	if embedModel, ok, _ := db.ReadIndexState(indexDB, "embed_model"); ok && embedModel != "" {
		switch {
		case qe == nil && embedModel != "nomic-v1.5":
			fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: index embed_model is %q but no embedding config is set — semantic layer will skip\n", embedModel)
		case qe != nil && qe.ModelName() != embedModel:
			fmt.Fprintf(cmd.ErrOrStderr(), "rekal: warning: query embedder model %q != index embed_model %q — semantic layer may skip\n", qe.ModelName(), embedModel)
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

	// Digest mode: the in-binary route.py — agent-facing seed text instead of
	// JSON. Exit 1 on SILENCE mirrors route.py so the same gating holds.
	if digestMode {
		text, code := formatDigest(&out)
		fmt.Fprint(cmd.OutOrStdout(), text)
		if filters.Lineage != nil {
			filters.Lineage.FlushResult(len(text))
		}
		if code == 1 {
			return NewSilentError(fmt.Errorf("silence"))
		}
		return nil
	}

	var data []byte
	if jsonCompact {
		data, err = json.Marshal(out)
	} else {
		data, err = json.MarshalIndent(out, "", "  ")
	}
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
