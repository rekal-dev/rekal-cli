package transport

import "testing"

// TestExport_WithholdsDuplicateOnlyWhenSurvivorShips guards a way the duplicate
// filter can lose a conversation entirely.
//
// A superseded copy and its survivor can sit on different checkpoints, and the
// merged-only gate judges those independently — an older, merged copy is
// shareable while the longer one is still held back on an unmerged branch.
// Withholding the copy anyway would ship neither, silently dropping the whole
// conversation from the wire. A filter whose job is to remove redundancy must
// never remove the last copy.
func TestExport_WithholdsDuplicateOnlyWhenSurvivorShips(t *testing.T) {
	t.Parallel()

	survivors := map[string]string{"old": "newer"}

	// Case 1: survivor is going out — the older copy is redundant, drop it.
	shipping := map[string]bool{"old": true, "newer": true}
	if !withheldSessions(survivors, shipping)["old"] {
		t.Error("with the survivor shipping, the superseded copy is redundant and must be withheld")
	}

	// Case 2: survivor is held back — the older copy is the only copy.
	shipping = map[string]bool{"old": true}
	if withheldSessions(survivors, shipping)["old"] {
		t.Error("the survivor is not in this export, so withholding the older copy drops the conversation entirely")
	}
}
