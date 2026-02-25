package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var selfOnly bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync team context from remote rekal branches",
		Long: `Fetch rekal branches from the remote and merge into the local data DB.

By default, fetches all rekal/* branches (whole team). Use --self to fetch
only your own rekal branch — useful when syncing across your own machines
(e.g. pulling context from your work laptop to your home machine) without
fetching the whole team's data.`,
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

			_ = selfOnly // will be used when implemented
			fmt.Fprintln(cmd.ErrOrStderr(), "rekal sync: not yet implemented")
			return nil
		},
	}

	cmd.Flags().BoolVar(&selfOnly, "self", false, "Only fetch your own rekal branch (not the whole team)")

	return cmd
}
