package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/graph"
	"github.com/spf13/cobra"
)

// logDrillEdge spools a drill edge for the L1 recall citation graph — the
// strong "this session was opened" signal. Best-effort; a graph failure never
// breaks a drill.
func logDrillEdge(gitRoot, sessionID string) {
	if sessionID == "" {
		return
	}
	_ = graph.Append(gitRoot, []graph.Edge{{ //nolint:errcheck // best-effort telemetry
		TS:     time.Now().UTC(),
		Kind:   "drill",
		Target: sessionID,
	}})
}

func newQueryCmd() *cobra.Command {
	var (
		useIndex  bool
		sessionID string
		sqlFlag   string
		full      bool
		offset    int
		limit     int
		role      string
		jsonFlag  bool
	)

	cmd := &cobra.Command{
		Use:   "query [--sql \"<statement>\" | <sql> | --session <id> [--full] [--offset N] [--limit N] [--role human|assistant|human_steering|summary]]",
		Short: "Run raw SQL or drill into a session",
		Long: `Run raw SQL against the data or index DB, or drill into a specific session.

Session drill-down (--session) prints a readable turn transcript by default
(compact text an agent can act on). Add --json for a single JSON object,
--full for tool calls and files touched. Use --offset, --limit, and --role to
paginate through turns or filter by role (human, assistant, human_steering, or
summary — the exact stored role; steering turns are the text a human typed
while the agent was already working, summary turns are the harness-written
compaction distillations of the conversation so far). Output always includes
child_session_ids — subagent/workflow transcripts whose parent_session_id
points at this session — so an agent can navigate from a trunk conversation
into the transcript that actually matched.

SQL mode is explicit: --sql "<statement>". A bare positional statement
(rekal query "SELECT …") is accepted as shorthand. One SELECT per call, run on
a read-only connection — the ledger is append-only, so no statement here can
write to it. Rows print as TSV by default, or NDJSON with --json. Use --index
to query the index DB. --sql, --session, and a positional statement are
mutually exclusive.

Flags (short forms first — they are what an agent should emit):
  -s, --session <id>    Drill a session by short handle (s35) or full ULID
  -o, --offset <int>    Skip the first N turns (with --session)
  -n, --limit <int>     Return at most N turns, 0 = all (with --session)
  -r, --role <role>     human | assistant | human_steering | summary
  -F, --full            Add tool calls and files touched
  -q, --sql "<stmt>"    Explicit SQL mode (a bare positional works too)
  -i, --index           Run the SQL against index.db instead of data.db
  -j, --json            One object for a drill, NDJSON for SQL rows

  --offset/--limit/--role require --session. Short ids (s35) come from the
  digest of the most recent recall; a ULID always works.

Full queryable schema (FTS-internal tables — dict/docs/fields/stats/stopwords/
terms — and state tables — schema_meta/checkpoint_state/index_state — are engine
internals; ignore them).

DATA DB SCHEMA (.rekal/data.db):

  sessions        id, parent_session_id, session_hash, captured_at, actor_type,
                  agent_id, user_email, branch, source, team_name, workflow_name,
                  agent_type, description, spawn_depth
                  (parent_session_id/team_name/workflow_name/agent_type/
                  description/spawn_depth are optional harness metadata — NULL
                  for agents without the concept)
  turns           id, session_id, turn_index, role, content, ts
  tool_calls      id, session_id, call_order, tool, path, cmd_prefix
  checkpoints     id, git_sha, git_branch, user_email, ts, actor_type, agent_id,
                  exported
  files_touched   id, checkpoint_id, file_path, change_type
  checkpoint_sessions  checkpoint_id, session_id
  recall_edges    id, ts, kind, query, target_session_id
                  (L1 recall citation graph: one row per session reached
                  (kind='recall'|'drill'). Local-only — never pushed/synced)

INDEX DB SCHEMA (.rekal/index.db):

  turns_ft             id, session_id, turn_index, role, content, ts
  tool_calls_index     id, session_id, call_order, tool, path, cmd_prefix
  files_index          checkpoint_id, session_id, file_path, change_type
  session_facets       session_id, user_email, git_branch, actor_type, agent_id,
                       captured_at, turn_count, tool_call_count, file_count,
                       checkpoint_id, git_sha, parent_session_id, team_name,
                       workflow_name, agent_type, description, spawn_depth,
                       origin, facet_text
  file_cooccurrence    file_a, file_b, count
  session_embeddings   session_id, embedding, model, generated_at
                       PK: (session_id, model). Models: lsa-v1, nomic-v1.5-c8k
  knowledge_chunks     id, path, anchor, breadcrumb, start_line, end_line,
                       content, content_hash, blob_sha
                       (heading-anchored prose sections of tracked files at HEAD)
  knowledge_embeddings content_hash, model, embedding
  session_reach        target_session_id, reach_count, drill_count, last_query,
                       top_query, last_ts
                       (L1 reach aggregate, derived from data.db.recall_edges.
                       reach_count is every edge, drill_count only the ones
                       where an agent opened the session)

Note: turns.ts is a TIMESTAMP, not text. "ts LIKE '2023-05%'" raises a Binder
error; use ts BETWEEN TIMESTAMP '2023-05-01' AND TIMESTAMP '2023-06-01', or
CAST(ts AS VARCHAR) LIKE '2023-05%'. turns_ft.ts is a VARCHAR holding the same
instant as text, so compare it as text (or CAST it to TIMESTAMP first).`,
		Example: `  # Short forms — the compact way to drill a seed from a digest
  rekal query -s s35                      # readable turns
  rekal query -s s35 -o 260 -n 5          # a window around turn 262
  rekal query -s s35 -r human_steering    # only what the human typed
  rekal query -s s35 -F                   # add tool calls and files
  rekal query -i -q "SELECT count(*) FROM turns_ft"

  # Explicit SQL mode
  rekal query -q "SELECT id, branch FROM sessions ORDER BY captured_at DESC LIMIT 5"

  # Positional shorthand (same as --sql)
  rekal query "SELECT count(*) FROM turns WHERE role = 'human'"

  # Drill into a session (turns only)
  rekal query -s 01JNQX...

  # Drill into a session (turns + tool calls + files)
  rekal query -s 01JNQX... -F

  # Paginate through turns
  rekal query -s 01JNQX... -n 5
  rekal query -s 01JNQX... -o 5 -n 5

  # Filter by role
  rekal query -s 01JNQX... -r human
  rekal query -s 01JNQX... -r human -n 5

  # Recent sessions
  rekal query "SELECT id, user_email, branch, captured_at FROM sessions ORDER BY captured_at DESC LIMIT 5"

  # Sessions that touched a file
  rekal query "SELECT DISTINCT s.id, s.user_email, s.captured_at FROM tool_calls t JOIN sessions s ON t.session_id = s.id WHERE t.path LIKE '%auth%'"

  # Most-edited files
  rekal query "SELECT path, count(*) as n FROM tool_calls WHERE tool IN ('Write','Edit') AND path IS NOT NULL GROUP BY path ORDER BY n DESC LIMIT 10"

  # File co-occurrence (index DB)
  rekal query -i "SELECT * FROM file_cooccurrence WHERE file_a LIKE '%auth%' ORDER BY count DESC LIMIT 10"

  # Transcripts of one dynamic workflow, grouped under their trunk session (index DB)
  rekal query -i "SELECT session_id, agent_id, parent_session_id FROM session_facets WHERE workflow_name = 'release-flow'"

  # Embedding model counts
  rekal query -i "SELECT model, count(*) FROM session_embeddings GROUP BY model"

  # Prose knowledge chunks matching a term (index DB, knowledge layer)
  rekal query -i "SELECT path, anchor, start_line, end_line FROM knowledge_chunks WHERE content ILIKE '%merged-only%' ORDER BY path"

  # Turns in a date window (ts is TIMESTAMP — use BETWEEN, not LIKE)
  rekal query "SELECT ts, session_id, content FROM turns WHERE ts BETWEEN TIMESTAMP '2023-05-01' AND TIMESTAMP '2023-06-01' AND role='human' ORDER BY ts"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}

			// SQL statement comes from explicit --sql or the positional
			// shorthand — never both.
			sqlStmt := sqlFlag
			if len(args) > 0 {
				if sqlFlag != "" {
					return fmt.Errorf("--sql and a positional SQL statement are mutually exclusive")
				}
				sqlStmt = args[0]
			}

			// SQL (--sql / positional) and --session are mutually exclusive modes.
			if sessionID != "" && sqlStmt != "" {
				return fmt.Errorf("--session and SQL (--sql / positional) are mutually exclusive")
			}

			// --offset, --limit, --role require --session.
			if sessionID == "" && (offset != 0 || limit != 0 || role != "") {
				return fmt.Errorf("--offset, --limit, and --role require --session")
			}

			// --role matches the stored role exactly.
			if role != "" && role != "human" && role != "assistant" && role != "human_steering" && role != "summary" {
				return fmt.Errorf("--role must be \"human\", \"assistant\", \"human_steering\", or \"summary\"")
			}

			if sessionID != "" {
				return runSessionDrilldown(cmd, gitRoot, sessionID, full, offset, limit, role, jsonFlag)
			}

			if sqlStmt == "" {
				return fmt.Errorf("provide SQL via --sql \"<statement>\" (or a positional statement), or drill with --session <id>")
			}

			return runQuery(cmd, gitRoot, sqlStmt, useIndex, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&sqlFlag, "sql", "q", "", "SQL SELECT statement to run (explicit SQL mode; a bare positional statement is accepted as shorthand)")
	cmd.Flags().BoolVarP(&useIndex, "index", "i", false, "Run SQL against the index DB instead of the data DB")
	cmd.Flags().StringVarP(&sessionID, "session", "s", "", "Show session conversation by short handle (s3) or ULID")
	cmd.Flags().BoolVarP(&full, "full", "F", false, "Include tool calls and files in session output")
	cmd.Flags().IntVarP(&offset, "offset", "o", 0, "Skip first N turns (requires --session)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 0, "Max turns to return, 0 = no limit (requires --session; note: the opposite of top-level -n/--limit, where 0 = no results)")
	cmd.Flags().StringVarP(&role, "role", "r", "", "Filter turns by role: human, assistant, human_steering, or summary (requires --session)")
	cmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "JSON instead of the default text/TSV (session → one object; SQL → NDJSON)")
	return cmd
}

// sessionOutput is the JSON structure for session drill-down.
type sessionOutput struct {
	SessionID string `json:"session_id"`
	// Sid is the query-time short handle (sN) when resolvable from index.db.
	Sid        string `json:"sid,omitempty"`
	Author     string `json:"author"`
	Actor      string `json:"actor"`
	Branch     string `json:"branch"`
	CapturedAt string `json:"captured_at"`

	// Optional harness metadata — omitted for agents without the concept
	// (see docs/agent-metadata.md).
	AgentID         string `json:"agent_id,omitempty"`
	TeamName        string `json:"team_name,omitempty"`
	WorkflowName    string `json:"workflow_name,omitempty"`
	ParentSessionID string `json:"parent_session_id,omitempty"`

	// Optional Task subagent meta.json sidecar fields — omitted for
	// sessions that aren't a subagent transcript (see claude.go's
	// subagentMeta doc).
	AgentType   string `json:"agent_type,omitempty"`
	Description string `json:"description,omitempty"`
	SpawnDepth  int    `json:"spawn_depth,omitempty"`

	// ChildSessionIDs are sessions whose parent_session_id points at this
	// session — the grouped structure a recall result collapses. Lets an
	// agent navigate from a trunk conversation into its subagent/workflow
	// transcripts. Empty when this session has no children.
	ChildSessionIDs []string `json:"child_session_ids,omitempty"`

	TotalTurns int          `json:"total_turns"`
	Offset     int          `json:"offset,omitempty"`
	Limit      int          `json:"limit,omitempty"`
	HasMore    bool         `json:"has_more,omitempty"`
	Turns      []turnOutput `json:"turns"`
	// Commits closes the loop back to the code. Recall already goes commit →
	// session (--commit); without this the drill is one-way, handing the agent
	// the reasoning with no pointer to the diff it produced.
	Commits   []commitRef      `json:"commits,omitempty"`
	ToolCalls []toolCallOutput `json:"tool_calls,omitempty"`
	Files     []string         `json:"files_touched,omitempty"`
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

// drilldownSource selects the queries used to render a session drill-down.
// Own sessions live in the data DB; teammate sessions pulled via `rekal sync`
// exist only in the index DB (data.db is owner-only by design).
type drilldownSource struct {
	turns     func(*sql.DB, string, db.TurnPageOptions) ([]db.TurnRow, int, error)
	toolCalls func(*sql.DB, string) ([]db.ToolCallRow, error)
	files     func(*sql.DB, string) ([]string, error)
	commits   func(*sql.DB, string) ([]string, error)
	children  func(*sql.DB, string) ([]string, error)
}

// commitRef is one commit a session produced: the short sha to hand to
// git show, and the subject line so the agent can tell them apart without
// running it.
type commitRef struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject,omitempty"`
}

var dataDrilldownSource = drilldownSource{
	turns:     db.QueryTurnsPage,
	toolCalls: db.QueryToolCalls,
	files:     querySessionFilesFromData,
	commits:   querySessionCommitsFromData,
	children:  db.QueryChildSessionIDs,
}

var indexDrilldownSource = drilldownSource{
	turns:     db.QueryTurnsPageFromIndex,
	toolCalls: db.QueryToolCallsFromIndex,
	files:     querySessionFilesFromIndex,
	commits:   querySessionCommitsFromIndex,
	children:  db.QueryChildSessionIDsFromIndex,
}

// sessionNotFoundError reports a missing session in plain words. A lookup
// miss is sql.ErrNoRows, which is not something to show a caller; any other
// error is a real DB failure and keeps its detail.
func sessionNotFoundError(handle string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("session not found: %q", handle)
	}
	return fmt.Errorf("session not found: %w", err)
}

