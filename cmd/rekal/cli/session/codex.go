package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CodexAdapter discovers and parses OpenAI Codex CLI sessions from JSONL files.
type CodexAdapter struct{}

func (a *CodexAdapter) Name() string { return "codex" }

func (a *CodexAdapter) Discover(repoPath string) ([]SessionRef, error) {
	var refs []SessionRef
	for _, dir := range codexSessionDirs() {
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

// codexSessionDirs is shared by repo-scoped capture and cross-repo import.
func codexSessionDirs() []string {
	root := os.Getenv("CODEX_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		root = filepath.Join(home, ".codex")
	}
	return []string{filepath.Join(root, "sessions"), filepath.Join(root, "archived_sessions")}
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
	var eventTurns, responseTurns []Turn
	hasEventMessages := false

	for scanner.Scan() {
		var entry codexRawEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		switch entry.Type {
		case "session_meta":
			var meta codexSessionMeta
			_ = json.Unmarshal(entry.Payload, &meta)
			if payload.SessionID == "" {
				payload.SessionID = entry.SessionID
				if payload.SessionID == "" {
					payload.SessionID = meta.ID
				}
			}
			if payload.CWD == "" {
				payload.CWD = entry.extractCWD()
			}
			if payload.Branch == "" {
				if meta.Git.Branch != "" {
					payload.Branch = meta.Git.Branch
				}
			}
		case "turn_context":
			if payload.CWD == "" {
				payload.CWD = entry.extractCWD()
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
					hasEventMessages = true
					eventTurns = append(eventTurns, Turn{
						Role: "human", Content: msg.Message, Timestamp: ts,
					})
				}
			case "agent_message":
				if msg.Message != "" {
					hasEventMessages = true
					eventTurns = append(eventTurns, Turn{
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
			case "message":
				role := ""
				switch item.Role {
				case "user":
					role = "human"
				case "assistant":
					role = "assistant"
				}
				if text := codexMessageText(item.Content); role != "" && text != "" {
					responseTurns = append(responseTurns, Turn{
						Role: role, Content: text, Timestamp: parseTimestamp(entry.Timestamp),
					})
				}
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
					turn := Turn{Role: "assistant", Content: strings.Join(summaries, "\n"), Timestamp: parseTimestamp(entry.Timestamp)}
					eventTurns = append(eventTurns, turn)
					responseTurns = append(responseTurns, turn)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan codex session: %w", err)
	}
	// Rollouts can mirror messages in both streams. Prefer the event stream;
	// response-only transcripts use message items without duplicating turns.
	payload.Turns = responseTurns
	if hasEventMessages {
		payload.Turns = eventTurns
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
	ID  string `json:"id"`
	Git struct {
		Branch string `json:"branch"`
	} `json:"git"`
}

type codexEventMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type codexResponseItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	Name      string          `json:"name"`
	Arguments string          `json:"arguments"`
	Summary   []struct {
		Text string `json:"text"`
	} `json:"summary"`
}

func codexMessageText(content json.RawMessage) string {
	var text string
	if json.Unmarshal(content, &text) == nil {
		return text
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, block := range blocks {
		if (block.Type == "input_text" || block.Type == "output_text" || block.Type == "text") && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
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
// by scanning for CWD in session_meta or turn_context entries.
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
		if cwd != "" {
			rel, err := filepath.Rel(repoPath, cwd)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
}
