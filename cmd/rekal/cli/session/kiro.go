package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// KiroAdapter discovers and parses Kiro (kiro.dev) chat sessions from both the
// CLI and the IDE. Kiro's schema is not officially published
// (github kirodotdev/Kiro#5094); both formats are verified against community
// readers (prabhugr/kiro-cli-history for the CLI, pajaydev/kiro-history for the
// IDE).
//
// CLI (v3 "JSONL"), under $KIRO_HOME (default ~/.kiro):
//
//	sessions/cli/<id>.json    metadata {session_id, cwd, title, created_at}
//	sessions/cli/<id>.jsonl   one event per line —
//	                          {"kind":"Prompt"|"AssistantMessage","data":{"content":[{"kind":"text","data":"…"}]}}
//
// IDE, under the Code-OSS global storage (macOS ~/Library/Application Support/
// Kiro, Linux ~/.config/Kiro, Windows %APPDATA%/Kiro) at
// User/globalStorage/kiro.kiroagent/workspace-sessions/<ws>/:
//
//	sessions.json     index: [{sessionId, dateCreated, workspaceDirectory, hidden}]
//	<sessionId>.json  {history:[{message:{role:"user"|"assistant", content: string | [{type,text}]}}]}
//
// Discovery is exact for both — the CLI's `cwd` and the IDE index's
// `workspaceDirectory` record the repo. Tool calls carry no documented shape,
// so they're extracted best-effort and fail soft; the conversation text is the
// reliable part. Parse dispatches on the ref's extension (.jsonl → CLI,
// .json → IDE).
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

func kiroCLIDir() string {
	base := kiroHome()
	if base == "" {
		return ""
	}
	return filepath.Join(base, "sessions", "cli")
}

func (a *KiroAdapter) Discover(repoPath string) ([]SessionRef, error) {
	var refs []SessionRef
	refs = append(refs, discoverKiroCLI(repoPath)...)
	refs = append(refs, discoverKiroIDE(repoPath)...)
	return refs, nil
}

func discoverKiroCLI(repoPath string) []SessionRef {
	cliDir := kiroCLIDir()
	if cliDir == "" {
		return nil
	}
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return nil // no Kiro CLI sessions on this machine
	}

	var refs []SessionRef
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		metaPath := filepath.Join(cliDir, e.Name())
		meta, err := readKiroMeta(metaPath)
		if err != nil {
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
	return refs
}

// kiroRepoMatch reports whether a session's cwd belongs to repoPath — the exact
// repo or a path inside it, never a sibling that merely shares the prefix
// (repo vs repo-2).
func kiroRepoMatch(cwd, repoPath string) bool {
	return cwd == repoPath || strings.HasPrefix(cwd, repoPath+string(os.PathSeparator))
}

func (a *KiroAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
	if strings.HasSuffix(ref.Path, ".jsonl") {
		return parseKiroCLI(ref)
	}
	return parseKiroIDE(ref.Path)
}

