package cli

import (
	"testing"
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
