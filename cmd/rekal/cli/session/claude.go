package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ClaudeAdapter discovers and parses Claude Code sessions from JSONL files.
type ClaudeAdapter struct{}

func (a *ClaudeAdapter) Name() string { return "claude" }

// Discover returns trunk session files first, followed by subagent transcript
// files, so a caller processing refs in order sees each parent session before
// the subagent sessions that link to it.
func (a *ClaudeAdapter) Discover(repoPath string) ([]SessionRef, error) {
	sessionDir := FindSessionDir(repoPath)
	if sessionDir == "" {
		return nil, nil
	}
	return discoverSessionRefs(sessionDir)
}

// discoverSessionRefs scans a Claude Code project session directory and
// returns refs for trunk sessions followed by their subagent transcripts.
func discoverSessionRefs(sessionDir string) ([]SessionRef, error) {
	files, err := FindSessionFiles(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	refs := make([]SessionRef, len(files))
	for i, f := range files {
		refs[i] = SessionRef{Path: f}
	}

	refs = append(refs, findSubagentRefs(sessionDir)...)
	return refs, nil
}

func (a *ClaudeAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	// Subagent transcripts consist of sidechain entries; include them.
	opts := TranscriptOptions{IncludeSidechain: ref.ParentPath != ""}
	payload, err := ParseTranscriptWithOptions(data, opts)
	if err != nil {
		return nil, err
	}
	payload.Source = "claude"
	payload.CapturedAt = time.Now().UTC()

	if ref.ParentPath != "" {
		payload.ActorType = "agent"
		payload.ParentSessionPath = ref.ParentPath
		payload.WorkflowName = workflowName(ref.Path)

		// Agent identity: the transcript entries' own agentId field (set by
		// Claude Code on every line of a real subagent transcript — verified
		// directly against captured sessions), falling back to the filename
		// for older formats that predate it. The meta.json sidecar carries
		// no agentId/teamName of its own (see subagentMeta doc) — team name
		// stays sourced purely from the entries' teamName field, same as
		// before, which remains unverified against a real teammates run (no
		// such run was available to capture; see docs/REVIEW_2026-07-03.md).
		if payload.AgentID == "" {
			payload.AgentID = subagentID(ref.Path)
		}

		if meta := readSubagentMeta(ref.Path); meta != nil {
			payload.AgentType = meta.AgentType
			payload.Description = meta.Description
			payload.SpawnDepth = meta.SpawnDepth
		}
	}
	return payload, nil
}

// subagentMeta is the agent-<id>.meta.json sidecar Claude Code writes next to
// a subagent transcript. Verified directly against real
// ~/.claude/projects/*/subagents/*.meta.json sidecars (captured from Task
// tool subagent runs — see cmd/rekal/cli/session/testdata/real): the shape
// is {agentType, description, toolUseId, spawnDepth}. Notably it carries no
// agentId or teamName field — an earlier version of this struct assumed
// both existed, which meant the sidecar contributed nothing (silently, no
// error) beyond a fallback that was already redundant with the transcript
// entries' own agentId. Only Task-tool subagent sidecars have been observed;
// whether a true Claude Code teammates run's sidecar (if one exists) carries
// additional fields is still unverified.
type subagentMeta struct {
	AgentType   string `json:"agentType"`
	Description string `json:"description"`
	ToolUseID   string `json:"toolUseId"`
	SpawnDepth  int    `json:"spawnDepth"`
}

// readSubagentMeta loads the meta sidecar for a subagent transcript, if
// present. Older session formats have no sidecars; returns nil.
func readSubagentMeta(transcriptPath string) *subagentMeta {
	metaPath := strings.TrimSuffix(transcriptPath, ".jsonl") + ".meta.json"
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil
	}
	var m subagentMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return &m
}

// FindSessionDir returns the Claude Code session directory for the given repo path.
// It honors CLAUDE_CONFIG_DIR when set (Claude Code's supported way to move
// storage off ~/.claude); otherwise it defaults to
// ~/.claude/projects/<sanitized-repo-path>/.
func FindSessionDir(repoPath string) string {
	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configDir = filepath.Join(home, ".claude")
	}
	sanitized := SanitizeRepoPath(repoPath)
	return filepath.Join(configDir, "projects", sanitized)
}

// FindSessionFiles lists all top-level .jsonl session files in the given directory.
func FindSessionFiles(sessionDir string) ([]string, error) {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".jsonl") {
			files = append(files, filepath.Join(sessionDir, e.Name()))
		}
	}
	return files, nil
}

// findSubagentRefs discovers subagent and dynamic-workflow transcripts under
// <sessionDir>/<session-stem>/subagents/. Claude Code writes Task/teammate
// transcripts as subagents/agent-<id>.jsonl and dynamic-workflow side-channel
// transcripts as subagents/workflows/<name>/*.jsonl, where <session-stem> is
// the parent session filename without .jsonl. Each ref carries ParentPath
// pointing at the trunk session file so the payload can be linked to it.
//
// The subagents/agent-<id>.jsonl layout and its .meta.json sidecar are now
// verified directly against real captured Task-tool subagent transcripts
// (cmd/rekal/cli/session/testdata/real); the workflows/<name>/ side-channel
// layout remains unverified — no dynamic-workflow transcript was available
// to capture — so it stays implemented defensively and degrades to a no-op
// when absent.
func findSubagentRefs(sessionDir string) []SessionRef {
	var refs []SessionRef

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		parentPath := filepath.Join(sessionDir, e.Name()+".jsonl")
		if _, err := os.Stat(parentPath); err != nil {
			// Not a session-stem directory (e.g. memory/); skip.
			continue
		}
		subagentsDir := filepath.Join(sessionDir, e.Name(), "subagents")

		// Ordinary subagent / teammate transcripts.
		agentFiles, _ := filepath.Glob(filepath.Join(subagentsDir, "agent-*.jsonl"))
		for _, f := range agentFiles {
			refs = append(refs, SessionRef{Path: f, ParentPath: parentPath})
		}

		// Dynamic-workflow side-channel transcripts.
		workflowFiles, _ := filepath.Glob(filepath.Join(subagentsDir, "workflows", "*", "*.jsonl"))
		for _, f := range workflowFiles {
			refs = append(refs, SessionRef{Path: f, ParentPath: parentPath})
		}
	}
	return refs
}

// subagentID derives a fallback agent identifier from a subagent transcript
// path when no meta sidecar or entry-level agentId is available.
// subagents/agent-<id>.jsonl → <id>; subagents/workflows/<name>/<file>.jsonl →
// <file> (the workflow name is stored separately as first-class metadata).
func subagentID(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	return strings.TrimPrefix(base, "agent-")
}

// workflowName returns the dynamic-workflow name for transcripts under
// subagents/workflows/<name>/, or "" for ordinary subagent transcripts.
func workflowName(path string) string {
	dir := filepath.Dir(path)
	if filepath.Base(filepath.Dir(dir)) == "workflows" {
		return filepath.Base(dir)
	}
	return ""
}
