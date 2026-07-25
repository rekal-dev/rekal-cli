package session

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// AllDiscoverer is implemented by adapters that can enumerate every local
// session (cross-repo import --include-all), not just those matching one repo.
type AllDiscoverer interface {
	DiscoverAll() ([]SessionRef, error)
}

// DiscoverAllSessions returns every local session for a, or nil when the
// adapter has nothing on this machine / does not implement AllDiscoverer.
func DiscoverAllSessions(a Adapter) ([]SessionRef, error) {
	if d, ok := a.(AllDiscoverer); ok {
		return d.DiscoverAll()
	}
	return nil, nil
}

// ContentHash is the dedup key shared by checkpoint and cross-repo import.
// File refs hash transcript bytes; DB refs (OpenCode) hash "name:DBID".
func ContentHash(a Adapter, ref SessionRef, fileData []byte) string {
	if ref.Path != "" {
		return sha256HexBytes(fileData)
	}
	return sha256HexBytes([]byte(a.Name() + ":" + ref.DBID))
}

func sha256HexBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// DiscoverAll returns every Claude Code project session on this machine
// (repos + shell), matching EnumerateProjectDirs used by the original import.
func (a *ClaudeAdapter) DiscoverAll() ([]SessionRef, error) {
	dirs, err := EnumerateProjectDirs()
	if err != nil {
		return nil, err
	}
	var refs []SessionRef
	for _, dir := range dirs {
		found, derr := discoverSessionRefs(dir)
		if derr != nil {
			continue
		}
		refs = append(refs, found...)
	}
	return refs, nil
}

// DiscoverAll returns every Cursor agent transcript under ~/.cursor/projects.
func (a *CursorAdapter) DiscoverAll() ([]SessionRef, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	root := filepath.Join(home, ".cursor", "projects")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil
	}
	var refs []SessionRef
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		found, derr := discoverCursorSessionRefs(filepath.Join(root, e.Name(), "agent-transcripts"))
		if derr != nil || len(found) == 0 {
			continue
		}
		refs = append(refs, found...)
	}
	return refs, nil
}

// DiscoverAll returns every Codex session JSONL (active + archived).
func (a *CodexAdapter) DiscoverAll() ([]SessionRef, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	var refs []SessionRef
	for _, dir := range []string{
		filepath.Join(home, ".codex", "sessions"),
		filepath.Join(home, ".codex", "archived_sessions"),
	} {
		matches, err := findJSONLFiles(dir)
		if err != nil {
			continue
		}
		for _, f := range matches {
			refs = append(refs, SessionRef{Path: f})
		}
	}
	return refs, nil
}

// DiscoverAll returns every Gemini CLI session under ~/.gemini/tmp/*/chats.
func (a *GeminiAdapter) DiscoverAll() ([]SessionRef, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	tmpDir := filepath.Join(home, ".gemini", "tmp")
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, nil
	}
	var refs []SessionRef
	for _, e := range entries {
		if !e.IsDir() {
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

// DiscoverAll returns every OpenCode session in the local SQLite store.
func (a *OpenCodeAdapter) DiscoverAll() ([]SessionRef, error) {
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

	rows, err := db.Query("SELECT id FROM session")
	if err != nil {
		return nil, nil
	}
	defer rows.Close() //nolint:errcheck

	var refs []SessionRef
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		refs = append(refs, SessionRef{DBID: id})
	}
	return refs, nil
}

// DiscoverAll returns every Copilot CLI events.jsonl on this machine.
func (a *CopilotAdapter) DiscoverAll() ([]SessionRef, error) {
	base := copilotHome()
	if base == "" {
		return nil, nil
	}
	stateDir := filepath.Join(base, "session-state")
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil, nil
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
		refs = append(refs, SessionRef{Path: events})
	}
	return refs, nil
}

// DiscoverAll returns every Kiro CLI + IDE session, unscoped by repo.
func (a *KiroAdapter) DiscoverAll() ([]SessionRef, error) {
	var refs []SessionRef
	refs = append(refs, discoverKiroCLIAll()...)
	refs = append(refs, discoverKiroIDEAll()...)
	return refs, nil
}

func discoverKiroCLIAll() []SessionRef {
	cliDir := kiroCLIDir()
	if cliDir == "" {
		return nil
	}
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return nil
	}
	var refs []SessionRef
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		metaPath := filepath.Join(cliDir, e.Name())
		jsonl := strings.TrimSuffix(metaPath, ".json") + ".jsonl"
		if _, err := os.Stat(jsonl); err != nil {
			continue
		}
		refs = append(refs, SessionRef{Path: jsonl})
	}
	return refs
}

func discoverKiroIDEAll() []SessionRef {
	base := kiroIDEStorageDir()
	if base == "" {
		return nil
	}
	wsRoot := filepath.Join(base, "workspace-sessions")
	wsDirs, err := os.ReadDir(wsRoot)
	if err != nil {
		return nil
	}
	var refs []SessionRef
	for _, ws := range wsDirs {
		if !ws.IsDir() {
			continue
		}
		wsDir := filepath.Join(wsRoot, ws.Name())
		idxData, err := os.ReadFile(filepath.Join(wsDir, "sessions.json"))
		if err != nil {
			continue
		}
		var index []kiroIDEIndex
		if err := json.Unmarshal(idxData, &index); err != nil {
			continue
		}
		for _, e := range index {
			if e.Hidden || e.SessionID == "" {
				continue
			}
			sf := filepath.Join(wsDir, e.SessionID+".json")
			if _, err := os.Stat(sf); err != nil {
				continue
			}
			refs = append(refs, SessionRef{Path: sf})
		}
	}
	return refs
}
