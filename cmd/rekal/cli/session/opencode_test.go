package session

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOpenCodeAdapter_Parse(t *testing.T) {
	t.Parallel()

	// Create a temp SQLite DB with OpenCode schema.
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "opencode.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// Create tables.
	_, err = db.Exec(`
		CREATE TABLE session (id TEXT PRIMARY KEY, directory TEXT);
		CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, data TEXT, time_created TEXT);
		CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT, data TEXT, time_created TEXT);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	// Insert test data.
	_, err = db.Exec(`INSERT INTO session (id, directory) VALUES ('oc-001', '/tmp/repo')`)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	userMsg, _ := json.Marshal(openCodeMessage{Role: "user"})
	assistantMsg, _ := json.Marshal(openCodeMessage{Role: "assistant"})

	_, err = db.Exec(`INSERT INTO message (id, session_id, data, time_created) VALUES
		('m1', 'oc-001', ?, '2025-06-01T10:00:00Z'),
		('m2', 'oc-001', ?, '2025-06-01T10:00:05Z')`,
		string(userMsg), string(assistantMsg))
	if err != nil {
		t.Fatalf("insert messages: %v", err)
	}

	userPart, _ := json.Marshal(openCodePart{Type: "text", Text: "Add a login page"})
	assistantPart, _ := json.Marshal(openCodePart{Type: "text", Text: "I'll create a login page."})
	toolPart, _ := json.Marshal(openCodePart{
		Type:  "tool_use",
		Name:  "write_file",
		Input: json.RawMessage(`{"file_path":"src/login.tsx"}`),
	})

	_, err = db.Exec(`INSERT INTO part (id, message_id, data, time_created) VALUES
		('p1', 'm1', ?, '2025-06-01T10:00:00Z'),
		('p2', 'm2', ?, '2025-06-01T10:00:05Z'),
		('p3', 'm2', ?, '2025-06-01T10:00:06Z')`,
		string(userPart), string(assistantPart), string(toolPart))
	if err != nil {
		t.Fatalf("insert parts: %v", err)
	}
	db.Close()

	// Parse using a custom adapter that points to our test DB.
	payload, err := parseOpenCodeDB(dbPath, "oc-001")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if payload.Source != "opencode" {
		t.Errorf("Source = %q, want opencode", payload.Source)
	}
	if payload.SessionID != "oc-001" {
		t.Errorf("SessionID = %q, want oc-001", payload.SessionID)
	}
	if payload.CWD != "/tmp/repo" {
		t.Errorf("CWD = %q, want /tmp/repo", payload.CWD)
	}

	if len(payload.Turns) != 2 {
		t.Fatalf("len(Turns) = %d, want 2", len(payload.Turns))
	}
	if payload.Turns[0].Role != "human" || payload.Turns[0].Content != "Add a login page" {
		t.Errorf("Turns[0] = %+v", payload.Turns[0])
	}
	if payload.Turns[1].Role != "assistant" || payload.Turns[1].Content != "I'll create a login page." {
		t.Errorf("Turns[1] = %+v", payload.Turns[1])
	}

	if len(payload.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(payload.ToolCalls))
	}
	if payload.ToolCalls[0].Tool != "write_file" || payload.ToolCalls[0].Path != "src/login.tsx" {
		t.Errorf("ToolCalls[0] = %+v", payload.ToolCalls[0])
	}
}
