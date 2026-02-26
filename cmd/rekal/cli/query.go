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
		Args:  cobra.MaximumNArgs(1),
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
