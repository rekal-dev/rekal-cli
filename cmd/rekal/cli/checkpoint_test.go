package cli

import (
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

func turns(contents ...string) []session.Turn {
	out := make([]session.Turn, len(contents))
	for i, c := range contents {
		out[i] = session.Turn{Role: "human", Content: c}
	}
	return out
}

// TestTurnsExtend guards the capture append path. Capture reuses an existing
// session only when the freshly parsed transcript is the stored turns plus more
// — the shape a live conversation has when checkpointed again at a later commit.
// Anything else must fall through to a new session, because appending onto a
// record that was rewritten would splice unrelated turns into history, and the
// ledger is append-only precisely so that cannot happen.
func TestTurnsExtend(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		stored []string
		parsed []session.Turn
		want   bool
	}{
		{"grew by one turn", []string{"a", "b"}, turns("a", "b", "c"), true},
		{"unchanged", []string{"a", "b"}, turns("a", "b"), true},
		{"first capture", nil, turns("a"), true},
		{"truncated", []string{"a", "b", "c"}, turns("a", "b"), false},
		{"rewritten mid-history", []string{"a", "b"}, turns("a", "different", "c"), false},
		{"different conversation, same opening", []string{"a", "b"}, turns("a", "z"), false},
		{"emptied", []string{"a"}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := turnsExtend(tc.stored, tc.parsed); got != tc.want {
				t.Errorf("turnsExtend(%v, %d parsed) = %v, want %v", tc.stored, len(tc.parsed), got, tc.want)
			}
		})
	}
}