func runSessionDrilldown(cmd *cobra.Command, gitRoot, handle string, full bool, offset, limit int, role string, jsonCompact bool) error {
	// Open index once: resolve sN→ULID, emit sid on output, and fall back
	// for teammate sessions that exist only in index.db.
	var indexDB *sql.DB
	var sidMap *db.SessionSIDMap
	if d, err := db.OpenIndex(gitRoot); err == nil {
		indexDB = d
		defer indexDB.Close()
		if err := db.MigrateIndexSchema(indexDB); err != nil {
			return fmt.Errorf("migrate index db: %w", err)
		}
		if m, err := db.LoadSessionSIDMap(indexDB); err == nil {
			sidMap = m
		}
	} else if db.IsShortSessionHandle(handle) {
		return fmt.Errorf("short session handle %q needs index.db — run 'rekal index'", handle)
	}

	sessionID, err := sidMap.Resolve(handle)
	if err != nil {
		return err
	}
	shortSid := sidMap.SID(sessionID)
	if shortSid == "" && db.IsShortSessionHandle(handle) {
		shortSid = handle
	}

	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data db: %w", err)
	}
	defer dataDB.Close()

	// The data DB may have been written by an older rekal (shared via git);
	// migrations are additive and idempotent.
	if err := db.MigrateDataSchema(dataDB); err != nil {
		return fmt.Errorf("migrate data db: %w", err)
	}

	session, dataErr := db.QuerySession(dataDB, sessionID)
	if dataErr == nil {
		// L1 recall graph: an explicit drill is the strong "this memory was
		// used" signal. Spool a drill edge for the checkpoint drain (no query
		// context — the recall that led here is a separate process).
		logDrillEdge(gitRoot, sessionID)
		return renderSessionDrilldown(cmd, gitRoot, dataDB, session, dataDrilldownSource, full, offset, limit, role, shortSid, jsonCompact)
	}

	// Not in the data DB — fall back to the index DB, where teammate
	// sessions from `rekal sync` live.
	if indexDB == nil {
		return sessionNotFoundError(handle, dataErr)
	}
	session, err = db.QuerySessionFromIndex(indexDB, sessionID)
	if err != nil {
		return sessionNotFoundError(handle, err)
	}
	logDrillEdge(gitRoot, sessionID)
	return renderSessionDrilldown(cmd, gitRoot, indexDB, session, indexDrilldownSource, full, offset, limit, role, shortSid, jsonCompact)
}

