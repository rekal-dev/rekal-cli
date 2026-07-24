package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// KiroAdapter discovers and parses Kiro (kiro.dev) CLI chat sessions.
//
// Layout (community-documented; the session schema is not yet an official
// contract — github kirodotdev/Kiro#5094):
//
//	$KIRO_HOME/sessions/cli/<session-id>.json    metadata {session_id, cwd, title}
//	$KIRO_HOME/sessions/cli/<session-id>.jsonl   event log — one JSON object per line
//
// where $KIRO_HOME defaults to ~/.kiro. Discovery is exact: the sibling .json's
// `cwd` records the repo the session was run in, so a session is matched to a
// repo the same way the OpenCode/Copilot adapters match theirs. The .jsonl
// message schema is undocumented, so turn/tool extraction is defensive across
// the plausible shapes (a role with either a plain-string content or
// Anthropic-style content parts) and fails soft — a line it can't read is
// skipped, never fatal. IDE sessions (stored elsewhere under sessions/) are not
// yet covered: they carry no cwd metadata to match a repo cleanly.
type KiroAdapter struct{}

func (a *KiroAdapter) Name() string { return "kiro" }

// kiroHome returns the Kiro state root: $KIRO_HOME, else ~/.kiro. Empty when the
// home dir can't be resolved.
func kiroHome() string {
	if h := strings.TrimSpace(os.Getenv("KIRO_HOME")); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kiro")
}

func (a *KiroAdapter) Discover(repoPath string) ([]SessionRef, error) {
	base := kiroHome()
	if base == "" {
		return nil, nil
	}
	cliDir := filepath.Join(base, "sessions", "cli")
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return nil, nil // no Kiro CLI sessions on this machine
	}

	var refs []SessionRef
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		metaPath := filepath.Join(cliDir, e.Name())
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta kiroSessionMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		if meta.CWD == "" || !kiroRepoMatch(meta.CWD, repoPath) {
			continue
		}
		jsonl := strings.TrimSuffix(metaPath, ".json") + ".jsonl"
		if _, err := os.Stat(jsonl); err != nil {
			continue
		}
		refs = append(refs, SessionRef{Path: jsonl})
	}
	return refs, nil
}

// kiroRepoMatch reports whether a session's cwd belongs to repoPath — the exact
// repo or a path inside it, never a sibling that merely shares the prefix
// (repo vs repo-2).
func kiroRepoMatch(cwd, repoPath string) bool {
	return cwd == repoPath || strings.HasPrefix(cwd, repoPath+string(os.PathSeparator))
}

func (a *KiroAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	payload := &SessionPayload{
		SessionID:  strings.TrimSuffix(filepath.Base(ref.Path), ".jsonl"),
		Source:     "kiro",
		ActorType:  "human",
		CapturedAt: time.Now().UTC(),
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		var ev kiroEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue // control / non-JSON line
		}
		role := kiroRole(ev.Role, ev.Type)
		if role == "" {
			continue
		}
		text, tools := kiroExtractContent(ev.Content)
		if text == "" && ev.Text != "" {
			text = strings.TrimSpace(ev.Text)
		}
		ts := parseTimestamp(ev.Timestamp)
		switch role {
		case "human":
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
	return payload, nil
}

// kiroSessionMeta is the <session-id>.json metadata sidecar.
type kiroSessionMeta struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	Title     string `json:"title"`
}

// kiroEvent is one line of the .jsonl event log. The schema is unofficial, so
// the struct is permissive: a role (or a type that names the speaker), plus
// content as either a plain string / object / array-of-parts (via Content) or a
// flat text field (via Text).
type kiroEvent struct {
	Role      string          `json:"role"`
	Type      string          `json:"type"`
	Content   json.RawMessage `json:"content"`
	Text      string          `json:"text"`
	Timestamp string          `json:"timestamp"`
}

// kiroRole maps a line's role/type onto the canonical human/assistant turn
// roles, or "" for lines that carry no conversational turn (tool results,
// status events, system prompts).
func kiroRole(role, typ string) string {
	for _, v := range []string{role, typ} {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "user", "human":
			return "human"
		case "assistant", "ai", "model":
			return "assistant"
		}
	}
	return ""
}

// kiroExtractContent decodes a message's content, which may be a plain string,
// an object carrying a "text" field, or an Anthropic-style array of parts
// (text + tool_use). Returns the joined text and any tool calls; empty on an
// unrecognized shape (fail soft).
func kiroExtractContent(raw json.RawMessage) (string, []ToolCall) {
	if len(raw) == 0 {
		return "", nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString), nil
	}
	var obj struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Text != "" {
		return strings.TrimSpace(obj.Text), nil
	}
	var parts []struct {
		Type  string                 `json:"type"`
		Text  string                 `json:"text"`
		Name  string                 `json:"name"`
		Input map[string]interface{} `json:"input"`
	}
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
			if p.Name != "" {
				tc := ToolCall{Tool: p.Name}
				toolCallArgsFromMap(&tc, p.Input)
				tools = append(tools, tc)
			}
		}
	}
	return strings.TrimSpace(strings.Join(texts, "\n")), tools
}
