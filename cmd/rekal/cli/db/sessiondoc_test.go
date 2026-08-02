package db

import (
	"strings"
	"testing"
)

// TestBuildSessionDoc_IntentSurvivesALongTranscript is the regression this
// exists for.
//
// The document used to be every turn concatenated in transcript order, so a
// session was truncated positionally at the embedder's window. On a real store
// assistant turns are 84.4% of the bytes against 1.0% for steering, which meant
// two things at once: mean pooling made the vector mostly execution noise, and
// a decision stated late was cut entirely while boilerplate near the top was
// kept. Decisions arrive late — that is the material the ledger exists to hold.
func TestBuildSessionDoc_IntentSurvivesALongTranscript(t *testing.T) {
	t.Parallel()

	filler := strings.Repeat("let me check that file and read the surrounding code ", 400)
	var turns []docTurn
	for i := 0; i < 60; i++ {
		turns = append(turns, docTurn{"assistant", filler})
	}
	// The decision, stated at the very end of a long session.
	turns = append(turns, docTurn{"human_steering", "we rejected the mutex because it deadlocks on rebase"})

	doc := buildSessionDoc(turns)

	if !strings.Contains(doc, "rejected the mutex because it deadlocks on rebase") {
		t.Error("a steering turn at the end of a long session did not reach the document — positional truncation is still dropping the decisions the ledger exists to hold")
	}
	if len(doc) > sessionDocBudgetChars+len(turns) {
		t.Errorf("document is %d chars, over the %d budget", len(doc), sessionDocBudgetChars)
	}
}

// TestBuildSessionDoc_AssistantIsBoundedNotDropped pins both halves of the
// bargain: execution text is capped so it cannot own the mean-pooled vector,
// but it is not discarded, because the reasoning lives there.
func TestBuildSessionDoc_AssistantIsBoundedNotDropped(t *testing.T) {
	t.Parallel()

	turns := []docTurn{
		{"human", "why is the export gate refusing this branch"},
		{"assistant", "MARKER_REASONING " + strings.Repeat("x", 5_000)},
		{"assistant", strings.Repeat("y", 5_000)},
	}
	doc := buildSessionDoc(turns)

	if !strings.Contains(doc, "why is the export gate refusing this branch") {
		t.Fatal("intent turn missing from the document")
	}
	if !strings.Contains(doc, "MARKER_REASONING") {
		t.Error("assistant turns were dropped entirely — the reasoning has to survive, only bounded")
	}
	assistantChars := strings.Count(doc, "x") + strings.Count(doc, "y")
	maxAssistant := sessionDocBudgetChars * assistantAllowanceNum / assistantAllowanceDen
	if assistantChars > maxAssistant {
		t.Errorf("assistant text is %d chars, over the %d allowance — it can dominate the vector again", assistantChars, maxAssistant)
	}
}

// TestBuildSessionDoc_ShortSessionIsUnchanged keeps the change honest for the
// common case: a session that fits under the budget must be assembled whole,
// so nothing is lost from the sessions that were never the problem.
func TestBuildSessionDoc_ShortSessionIsUnchanged(t *testing.T) {
	t.Parallel()

	turns := []docTurn{
		{"human", "why did we pick duckdb"},
		{"assistant", "because it embeds and does analytical SQL in-process"},
		{"human_steering", "keep it"},
	}
	doc := buildSessionDoc(turns)
	for _, want := range []string{"why did we pick duckdb", "because it embeds and does analytical SQL in-process", "keep it"} {
		if !strings.Contains(doc, want) {
			t.Errorf("short session lost %q — sessions under the budget must be assembled whole", want)
		}
	}
}

// TestBuildSessionDoc_TruncationIsSpreadNotTailCut pins that the assistant cut
// lands across the session rather than amputating its end. Filling in
// transcript order would spend the allowance on the first turns and reproduce
// exactly the bias this replaced.
func TestBuildSessionDoc_TruncationIsSpreadNotTailCut(t *testing.T) {
	t.Parallel()

	var turns []docTurn
	for i := 0; i < 40; i++ {
		turns = append(turns, docTurn{"assistant", "HEAD_" + strings.Repeat("a", 4_000)})
	}
	turns = append(turns, docTurn{"assistant", "LAST_" + strings.Repeat("b", 4_000)})

	doc := buildSessionDoc(turns)
	if !strings.Contains(doc, "LAST_") {
		t.Error("the final assistant turn contributed nothing — the allowance was spent front-to-back, so the tail is still being amputated")
	}
}
