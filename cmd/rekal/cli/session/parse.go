package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// SessionPayload is the parsed, filtered representation of an AI agent session.
type SessionPayload struct {
	SessionID  string     `json:"session_id"`
	Source     string     `json:"source"` // "claude", "codex", "gemini", "opencode"
	Turns      []Turn     `json:"turns"`
	ToolCalls  []ToolCall `json:"tool_calls"`
	Branch     string     `json:"branch"`
	CapturedAt time.Time  `json:"captured_at"`

	// CWD is the working directory the session was recorded in (from the
	// transcript's cwd field). Used to label the origin of cross-repo local
	// imports; empty when the transcript carries no cwd.
	CWD       string `json:"cwd,omitempty"`
	ActorType string `json:"actor_type"` // "human" | "agent"
	AgentID   string `json:"agent_id"`   // empty for human

	// TeamName is the active team name when the session was part of a
	// Claude Code teammates run (from the entries' teamName field or the
	// subagent meta sidecar).
	TeamName string `json:"team_name,omitempty"`

	// WorkflowName is the dynamic-workflow name for transcripts found under
	// subagents/workflows/<name>/; empty otherwise.
	WorkflowName string `json:"workflow_name,omitempty"`

	// AgentType, Description, and SpawnDepth come from a subagent's
	// meta.json sidecar (agentType/description/spawnDepth — the real
	// observed shape, see claude.go's subagentMeta doc). Empty/zero for
	// non-subagent sessions or subagents captured without a sidecar.
	AgentType   string `json:"agent_type,omitempty"`
	Description string `json:"description,omitempty"`
	SpawnDepth  int    `json:"spawn_depth,omitempty"`

	// ParentSessionPath is the local path of the trunk session file when this
	// payload was parsed from a subagent transcript. Checkpoint uses it to
	// link the subagent session to its parent row; never serialized.
	ParentSessionPath string `json:"-"`
}

