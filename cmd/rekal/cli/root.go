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

const readingTheDigest = `

Reading the seed digest:
  Recall answers with one header line, then one line per session it found.

    INJECT top=0.87 gap=0.24 20 seeds
      s35 conf=0.87 t262 [reached 9× drilled 2×· "rebase capture"] "Now the integration test proving…"
      s12 conf=0.63 t54 "All done. Here's what was implemented: checkpoint now only writes…"
    KNOWLEDGE docs/design/merged-only-sharing.md=0.88 docs/usage.md=0.78

  Header
    INJECT      sessions worth reading were found
    KNOWLEDGE   no session stood out, but prose at HEAD looks relevant
    SILENCE     nothing above the floor; exits 1
    top=        highest absolute confidence in the set (corpus-independent)
    gap=        top minus the runner-up, so one clear answer reads apart from a tie

  Seed line
    s35         short id for this call; pass it to 'rekal query -s s35'
    conf=0.87   absolute confidence — judge relevance from this plus the snippet
    t262        turn index of the match; drill around it with -o 260 -n 5
    [reached…]  usage history: how often search surfaced it, how often an agent
                opened it. A drill is the load-bearing signal; surfacing is not.

  The verdict is a recommendation, not a decision. You judge.

Filters and flags (short forms are the cheapest to type and to emit):
  -p, --file <regex>    Only sessions that touched a matching path
  -c, --commit <sha>    Only sessions behind a commit (prefix or full SHA)
  -a, --author <email>  Only one person's sessions
  -A, --actor <type>    human or agent
  -n, --limit <int>     Results per framing (default 20; 0 = none). Soft:
                        the fused union across framings may exceed it
  -j, --json            Raw structured JSON instead of the digest
  -e, --explain         Per-layer scores and related sessions (needs --json)

  Filters combine, and they narrow the session search only — the KNOWLEDGE
  line is prose at HEAD and ignores them.

Workflow:
  rekal "keyword"                Search sessions → seed digest (-j for raw)
  rekal -p auth "token refresh"  Filter by file path
  rekal find "auth middleware"   Every ledger mention of a term (complete, time order)
  rekal query -s <id>            Drill into a session (readable turns; -j for raw)
  rekal query -s <id> -F         Include tool calls and files
  rekal query "SELECT ..."       Raw SQL → TSV rows (complete-set / temporal / analytical;
                                 -j for NDJSON; see 'rekal query --help' for schema)

Getting Started:
  rekal init                        Initialize Rekal in a git repository
  rekal checkpoint                  Capture the current session
  rekal push                        Share context with the team (merged work only)
  rekal sync                        Pull team context
  rekal index --include-all         Also recall your other repos' sessions (local only, never pushed)
  rekal embed                       Fill missing semantic embeddings (also runs after index/sync)
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
		jsonFlag     bool
	)

	cmd := &cobra.Command{
		Use:   "rekal [filters...] [query]",
		Short: "Rekal — gives your agent precise memory",
		Long:  "Rekal gives your agent precise memory — the exact context it needs for what it's working on." + readingTheDigest,
		Example: `  # Ask why something is the way it is, then read the conversation behind it
  rekal "why is the ledger append-only"
  rekal query -s s35 -o 260 -n 5

  # Narrow to one file's history, or to the work behind one commit
  rekal -p "cmd/rekal/cli/push.go" "force flag"
  rekal -c 93952c0 "what did we decide"

  # Only a teammate's sessions, or only what a human typed
  rekal -a frank@rekal.dev "merge gate"
  rekal -A human "why not squash"

  # Combine filters, and cap the result set
  rekal -p "search/" -a frank@rekal.dev -n 5 "ranking weights"

  # Machine-readable instead of the digest, with per-layer scores
  rekal -j "merge gate"
  rekal -j -e "merge gate"

  # Long forms work identically — clearer in scripts someone reads later
  rekal --file "search/" --limit 5 "ranking weights"`,
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
				TextQuery:     len(args) > 0,
				File:          fileFilter,
				Commit:        commitFilter,
				Author:        authorFilter,
				Actor:         actorFilter,
				Limit:         limit,
				LimitExplicit: explicit,
				Explain:       explainFlag,
			}

			return runRecall(cmd, gitRoot, filters, jsonFlag)
		},
	}

	// Recall filter flags on root command.
	cmd.Flags().StringVarP(&fileFilter, "file", "p", "", "Filter by file path (regex)")
	cmd.Flags().StringVarP(&commitFilter, "commit", "c", "", "Filter by git commit SHA")
	cmd.Flags().StringVarP(&authorFilter, "author", "a", "", "Filter by author email")
	cmd.Flags().StringVarP(&actorFilter, "actor", "A", "", "Filter by actor type (human|agent)")
	cmd.Flags().IntVarP(&limitFlag, "limit", "n", 0, "Results per framing (default 20; 0 = none; negative rejected). Soft: the RRF-fused union across framings may exceed it")
	cmd.Flags().BoolVarP(&explainFlag, "explain", "e", false, "Add per-layer scores and related-session joins to results")
	cmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Raw structured JSON instead of the default seed digest (for machine consumers)")

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
	findCmd := newFindCmd()
	findCmd.GroupID = "advanced"
	indexCmd := newIndexCmd()
	indexCmd.GroupID = "advanced"
	embedCmd := newEmbedCmd()
	embedCmd.GroupID = "advanced"

	cmd.AddCommand(initCmd, cleanCmd, versionCmd)
	cmd.AddCommand(checkpointCmd, pushCmd, syncCmd, logCmd)
	cmd.AddCommand(queryCmd, findCmd, indexCmd, embedCmd)
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
