package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rekal-dev/cli/cmd/rekal/cli/db"
	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	var (
		useIndex  bool
		sessionID string
		full      bool
	)

	cmd := &cobra.Command{
		Use:   "query [<sql> | --session <id>]",
		Short: "Run raw SQL or drill into a session",
		Long: `Run raw SQL against the data or index DB, or drill into a specific session.

Session drill-down (--session) returns the full conversation as JSON. Add --full
to include tool calls and files touched.

Raw SQL mode accepts SELECT statements only. Output is one JSON object per row.
Use --index to query the index DB instead of the data DB.

DATA DB SCHEMA (.rekal/data.db):

  sessions        id, parent_session_id, session_hash, captured_at, actor_type,
                  agent_id, user_email, branch
  turns           id, session_id, turn_index, role, content, ts
  tool_calls      id, session_id, call_order, tool, path, cmd_prefix
  checkpoints     id, git_sha, git_branch, user_email, ts, actor_type, agent_id,
                  exported
  files_touched   id, checkpoint_id, file_path, change_type
  checkpoint_sessions  checkpoint_id, session_id

INDEX DB SCHEMA (.rekal/index.db):

  turns_ft             id, session_id, turn_index, role, content, ts
  tool_calls_index     id, session_id, call_order, tool, path, cmd_prefix
  files_index          checkpoint_id, session_id, file_path, change_type
  session_facets       session_id, user_email, git_branch, actor_type, agent_id,
                       captured_at, turn_count, tool_call_count, file_count,
                       checkpoint_id, git_sha
  file_cooccurrence    file_a, file_b, count
  session_embeddings   session_id, embedding, model, generated_at`,
		Example: `  # Drill into a session (turns only)
  rekal query --session 01JNQX...

  # Drill into a session (turns + tool calls + files)
  rekal query --session 01JNQX... --full

  # Recent sessions
  rekal query "SELECT id, user_email, branch, captured_at FROM sessions ORDER BY captured_at DESC LIMIT 5"

  # Sessions that touched a file
  rekal query "SELECT DISTINCT s.id, s.user_email, s.captured_at FROM tool_calls t JOIN sessions s ON t.session_id = s.id WHERE t.path LIKE '%auth%'"

  # Most-edited files
  rekal query "SELECT path, count(*) as n FROM tool_calls WHERE tool IN ('Write','Edit') AND path IS NOT NULL GROUP BY path ORDER BY n DESC LIMIT 10"

  # File co-occurrence (index DB)
  rekal query --index "SELECT * FROM file_cooccurrence WHERE file_a LIKE '%auth%' ORDER BY count DESC LIMIT 10"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			// --session and positional SQL are mutually exclusive.
			if sessionID != "" && len(args) > 0 {
				return fmt.Errorf("--session and SQL argument are mutually exclusive")
			}

			if sessionID != "" {
				return runSessionDrilldown(cmd, gitRoot, sessionID, full)
			}

			if len(args) == 0 {
				return fmt.Errorf("provide a SQL query or use --session <id>")
			}

			return runQuery(cmd, gitRoot, args[0], useIndex)
		},
	}

	cmd.Flags().BoolVar(&useIndex, "index", false, "Run SQL against the index DB instead of the data DB")
	cmd.Flags().StringVar(&sessionID, "session", "", "Show session conversation by ID")
	cmd.Flags().BoolVar(&full, "full", false, "Include tool calls and files in session output")
	return cmd
}

// sessionOutput is the JSON structure for session drill-down.
type sessionOutput struct {
	SessionID  string           `json:"session_id"`
	Author     string           `json:"author"`
	Actor      string           `json:"actor"`
	Branch     string           `json:"branch"`
	CapturedAt string           `json:"captured_at"`
	Turns      []turnOutput     `json:"turns"`
	ToolCalls  []toolCallOutput `json:"tool_calls,omitempty"`
	Files      []string         `json:"files_touched,omitempty"`
}

type turnOutput struct {
	Index   int    `json:"index"`
	Role    string `json:"role"`
	Content string `json:"content"`
	Ts      string `json:"ts,omitempty"`
}

type toolCallOutput struct {
	Order int    `json:"order"`
	Tool  string `json:"tool"`
	Path  string `json:"path,omitempty"`
}

func runSessionDrilldown(cmd *cobra.Command, gitRoot, sessionID string, full bool) error {
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data db: %w", err)
	}
	defer dataDB.Close()

	session, err := db.QuerySession(dataDB, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	turns, err := db.QueryTurns(dataDB, sessionID)
	if err != nil {
		return fmt.Errorf("query turns: %w", err)
	}

	output := sessionOutput{
		SessionID:  session.ID,
		Author:     session.Email,
		Actor:      session.ActorType,
		Branch:     session.Branch,
		CapturedAt: session.CapturedAt,
	}

	for _, t := range turns {
		output.Turns = append(output.Turns, turnOutput{
			Index:   t.TurnIndex,
			Role:    t.Role,
			Content: t.Content,
			Ts:      t.Ts,
		})
	}

	if full {
		toolCalls, err := db.QueryToolCalls(dataDB, sessionID)
		if err != nil {
			return fmt.Errorf("query tool_calls: %w", err)
		}
		for _, tc := range toolCalls {
			output.ToolCalls = append(output.ToolCalls, toolCallOutput{
				Order: tc.CallOrder,
				Tool:  tc.Tool,
				Path:  tc.Path,
			})
		}

		// Get files from checkpoint_sessions → files_touched.
		files, err := querySessionFilesFromData(dataDB, sessionID)
		if err != nil {
			return fmt.Errorf("query files: %w", err)
		}
		output.Files = files
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func querySessionFilesFromData(dataDB *sql.DB, sessionID string) ([]string, error) {
	rows, err := dataDB.Query(`
		SELECT DISTINCT ft.file_path
		FROM checkpoint_sessions cs
		JOIN files_touched ft ON ft.checkpoint_id = cs.checkpoint_id
		WHERE cs.session_id = $1
		ORDER BY ft.file_path
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var files []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func runQuery(cmd *cobra.Command, gitRoot, query string, useIndex bool) error {
	// Read-only: only allow SELECT statements.
	normalized := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(normalized, "SELECT") {
		return fmt.Errorf("only SELECT statements are allowed")
	}

	var d *sql.DB
	var err error
	if useIndex {
		d, err = db.OpenIndex(gitRoot)
	} else {
		d, err = db.OpenData(gitRoot)
	}
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer d.Close()

	rows, err := d.Query(query)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("columns: %w", err)
	}

	out := cmd.OutOrStdout()
	first := true

	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			v := values[i]
			// Convert []byte to string for JSON output.
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			row[col] = v
		}

		data, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}

		if !first {
			fmt.Fprintln(out)
		}
		fmt.Fprint(out, string(data))
		first = false
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	// Trailing newline if we printed anything.
	if !first {
		fmt.Fprintln(out)
	}

	return nil
}
