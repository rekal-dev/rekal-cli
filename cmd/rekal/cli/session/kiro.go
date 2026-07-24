package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// KiroAdapter discovers and parses Kiro (kiro.dev) CLI chat sessions.
//
// Layout (Kiro CLI v3 "JSONL" format; verified against the community reader
// prabhugr/kiro-cli-history since Kiro's schema is not officially published —
// github kirodotdev/Kiro#5094):
//
//	$KIRO_HOME/sessions/cli/<session-id>.json    metadata {session_id, cwd, title, created_at, updated_at}
//	$KIRO_HOME/sessions/cli/<session-id>.jsonl   one event object per line
//
// where $KIRO_HOME defaults to ~/.kiro. Each .jsonl line is
// `{"kind": <event>, "data": {"content": [ {"kind":"text","data":"…"}, … ]}}`;
// the conversational events are `kind:"Prompt"` (the human) and
// `kind:"AssistantMessage"` (the model), and text lives in content blocks whose
// own `kind` is `"text"` (their `data` is the string). Discovery is exact — the
// sibling .json's `cwd` records the repo the session ran in. Tool-call blocks
// carry no publicly documented shape, so they are extracted best-effort and
// fail soft. IDE sessions (stored without cwd metadata) are not yet covered.
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
	cliDir := kiroCLIDir()
	if cliDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return nil, nil // no Kiro CLI sessions on this machine
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
	return refs, nil
}

// kiroRepoMatch reports whether a session's cwd belongs to repoPath — the exact
// repo or a path inside it, never a sibling that merely shares the prefix
// (repo vs repo-2).
func kiroRepoMatch(cwd, repoPath string) bool {
	return cwd == repoPath || strings.HasPrefix(cwd, repoPath+string(os.PathSeparator))
}

func (a *KiroAdapter) Parse(ref SessionRef) (*SessionPayload, error) {
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
