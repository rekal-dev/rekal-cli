package cli

import (
	"testing"
	"time"
)

func TestResolveWhen(t *testing.T) {
	t.Parallel()

	mustDate := func(s string) time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("bad test date %q: %v", s, err)
		}
		return d
	}

	// Vectors ported from the skill's when.py test set (byte-identical output).
	cases := []struct {
		anchor, phrase, want string
	}{
		{"2023-05-25", "last Saturday", "2023-05-20 (Saturday)"},
		{"2023-05-25", "yesterday", "2023-05-24 (Wednesday)"},
		{"2023-05-25", "three weeks ago", "2023-05-04 (Thursday)"},
		{"2023-05-25", "2 days ago", "2023-05-23 (Tuesday)"},
		{"2023-05-25", "next Friday", "2023-05-26 (Friday)"},
		{"2023-05-25", "last month", "2023-04-01..2023-04-30 (month)"},
		{"2023-05-25", "a few days ago", "2023-05-20..2023-05-24 (approx)"},
		{"2023-01-01", "last year", "2022-01-01..2022-12-31 (year)"},
		{"2023-08-14", "last night", "2023-08-13 (Sunday)"},
		{"2023-05-25", "this morning", "2023-05-25 (Thursday)"},
		{"2023-11-22", "a few days before", "2023-11-17..2023-11-21 (approx)"},
		{"2023-11-22", "several days earlier", "2023-11-17..2023-11-21 (approx)"},
		{"2023-05-25", "  Last   SATURDAY  .", "2023-05-20 (Saturday)"}, // normalization
		{"2024-01-31", "one month ago", "2023-12-31 (Sunday)"},          // day clamp
	}
	for _, c := range cases {
		got, ok := resolveWhen(mustDate(c.anchor), c.phrase)
		if !ok || got != c.want {
			t.Errorf("resolveWhen(%s, %q) = %q ok=%v, want %q", c.anchor, c.phrase, got, ok, c.want)
		}
	}

	// Unrecognized phrases return ok=false (say so, don't guess).
	for _, p := range []string{"gibberish", "", "sometime maybe"} {
		if got, ok := resolveWhen(mustDate("2023-05-25"), p); ok {
			t.Errorf("resolveWhen unrecognized %q returned %q ok=true", p, got)
		}
	}
}
