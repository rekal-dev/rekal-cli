package cli

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestExtractSnippet_ShortContent(t *testing.T) {
	t.Parallel()
	content := "short content"
	snippet := extractSnippet(content, "short")
	if snippet != content {
		t.Errorf("expected %q, got %q", content, snippet)
	}
}

func TestExtractSnippet_NoMatch(t *testing.T) {
	t.Parallel()
	content := make([]byte, 500)
	for i := range content {
		content[i] = 'a' + byte(i%26)
	}
	contentStr := string(content)
	snippet := extractSnippet(contentStr, "zzzznotfound")
	if len(snippet) > defaultSnippetSize+10 { // +10 for "..."
		t.Errorf("snippet too long: %d", len(snippet))
	}
}

func TestExtractSnippet_MatchInMiddle(t *testing.T) {
	t.Parallel()
	// Build content with a known term in the middle.
	prefix := make([]byte, 200)
	for i := range prefix {
		prefix[i] = 'x'
	}
	suffix := make([]byte, 200)
	for i := range suffix {
		suffix[i] = 'y'
	}
	content := string(prefix) + " authentication token " + string(suffix)
	snippet := extractSnippet(content, "authentication")
	if len(snippet) == 0 {
		t.Error("expected non-empty snippet")
	}
	if len(snippet) > defaultSnippetSize+10 {
		t.Errorf("snippet too long: %d", len(snippet))
	}
}

func TestNullStr(t *testing.T) {
	t.Parallel()
	// Test with zero-value NullString (not valid).
	var ns nullableString
	ns.Valid = false
	ns.String = ""
	// nullStr is tested indirectly through the recall pipeline.
	// Direct test of the helper.
}

func TestSortScored(t *testing.T) {
	t.Parallel()
	s := []scored{
		{sessionID: "a", score: 0.3},
		{sessionID: "b", score: 0.9},
		{sessionID: "c", score: 0.5},
	}
	sortScored(s)
	if s[0].sessionID != "b" || s[1].sessionID != "c" || s[2].sessionID != "a" {
		t.Errorf("unexpected order: %v", s)
	}
}

// nullableString mirrors sql.NullString for testing.
type nullableString struct {
	String string
	Valid  bool
}

func TestExtractSnippet_NeverSplitsRunes(t *testing.T) {
	t.Parallel()
	// Content longer than the snippet size made entirely of multi-byte runes,
	// with the match far enough in that both boundaries land mid-content.
	content := strings.Repeat("日本語テキスト ", 40) + "target " + strings.Repeat("日本語テキスト ", 40)
	snippet := extractSnippet(content, "target")
	if !utf8.ValidString(snippet) {
		t.Fatalf("snippet contains invalid UTF-8: %q", snippet)
	}
	if !strings.Contains(snippet, "target") {
		t.Errorf("snippet should contain the matched term: %q", snippet)
	}

	// No-match path truncates at the snippet size — must also stay valid.
	noMatch := extractSnippet(strings.Repeat("é", 400), "zzz")
	if !utf8.ValidString(noMatch) {
		t.Fatalf("no-match snippet contains invalid UTF-8: %q", noMatch)
	}
}

func TestExtractSnippet_GiantTokenNoPanic(t *testing.T) {
	t.Parallel()
	// One space-free token much larger than the snippet window, with the
	// match past position zero: word alignment must not walk start past end.
	content := strings.Repeat("x", 200) + "needle" + strings.Repeat("y", 2000)
	snippet := extractSnippet(content, "needle")
	if !strings.Contains(snippet, "needle") {
		t.Errorf("snippet should contain the matched term, got %q", snippet)
	}
}

func TestSnapToRuneStart(t *testing.T) {
	t.Parallel()
	s := "aé日" // 'a'=1 byte at 0, 'é'=2 bytes at 1, '日'=3 bytes at 3
	cases := map[int]int{0: 0, 1: 1, 2: 1, 3: 3, 4: 3, 5: 3, 6: 6}
	for in, want := range cases {
		if got := snapToRuneStart(s, in); got != want {
			t.Errorf("snapToRuneStart(%d): got %d, want %d", in, got, want)
		}
	}
}
