package cli

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"

	_ "modernc.org/sqlite"
)

func TestOriginLabel(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	shellDir := t.TempDir()

	cases := []struct {
		name, cwd, fallback, want string
	}{
		{"git repo cwd", repoDir, "/proj", "repo:" + repoDir},
		{"non-repo cwd", shellDir, "/proj", "shell:" + shellDir},
		{"no cwd non-git fallback", "", "/proj/x", "local:/proj/x"},
		{"no cwd git fallback", "", repoDir, "repo:" + repoDir},
		{"empty", "", "", ""},
	}
	for _, tc := range cases {
		if got := originLabel(tc.cwd, tc.fallback); got != tc.want {
			t.Errorf("%s: originLabel(%q,%q) = %q, want %q", tc.name, tc.cwd, tc.fallback, got, tc.want)
		}
	}
}

// setupLocalImport stages an import: a fake CLAUDE_CONFIG_DIR whose projects
// tree has a dir for a (fake) other git repo, and an importing repo's gitRoot
// with an initialized data.db (the dedup source). Tests write transcripts into
// the project dir with writeTranscript.
func setupLocalImport(t *testing.T) (gitRoot, otherRepo string) {
	t.Helper()

	cfg := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", cfg)

	// The "other repo" the sessions came from — has a .git dir so the origin
	// label classifies it as repo:.
	otherRepo = t.TempDir()
	if err := os.MkdirAll(filepath.Join(otherRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// The importing repo: .rekal with an initialized data.db.
	gitRoot = t.TempDir()
	if err := os.MkdirAll(RekalDir(gitRoot), 0o755); err != nil {
		t.Fatal(err)
	}
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := db.InitDataSchema(dataDB); err != nil {
		dataDB.Close()
		t.Fatalf("InitDataSchema: %v", err)
	}
	dataDB.Close()

	return gitRoot, otherRepo
}

// openTestIndex opens a fresh index DB (with schema) in the repo's .rekal dir.
func openTestIndex(t *testing.T, gitRoot string) *sql.DB {
	t.Helper()
	indexDB, err := db.OpenIndexAt(filepath.Join(RekalDir(gitRoot), "index.db"))
	if err != nil {
		t.Fatalf("OpenIndexAt: %v", err)
	}
	if err := db.InitIndexSchema(indexDB); err != nil {
		indexDB.Close()
		t.Fatalf("InitIndexSchema: %v", err)
	}
	return indexDB
}

// transcriptFor renders a minimal two-turn Claude transcript recorded in cwd.
func transcriptFor(cwd, userText, assistantText string) string {
	return `{"uuid":"u1","sessionId":"s-1","timestamp":"2026-01-01T10:00:00Z","type":"user","message":{"role":"user","content":"` + userText + `"},"cwd":"` + cwd + `","gitBranch":"main"}
{"uuid":"u2","sessionId":"s-1","timestamp":"2026-01-01T10:00:05Z","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"` + assistantText + `"}]},"cwd":"` + cwd + `","gitBranch":"main"}
`
}

func TestImportLocalSessions_EndToEnd(t *testing.T) {
	gitRoot, otherRepo := setupLocalImport(t)
	transcript := transcriptFor(otherRepo, "how did we fix the auth race", "we serialized the token refresh")
	writeTranscript(t, otherRepo, "sess.jsonl", transcript)

	indexDB := openTestIndex(t, gitRoot)
	defer indexDB.Close()

	sessions, projects, err := importLocalSessions(indexDB, gitRoot, localPref{Repos: []string{otherRepo}}, io.Discard)
	if err != nil {
		t.Fatalf("importLocalSessions: %v", err)
	}
	if sessions != 1 || projects != 1 {
		t.Fatalf("imported %d sessions from %d projects, want 1 from 1", sessions, projects)
	}

	// The facet row carries the origin label; the turns are searchable.
	var origin string
	if err := indexDB.QueryRow("SELECT origin FROM session_facets").Scan(&origin); err != nil {
		t.Fatalf("query origin: %v", err)
	}
	if origin != "repo:"+otherRepo {
		t.Fatalf("origin = %q, want %q", origin, "repo:"+otherRepo)
	}
	var turns int
	if err := indexDB.QueryRow("SELECT count(*) FROM turns_ft").Scan(&turns); err != nil {
		t.Fatalf("count turns: %v", err)
	}
	if turns != 2 {
		t.Fatalf("turns_ft has %d rows, want 2", turns)
	}
}

func TestImportLocalSessions_DedupAgainstDataDB(t *testing.T) {
	gitRoot, otherRepo := setupLocalImport(t)
	transcript := transcriptFor(otherRepo, "already captured here", "yes")
	writeTranscript(t, otherRepo, "sess.jsonl", transcript)

	// Pretend this repo already captured the same content: insert a session
	// whose hash matches the transcript bytes.
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := db.InsertSession(dataDB, "existing", "", sha256Hex([]byte(transcript)), "human", "", "dev@example.com", "main", "2026-01-01T10:00:00Z", "claude"); err != nil {
		dataDB.Close()
		t.Fatalf("InsertSession: %v", err)
	}
	dataDB.Close()

	indexDB := openTestIndex(t, gitRoot)
	defer indexDB.Close()

	sessions, projects, err := importLocalSessions(indexDB, gitRoot, localPref{Repos: []string{otherRepo}}, io.Discard)
	if err != nil {
		t.Fatalf("importLocalSessions: %v", err)
	}
	if sessions != 0 || projects != 0 {
		t.Fatalf("imported %d sessions from %d projects, want 0 (deduped by content hash)", sessions, projects)
	}
}

func TestImportLocalSessions_ScrubsSecrets(t *testing.T) {
	gitRoot, otherRepo := setupLocalImport(t)
	secret := "ghp_abcdefghijklmnopqrstuvwxyz123456"
	transcript := transcriptFor(otherRepo, "use token "+secret, "done")
	writeTranscript(t, otherRepo, "sess.jsonl", transcript)

	indexDB := openTestIndex(t, gitRoot)
	defer indexDB.Close()

	if _, _, err := importLocalSessions(indexDB, gitRoot, localPref{Repos: []string{otherRepo}}, io.Discard); err != nil {
		t.Fatalf("importLocalSessions: %v", err)
	}

	var content string
	if err := indexDB.QueryRow("SELECT content FROM turns_ft WHERE turn_index = 0").Scan(&content); err != nil {
		t.Fatalf("query content: %v", err)
	}
	if strings.Contains(content, secret) {
		t.Fatal("secret survived import — scrub must run before any index insert")
	}
	if !strings.Contains(content, "[REDACTED]") {
		t.Fatalf("content = %q, want the secret replaced with [REDACTED]", content)
	}
}

func TestImportLocalSessions_DisabledPrefIsNoop(t *testing.T) {
	gitRoot, _ := setupLocalImport(t)

	indexDB := openTestIndex(t, gitRoot)
	defer indexDB.Close()

	sessions, projects, err := importLocalSessions(indexDB, gitRoot, localPref{}, io.Discard)
	if err != nil {
		t.Fatalf("importLocalSessions: %v", err)
	}
	if sessions != 0 || projects != 0 {
		t.Fatalf("disabled pref imported %d/%d, want 0/0", sessions, projects)
	}
}

// TestImportLocalSessions_AllAgentsInclude imports Cursor + Copilot sessions
// for an --include repo path (not Claude-only).
func TestImportLocalSessions_AllAgentsInclude(t *testing.T) {
	gitRoot, otherRepo := setupLocalImport(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("COPILOT_HOME", filepath.Join(home, "copilot"))

	// Cursor transcript for otherRepo.
	stem := "cursor-sess"
	cursorBase := filepath.Join(home, ".cursor", "projects", session.SanitizeCursorRepoPath(otherRepo), "agent-transcripts", stem)
	if err := os.MkdirAll(cursorBase, 0o755); err != nil {
		t.Fatal(err)
	}
	cursorBody := `{"role":"user","message":{"content":[{"type":"text","text":"cursor cross-repo question"}]}}
{"role":"assistant","message":{"content":[{"type":"text","text":"cursor answer"}]}}
`
	if err := os.WriteFile(filepath.Join(cursorBase, stem+".jsonl"), []byte(cursorBody), 0o644); err != nil {
		t.Fatal(err)
	}

	// Copilot session whose cwd is otherRepo.
	copilotDir := filepath.Join(home, "copilot", "session-state", "cp1")
	if err := os.MkdirAll(copilotDir, 0o755); err != nil {
		t.Fatal(err)
	}
	copilotBody := `{"type":"session.start","timestamp":"2026-05-07T10:00:00Z","data":{"sessionId":"cp1","context":{"cwd":"` + otherRepo + `"}}}
{"type":"user.message","timestamp":"2026-05-07T10:00:01Z","data":{"content":"copilot cross-repo question"}}
{"type":"assistant.message","timestamp":"2026-05-07T10:00:02Z","data":{"content":{"text":"copilot answer","length":14}}}
`
	if err := os.WriteFile(filepath.Join(copilotDir, "events.jsonl"), []byte(copilotBody), 0o644); err != nil {
		t.Fatal(err)
	}

	indexDB := openTestIndex(t, gitRoot)
	defer indexDB.Close()

	sessions, projects, err := importLocalSessions(indexDB, gitRoot, localPref{Repos: []string{otherRepo}}, io.Discard)
	if err != nil {
		t.Fatalf("importLocalSessions: %v", err)
	}
	if sessions < 2 {
		t.Fatalf("imported %d sessions from %d origins, want ≥2 (cursor+copilot)", sessions, projects)
	}

	var n int
	if err := indexDB.QueryRow(`SELECT count(*) FROM turns_ft WHERE content LIKE '%cross-repo question%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 2 {
		t.Fatalf("turns matching cross-repo question = %d, want ≥2", n)
	}
}

// TestImportLocalSessions_IncludeAllDiscoversUnscopedSessions verifies
// --include-all pulls sessions that Discover(repo) would miss (wrong cwd).
func TestImportLocalSessions_IncludeAllDiscoversUnscopedSessions(t *testing.T) {
	gitRoot, _ := setupLocalImport(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("COPILOT_HOME", filepath.Join(home, "copilot"))
	// Isolate Claude so include-all doesn't walk the setupLocalImport projects tree only.
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "no-claude"))

	foreign := "/work/foreign-app"
	copilotDir := filepath.Join(home, "copilot", "session-state", "foreign")
	if err := os.MkdirAll(copilotDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"session.start","timestamp":"2026-05-07T10:00:00Z","data":{"sessionId":"foreign","context":{"cwd":"` + foreign + `"}}}
{"type":"user.message","timestamp":"2026-05-07T10:00:01Z","data":{"content":"include-all finds me"}}
{"type":"assistant.message","timestamp":"2026-05-07T10:00:02Z","data":{"content":{"text":"yes","length":3}}}
`
	if err := os.WriteFile(filepath.Join(copilotDir, "events.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	indexDB := openTestIndex(t, gitRoot)
	defer indexDB.Close()

	sessions, _, err := importLocalSessions(indexDB, gitRoot, localPref{All: true}, io.Discard)
	if err != nil {
		t.Fatalf("importLocalSessions: %v", err)
	}
	if sessions < 1 {
		t.Fatal("include-all imported 0 sessions, want the foreign Copilot session")
	}
	var content string
	if err := indexDB.QueryRow(`SELECT content FROM turns_ft WHERE content LIKE '%include-all finds me%'`).Scan(&content); err != nil {
		t.Fatalf("foreign session not in index: %v", err)
	}
}

// TestImportLocalSessions_OpenCodeDedupUsesAdapterDBHash ensures DB-backed
// OpenCode sessions share checkpoint's sha256("opencode:"+id) dedup key.
func TestImportLocalSessions_OpenCodeDedupUsesAdapterDBHash(t *testing.T) {
	gitRoot, otherRepo := setupLocalImport(t)

	home := t.TempDir()
	t.Setenv("HOME", home)
	dbPath := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	odb, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := odb.Exec(`
		CREATE TABLE session (id TEXT PRIMARY KEY, directory TEXT);
		CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, data TEXT, time_created TEXT);
		CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT, data TEXT, time_created TEXT);
		INSERT INTO session (id, directory) VALUES ('oc-dup', ?);
		INSERT INTO message (id, session_id, data, time_created) VALUES
			('m1', 'oc-dup', '{"role":"user"}', '2026-01-01T10:00:00Z'),
			('m2', 'oc-dup', '{"role":"assistant"}', '2026-01-01T10:00:01Z');
		INSERT INTO part (id, message_id, data, time_created) VALUES
			('p1', 'm1', '{"type":"text","text":"already in data.db"}', '2026-01-01T10:00:00Z'),
			('p2', 'm2', '{"type":"text","text":"ok"}', '2026-01-01T10:00:01Z');
	`, otherRepo); err != nil {
		odb.Close()
		t.Fatal(err)
	}
	odb.Close()

	hash := session.ContentHash(&session.OpenCodeAdapter{}, session.SessionRef{DBID: "oc-dup"}, nil)
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InsertSession(dataDB, "existing-oc", "", hash, "human", "", "dev@example.com", "main", "2026-01-01T10:00:00Z", "opencode"); err != nil {
		dataDB.Close()
		t.Fatal(err)
	}
	dataDB.Close()

	indexDB := openTestIndex(t, gitRoot)
	defer indexDB.Close()

	sessions, _, err := importLocalSessions(indexDB, gitRoot, localPref{Repos: []string{otherRepo}}, io.Discard)
	if err != nil {
		t.Fatalf("importLocalSessions: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("imported %d, want 0 (OpenCode hash deduped against data.db)", sessions)
	}
}

// writeTranscript (re)writes a transcript file in otherRepo's project dir.
func writeTranscript(t *testing.T, otherRepo, name, content string) {
	t.Helper()
	projectDir := session.ProjectDirForRepo(otherRepo)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
