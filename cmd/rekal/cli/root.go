package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd returns the root command for the rekal CLI.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rekal",
		Short: "Rekal — gives your agent precise memory",
		Long:  "Rekal gives your agent precise memory — the exact context it needs for what it's working on.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// M0: no default behavior; show usage
			return cmd.Help()
		},
	}
	cmd.SetVersionTemplate("rekal {{.Version}}\n")
	cmd.Version = Version
	cmd.AddCommand(newVersionCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "rekal", Version)
			return nil
		},
	}
}

// Run executes the root command and exits with the appropriate code.
func Run() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
