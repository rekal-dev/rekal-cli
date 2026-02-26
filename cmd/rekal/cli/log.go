package cli

import (
	"fmt"

	"github.com/rekal-dev/cli/cmd/rekal/cli/db"
	"github.com/spf13/cobra"
)

func newLogCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show recent checkpoints",
		Long: `Show recent checkpoints from the data DB, newest first.

Each entry shows the checkpoint ID, timestamp, git commit SHA, branch,
author email, and number of sessions captured. Use --limit to control
how many entries are shown.`,
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

			return runLog(cmd, gitRoot, limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Max entries to show")
	return cmd
}

func runLog(cmd *cobra.Command, gitRoot string, limit int) error {
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	rows, err := dataDB.Query(
		`SELECT c.id, c.git_sha, c.git_branch, c.user_email, c.ts, c.actor_type,
		        count(cs.session_id) as n_sessions
		 FROM checkpoints c
		 LEFT JOIN checkpoint_sessions cs ON cs.checkpoint_id = c.id
		 GROUP BY c.id, c.git_sha, c.git_branch, c.user_email, c.ts, c.actor_type
		 ORDER BY c.ts DESC
		 LIMIT $1`, limit,
	)
	if err != nil {
		return fmt.Errorf("query checkpoints: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var id, gitSHA, branch, email, ts, actorType string
		var nSessions int
		if err := rows.Scan(&id, &gitSHA, &branch, &email, &ts, &actorType, &nSessions); err != nil {
			return fmt.Errorf("scan checkpoint: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "checkpoint %s\n", id)
		fmt.Fprintf(cmd.OutOrStdout(), "Date:     %s\n", ts)
		fmt.Fprintf(cmd.OutOrStdout(), "Commit:   %s\n", gitSHA)
		fmt.Fprintf(cmd.OutOrStdout(), "Branch:   %s\n", branch)
		fmt.Fprintf(cmd.OutOrStdout(), "Author:   %s\n", email)
		fmt.Fprintf(cmd.OutOrStdout(), "Sessions: %d\n", nSessions)
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return rows.Err()
}