func parseKiroCLI(ref SessionRef) (*SessionPayload, error) {
	data, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	payload := &SessionPayload{
		SessionID: strings.TrimSuffix(filepath.Base(ref.Path), ".jsonl"),
		Source:    "kiro",
		ActorType: "human",
	}
	// captured_at comes from the sibling metadata's created_at when present;
	// wall clock is the fallback.
	payload.CapturedAt = time.Now().UTC()
	metaPath := strings.TrimSuffix(ref.Path, ".jsonl") + ".json"
	if meta, err := readKiroMeta(metaPath); err == nil {
		if ts := parseTimestamp(meta.CreatedAt); !ts.IsZero() {
			payload.CapturedAt = ts
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		var ev kiroEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		var role string
		switch ev.Kind {
		case "Prompt":
			role = "human"
		case "AssistantMessage":
			role = "assistant"
		default:
			continue // tool results, status, and other event kinds carry no turn
		}
		text, tools := kiroBlocks(ev.Data.Content)
		if text != "" {
			payload.Turns = append(payload.Turns, Turn{Role: role, Content: text})
		}
		if role == "assistant" {
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
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func readKiroMeta(path string) (kiroSessionMeta, error) {
	var meta kiroSessionMeta
	data, err := os.ReadFile(path)
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

// kiroEvent is one line of the .jsonl event log. `kind` names the event
// ("Prompt" / "AssistantMessage" / tool + status kinds); `data.content` is the
// array of content blocks.
type kiroEvent struct {
	Kind string `json:"kind"`
	Data struct {
		Content []kiroBlock `json:"content"`
	} `json:"data"`
}

// kiroBlock is one content block. For a text block, kind == "text" and Data is
// the plain string. Other kinds (tool use) carry an object whose shape is not
// publicly documented.
type kiroBlock struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

// kiroBlocks joins the text of a message's content blocks and best-effort
// extracts tool calls from the non-text blocks (fail soft — the text is the
// documented, reliable part).
func kiroBlocks(blocks []kiroBlock) (string, []ToolCall) {
	var texts []string
	var tools []ToolCall
	for _, b := range blocks {
		if b.Kind == "text" {
			var s string
			if err := json.Unmarshal(b.Data, &s); err == nil {
				if s = strings.TrimSpace(s); s != "" {
					texts = append(texts, s)
				}
			}
			continue
		}
		if !strings.Contains(strings.ToLower(b.Kind), "tool") {
			continue
		}
		// Best-effort tool block: pull a name and argument map under any of the
		// plausible keys. An unrecognized shape yields no tool (never fatal).
		var td struct {
			Name     string                 `json:"name"`
			ToolName string                 `json:"toolName"`
			Tool     string                 `json:"tool"`
			Input    map[string]interface{} `json:"input"`
			Args     map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(b.Data, &td); err != nil {
			continue
		}
		name := firstNonEmpty(td.Name, td.ToolName, td.Tool)
		if name == "" {
			continue
		}
		tc := ToolCall{Tool: name}
		if td.Input != nil {
			toolCallArgsFromMap(&tc, td.Input)
		} else if td.Args != nil {
			toolCallArgsFromMap(&tc, td.Args)
		}
		tools = append(tools, tc)
	}
	return strings.TrimSpace(strings.Join(texts, "\n")), tools
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// kiroRole maps an IDE message role onto the canonical human/assistant turn
// roles, or "" for a role that carries no conversational turn.
func kiroRole(role, _ string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user", "human":
		return "human"
	case "assistant", "ai", "model":
		return "assistant"
	}
	return ""
}

// --- Kiro IDE sessions (Code-OSS global storage) ---

// kiroIDEStorageDir returns Kiro IDE's globalStorage/kiro.kiroagent directory
// for this platform, or the $KIRO_IDE_STORAGE override (used by tests). Empty
// when it can't be resolved.
func kiroIDEStorageDir() string {
	if s := strings.TrimSpace(os.Getenv("KIRO_IDE_STORAGE")); s != "" {
		return s
	}
	if runtime.GOOS == "windows" {
		appdata := os.Getenv("APPDATA")
		if appdata == "" {
			return ""
		}
		return filepath.Join(appdata, "Kiro", "User", "globalStorage", "kiro.kiroagent")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "Kiro", "User", "globalStorage", "kiro.kiroagent")
	}
	return filepath.Join(home, ".config", "Kiro", "User", "globalStorage", "kiro.kiroagent")
}

func discoverKiroIDE(repoPath string) []SessionRef {
	base := kiroIDEStorageDir()
	if base == "" {
		return nil
	}
	wsRoot := filepath.Join(base, "workspace-sessions")
	wsDirs, err := os.ReadDir(wsRoot)
	if err != nil {
		return nil // no Kiro IDE sessions on this machine
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
			// Prefer the index's workspaceDirectory; fall back to the session
			// file's own workspacePath/workspaceDirectory when the index omits it.
			wsPath := e.WorkspaceDirectory
			if wsPath == "" {
				if s, err := readKiroIDESession(sf); err == nil {
					wsPath = firstNonEmpty(s.WorkspacePath, s.WorkspaceDirectory)
				}
			}
			if wsPath == "" || !kiroRepoMatch(kiroStripFileScheme(wsPath), repoPath) {
				continue
			}
			if _, err := os.Stat(sf); err != nil {
				continue
			}
			refs = append(refs, SessionRef{Path: sf})
		}
	}
	return refs
}

// kiroStripFileScheme drops a leading file:// so a VS Code-style workspace URI
// compares against a plain repo path.
func kiroStripFileScheme(dir string) string {
	return strings.TrimPrefix(strings.TrimPrefix(dir, "file://"), "localhost")
}

func parseKiroIDE(path string) (*SessionPayload, error) {
	sess, err := readKiroIDESession(path)
	if err != nil {
		return nil, nil // not a recognizable IDE session file — skip, never fatal
	}
	sessionID := strings.TrimSuffix(filepath.Base(path), ".json")
	payload := &SessionPayload{
		SessionID:  sessionID,
		Source:     "kiro",
		ActorType:  "human",
		CapturedAt: time.Now().UTC(),
	}
	// captured_at from the workspace's sessions.json index (dateCreated).
	if ts := kiroIDECreatedAt(filepath.Dir(path), sessionID); !ts.IsZero() {
		payload.CapturedAt = ts
	}
	for _, h := range sess.History {
		role := kiroRole(h.Message.Role, "")
		if role == "" {
			continue
		}
		if text := kiroIDEText(h.Message.Content); text != "" {
			payload.Turns = append(payload.Turns, Turn{Role: role, Content: text})
		}
		if role == "assistant" {
			payload.ToolCalls = append(payload.ToolCalls, kiroIDETools(h.ToolUses)...)
		}
	}
	return payload, nil
}

func readKiroIDESession(path string) (kiroIDESession, error) {
	var sess kiroIDESession
	data, err := os.ReadFile(path)
	if err != nil {
		return sess, err
	}
	if err := json.Unmarshal(data, &sess); err != nil {
		return sess, err
	}
	return sess, nil
}

// kiroIDECreatedAt reads the workspace sessions.json index in dir and returns
// the dateCreated of the given session, or zero when unavailable.
func kiroIDECreatedAt(dir, sessionID string) time.Time {
	data, err := os.ReadFile(filepath.Join(dir, "sessions.json"))
	if err != nil {
		return time.Time{}
	}
	var index []kiroIDEIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return time.Time{}
	}
	for _, e := range index {
		if e.SessionID == sessionID {
			return parseTimestamp(e.DateCreated)
		}
	}
	return time.Time{}
}

// kiroIDEIndex is one entry of a workspace's sessions.json index.
type kiroIDEIndex struct {
	SessionID          string `json:"sessionId"`
	Title              string `json:"title"`
	DateCreated        string `json:"dateCreated"`
	WorkspaceDirectory string `json:"workspaceDirectory"`
	Hidden             bool   `json:"hidden"`
}

// kiroIDESession is a <sessionId>.json IDE session file.
type kiroIDESession struct {
	History            []kiroIDEEntry `json:"history"`
	Title              string         `json:"title"`
	SessionID          string         `json:"sessionId"`
	WorkspacePath      string         `json:"workspacePath"`
	WorkspaceDirectory string         `json:"workspaceDirectory"`
}

// kiroIDEEntry is one history entry. Content is a string or an array of
// {type,text} parts; ToolUses (when present) carry the turn's tool calls.
type kiroIDEEntry struct {
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
		ID      string          `json:"id"`
	} `json:"message"`
	ExecutionID string          `json:"executionId"`
	ToolUses    json.RawMessage `json:"toolUses"`
}

// kiroIDEText extracts a message's text — a plain string, or the joined text of
// an array of {type,text} parts.
func kiroIDEText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	var texts []string
	for _, p := range parts {
		if p.Text != "" {
			texts = append(texts, p.Text)
		}
	}
	return strings.TrimSpace(strings.Join(texts, "\n"))
}

// kiroIDETools best-effort extracts tool calls from an assistant entry's
// toolUses. The block shape is undocumented, so it accepts the plausible name /
// argument keys and fails soft.
func kiroIDETools(raw json.RawMessage) []ToolCall {
	if len(raw) == 0 {
		return nil
	}
	var uses []struct {
		Name     string                 `json:"name"`
		ToolName string                 `json:"toolName"`
		Tool     string                 `json:"tool"`
		Input    map[string]interface{} `json:"input"`
		Args     map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &uses); err != nil {
		return nil
	}
	var tools []ToolCall
	for _, u := range uses {
		name := firstNonEmpty(u.Name, u.ToolName, u.Tool)
		if name == "" {
			continue
		}
		tc := ToolCall{Tool: name}
		if u.Input != nil {
			toolCallArgsFromMap(&tc, u.Input)
		} else if u.Args != nil {
			toolCallArgsFromMap(&tc, u.Args)
		}
		tools = append(tools, tc)
	}
	return tools
}
