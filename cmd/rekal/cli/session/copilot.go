package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CopilotAdapter discovers and parses GitHub Copilot CLI sessions.
//
// Layout (observed): the CLI writes an event stream per session to
//
//	$COPILOT_HOME/session-state/<session-id>/events.jsonl
//
// where $COPILOT_HOME defaults to ~/.copilot. Each line is one JSON event with
// a common envelope — {"type": "<dotted>", "timestamp": "<ISO8601>", "data":
// {...}} — whose data payload shape depends on the type. Rekal reads the turn
// and tool-call events; the working directory (for repo matching) lives on the
// session.start event at data.context.cwd. The event schema is not yet an
// official contract (github/copilot-cli#3551), so message text is extracted
// defensively across the plausible shapes.
type CopilotAdapter struct{}

func (a *CopilotAdapter) Name() string { return "copilot" }

// copilotHome returns the Copilot CLI config/state root: $COPILOT_HOME, else
// ~/.copilot. Empty when the home dir can't be resolved.
func copilotHome() string {
	if h := strings.TrimSpace(os.Getenv("COPILOT_HOME")); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".copilot")
}

func (a *CopilotAdapter) Discover(repoPath string) ([]SessionRef, error) {
	base := copilotHome()
	if base == "" {
		return nil, nil
	}
	stateDir := filepath.Join(base, "session-state")
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil, nil // no Copilot sessions on this machine
	}

	var refs []SessionRef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		events := filepath.Join(stateDir, e.Name(), "events.jsonl")
		if _, err := os.Stat(events); err != nil {
			continue
		}
		if copilotSessionMatchesRepo(events, repoPath) {
			refs = append(refs, SessionRef{Path: events})
		}
	}
	return refs, nil
}

func (a *CopilotAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	payload := &SessionPayload{
		Source:    "copilot",
		ActorType: "human",
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		var ev copilotEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		ts := parseTimestamp(ev.Timestamp)

		switch ev.Type {
		case "session.start":
			var d copilotStartData
			if err := json.Unmarshal(ev.Data, &d); err == nil {
				if payload.SessionID == "" {
					payload.SessionID = d.SessionID
				}
				if payload.CWD == "" {
					payload.CWD = d.Context.CWD
				}
				if payload.Branch == "" {
					payload.Branch = d.Repository.Branch
				}
			}

		case "user.message":
			if txt := copilotMessageText(ev.Data); txt != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "human", Content: txt, Timestamp: ts,
				})
			}

		case "assistant.message":
			if txt := copilotMessageText(ev.Data); txt != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "assistant", Content: txt, Timestamp: ts,
				})
			}

		case "tool.execution_start":
			var d copilotToolData
			if err := json.Unmarshal(ev.Data, &d); err == nil && d.ToolName != "" {
				tc := ToolCall{Tool: d.ToolName}
				copilotFillToolArgs(&tc, d.Arguments)
				payload.ToolCalls = append(payload.ToolCalls, tc)
			}
		}
	}

	if payload.CapturedAt.IsZero() {
		payload.CapturedAt = time.Now().UTC()
	}
	return payload, nil
}

// Copilot events.jsonl types.

type copilotEvent struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

type copilotStartData struct {
	SessionID string `json:"sessionId"`
	Context   struct {
		CWD string `json:"cwd"`
	} `json:"context"`
	Repository struct {
		Branch string `json:"branch"`
	} `json:"repository"`
}

type copilotToolData struct {
	ToolName  string          `json:"toolName"`
	Arguments json.RawMessage `json:"arguments"`
}

// copilotMessageText pulls the human-readable text out of a user.message /
// assistant.message data payload. The schema is unofficial and the content has
// been observed under a few shapes, so this tries each candidate field
// (content, message, text) and each candidate value form (a plain string, an
// object carrying a "text" field, or an array of {text} parts) and returns the
// first non-empty result.
func copilotMessageText(data json.RawMessage) string {
	if len(data) == 0 {
		return ""
	}
	var env struct {
		Content json.RawMessage `json:"content"`
		Message json.RawMessage `json:"message"`
		Text    json.RawMessage `json:"text"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return ""
	}
	for _, cand := range []json.RawMessage{env.Content, env.Message, env.Text} {
		if txt := copilotRawText(cand); txt != "" {
			return txt
		}
	}
	return ""
}

// copilotRawText decodes a text value that may be a string, an object with a
// "text" field, or an array of parts each with a "text" field.
func copilotRawText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var obj struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Text != "" {
		return strings.TrimSpace(obj.Text)
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var texts []string
		for _, p := range parts {
			if p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
		return strings.TrimSpace(strings.Join(texts, "\n"))
	}
	return ""
}

// copilotFillToolArgs fills tc.Path / tc.CmdPrefix from a tool event's
// arguments, which may be a JSON object or a JSON-encoded string (both shapes
// have been seen across harnesses).
func copilotFillToolArgs(tc *ToolCall, args json.RawMessage) {
	if len(args) == 0 {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(args, &m); err == nil {
		toolCallArgsFromMap(tc, m)
		return
	}
	// arguments may be a JSON string containing the object.
	var s string
	if err := json.Unmarshal(args, &s); err == nil && s != "" {
		if err := json.Unmarshal([]byte(s), &m); err == nil {
			toolCallArgsFromMap(tc, m)
		}
	}
}

// copilotSessionMatchesRepo reports whether a Copilot events.jsonl belongs to
// repoPath, by finding the session.start event's data.context.cwd and
// prefix-matching it (same approach as the Codex adapter). Scans only until the
// cwd is found — normally the first line.
func copilotSessionMatchesRepo(eventsPath, repoPath string) bool {
	f, err := os.Open(eventsPath)
	if err != nil {
		return false
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		var ev copilotEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type != "session.start" {
			continue
		}
		var d copilotStartData
		if err := json.Unmarshal(ev.Data, &d); err == nil && d.Context.CWD != "" {
			return strings.HasPrefix(d.Context.CWD, repoPath)
		}
	}
	return false
}
