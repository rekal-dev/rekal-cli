package db

import (
	"database/sql"
	"fmt"
)

// MergeGateVersion identifies the merge-gate implementation whose verdicts are
// cached. Bump it whenever filterMerged's logic changes, so a decision made by
// an older rule is never reused — a cache that outlives the reasoning behind it
// is worse than no cache.
const MergeGateVersion = 1

// mergeGateCacheDDL is local-only bookkeeping, in the same category as
// recall_edges and checkpoint_state: it never crosses the wire. Nothing in
// export encodes it, and it must stay that way — it is a record of what this
// machine's git could see, which says nothing useful to anyone else.
const mergeGateCacheDDL = `
CREATE TABLE IF NOT EXISTS merge_gate_cache (
	git_sha      VARCHAR NOT NULL,
	target_tip   VARCHAR NOT NULL,
	gate_version INTEGER NOT NULL,
	shareable    BOOLEAN NOT NULL,
	PRIMARY KEY (git_sha, target_tip, gate_version)
);`

// EnsureMergeGateCache creates the cache table on demand, so a data.db written
// by an older rekal gains it in place.
func EnsureMergeGateCache(d *sql.DB) error {
	if _, err := d.Exec(mergeGateCacheDDL); err != nil {
		return fmt.Errorf("create merge_gate_cache: %w", err)
	}
	return nil
}

// LoadMergeGateVerdicts returns git_sha → shareable for decisions already made
// against this target tip and gate version.
//
// The key includes the tip because the answer genuinely changes when the
// mainline moves: work that was unmerged at one tip may have landed by the
// next. It includes the gate version because the answer also changes when the
// rule does. Anything else is a repeat of a question already answered — and
// answering it costs a commit-tree and a git cherry per held checkpoint, on
// every push, forever, for work that was abandoned and never will merge.
func LoadMergeGateVerdicts(d *sql.DB, targetTip string) (map[string]bool, error) {
	if targetTip == "" {
		return map[string]bool{}, nil
	}
	if err := EnsureMergeGateCache(d); err != nil {
		return nil, err
	}
	rows, err := d.Query(
		`SELECT git_sha, shareable FROM merge_gate_cache
		 WHERE target_tip = $1 AND gate_version = $2`, targetTip, MergeGateVersion)
	if err != nil {
		return nil, fmt.Errorf("read merge gate cache: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	out := map[string]bool{}
	for rows.Next() {
		var sha string
		var shareable bool
		if err := rows.Scan(&sha, &shareable); err != nil {
			return nil, fmt.Errorf("scan merge gate cache: %w", err)
		}
		out[sha] = shareable
	}
	return out, rows.Err()
}

// SaveMergeGateVerdicts records decisions for this target tip. Best-effort by
// contract: the caller must treat a failure as "no cache", never as a reason to
// withhold work, because the gate's answer is what decides sharing and a cache
// must not be able to change it.
func SaveMergeGateVerdicts(d *sql.DB, targetTip string, verdicts map[string]bool) error {
	if targetTip == "" || len(verdicts) == 0 {
		return nil
	}
	if err := EnsureMergeGateCache(d); err != nil {
		return err
	}
	for sha, shareable := range verdicts {
		if sha == "" {
			continue
		}
		if _, err := d.Exec(`INSERT INTO merge_gate_cache
			(git_sha, target_tip, gate_version, shareable) VALUES ($1, $2, $3, $4)
			ON CONFLICT (git_sha, target_tip, gate_version) DO UPDATE SET shareable = excluded.shareable`,
			sha, targetTip, MergeGateVersion, shareable); err != nil {
			return fmt.Errorf("write merge gate cache: %w", err)
		}
	}
	return nil
}

// PruneMergeGateCache drops rows for tips other than the current one, so the
// table stays proportional to the checkpoints rather than to the number of
// commits the mainline has ever had.
func PruneMergeGateCache(d *sql.DB, keepTip string) error {
	if keepTip == "" {
		return nil
	}
	if _, err := d.Exec(
		`DELETE FROM merge_gate_cache WHERE target_tip <> $1 OR gate_version <> $2`,
		keepTip, MergeGateVersion); err != nil {
		return fmt.Errorf("prune merge gate cache: %w", err)
	}
	return nil
}
