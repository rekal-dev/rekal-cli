package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GeminiAdapter discovers and parses Gemini CLI sessions from JSON files.
type GeminiAdapter struct{}

func (a *GeminiAdapter) Name() string { return "gemini" }

func (a *GeminiAdapter) Discover(repoPath string) ([]SessionRef, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}

	tmpDir := filepath.Join(home, ".gemini", "tmp")
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, nil
	}

	// Build a hash→path map for first-level dirs under $HOME to reverse the project hash.
	targetHash := geminiProjectHash(repoPath)

	var refs []SessionRef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if e.Name() != targetHash {
			continue
		}
		chatsDir := filepath.Join(tmpDir, e.Name(), "chats")
		chatEntries, err := os.ReadDir(chatsDir)
		if err != nil {
			continue
		}
		for _, ce := range chatEntries {
			if ce.IsDir() || !strings.HasPrefix(ce.Name(), "session-") || !strings.HasSuffix(ce.Name(), ".json") {
				continue
			}
			refs = append(refs, SessionRef{Path: filepath.Join(chatsDir, ce.Name())})
		}
	}
	return refs, nil
}

func (a *GeminiAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	var raw geminiSession
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	payload := &SessionPayload{
		SessionID: raw.SessionID,
		Source:    "gemini",
		ActorType: "human",
	}

	if raw.StartTime != "" {
		payload.CapturedAt = parseTimestamp(raw.StartTime)
	}
	if payload.CapturedAt.IsZero() {
		payload.CapturedAt = time.Now().UTC()
	}

	for _, msg := range raw.Messages {
		switch msg.Type {
		case "user":
			text := geminiExtractText(msg.Content)
			if text != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "human", Content: text,
				})
			}
		case "gemini":
			text := geminiExtractText(msg.Content)
			if text != "" {
				payload.Turns = append(payload.Turns, Turn{
					Role: "assistant", Content: text,
				})
			}
			for _, tc := range msg.ToolCalls {
				toolCall := ToolCall{Tool: tc.Name}
				toolCallArgsFromMap(&toolCall, tc.Args)
				payload.ToolCalls = append(payload.ToolCalls, toolCall)
			}
		}
	}

	return payload, nil
}

// geminiProjectHash computes the SHA-256 hex hash of a project path,
// matching Gemini CLI's directory hashing convention.
func geminiProjectHash(path string) string {
	h := sha256.Sum256([]byte(path))
	return hex.EncodeToString(h[:])
}

// geminiExtractText extracts text from Gemini message content.
// Content can be a string or structured.
func geminiExtractText(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}
	// Try as string.
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return s
	}
	// Try as array of parts.
	var parts []geminiContentPart
	if err := json.Unmarshal(content, &parts); err == nil {
		var texts []string
		for _, p := range parts {
			if p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
		return strings.Join(texts, "\n")
	}
	return ""
}

// Gemini JSON types.

type geminiSession struct {
	SessionID string          `json:"sessionId"`
	StartTime string          `json:"startTime"`
	Messages  []geminiMessage `json:"messages"`
}

type geminiMessage struct {
	Type      string           `json:"type"`
	Content   json.RawMessage  `json:"content"`
	ToolCalls []geminiToolCall `json:"toolCalls"`
}

type geminiToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type geminiContentPart struct {
	Text string `json:"text"`
}
