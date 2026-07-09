package scrub

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

func TestSanitizeText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, in string
		check    func(string) bool
	}{
		{"valid untouched", "hello, 世界", func(s string) bool { return s == "hello, 世界" }},
		{"empty", "", func(s string) bool { return s == "" }},
		{"strips NUL", "a\x00b", func(s string) bool { return s == "ab" }},
		{"invalid utf8 replaced", "abc\xff\xfedef", func(s string) bool {
			return utf8.ValidString(s) && strings.HasPrefix(s, "abc") && strings.HasSuffix(s, "def")
		}},
		{"split multibyte rune", "café"[:5], func(s string) bool { return utf8.ValidString(s) }}, // "café" is 5 bytes; [:5] is whole, use next
	}
	for _, tc := range cases {
		if got := SanitizeText(tc.in); !tc.check(got) {
			t.Errorf("%s: SanitizeText(%q) = %q failed check", tc.name, tc.in, got)
		}
	}

	// A genuinely split multi-byte rune must come out valid.
	split := strings.Repeat("é", 60)[:101] // odd byte count splits the last "é"
	if got := SanitizeText(split); !utf8.ValidString(got) {
		t.Fatalf("SanitizeText left invalid UTF-8: %q", got)
	}
}

// TestScrub_GuaranteesValidUTF8 is the regression test for the checkpoint
// "insert tool_call: could not bind parameter" failure: a tool call whose
// command carried invalid UTF-8 (e.g. binary output, or a mid-rune truncation)
// must be sanitized so DuckDB can bind it.
func TestScrub_GuaranteesValidUTF8(t *testing.T) {
	t.Parallel()
	payload := &session.SessionPayload{
		Turns:     []session.Turn{{Role: "human", Content: "ok\xff\xfe bytes"}},
		ToolCalls: []session.ToolCall{{Tool: "Bash", Path: "p\xffath", CmdPrefix: "run \xc3 truncated"}},
	}
	Scrub(payload)

	if !utf8.ValidString(payload.Turns[0].Content) {
		t.Errorf("turn content still invalid UTF-8: %q", payload.Turns[0].Content)
	}
	tc := payload.ToolCalls[0]
	for name, v := range map[string]string{"tool": tc.Tool, "path": tc.Path, "cmd_prefix": tc.CmdPrefix} {
		if !utf8.ValidString(v) {
			t.Errorf("tool_call %s still invalid UTF-8: %q", name, v)
		}
	}
}
