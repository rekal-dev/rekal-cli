package session

// Adapter discovers and parses sessions for a specific AI agent.
type Adapter interface {
	// Name returns the agent identifier (e.g. "claude", "cursor", "codex", "gemini", "opencode").
	Name() string
	// Discover returns session references for the given repo path.
	Discover(repoPath string) ([]SessionRef, error)
	// Parse reads a session ref and returns a parsed payload.
	Parse(ref SessionRef) (*SessionPayload, error)
}

// SessionRef identifies a session to parse. For file-based agents, Path is set.
// For DB-based agents (OpenCode), DBID is set.
type SessionRef struct {
	Path string // file path for JSONL/JSON agents
	DBID string // session ID for DB-based agents

	// ParentPath is the trunk session file this ref belongs to when the ref
	// points at a subagent transcript (Claude Code subagents/ directory).
	// Empty for top-level sessions.
	ParentPath string
}

// Adapters is the registry of all known agent adapters.
// Checkpoint iterates this list to discover sessions from all agents.
var Adapters = []Adapter{
	&ClaudeAdapter{},
	&CursorAdapter{},
	&CodexAdapter{},
	&GeminiAdapter{},
	&OpenCodeAdapter{},
}

// toolCallArgsFromMap fills in tc.Path (preferring file_path, falling back
// to path) and tc.CmdPrefix (truncated command) from a generic tool-call
// argument map — the shape Codex, OpenCode, and Gemini's JSON tool-call
// blocks all use (unlike Claude's, which has a more precisely known input
// schema and uses a typed struct instead — see parse.go's extractToolCall).
// Centralizing this here means a future change to path/command extraction
// only needs to happen once for the three harnesses that share this shape,
// instead of independently in each adapter.
func toolCallArgsFromMap(tc *ToolCall, args map[string]interface{}) {
	if p, ok := args["file_path"].(string); ok {
		tc.Path = p
	} else if p, ok := args["path"].(string); ok {
		tc.Path = p
	}
	if cmd, ok := args["command"].(string); ok {
		tc.CmdPrefix = truncate(cmd, 100)
	}
}