// Turn represents a single conversation turn (human prompt or assistant reply).
type Turn struct {
	Role      string    `json:"role"` // "human" | "human_steering" | "assistant"
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

	// isMeta marks harness-injected user turns (e.g. skill bodies, command
	// wrappers) rather than real human input. Filtered out.
	IsMeta bool `json:"isMeta"`

	// TeamName is the active team name on entries written during a
	// teammates run; AgentID is transcript membership (which agent's
	// transcript this entry belongs to). Absent in older session files.
	TeamName string `json:"teamName"`
	AgentID  string `json:"agentId"`

	// Operation and Content are only populated on "queue-operation" lines.
	// Verified directly against real ~/.claude/projects/*.jsonl session
	// files: "enqueue" carries an out-of-band user steering message typed
	// while the agent was working (Content is the message text); "dequeue"
	// fires when it's delivered and carries no content. Content is
	// RawMessage because it can be a plain string or a content-block array.
	//
	// A queued message can also be cancelled/removed by the user before
	// delivery (Claude Code's queue UI allows this). That variant's exact
	// operation name and payload shape have not been observed in a captured
	// transcript, so it is handled defensively in ParseTranscriptWithOptions:
	// any operation other than "enqueue"/"dequeue" is treated as a
	// removal and, if Content matches a pending enqueued message (same
	// string-or-block-array shape as "enqueue"), the phantom human_steering
	// turn already emitted for it is retracted. Without this, a cancelled
	// draft would otherwise sit in the append-only ledger forever as if the
	// user had actually typed it — worse, weighted up in recall ranking.
	Operation string          `json:"operation"`
	Content   json.RawMessage `json:"content"`
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

// TranscriptOptions controls transcript parsing behavior.
type TranscriptOptions struct {
	// IncludeSidechain keeps isSidechain entries. Trunk transcripts drop
	// them (legacy inline subagent duplicates); subagent transcript files
	// consist of them and must keep them.
	IncludeSidechain bool
}

// ParseTranscript parses raw JSONL bytes into a SessionPayload.
// It extracts conversation turns and tool calls, discarding tool results,
// thinking blocks, system content, file-history-snapshots, sidechain messages,
// and harness-injected (isMeta) user turns. Out-of-band steering messages
// (queue-operation/enqueue) are extracted as human turns; the later ordinary
// "user" entry carrying the same delivered text is deduplicated against it.
// A queue-operation that cancels/removes a message before delivery retracts
// its steering turn instead of leaving a phantom entry in the ledger.
func ParseTranscript(data []byte) (*SessionPayload, error) {
	return ParseTranscriptWithOptions(data, TranscriptOptions{})
}

// ParseTranscriptWithOptions is ParseTranscript with explicit options.
func ParseTranscriptWithOptions(data []byte, opts TranscriptOptions) (*SessionPayload, error) {
	payload := &SessionPayload{
		ActorType: "human",
	}

	// pendingPlanReads tracks tool_use IDs for Read calls targeting .claude/plans/ files.
	// When the corresponding tool_result arrives in a user message, we extract the plan text.
	pendingPlanReads := make(map[string]bool)

	// pendingQueueTexts tracks steering message text already emitted from a
	// queue-operation/enqueue line. Claude Code also writes the same text as
	// an ordinary "user" entry once the message is delivered (verified
	// against real session files); when that entry arrives we skip
	// re-emitting it as a duplicate turn.
	pendingQueueTexts := make(map[string]bool)

	// pendingQueueTurnIdx maps that same pending steering text to its index
	// in payload.Turns, so a later cancel/remove queue-operation (see
	// rawLine.Operation doc) can retract the turn instead of leaving a
	// phantom human_steering entry for a message the user never actually
	// sent. If the same text is queued more than once before either is
	// resolved, only the most recent occurrence is tracked — an accepted
	// edge case given real removal payloads have not been observed yet.
	pendingQueueTurnIdx := make(map[string]int)

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
		if raw.IsSidechain && !opts.IncludeSidechain {
			continue
		}
		if raw.Type == "file-history-snapshot" {
			continue
		}
		// isMeta user turns are harness-injected (skill bodies, command
		// wrappers), not real human input.
		if raw.Type == "user" && raw.IsMeta {
			continue
		}

		// Capture session metadata from first line that has it.
		if payload.SessionID == "" && raw.SessionID != "" {
			payload.SessionID = raw.SessionID
		}
		if payload.Branch == "" && raw.GitBranch != "" {
			payload.Branch = raw.GitBranch
		}
		if payload.CWD == "" && raw.CWD != "" {
			payload.CWD = raw.CWD
		}
		if payload.TeamName == "" && raw.TeamName != "" {
			payload.TeamName = raw.TeamName
		}
		if payload.AgentID == "" && raw.AgentID != "" {
			payload.AgentID = raw.AgentID
		}

		ts := parseTimestamp(raw.Timestamp)

		switch raw.Type {
		case "user":
			turns, err := parseUserTurn(raw.Message, ts, pendingPlanReads)
			if err != nil {
				continue
			}
			for _, t := range turns {
				if t.Role == "human" && pendingQueueTexts[t.Content] {
					delete(pendingQueueTexts, t.Content)
					delete(pendingQueueTurnIdx, t.Content)
					continue
				}
				payload.Turns = append(payload.Turns, t)
			}

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

		case "queue-operation":
			switch raw.Operation {
			case "enqueue":
				// Content can be a plain string or a content-block array.
				// Tagged "human_steering" (distinct from "human") — it is
				// the highest-intent text in the corpus and recall ranking
				// boosts it.
				if text := extractTextContent(raw.Content); text != "" {
					pendingQueueTexts[text] = true
					payload.Turns = append(payload.Turns, Turn{
						Role:      "human_steering",
						Content:   text,
						Timestamp: ts,
					})
					pendingQueueTurnIdx[text] = len(payload.Turns) - 1
				}
			case "dequeue":
				// Fires once the queued message is delivered; carries no
				// content of its own. The subsequent ordinary "user" entry
				// with the same text is deduplicated above.
			default:
				// A queued message can be cancelled/removed before delivery
				// — the operation name and payload shape for that case have
				// not been observed in a real transcript (see rawLine.
				// Operation doc), so any operation other than the two known
				// values is treated defensively as a removal: try to
				// extract text the same way "enqueue" does, and if it
				// matches a pending steering turn, retract that turn
				// instead of leaving a phantom human_steering entry for a
				// message the user took back.
				if text := extractTextContent(raw.Content); text != "" {
					if idx, ok := pendingQueueTurnIdx[text]; ok {
						payload.Turns = append(payload.Turns[:idx], payload.Turns[idx+1:]...)
						delete(pendingQueueTurnIdx, text)
						delete(pendingQueueTexts, text)
						for k, v := range pendingQueueTurnIdx {
							if v > idx {
								pendingQueueTurnIdx[k] = v - 1
							}
						}
					}
				}
				// If Content doesn't match any pending turn (unknown shape,
				// or the real removal payload references the queued item
				// by ID rather than text), there is nothing safe to
				// retract — leaving the turn in place matches today's
				// behavior rather than risking dropping a real message.
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

// truncate returns at most maxLen bytes of s without splitting a multi-byte
// UTF-8 rune at the boundary. A split rune is invalid UTF-8, which DuckDB
// rejects on insert ("could not bind parameter") — so a naive s[:maxLen] made
// checkpoints intermittently fail on commands whose byte-maxLen boundary
// landed mid-rune. Returns s unchanged when it already fits.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	cut := maxLen
	// Back up off any UTF-8 continuation byte so the last kept rune is whole.
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
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
