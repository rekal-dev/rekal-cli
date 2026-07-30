package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// TestSupersededSessions_ScalesWithoutPairwiseSQL is a performance fixture, not
// a microbenchmark: it pins the shape of the algorithm, not a wall-clock number.
//
// The prefix test used to be a FULL OUTER JOIN over the turns table per
// candidate *pair*. Pairs grow quadratically inside a group, so a store with a
// few hundred duplicate candidates issued thousands of joins, each scanning
// every turn — minutes of work with data.db held open, blocking every other
// rekal process. The comparison never needed the database.
//
// The fixture is deliberately built as one large group of mutual prefixes,
// which is the worst case: every session is a candidate against every longer
// one. If the pairwise SQL ever comes back, this does not fail by a few
// percent — it fails by orders of magnitude.
func TestSupersededSessions_ScalesWithoutPairwiseSQL(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("performance fixture")
	}
	dir, _ := openTempDB(t)

	d, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer d.Close() //nolint:errcheck
	if err := InitDataSchema(d); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	// 100 growing prefixes of one conversation, 150 turns at the longest:
	// ~10k turns and ~4,950 candidate pairs in a single group. Sized so the
	// fixture's own inserts stay cheap — the pairwise version needed ~1.2s per
	// pair on this shape, so even this many pairs is over an hour of joins,
	// which is the difference this guards.
	const sessions, maxTurns = 100, 150
	turns := make([]string, maxTurns)
	for i := range turns {
		turns[i] = fmt.Sprintf("turn %d: a reasonably sized line of conversation about the retry path", i)
	}
	tx, err := d.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for s := 0; s < sessions; s++ {
		id := fmt.Sprintf("s%04d", s)
		if _, err := tx.Exec(
			`INSERT INTO sessions (id, session_hash, captured_at, actor_type, user_email, branch, source)
			 VALUES ($1, $2, $3, 'human', 'a@b.c', 'main', 'claude')`,
			id, "h-"+id, "2026-07-11T00:00:00Z"); err != nil {
			t.Fatalf("insert session: %v", err)
		}
		n := maxTurns - (sessions - 1 - s) // each one turn longer than the last
		for i := 0; i < n; i++ {
			if _, err := tx.Exec(
				`INSERT INTO turns (id, session_id, turn_index, role, content)
				 VALUES ($1, $2, $3, 'human', $4)`,
				fmt.Sprintf("%s:%d", id, i), id, i, turns[i]); err != nil {
				t.Fatalf("insert turn: %v", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	start := time.Now()
	got, err := supersededSessions(d, "turns", "sessions", "")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("supersededSessions: %v", err)
	}

	// Correctness first: every session but the longest is a strict prefix.
	if len(got) != sessions-1 {
		t.Errorf("collapsed %d sessions, want %d — the fast path must find exactly what the pairwise one did", len(got), sessions-1)
	}
	longest := fmt.Sprintf("s%04d", sessions-1)
	for old, survivor := range got {
		if survivor != longest {
			t.Errorf("%s → %s, want %s", old, survivor, longest)
			break
		}
	}

	// ~19,900 pairs. One database round trip per pair is what this guards
	// against; the bound is loose on purpose so it fails on the algorithm
	// rather than on a slow machine.
	if elapsed > 30*time.Second {
		t.Errorf("supersession took %s for %d sessions — that is pairwise-SQL shaped, not a bulk in-memory comparison", elapsed, sessions)
	}
	t.Logf("%d sessions / ~%d pairs in %s", sessions, sessions*(sessions-1)/2, elapsed)
}

var _ = sql.ErrNoRows
