package session

import "testing"

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
