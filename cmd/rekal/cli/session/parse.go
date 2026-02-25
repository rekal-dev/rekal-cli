package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SessionPayload is the parsed, filtered representation of a Claude Code session.
type SessionPayload struct {
	SessionID  string     `json:"session_id"`
	Turns      []Turn     `json:"turns"`
	ToolCalls  []ToolCall `json:"tool_calls"`
	Branch     string     `json:"branch"`
	CapturedAt time.Time  `json:"captured_at"`
	ActorType  string     `json:"actor_type"` // "human" | "agent"
	AgentID    string     `json:"agent_id"`   // empty for human
}

// Turn represents a single conversation turn (human prompt or assistant reply).
type Turn struct {
	Role      string    `json:"role"` // "human" | "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ToolCall represents a tool invocation extracted from assistant content.
type ToolCall struct {
	Tool      string `json:"tool"`       // Write, Edit, Read, Bash, etc.
	Path      string `json:"path"`       // file path if applicable
	CmdPrefix string `json:"cmd_prefix"` // first 100 chars of bash command if applicable
}

// rawLine is the top-level structure of a JSONL line from a Claude Code session.
type rawLine struct {
	UUID      string          `json:"uuid"`
	SessionID string          `json:"sessionId"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Message   json.RawMessage `json:"message"`
	CWD       string          `json:"cwd"`
	GitBranch string          `json:"gitBranch"`

	// isSidechain lines are filtered out
	IsSidechain bool `json:"isSidechain"`
}

// rawMessage is the message field within a JSONL line.
type rawMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// contentBlock represents a single block in an assistant message's content array.
// Also used for tool_result blocks in user messages.
type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`          // tool_use block ID
	ToolUseID string          `json:"tool_use_id"` // tool_result reference
	Input     json.RawMessage `json:"input"`
	Content   json.RawMessage `json:"content"` // tool_result content (string or array)
}

// toolInput holds common fields from tool_use input blocks.
type toolInput struct {
	FilePath string `json:"file_path"`
	Path     string `json:"path"`
	Command  string `json:"command"`
	Content  string `json:"content"`
}

// ParseTranscript parses raw JSONL bytes into a SessionPayload.
// It extracts conversation turns and tool calls, discarding tool results,
// thinking blocks, system content, file-history-snapshots, and sidechain messages.
func ParseTranscript(data []byte) (*SessionPayload, error) {
	payload := &SessionPayload{
		ActorType: "human",
	}

	// pendingPlanReads tracks tool_use IDs for Read calls targeting .claude/plans/ files.
	// When the corresponding tool_result arrives in a user message, we extract the plan text.
	pendingPlanReads := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Increase scanner buffer for large lines (tool results can be huge).
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw rawLine
		if err := json.Unmarshal(line, &raw); err != nil {
			// Skip malformed lines rather than failing the whole parse.
			continue
		}

		// Discard filtered line types.
		if raw.IsSidechain {
			continue
		}
		if raw.Type == "file-history-snapshot" {
			continue
		}

		// Capture session metadata from first line that has it.
		if payload.SessionID == "" && raw.SessionID != "" {
			payload.SessionID = raw.SessionID
		}
		if payload.Branch == "" && raw.GitBranch != "" {
			payload.Branch = raw.GitBranch
		}

		ts := parseTimestamp(raw.Timestamp)

		switch raw.Type {
		case "user":
			turns, err := parseUserTurn(raw.Message, ts, pendingPlanReads)
			if err != nil {
				continue
			}
			payload.Turns = append(payload.Turns, turns...)

		case "assistant":
			turns, toolCalls, planReadIDs, err := parseAssistantMessage(raw.Message, ts)
			if err != nil {
				continue
			}
			payload.Turns = append(payload.Turns, turns...)
			payload.ToolCalls = append(payload.ToolCalls, toolCalls...)
			for _, id := range planReadIDs {
				pendingPlanReads[id] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan JSONL: %w", err)
	}

	payload.CapturedAt = time.Now().UTC()
	return payload, nil
}

// parseUserTurn extracts the text content from a user message.
// It skips tool_result blocks (which contain file bodies, command outputs),
// except for tool_results matching pendingPlanReads — those contain plan file
// content that should be indexed.
func parseUserTurn(msgRaw json.RawMessage, ts time.Time, pendingPlanReads map[string]bool) ([]Turn, error) {
	if len(msgRaw) == 0 {
		return nil, nil
	}

	var msg rawMessage
	if err := json.Unmarshal(msgRaw, &msg); err != nil {
		return nil, err
	}

	if msg.Role != "user" {
		return nil, nil
	}

	var turns []Turn

	// Extract plan content from tool_result blocks matching pending plan reads.
	if len(pendingPlanReads) > 0 {
		planTurns := extractPlanToolResults(msg.Content, ts, pendingPlanReads)
		turns = append(turns, planTurns...)
	}

	text := extractTextContent(msg.Content)
	if text != "" {
		turns = append(turns, Turn{
			Role:      "human",
			Content:   text,
			Timestamp: ts,
		})
	}

	return turns, nil
}

// parseAssistantMessage extracts text turns and tool calls from an assistant message.
// It discards thinking blocks and tool results.
// It also returns IDs of Read tool_use blocks targeting .claude/plans/ files,
// so the caller can match them against subsequent tool_result blocks.
func parseAssistantMessage(msgRaw json.RawMessage, ts time.Time) ([]Turn, []ToolCall, []string, error) {
	if len(msgRaw) == 0 {
		return nil, nil, nil, nil
	}

	var msg rawMessage
	if err := json.Unmarshal(msgRaw, &msg); err != nil {
		return nil, nil, nil, err
	}

	if msg.Role != "assistant" {
		return nil, nil, nil, nil
	}

	// Content can be a string or an array of blocks.
	var turns []Turn
	var toolCalls []ToolCall
	var planReadIDs []string

	// Try as string first.
	var textContent string
	if err := json.Unmarshal(msg.Content, &textContent); err == nil {
		if textContent != "" {
			turns = append(turns, Turn{
				Role:      "assistant",
				Content:   textContent,
				Timestamp: ts,
			})
		}
		return turns, nil, nil, nil
	}

	// Parse as array of content blocks.
	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil, nil, nil, err
	}

	var textParts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				textParts = append(textParts, b.Text)
			}
		case "tool_use":
			tc := extractToolCall(b)
			toolCalls = append(toolCalls, tc)
			// Capture plan file content as an assistant turn so it's searchable.
			if planText := extractPlanContent(b); planText != "" {
				turns = append(turns, Turn{
					Role:      "assistant",
					Content:   planText,
					Timestamp: ts,
				})
			}
			// Track Read calls targeting plan files.
			if id := extractPlanReadID(b); id != "" {
				planReadIDs = append(planReadIDs, id)
			}
			// Discard: "thinking", "tool_result", etc.
		}
	}

	if len(textParts) > 0 {
		combined := ""
		for i, p := range textParts {
			if i > 0 {
				combined += "\n"
			}
			combined += p
		}
		turns = append(turns, Turn{
			Role:      "assistant",
			Content:   combined,
			Timestamp: ts,
		})
	}

	return turns, toolCalls, planReadIDs, nil
}

