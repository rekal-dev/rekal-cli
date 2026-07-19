package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CursorAdapter discovers and parses Cursor agent transcripts from JSONL files.
//
// Layout (observed under ~/.cursor/projects/):
//
//	~/.cursor/projects/<sanitized-repo>/agent-transcripts/<uuid>/<uuid>.jsonl
//	~/.cursor/projects/<sanitized-repo>/agent-transcripts/<uuid>/subagents/<id>.jsonl
//
// Sanitization strips a leading slash and replaces path separators with dashes
// (e.g. /Users/frank/projects/rekal → Users-frank-projects-rekal). Unlike Claude
// Code, there is no leading dash.
type CursorAdapter struct{}

func (a *CursorAdapter) Name() string { return "cursor" }

func (a *CursorAdapter) Discover(repoPath string) ([]SessionRef, error) {
	sessionDir := FindCursorSessionDir(repoPath)
	if sessionDir == "" {
		return nil, nil
	}
	return discoverCursorSessionRefs(sessionDir)
}

// FindCursorSessionDir returns the Cursor agent-transcripts directory for repoPath.
func FindCursorSessionDir(repoPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cursor", "projects", SanitizeCursorRepoPath(repoPath), "agent-transcripts")
}

// SanitizeCursorRepoPath matches Cursor's project-directory naming:
// leading slash removed, remaining "/" replaced with "-".
// e.g. /Users/frank/projects/rekal → Users-frank-projects-rekal
func SanitizeCursorRepoPath(repoPath string) string {
	p := filepath.ToSlash(filepath.Clean(repoPath))
	p = strings.TrimPrefix(p, "/")
	return strings.ReplaceAll(p, "/", "-")
}

func discoverCursorSessionRefs(transcriptsDir string) ([]SessionRef, error) {
	entries, err := os.ReadDir(transcriptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var refs []SessionRef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		stem := e.Name()
		trunk := filepath.Join(transcriptsDir, stem, stem+".jsonl")
		if _, err := os.Stat(trunk); err != nil {
			continue
		}
		refs = append(refs, SessionRef{Path: trunk})

		subDir := filepath.Join(transcriptsDir, stem, "subagents")
		subEntries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if se.IsDir() || !strings.HasSuffix(se.Name(), ".jsonl") {
				continue
			}
			refs = append(refs, SessionRef{
				Path:       filepath.Join(subDir, se.Name()),
				ParentPath: trunk,
			})
		}
	}
	return refs, nil
}

func (a *CursorAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	payload := &SessionPayload{
		SessionID:  strings.TrimSuffix(filepath.Base(ref.Path), ".jsonl"),
		Source:     "cursor",
		ActorType:  "human",
		CapturedAt: time.Now().UTC(),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry cursorRawEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		// Control / status lines (e.g. turn_ended) carry no conversation.
		if entry.Role == "" {
			continue
		}

		text, tools := cursorExtractContent(entry.Message.Content)
		ts := cursorTimestampFromText(text)
		switch entry.Role {
		case "user":
			if text != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "human", Content: text, Timestamp: ts,
				})
			}
		case "assistant":
			if text != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "assistant", Content: text, Timestamp: ts,
				})
			}
			payload.ToolCalls = append(payload.ToolCalls, tools...)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if ref.ParentPath != "" {
		payload.ActorType = "agent"
		payload.ParentSessionPath = ref.ParentPath
		payload.AgentID = strings.TrimSuffix(filepath.Base(ref.Path), ".jsonl")
	}

	return payload, nil
}

type cursorRawEntry struct {
	Role    string `json:"role"`
	Type    string `json:"type"`
	Message struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type cursorContentPart struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func cursorExtractContent(raw json.RawMessage) (string, []ToolCall) {
	if len(raw) == 0 {
		return "", nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return asString, nil
	}
	var parts []cursorContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", nil
	}
	var texts []string
	var tools []ToolCall
	for _, p := range parts {
		switch p.Type {
		case "text":
			if p.Text != "" {
				texts = append(texts, p.Text)
			}
		case "tool_use":
			tc := ToolCall{Tool: p.Name}
			toolCallArgsFromMap(&tc, p.Input)
			tools = append(tools, tc)
		}
	}
	return strings.Join(texts, "\n"), tools
}

// cursorTimestampFromText pulls the optional leading <timestamp>…</timestamp>
// Cursor wraps around user_query blocks. Best-effort; zero time on miss.
func cursorTimestampFromText(text string) time.Time {
	const open, close = "<timestamp>", "</timestamp>"
	start := strings.Index(text, open)
	if start < 0 {
		return time.Time{}
	}
	start += len(open)
	end := strings.Index(text[start:], close)
	if end < 0 {
		return time.Time{}
	}
	raw := strings.TrimSpace(text[start : start+end])
	// Cursor writes e.g. "Friday, Jul 17, 2026, 6:10 PM (UTC+10)".
	// Strip the parenthetical zone — offsets like UTC+10 are not Go layout zones.
	if i := strings.LastIndex(raw, " ("); i > 0 {
		raw = raw[:i]
	}
	if t, err := time.Parse("Monday, Jan 2, 2006, 3:04 PM", raw); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC()
	}
	return time.Time{}
}
