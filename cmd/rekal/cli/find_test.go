package cli

import (
	"strings"
	"testing"
)

func TestFindSnippet(t *testing.T) {
	t.Parallel()
	// Expectations computed from the skill's find.py snippet() (byte-identical).
	cases := []struct {
		content, term, want string
	}{
		{"the quick brown fox jumps over the lazy dog", "fox", "the quick brown fox jumps over the lazy dog"},
		{strings.Repeat("word ", 30) + "TARGET " + strings.Repeat("tail ", 30), "TARGET",
			"…word word word word word word word word word word word word TARGET tail tail tail tail tail tail tail tail tail tail tail tail…"},
		{"short text here", "missing", "short text here"},
		{"Multi  whitespace\n\tcollapsed HIT around", "hit", "Multi w hitespace collapsed HIT around"},
		{"prefixHITsuffix inline", "hit", "prefix HITsuffix inline"},
	}
	for _, c := range cases {
		if got := findSnippet(c.content, c.term); got != c.want {
			t.Errorf("findSnippet(%q, %q)\n = %q\nwant %q", c.content, c.term, got, c.want)
		}
	}
}

func TestEscapeLike(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"plain":       "plain",
		"50%":         `50\%`,
		"under_score": `under\_score`,
		`back\slash`:  `back\\slash`,
		`%_\`:         `\%\_\\`,
	}
	for in, want := range cases {
		if got := escapeLike(in); got != want {
			t.Errorf("escapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}
