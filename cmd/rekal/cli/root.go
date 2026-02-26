package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/rekal-dev/cli/cmd/rekal/cli/versioncheck"
	"github.com/spf13/cobra"
)

const gettingStarted = `

Getting Started:
  rekal init          Initialize Rekal in a git repository
  rekal checkpoint    Capture the current session
  rekal push          Share context with the team
  rekal sync          Pull team context
  rekal "query"       Recall sessions by keyword
`

// NewRootCmd returns the root command for the rekal CLI.
func NewRootCmd() *cobra.Command {
	var (
		fileFilter       string
		commitFilter     string
		checkpointFilter string
		authorFilter     string
		actorFilter      string
		limitFlag        int
	)

	cmd := &cobra.Command{
		Use:           "rekal [filters...] [query]",
		Short:         "Rekal — gives your agent precise memory",
		Long:          "Rekal gives your agent precise memory — the exact context it needs for what it's working on." + gettingStarted,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		PersistentPostRun: func(cmd *cobra.Command, _ []string) {
			versioncheck.CheckAndNotify(cmd.OutOrStdout(), Version)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no args and no filters, show help.
			if len(args) == 0 && fileFilter == "" && commitFilter == "" &&
				checkpointFilter == "" && authorFilter == "" && actorFilter == "" {
				return cmd.Help()
			}

			// Recall: preconditions required.
			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}
			if err := EnsureInitDone(gitRoot); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			filters := RecallFilters{
				Query:  strings.Join(args, " "),
				File:   fileFilter,
				Commit: commitFilter,
				Author: authorFilter,
				Actor:  actorFilter,
				Limit:  limitFlag,
			}

			_ = checkpointFilter // reserved for future use

			return runRecall(cmd, gitRoot, filters)
		},
	}

	// Recall filter flags on root command.
	cmd.Flags().StringVar(&fileFilter, "file", "", "Filter by file path (regex)")
	cmd.Flags().StringVar(&commitFilter, "commit", "", "Filter by git commit SHA")
	cmd.Flags().StringVar(&checkpointFilter, "checkpoint", "", "Query as of checkpoint ref")
	cmd.Flags().StringVar(&authorFilter, "author", "", "Filter by author email")
	cmd.Flags().StringVar(&actorFilter, "actor", "", "Filter by actor type (human|agent)")
	cmd.Flags().IntVarP(&limitFlag, "limit", "n", 0, "Max results (0 = no limit)")

	cmd.SetVersionTemplate("rekal {{.Version}}\n")
	cmd.Version = Version

	// Command groups.
	coreGroup := &cobra.Group{ID: "core", Title: "Core Commands:"}
	workflowGroup := &cobra.Group{ID: "workflow", Title: "Workflow Commands:"}
	advancedGroup := &cobra.Group{ID: "advanced", Title: "Advanced Commands:"}
	cmd.AddGroup(coreGroup, workflowGroup, advancedGroup)

	initCmd := newInitCmd()
	initCmd.GroupID = "core"
	cleanCmd := newCleanCmd()
	cleanCmd.GroupID = "core"
	versionCmd := newVersionCmd()
	versionCmd.GroupID = "core"

	checkpointCmd := newCheckpointCmd()
	checkpointCmd.GroupID = "workflow"
	pushCmd := newPushCmd()
	pushCmd.GroupID = "workflow"
	syncCmd := newSyncCmd()
	syncCmd.GroupID = "workflow"
	logCmd := newLogCmd()
	logCmd.GroupID = "workflow"

	queryCmd := newQueryCmd()
	queryCmd.GroupID = "advanced"
	indexCmd := newIndexCmd()
	indexCmd.GroupID = "advanced"

	cmd.AddCommand(initCmd, cleanCmd, versionCmd)
	cmd.AddCommand(checkpointCmd, pushCmd, syncCmd, logCmd)
	cmd.AddCommand(queryCmd, indexCmd)

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "rekal", Version)
			return nil
		},
	}
}

// Run executes the root command and exits with the appropriate code.
func Run() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		if !IsSilentError(err) {
			fmt.Fprintln(rootCmd.ErrOrStderr(), err)
		}
		os.Exit(1)
	}
}
