package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CodexAdapter discovers and parses OpenAI Codex CLI sessions from JSONL files.
type CodexAdapter struct{}

func (a *CodexAdapter) Name() string { return "codex" }

func (a *CodexAdapter) Discover(repoPath string) ([]SessionRef, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}

	var refs []SessionRef
	dirs := []string{
		filepath.Join(home, ".codex", "sessions"),
		filepath.Join(home, ".codex", "archived_sessions"),
	}

	for _, dir := range dirs {
		matches, err := findJSONLFiles(dir)
		if err != nil {
			continue
		}
		for _, f := range matches {
			if codexSessionMatchesRepo(f, repoPath) {
				refs = append(refs, SessionRef{Path: f})
			}
		}
	}
	return refs, nil
}

func (a *CodexAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	payload := &SessionPayload{
		Source:    "codex",
		ActorType: "human",
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		var entry codexRawEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		switch entry.Type {
		case "session_meta":
			if payload.SessionID == "" {
				payload.SessionID = entry.SessionID
			}
			if payload.Branch == "" {
				var meta codexSessionMeta
				if err := json.Unmarshal(entry.Payload, &meta); err == nil && meta.Git.Branch != "" {
					payload.Branch = meta.Git.Branch
				}
			}

		case "event_msg":
			var msg codexEventMsg
			if err := json.Unmarshal(entry.Payload, &msg); err != nil {
				continue
			}
			ts := parseTimestamp(entry.Timestamp)
			switch msg.Type {
			case "user_message":
				if msg.Message != "" {
					payload.Turns = append(payload.Turns, Turn{
						Role: "human", Content: msg.Message, Timestamp: ts,
					})
				}
			case "agent_message":
				if msg.Message != "" {
					payload.Turns = append(payload.Turns, Turn{
						Role: "assistant", Content: msg.Message, Timestamp: ts,
					})
				}
			}

		case "response_item":
			var item codexResponseItem
			if err := json.Unmarshal(entry.Payload, &item); err != nil {
				continue
			}
			switch item.Type {
			case "function_call":
				tc := ToolCall{Tool: item.Name}
				if item.Arguments != "" {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(item.Arguments), &args); err == nil {
						toolCallArgsFromMap(&tc, args)
					}
				}
				payload.ToolCalls = append(payload.ToolCalls, tc)
			case "reasoning":
				var summaries []string
				if err := json.Unmarshal(entry.Payload, &item); err == nil {
					for _, s := range item.Summary {
						if s.Text != "" {
							summaries = append(summaries, s.Text)
						}
					}
				}
				if len(summaries) > 0 {
					payload.Turns = append(payload.Turns, Turn{
						Role:    "assistant",
						Content: strings.Join(summaries, "\n"),
					})
				}
			}
		}
	}

	payload.CapturedAt = time.Now().UTC()
	return payload, nil
}

// Codex JSONL types.

type codexRawEntry struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

func (e *codexRawEntry) extractCWD() string {
	if e.Type == "session_meta" || e.Type == "turn_context" {
		var ctx struct {
			CWD string `json:"cwd"`
		}
		if err := json.Unmarshal(e.Payload, &ctx); err == nil && ctx.CWD != "" {
			return ctx.CWD
		}
	}
	return ""
}

type codexSessionMeta struct {
	Git struct {
		Branch string `json:"branch"`
	} `json:"git"`
}

type codexEventMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type codexResponseItem struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Summary   []struct {
		Text string `json:"text"`
	} `json:"summary"`
}

// findJSONLFiles recursively finds all .jsonl files under dir.
func findJSONLFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return files, err
}

// codexSessionMatchesRepo checks if a Codex session file belongs to the given repo
// by scanning for CWD in session_meta or event_msg entries.
func codexSessionMatchesRepo(filePath, repoPath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		var entry codexRawEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		cwd := entry.extractCWD()
		if cwd != "" && strings.HasPrefix(cwd, repoPath) {
			return true
		}
	}
	return false
}
