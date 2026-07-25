package session

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// OpenCodeAdapter discovers and parses OpenCode sessions from a SQLite database.
type OpenCodeAdapter struct{}

func (a *OpenCodeAdapter) Name() string { return "opencode" }

func (a *OpenCodeAdapter) Discover(repoPath string) ([]SessionRef, error) {
	dbPath := openCodeDBPath()
	if dbPath == "" {
		return nil, nil
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, nil
	}

	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, nil
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, directory FROM session")
	if err != nil {
		return nil, nil
	}
	defer rows.Close() //nolint:errcheck

	var refs []SessionRef
	for rows.Next() {
		var id, dir string
		if err := rows.Scan(&id, &dir); err != nil {
			continue
		}
		if strings.HasPrefix(dir, repoPath) {
			refs = append(refs, SessionRef{DBID: id})
		}
	}
	return refs, nil
}

func (a *OpenCodeAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
	dbPath := openCodeDBPath()
	if dbPath == "" {
		return nil, nil
	}
	return parseOpenCodeDB(dbPath, ref.DBID)
}

// parseOpenCodeDB reads a session from an OpenCode SQLite database.
// Extracted for testability.
func parseOpenCodeDB(dbPath, sessionID string) (*SessionPayload, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	payload := &SessionPayload{
		SessionID:  sessionID,
		Source:     "opencode",
		ActorType:  "human",
		CapturedAt: time.Now().UTC(),
	}
	if err := db.QueryRow("SELECT directory FROM session WHERE id = ?", sessionID).Scan(&payload.CWD); err != nil {
		payload.CWD = ""
	}

	// Query messages ordered by creation time.
	rows, err := db.Query(
		"SELECT id, data, time_created FROM message WHERE session_id = ? ORDER BY time_created ASC",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var msgID, dataStr, timeCreated string
		if err := rows.Scan(&msgID, &dataStr, &timeCreated); err != nil {
			continue
		}

		var msg openCodeMessage
		if err := json.Unmarshal([]byte(dataStr), &msg); err != nil {
			continue
		}

		ts := parseTimestamp(timeCreated)

		// Query parts for this message.
		partRows, err := db.Query(
			"SELECT data FROM part WHERE message_id = ? ORDER BY time_created ASC",
			msgID,
		)
		if err != nil {
			continue
		}

		var textParts []string
		for partRows.Next() {
			var partData string
			if err := partRows.Scan(&partData); err != nil {
				continue
			}
			var part openCodePart
			if err := json.Unmarshal([]byte(partData), &part); err != nil {
				continue
			}

			switch part.Type {
			case "text":
				if part.Text != "" {
					textParts = append(textParts, part.Text)
				}
			case "tool_use", "tool-use":
				tc := ToolCall{Tool: part.Name}
				if len(part.Input) > 0 {
					var inp map[string]interface{}
					if err := json.Unmarshal(part.Input, &inp); err == nil {
						toolCallArgsFromMap(&tc, inp)
					}
				}
				payload.ToolCalls = append(payload.ToolCalls, tc)
			}
		}
		partRows.Close() //nolint:errcheck

		if len(textParts) > 0 {
			role := "assistant"
			if msg.Role == "user" {
				role = "human"
			}
			payload.Turns = append(payload.Turns, Turn{
				Role:      role,
				Content:   strings.Join(textParts, "\n"),
				Timestamp: ts,
			})
		}
	}

	return payload, nil
}

func openCodeDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

// OpenCode SQLite types.

type openCodeMessage struct {
	Role string `json:"role"`
}

type openCodePart struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}