// extractTextContent pulls text from a message content field.
// Content can be a plain string or an array of content blocks.
// Only text blocks are extracted; tool_result blocks are discarded.
func extractTextContent(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}

	// Try string.
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return s
	}

	// Try array of blocks.
	var blocks []contentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}

	combined := ""
	for i, p := range parts {
		if i > 0 {
			combined += "\n"
		}
		combined += p
	}
	return combined
}

// extractToolCall builds a ToolCall from a tool_use content block.
func extractToolCall(b contentBlock) ToolCall {
	tc := ToolCall{
		Tool: b.Name,
	}

	if len(b.Input) == 0 {
		return tc
	}

	var inp toolInput
	if err := json.Unmarshal(b.Input, &inp); err != nil {
		return tc
	}

	// Prefer file_path, fall back to path.
	if inp.FilePath != "" {
		tc.Path = inp.FilePath
	} else if inp.Path != "" {
		tc.Path = inp.Path
	}

	// For Bash tool, capture first 100 chars of command.
	if inp.Command != "" {
		tc.CmdPrefix = truncate(inp.Command, 100)
	}

	return tc
}

// extractPlanContent returns the file content from a Write/Edit tool_use block
// if the target path is a .claude/plans/ file. This captures plan text as a
// searchable assistant turn.
func extractPlanContent(b contentBlock) string {
	if b.Name != "Write" && b.Name != "Edit" {
		return ""
	}
	if len(b.Input) == 0 {
		return ""
	}

	var inp toolInput
	if err := json.Unmarshal(b.Input, &inp); err != nil {
		return ""
	}

	path := inp.FilePath
	if path == "" {
		path = inp.Path
	}
	if !strings.Contains(path, ".claude/plans/") {
		return ""
	}

	return inp.Content
}

// extractPlanReadID returns the tool_use ID if this is a Read tool call
// targeting a .claude/plans/ file. The caller uses this to match the
// corresponding tool_result in the next user message.
func extractPlanReadID(b contentBlock) string {
	if b.Name != "Read" {
		return ""
	}
	if len(b.Input) == 0 || b.ID == "" {
		return ""
	}

	var inp toolInput
	if err := json.Unmarshal(b.Input, &inp); err != nil {
		return ""
	}

	path := inp.FilePath
	if path == "" {
		path = inp.Path
	}
	if !strings.Contains(path, ".claude/plans/") {
		return ""
	}

	return b.ID
}

// extractPlanToolResults scans user message content blocks for tool_result
// blocks whose tool_use_id matches a pending plan read. For each match, it
// extracts the text and emits it as an assistant turn (the content originated
// from the assistant's Read call). Matched IDs are removed from the map.
func extractPlanToolResults(content json.RawMessage, ts time.Time, pending map[string]bool) []Turn {
	if len(content) == 0 {
		return nil
	}

	var blocks []contentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return nil
	}

	var turns []Turn
	for _, b := range blocks {
		if b.Type != "tool_result" {
			continue
		}
		if !pending[b.ToolUseID] {
			continue
		}

		text := extractToolResultText(b.Content)
		if text != "" {
			turns = append(turns, Turn{
				Role:      "assistant",
				Content:   text,
				Timestamp: ts,
			})
		}
		delete(pending, b.ToolUseID)
	}
	return turns
}

// extractToolResultText extracts text from a tool_result content field,
// which can be a plain string or an array of content blocks.
func extractToolResultText(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}

	// Try string.
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return s
	}

	// Try array of blocks.
	var blocks []contentBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		return ""
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}

	combined := ""
	for i, p := range parts {
		if i > 0 {
			combined += "\n"
		}
		combined += p
	}
	return combined
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Claude uses ISO 8601 format.
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Try without nanoseconds.
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}
