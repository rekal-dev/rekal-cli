package session

import (
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
		payload.AgentID = subagentID(ref.Path)
		payload.ParentSessionPath = ref.ParentPath
	}
	return payload, nil
}

// FindSessionDir returns the Claude Code session directory for the given repo path.
// Returns ~/.claude/projects/<sanitized-repo-path>/.
func FindSessionDir(repoPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	sanitized := SanitizeRepoPath(repoPath)
	return filepath.Join(home, ".claude", "projects", sanitized)
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

// subagentID derives a stable agent identifier from a subagent transcript path.
// subagents/agent-<id>.jsonl → <id>; subagents/workflows/<name>/<file>.jsonl →
// workflow:<name>/<file>.
func subagentID(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	dir := filepath.Dir(path)
	if filepath.Base(filepath.Dir(dir)) == "workflows" {
		return "workflow:" + filepath.Base(dir) + "/" + base
	}
	return strings.TrimPrefix(base, "agent-")
}
