package cli

import (
	"fmt"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
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
how many entries are shown.

A checkpoint is the link between a commit and the conversation that produced
it. Use log to find that commit, then recall or drill to read the reasoning.

Flags:
  -n, --limit <int>     How many checkpoints to show (default 20)`,
		Example: `  # The last few captures
  rekal log
    checkpoint 01KZ0HC2QGPQNYW46X9BW9V4XP
    Date:     2026-08-02T06:07:55Z
    Commit:   f2d7660014a853a57eb363461221cf8a077b9425
    Branch:   claude/plugin-pr-request-fz3r1z
    Author:   noreply@anthropic.com
    Sessions: 1

  # Widen the window (short form)
  rekal log -n 50

  # From a commit seen here, read the conversation behind it
  rekal -c f2d7660 "what were we fixing"`,
		Args: rejectExtraArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}

			return runLog(cmd, gitRoot, limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Max entries to show")
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
