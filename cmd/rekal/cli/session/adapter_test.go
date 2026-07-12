package session

import "testing"

// TestNonClaudeAdapters_NeverEmitSummaryRole pins that the "summary" role is
// a Claude-parser concern only: Codex, Gemini, and OpenCode have no
// compaction-summary concept, and a user message that merely starts with
// Claude's continuation boilerplate must parse as an ordinary human turn.
// (The db-side fingerprint reclassification is likewise scoped to
// source = 'claude' — see db.SummaryFingerprint.)
func TestNonClaudeAdapters_NeverEmitSummaryRole(t *testing.T) {
	t.Parallel()

	const boiler = "This session is being continued from a previous conversation that ran out of context."

	codexFixture := `{"type":"session_meta","session_id":"codex-fp","timestamp":"2025-06-01T10:00:00Z","payload":{"cwd":"/tmp/repo"}}
{"type":"event_msg","session_id":"codex-fp","timestamp":"2025-06-01T10:00:01Z","payload":{"type":"user_message","message":"` + boiler + `"}}
`
	geminiFixture := `{"sessionId":"gemini-fp","startTime":"2025-06-01T10:00:00Z","messages":[{"type":"user","content":"` + boiler + `"}]}`

	dir := t.TempDir()
	cases := []struct {
		name    string
		adapter Adapter
		file    string
		content string
	}{
		{"codex", &CodexAdapter{}, dir + "/codex.jsonl", codexFixture},
		{"gemini", &GeminiAdapter{}, dir + "/gemini-fp.json", geminiFixture},
	}
	for _, tc := range cases {
		if err := writeTestFile(tc.file, tc.content); err != nil {
			t.Fatal(err)
		}
		payload, err := tc.adapter.Parse(SessionRef{Path: tc.file})
		if err != nil {
			t.Fatalf("%s Parse: %v", tc.name, err)
		}
		if len(payload.Turns) == 0 {
			t.Fatalf("%s: no turns parsed", tc.name)
		}
		for i, turn := range payload.Turns {
			if turn.Role == "summary" {
				t.Errorf("%s turn %d: role summary must never come from a non-Claude adapter", tc.name, i)
			}
		}
		if payload.Turns[0].Role != "human" {
			t.Errorf("%s Turns[0].Role = %q, want human", tc.name, payload.Turns[0].Role)
		}
	}
}

// TestToolCallArgsFromMap covers the shared file_path/path precedence and
// command-truncation logic factored out of codex.go, opencode.go, and
// gemini.go — previously three independent, identical copies of this same
// extraction (Claude's adapter uses a typed struct instead, for its more
// precisely known tool_use input schema, and isn't part of this dedup).
func TestToolCallArgsFromMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     map[string]interface{}
		wantPath string
		wantCmd  string
	}{
		{
			name:     "file_path preferred over path",
			args:     map[string]interface{}{"file_path": "a.go", "path": "b.go"},
			wantPath: "a.go",
		},
		{
			name:     "falls back to path when file_path absent",
			args:     map[string]interface{}{"path": "b.go"},
			wantPath: "b.go",
		},
		{
			name: "command captured and truncated",
			args: map[string]interface{}{"command": func() string {
				s := make([]byte, 150)
				for i := range s {
					s[i] = 'x'
				}
				return string(s)
			}()},
			wantCmd: func() string {
				s := make([]byte, 100)
				for i := range s {
					s[i] = 'x'
				}
				return string(s)
			}(),
		},
		{
			name:     "non-string values ignored, not panicked on",
			args:     map[string]interface{}{"file_path": 42, "path": true, "command": []string{"x"}},
			wantPath: "",
			wantCmd:  "",
		},
		{
			name: "empty map leaves fields empty",
			args: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var tc ToolCall
			toolCallArgsFromMap(&tc, tt.args)
			if tc.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", tc.Path, tt.wantPath)
			}
			if tc.CmdPrefix != tt.wantCmd {
				t.Errorf("CmdPrefix = %q, want %q", tc.CmdPrefix, tt.wantCmd)
			}
		})
	}
}