func renderSessionDrilldown(cmd *cobra.Command, gitRoot string, d *sql.DB, session *db.SessionRow, src drilldownSource, full bool, offset, limit int, role, shortSid string, jsonCompact bool) error {
	sessionID := session.ID

	turns, total, err := src.turns(d, sessionID, db.TurnPageOptions{
		Offset: offset,
		Limit:  limit,
		Role:   role,
	})
	if err != nil {
		return fmt.Errorf("query turns: %w", err)
	}

	output := sessionOutput{
		SessionID:  session.ID,
		Sid:        shortSid,
		Author:     session.Email,
		Actor:      session.ActorType,
		Branch:     session.Branch,
		CapturedAt: session.CapturedAt,
		TotalTurns: total,
		Offset:     offset,
		Limit:      limit,
		// Always an array, never null — a role filter that matches nothing
		// (e.g. --role summary on a session with no summary) must still emit
		// `"turns": []` so agents can iterate without a null guard.
		Turns: []turnOutput{},

		// Optional harness metadata; empty (and omitted from JSON) for
		// agents without the concept.
		AgentID:         session.AgentID,
		TeamName:        session.TeamName,
		WorkflowName:    session.WorkflowName,
		ParentSessionID: session.ParentSessionID,
		AgentType:       session.AgentType,
		Description:     session.Description,
		SpawnDepth:      session.SpawnDepth,
	}

	if children, err := src.children(d, sessionID); err == nil {
		output.ChildSessionIDs = children
	}

	// has_more is true when there are more turns beyond this page.
	if limit > 0 {
		output.HasMore = offset+len(turns) < total
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
		toolCalls, err := src.toolCalls(d, sessionID)
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

		// Get files touched by this session.
		files, err := src.files(d, sessionID)
		if err != nil {
			return fmt.Errorf("query files: %w", err)
		}
		output.Files = files
	}

	if shas, err := src.commits(d, sessionID); err == nil && len(shas) > 0 {
		subjects := gitx.CommitMessages(gitRoot, shas)
		for _, sha := range shas {
			ref := commitRef{SHA: sha}
			if msg, ok := subjects[sha]; ok {
				ref.Subject, _, _ = strings.Cut(msg, "\n")
			}
			output.Commits = append(output.Commits, ref)
		}
	}

	if !jsonCompact {
		// Default: agent-readable dialogue (view.py's session mode).
		fmt.Fprintln(cmd.OutOrStdout(), viewSession(&output))
		return nil
	}
	data, err := json.Marshal(output)
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

// querySessionFilesFromIndex returns files touched by a session from the
// index DB (files_index) — used for remote/teammate sessions.
func querySessionFilesFromIndex(indexDB *sql.DB, sessionID string) ([]string, error) {
	rows, err := indexDB.Query(`
		SELECT DISTINCT file_path
		FROM files_index
		WHERE session_id = $1
		ORDER BY file_path
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

// hasTrailingStatement reports whether q carries a second statement after the
// first — a ';' outside string literals, quoted identifiers and comments, with
// anything but whitespace and comments behind it. A plain trailing semicolon
// ("SELECT 1;") and a semicolon inside a literal ("SELECT ';'") are fine.
//
// Byte scanning is safe here: every character it looks for is ASCII, and no
// byte of a multi-byte UTF-8 rune is ever an ASCII byte.
func hasTrailingStatement(q string) bool {
	i := firstTopLevelSemicolon(q)
	return i >= 0 && !onlyBlankOrComments(q[i+1:])
}

// firstTopLevelSemicolon returns the index of the first ';' in q that is not
// inside a string literal, a quoted identifier or a comment, or -1.
func firstTopLevelSemicolon(q string) int {
	for i := 0; i < len(q); i++ {
		switch q[i] {
		case ';':
			return i
		case '\'', '"':
			if end := skipQuoted(q, i); end > i {
				i = end
			}
		case '-':
			if i+1 < len(q) && q[i+1] == '-' {
				i = skipLineComment(q, i)
			}
		case '/':
			if i+1 < len(q) && q[i+1] == '*' {
				i = skipBlockComment(q, i)
			}
		}
	}
	return -1
}

// onlyBlankOrComments reports whether s holds nothing but whitespace, comments
// and empty statements.
func onlyBlankOrComments(s string) bool {
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == ' ', c == '\t', c == '\n', c == '\r', c == ';':
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			i = skipLineComment(s, i)
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			i = skipBlockComment(s, i)
		default:
			return false
		}
	}
	return true
}

// skipQuoted returns the index of the closing quote of the literal or quoted
// identifier that starts at i, or len(s)-1 when it is never closed. A doubled
// quote (” / "") is an escaped quote and does not close the literal.
func skipQuoted(s string, i int) int {
	quote := s[i]
	for j := i + 1; j < len(s); j++ {
		if s[j] != quote {
			continue
		}
		if j+1 < len(s) && s[j+1] == quote {
			j++
			continue
		}
		return j
	}
	return len(s) - 1
}

// skipLineComment returns the index of the newline ending the -- comment that
// starts at i, or len(s)-1 when the comment runs to the end.
func skipLineComment(s string, i int) int {
	if n := strings.IndexByte(s[i:], '\n'); n >= 0 {
		return i + n
	}
	return len(s) - 1
}

// skipBlockComment returns the index of the '/' closing the /* comment that
// starts at i, or len(s)-1 when it is never closed.
func skipBlockComment(s string, i int) int {
	if n := strings.Index(s[i+2:], "*/"); n >= 0 {
		return i + 2 + n + 1
	}
	return len(s) - 1
}

func runQuery(cmd *cobra.Command, gitRoot, query string, useIndex, jsonOut bool) error {
	// Read-only: only allow SELECT statements.
	normalized := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(normalized, "SELECT") {
		return fmt.Errorf("only SELECT statements are allowed")
	}
	// ...and only one of them. The driver runs every statement in the string it
	// is handed, so a leading-keyword check alone accepts
	// "SELECT 1; DELETE FROM turns". The read-only handle below is what makes
	// the ledger safe; this check is here so the answer is a plain sentence
	// rather than a driver error about a read-only database.
	if hasTrailingStatement(query) {
		return fmt.Errorf("one statement per query (found another statement after the ';')")
	}

	var d *sql.DB
	var err error
	if useIndex {
		d, err = db.OpenIndexReadOnly(gitRoot)
	} else {
		d, err = db.OpenDataReadOnly(gitRoot)
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

	collected := make([]map[string]interface{}, 0)
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
			if b, ok := v.([]byte); ok {
				v = string(b) // []byte → string for output
			}
			row[col] = v
		}
		collected = append(collected, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	out := cmd.OutOrStdout()
	if jsonOut {
		// Raw NDJSON — one object per row (machine consumers).
		for i, row := range collected {
			data, merr := json.Marshal(row)
			if merr != nil {
				return fmt.Errorf("marshal: %w", merr)
			}
			if i > 0 {
				fmt.Fprintln(out)
			}
			fmt.Fprint(out, string(data))
		}
		if len(collected) > 0 {
			fmt.Fprintln(out)
		}
		return nil
	}

	// Default: agent-readable TSV (view.py's row mode, in the query's own
	// column order).
	fmt.Fprintln(out, viewRows(cols, collected))
	return nil
}

// querySessionCommitsFromData returns the git SHAs a session produced, oldest
// first, from the data DB.
func querySessionCommitsFromData(dataDB *sql.DB, sessionID string) ([]string, error) {
	return scanCommitSHAs(dataDB, `
		SELECT DISTINCT c.git_sha, min(c.ts) AS first_ts
		FROM checkpoint_sessions cs
		JOIN checkpoints c ON c.id = cs.checkpoint_id
		WHERE cs.session_id = $1 AND c.git_sha IS NOT NULL AND c.git_sha <> ''
		GROUP BY c.git_sha ORDER BY first_ts`, sessionID)
}

// querySessionCommitsFromIndex is the same for a teammate session that exists
// only in the index, where the anchor lives on session_facets.
func querySessionCommitsFromIndex(indexDB *sql.DB, sessionID string) ([]string, error) {
	return scanCommitSHAs(indexDB, `
		SELECT git_sha FROM session_facets
		WHERE session_id = $1 AND git_sha IS NOT NULL AND git_sha <> ''`, sessionID)
}

func scanCommitSHAs(d *sql.DB, query, sessionID string) ([]string, error) {
	rows, err := d.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var shas []string
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sha string
		if len(cols) == 1 {
			if err := rows.Scan(&sha); err != nil {
				return nil, err
			}
		} else {
			var ignored interface{}
			if err := rows.Scan(&sha, &ignored); err != nil {
				return nil, err
			}
		}
		shas = append(shas, sha)
	}
	return shas, rows.Err()
}
