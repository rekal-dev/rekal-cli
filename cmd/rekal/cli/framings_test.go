package cli

import (
	"reflect"
	"testing"
)

func TestDeriveFramings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{
			// No reformulation possible → single framing → single search
			// (byte-identical to a plain recall).
			name:  "single word stays single",
			query: "auth",
			want:  []string{"auth"},
		},
		{
			// Keyword variant equals the query (no stopwords) → deduped away.
			name:  "no stopwords no extra framing",
			query: "jwt expiry rotation",
			want:  []string{"jwt expiry rotation"},
		},
		{
			// Stopwords present → adds a keyword-only variant.
			name:  "keyword variant added",
			query: "how does the jwt refresh work",
			want:  []string{"how does the jwt refresh work", "jwt refresh work"},
		},
		{
			// Conjunction → clause split framings, plus keyword variant.
			name:  "clause split on and",
			query: "add jwt auth and rotate the refresh token",
			want: []string{
				"add jwt auth and rotate the refresh token",
				"add jwt auth rotate refresh token",
				"add jwt auth",
				"rotate the refresh token",
			},
		},
		{
			// Temporal cue → time-emphasis variant.
			name:  "temporal cue adds time variant",
			query: "when did we change expiry",
			want: []string{
				"when did we change expiry",
				"change expiry",
				"change expiry date time when",
			},
		},
		{
			name:  "empty stays empty single",
			query: "",
			want:  []string{""},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := deriveFramings(c.query)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("deriveFramings(%q) =\n  %#v\nwant\n  %#v", c.query, got, c.want)
			}
			if len(got) > maxFramings {
				t.Errorf("deriveFramings(%q) returned %d framings, exceeds cap %d", c.query, len(got), maxFramings)
			}
		})
	}
}
