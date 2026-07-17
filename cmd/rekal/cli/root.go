package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/versioncheck"
	"github.com/spf13/cobra"
)

const gettingStarted = `

Workflow:
  rekal "keyword"                   Search sessions (BM25 + LSA + Nomic hybrid)
  rekal --file auth "token refresh" Filter by file path
  rekal query --session <id>        Drill into a session (full turns)
  rekal query --session <id> --full Include tool calls and files
  rekal query "SELECT ..."          Raw SQL for edge cases

Getting Started:
  rekal init                        Initialize Rekal in a git repository
  rekal checkpoint                  Capture the current session
  rekal push                        Share context with the team (merged work only)
  rekal sync                        Pull team context
  rekal index --include-all         Also recall your other repos' sessions (local only, never pushed)
`

// resolveRecallLimit interprets -n/--limit. Unset → (DefaultLimit, false).
// Explicit 0 → empty result set. Negative → error (BUG 7).
func resolveRecallLimit(cmd *cobra.Command, flag int) (limit int, explicit bool, err error) {
	if !cmd.Flags().Changed("limit") {
		return search.DefaultLimit, false, nil
	}
	if flag < 0 {
		return 0, true, fmt.Errorf("rekal: --limit must be >= 0 (got %d)", flag)
	}
	return flag, true, nil
}

// NewRootCmd returns the root command for the rekal CLI.
func NewRootCmd() *cobra.Command {
	var (
		fileFilter   string
		commitFilter string
		authorFilter string
		actorFilter  string
		limitFlag    int
		explainFlag  bool
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
		// The update notice goes to stderr: stdout carries JSON that agents
		// parse (recall/query output), and appending the notice there breaks
		// the parse once per check interval.
		PersistentPostRun: func(cmd *cobra.Command, _ []string) {
			versioncheck.CheckAndNotify(cmd.ErrOrStderr(), Version)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no args and no filters, show help.
			if len(args) == 0 && fileFilter == "" && commitFilter == "" &&
				authorFilter == "" && actorFilter == "" {
				return cmd.Help()
			}

			// Recall: preconditions required.
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}

			limit, explicit, err := resolveRecallLimit(cmd, limitFlag)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			filters := search.Filters{
				Query:         strings.Join(args, " "),
				File:          fileFilter,
				Commit:        commitFilter,
				Author:        authorFilter,
				Actor:         actorFilter,
				Limit:         limit,
				LimitExplicit: explicit,
				Explain:       explainFlag,
			}

			return runRecall(cmd, gitRoot, filters)
		},
	}

	// Recall filter flags on root command.
	cmd.Flags().StringVar(&fileFilter, "file", "", "Filter by file path (regex)")
	cmd.Flags().StringVar(&commitFilter, "commit", "", "Filter by git commit SHA")
	cmd.Flags().StringVar(&authorFilter, "author", "", "Filter by author email")
	cmd.Flags().StringVar(&actorFilter, "actor", "", "Filter by actor type (human|agent)")
	cmd.Flags().IntVarP(&limitFlag, "limit", "n", 0, "Max results (default 20; 0 = none; negative rejected)")
	cmd.Flags().BoolVar(&explainFlag, "explain", false, "Add per-layer scores and related-session joins to results")

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
	embedCmd := newEmbedCmd()
	embedCmd.GroupID = "advanced"

	cmd.AddCommand(initCmd, cleanCmd, versionCmd)
	cmd.AddCommand(checkpointCmd, pushCmd, syncCmd, logCmd)
	cmd.AddCommand(queryCmd, indexCmd, embedCmd)
	cmd.AddCommand(nomic.NewDaemonCmd())

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
