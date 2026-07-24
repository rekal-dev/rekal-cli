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

	// seenTools dedupes tool calls by toolCallId across the two sources that
	// carry them — assistant.message.toolRequests (the intent) and
	// tool.execution_start (the execution). Whichever event lands first records
	// the tool; the other skips it.
	seenTools := make(map[string]bool)

	for scanner.Scan() {
		var ev copilotEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		ts := parseTimestamp(ev.Timestamp)

		// captured_at is the session's real start, the earliest event time —
		// not the ingestion wall clock (that is only the final fallback below).
		if !ts.IsZero() && (payload.CapturedAt.IsZero() || ts.Before(payload.CapturedAt)) {
			payload.CapturedAt = ts
		}

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
				// The branch lives at data.context.branch on real payloads;
				// data.repository.branch is the legacy/observed fallback.
				if payload.Branch == "" {
					if d.Context.Branch != "" {
						payload.Branch = d.Context.Branch
					} else {
						payload.Branch = d.Repository.Branch
					}
				}
			}

		case "system.message":
			// Copilot's system prompt / task instructions — the steering
			// equivalent. Ingest as a human_steering turn so it is queryable.
			if txt := copilotMessageText(ev.Data); txt != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "human_steering", Content: txt, Timestamp: ts,
				})
			}

		case "user.message":
			if txt := copilotMessageText(ev.Data); txt != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "human", Content: txt, Timestamp: ts,
				})
			}

		case "assistant.message":
			var d copilotAssistantData
			_ = json.Unmarshal(ev.Data, &d)
			// A tool-only assistant turn has empty content; synthesize a
			// readable stand-in from its tool requests so the turn is not
			// dropped (the majority of assistant turns are tool-only).
			txt := copilotRawText(d.Content)
			if txt == "" {
				txt = copilotToolRequestSummary(d.ToolRequests)
			}
			if txt != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "assistant", Content: txt, Timestamp: ts,
				})
			}
			// Tool intents also come from toolRequests, not just
			// tool.execution_start — record them, deduped by toolCallId.
			for _, tr := range d.ToolRequests {
				if tr.Name == "" {
					continue
				}
				if tr.ToolCallID != "" {
					if seenTools[tr.ToolCallID] {
						continue
					}
					seenTools[tr.ToolCallID] = true
				}
				tc := ToolCall{Tool: tr.Name}
				copilotFillToolArgs(&tc, tr.Arguments)
				payload.ToolCalls = append(payload.ToolCalls, tc)
			}

		case "tool.execution_start":
			var d copilotToolData
			if err := json.Unmarshal(ev.Data, &d); err == nil && d.ToolName != "" {
				if d.ToolCallID != "" {
					if seenTools[d.ToolCallID] {
						continue
					}
					seenTools[d.ToolCallID] = true
				}
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
		CWD    string `json:"cwd"`
		Branch string `json:"branch"`
	} `json:"context"`
	Repository struct {
		Branch string `json:"branch"`
	} `json:"repository"`
}

// copilotAssistantData is the assistant.message payload: free-text content
// (often "") plus the tool requests the turn issued.
type copilotAssistantData struct {
	Content      json.RawMessage      `json:"content"` // string (may be "")
	ToolRequests []copilotToolRequest `json:"toolRequests"`
}

// copilotToolRequest is one tool intent carried on an assistant.message. The
// toolCallId ties it to the matching tool.execution_start for dedupe.
type copilotToolRequest struct {
	ToolCallID string          `json:"toolCallId"`
	Name       string          `json:"name"`
	Arguments  json.RawMessage `json:"arguments"`
}

type copilotToolData struct {
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Arguments  json.RawMessage `json:"arguments"`
}

// copilotToolRequestSummary renders a tool-only assistant turn's intent as
// readable text (e.g. `[tool: shell] [tool: write_file]`) so the turn carries
// content and is not dropped. Returns "" when no request names a tool.
func copilotToolRequestSummary(reqs []copilotToolRequest) string {
	var parts []string
	for _, tr := range reqs {
		if tr.Name != "" {
			parts = append(parts, "[tool: "+tr.Name+"]")
		}
	}
	return strings.Join(parts, " ")
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
			// Exact repo or a path *inside* it — never a sibling that merely
			// shares the prefix (repo vs repo-2).
			return d.Context.CWD == repoPath ||
				strings.HasPrefix(d.Context.CWD, repoPath+string(os.PathSeparator))
		}
	}
	return false
}
