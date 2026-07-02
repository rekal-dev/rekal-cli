package session

// Adapter discovers and parses sessions for a specific AI agent.
type Adapter interface {
	// Name returns the agent identifier (e.g. "claude", "codex", "gemini", "opencode").
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
	&CodexAdapter{},
	&GeminiAdapter{},
	&OpenCodeAdapter{},
}
