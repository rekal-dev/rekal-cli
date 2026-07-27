package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

// seedSession inserts a session and n turns whose contents come from body.
func seedSession(t *testing.T, d *sql.DB, id, source, email string, body []string) {
	t.Helper()
	if err := InsertSession(d, id, "", "h-"+id, "human", "", email, "main", "2026-07-11T00:00:00Z", source); err != nil {
		t.Fatalf("InsertSession %s: %v", id, err)
	}
	for i, c := range body {
		if err := InsertTurn(d, fmt.Sprintf("%s:%d", id, i), id, i, "human", c, ""); err != nil {
			t.Fatalf("InsertTurn %s:%d: %v", id, i, err)
		}
	}
}

// TestPurgeSuperseded_CollapsesRecaptures covers the duplicate-session bug: before
// capture keyed on transcript identity, every commit during a live session stored
// the whole conversation again. The ledger keeps those copies (append-only, and
// already on the wire), so the index must collapse a chain of prefixes down to the
// longest — while never merging two conversations that merely start alike.
func TestPurgeSuperseded_CollapsesRecaptures(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	// One conversation captured at three commits — a prefix chain.
	grow := []string{"start the work", "keep going", "and finish"}
	seedSession(t, dataDB, "cap1", "claude", "a@b.c", grow[:1])
	seedSession(t, dataDB, "cap2", "claude", "a@b.c", grow[:2])
	seedSession(t, dataDB, "cap3", "claude", "a@b.c", grow)

	// Same opening turn, different agent — must survive.
	seedSession(t, dataDB, "other", "codex", "a@b.c", grow[:1])
	// Same opening turn, same agent, but it diverges — must survive.
	seedSession(t, dataDB, "fork", "claude", "a@b.c", []string{"start the work", "a different path"})
	dataDB.Close() //nolint:errcheck

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close() //nolint:errcheck
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := PopulateIndex(indexDB, dir); err != nil {
		t.Fatalf("PopulateIndex: %v", err)
	}

	live := map[string]bool{}
	rows, err := indexDB.Query(`SELECT DISTINCT session_id FROM turns_ft`)
	if err != nil {
		t.Fatalf("query turns_ft: %v", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		live[id] = true
	}
	rows.Close() //nolint:errcheck

	for _, gone := range []string{"cap1", "cap2"} {
		if live[gone] {
			t.Errorf("%s is a strict prefix of cap3 and should have been collapsed", gone)
		}
	}
	for _, kept := range []string{"cap3", "other", "fork"} {
		if !live[kept] {
			t.Errorf("%s must survive: it is either the longest capture or a distinct conversation", kept)
		}
	}
}

// TestSupersededSessionIDs_DataDB covers the export-side use of the same
// detection. It reads the data DB's own tables rather than the index's, and a
// silent failure there is worse than a loud one: export would ship the
// duplicates the filter exists to withhold, onto a branch other people fetch.
func TestSupersededSessionIDs_DataDB(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	defer dataDB.Close() //nolint:errcheck
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	grow := []string{"opening turn", "second", "third"}
	seedSession(t, dataDB, "short", "claude", "a@b.c", grow[:1])
	seedSession(t, dataDB, "mid", "claude", "a@b.c", grow[:2])
	seedSession(t, dataDB, "full", "claude", "a@b.c", grow)
	seedSession(t, dataDB, "elsewhere", "codex", "a@b.c", grow[:1])

	got, err := SupersededSessionIDs(dataDB)
	if err != nil {
		t.Fatalf("SupersededSessionIDs: %v", err)
	}
	for _, id := range []string{"short", "mid"} {
		if !got[id] {
			t.Errorf("%s is a strict prefix of full and must be withheld from the wire", id)
		}
	}
	for _, id := range []string{"full", "elsewhere"} {
		if got[id] {
			t.Errorf("%s must be shipped: it is the longest capture or a different agent's session", id)
		}
	}
}

// TestPurgeSuperseded_RepointsSubagentParents covers the interaction between the
// supersession collapse and subagent linkage.
//
// Under the duplicate-session bug a trunk was re-stored at every commit, so a
// subagent captured mid-conversation was linked to whichever trunk copy existed
// at that moment. Collapsing the copies then strands it: search groups sessions
// by walking parent_session_id to the root, and a parent that is no longer in the
// index breaks that walk — the subagent's work either detaches into its own
// result or vanishes from its conversation.
//
// The surviving copy is the same conversation, so children must follow it.
func TestPurgeSuperseded_RepointsSubagentParents(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	grow := []string{"trunk opening", "trunk continues", "trunk finishes"}
	seedSession(t, dataDB, "trunk-early", "claude", "a@b.c", grow[:2])
	seedSession(t, dataDB, "trunk-full", "claude", "a@b.c", grow)

	// A subagent captured while the trunk was still the early copy.
	if err := InsertSession(dataDB, "kid", "trunk-early", "h-kid", "agent", "",
		"a@b.c", "main", "2026-07-11T00:00:00Z", "claude"); err != nil {
		t.Fatalf("InsertSession kid: %v", err)
	}
	if err := InsertTurn(dataDB, "kid:0", "kid", 0, "assistant", "subagent did the work", ""); err != nil {
		t.Fatalf("InsertTurn kid: %v", err)
	}
	dataDB.Close() //nolint:errcheck

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close() //nolint:errcheck
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := PopulateIndex(indexDB, dir); err != nil {
		t.Fatalf("PopulateIndex: %v", err)
	}

	var parent string
	if err := indexDB.QueryRow(
		`SELECT COALESCE(parent_session_id, '') FROM session_facets WHERE session_id = 'kid'`,
	).Scan(&parent); err != nil {
		t.Fatalf("read kid's parent: %v", err)
	}

	var trunkEarlyLives int
	if err := indexDB.QueryRow(
		`SELECT count(*) FROM turns_ft WHERE session_id = 'trunk-early'`,
	).Scan(&trunkEarlyLives); err != nil {
		t.Fatalf("check purged trunk: %v", err)
	}
	if trunkEarlyLives != 0 {
		t.Fatal("setup: trunk-early should have been collapsed into trunk-full")
	}

	if parent != "trunk-full" {
		t.Errorf("subagent parent = %q, want trunk-full — its trunk was collapsed away, so it must follow the surviving copy or it detaches from its conversation", parent)
	}
}

// TestPurgeSuperseded_CarriesReachToSurvivor covers the personalisation signal
// across the collapse. Reach counts are the "this memory was used N times" hint
// that nudges load-bearing conversations up the ranking — and they are keyed by
// the session id that was surfaced at recall time.
//
// Under the duplicate bug that was whichever copy existed then, so collapsing the
// copies strands every recall the user ever made: the surviving conversation
// shows zero reach and looks brand new, precisely because it was used a lot.
func TestPurgeSuperseded_CarriesReachToSurvivor(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	grow := []string{"why we dropped batching", "because of tail latency"}
	seedSession(t, dataDB, "old-copy", "claude", "a@b.c", grow[:1])
	seedSession(t, dataDB, "survivor", "claude", "a@b.c", grow)

	// The user reached the earlier copy twice before it was collapsed.
	ts := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	if err := InsertRecallEdges(dataDB, []RecallEdge{
		{ID: "e1", TS: ts, Kind: "recall", Query: "batching", Target: "old-copy"},
		{ID: "e2", TS: ts.Add(time.Hour), Kind: "drill", Query: "batching", Target: "old-copy"},
	}); err != nil {
		t.Fatalf("InsertRecallEdges: %v", err)
	}
	dataDB.Close() //nolint:errcheck

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close() //nolint:errcheck
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := PopulateIndex(indexDB, dir); err != nil {
		t.Fatalf("PopulateIndex: %v", err)
	}

	var reach int
	if err := indexDB.QueryRow(
		`SELECT COALESCE(sum(reach_count), 0) FROM session_reach WHERE target_session_id = 'survivor'`,
	).Scan(&reach); err != nil {
		t.Fatalf("read reach: %v", err)
	}
	if reach != 2 {
		t.Errorf("survivor reach = %d, want 2 — recalls made against the collapsed copy must follow the conversation, or a well-used memory reads as never used", reach)
	}
}

// seedIndexOnlySession writes a session that exists only in the index — turns_ft
// plus session_facets, never data.db. That is exactly what `rekal sync` produces
// for a teammate's conversation arriving over the wire.
func seedIndexOnlySession(t *testing.T, d *sql.DB, id, email string, body []string) {
	t.Helper()
	for i, c := range body {
		if _, err := d.Exec(
			`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
			 VALUES ($1, $2, $3, 'human', $4, '')`,
			fmt.Sprintf("%s:%d", id, i), id, i, c,
		); err != nil {
			t.Fatalf("insert turn_ft %s:%d: %v", id, i, err)
		}
	}
	if _, err := d.Exec(
		`INSERT INTO session_facets (session_id, user_email, turn_count, captured_at, actor_type)
		 VALUES ($1, $2, $3, $4, 'human')`,
		id, email, len(body), "2026-07-11T00:00:00Z",
	); err != nil {
		t.Fatalf("insert session_facets %s: %v", id, err)
	}
}

// TestPurgeSuperseded_CollapsesSyncedSessions covers the sync path. A teammate
// whose store predates append-on-recapture ships one conversation as a chain of
// growing prefixes; every link arrives as its own session and lands in the index
// only. PopulateIndex's own pass runs before any of them exist, so without a
// second pass after import the reader keeps all of them and the recall digest
// fills with copies of one conversation.
//
// The discriminators must still hold: these sessions are absent from
// data.db.sessions, so the key has to come from session_facets or two people
// who opened with the same prompt would be merged into one.
func TestPurgeSuperseded_CollapsesSyncedSessions(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}
	dataDB.Close() //nolint:errcheck

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close() //nolint:errcheck
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}

	// Arrivals: one conversation as a prefix chain, plus a different author
	// who opened with the identical prompt and must not be folded in.
	grow := []string{"implement the plan", "next step", "and done"}
	seedIndexOnlySession(t, indexDB, "remote1", "teammate@x.dev", grow[:1])
	seedIndexOnlySession(t, indexDB, "remote2", "teammate@x.dev", grow[:2])
	seedIndexOnlySession(t, indexDB, "remote3", "teammate@x.dev", grow)
	seedIndexOnlySession(t, indexDB, "someone", "other@x.dev", grow[:1])

	if err := PurgeSupersededSessionsWithData(indexDB, dir); err != nil {
		t.Fatalf("PurgeSupersededSessionsWithData: %v", err)
	}

	live := map[string]bool{}
	rows, err := indexDB.Query(`SELECT DISTINCT session_id FROM turns_ft`)
	if err != nil {
		t.Fatalf("query turns_ft: %v", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		live[id] = true
	}
	rows.Close() //nolint:errcheck

	for _, gone := range []string{"remote1", "remote2"} {
		if live[gone] {
			t.Errorf("%s is a strict prefix of remote3 and should have been collapsed", gone)
		}
	}
	for _, kept := range []string{"remote3", "someone"} {
		if !live[kept] {
			t.Errorf("%s must survive: longest capture, or a different author's conversation", kept)
		}
	}
}

// TestPurgeSuperseded_CollapsesExactDuplicates covers the copy prefix detection
// is blind to: two sessions with byte-identical transcripts and the *same*
// length. The prefix pass compares each session only against strictly longer
// ones, so equal-length copies never meet and every one of them survives.
//
// They arise wherever the same conversation reaches the index by two routes —
// a wire import landing beside the local capture it came from, a store synced
// from two peers that both carry it, a cross-repo local import of a repo that
// already pushed. Recall then spends several of its seed budget on one
// conversation, which is the whole defect the supersession pass exists to stop.
func TestPurgeSuperseded_CollapsesExactDuplicates(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	body := []string{"why we dropped the queue", "because redelivery was silent", "so we made it explicit"}
	for _, id := range []string{"copy-a", "copy-b", "copy-c"} {
		seedSession(t, dataDB, id, "claude", "a@b.c", body)
	}
	// Same length, same opening turn, genuinely different conversation.
	seedSession(t, dataDB, "distinct", "claude", "a@b.c",
		[]string{"why we dropped the queue", "a different reason entirely", "and a different end"})
	dataDB.Close() //nolint:errcheck

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close() //nolint:errcheck
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := PopulateIndex(indexDB, dir); err != nil {
		t.Fatalf("PopulateIndex: %v", err)
	}

	live := map[string]bool{}
	rows, err := indexDB.Query(`SELECT DISTINCT session_id FROM turns_ft`)
	if err != nil {
		t.Fatalf("query turns_ft: %v", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		live[id] = true
	}
	rows.Close() //nolint:errcheck

	copies := 0
	for _, id := range []string{"copy-a", "copy-b", "copy-c"} {
		if live[id] {
			copies++
		}
	}
	if copies != 1 {
		t.Errorf("%d of the 3 identical copies survived, want exactly 1 — recall spends a seed on each", copies)
	}
	if !live["distinct"] {
		t.Error("distinct must survive: same length and same opening turn, but not the same conversation")
	}
}

// TestPurgeSuperseded_FlattensAcrossBothPasses pins the seam between the two
// collapse rules. An exact copy is mapped onto its twin, and that twin is then
// mapped onto a longer capture that extends it — so the raw result chains.
//
// Every consumer reads the map as old → *final* survivor: the index deletes all
// of the keys, so a value that is itself a key would re-point a subagent's
// parent or a reach count at a session that is no longer in the index. That is
// the bug the prefix-only pass could never produce, because it always jumped
// straight to the longest member of a chain.
func TestPurgeSuperseded_FlattensAcrossBothPasses(t *testing.T) {
	t.Parallel()
	dir, _ := openTempDB(t)

	dataDB, err := OpenData(dir)
	if err != nil {
		t.Fatalf("OpenData: %v", err)
	}
	if err := InitDataSchema(dataDB); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	grow := []string{"the shared opening", "the shared second", "only the long one has this"}
	// a-copy and b-copy are byte-identical; long extends both.
	seedSession(t, dataDB, "a-copy", "claude", "a@b.c", grow[:2])
	seedSession(t, dataDB, "b-copy", "claude", "a@b.c", grow[:2])
	seedSession(t, dataDB, "long", "claude", "a@b.c", grow)

	// A subagent pinned to the copy that loses both passes.
	if err := InsertSession(dataDB, "kid", "b-copy", "h-kid", "agent", "",
		"a@b.c", "main", "2026-07-11T00:00:00Z", "claude"); err != nil {
		t.Fatalf("InsertSession kid: %v", err)
	}
	if err := InsertTurn(dataDB, "kid:0", "kid", 0, "assistant", "subagent output", ""); err != nil {
		t.Fatalf("InsertTurn kid: %v", err)
	}
	dataDB.Close() //nolint:errcheck

	indexDB, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	defer indexDB.Close() //nolint:errcheck
	if err := InitIndexSchema(indexDB); err != nil {
		t.Fatalf("InitIndexSchema: %v", err)
	}
	if err := PopulateIndex(indexDB, dir); err != nil {
		t.Fatalf("PopulateIndex: %v", err)
	}

	// No survivor may itself be superseded.
	rows, err := indexDB.Query(`SELECT old_session_id, survivor_session_id FROM session_supersedes`)
	if err != nil {
		t.Fatalf("query session_supersedes: %v", err)
	}
	mapped := map[string]string{}
	for rows.Next() {
		var old, survivor string
		if err := rows.Scan(&old, &survivor); err != nil {
			t.Fatalf("scan: %v", err)
		}
		mapped[old] = survivor
	}
	rows.Close() //nolint:errcheck

	for old, survivor := range mapped {
		if _, chained := mapped[survivor]; chained {
			t.Errorf("%s → %s, but %s is itself superseded — the map must point at the final survivor",
				old, survivor, survivor)
		}
		if survivor != "long" {
			t.Errorf("%s → %s, want long", old, survivor)
		}
	}
	if len(mapped) != 2 {
		t.Errorf("mapped %d sessions, want 2 (a-copy and b-copy)", len(mapped))
	}

	// And the subagent follows all the way to the end of the chain.
	var parent string
	if err := indexDB.QueryRow(
		`SELECT COALESCE(parent_session_id, '') FROM session_facets WHERE session_id = 'kid'`,
	).Scan(&parent); err != nil {
		t.Fatalf("read kid's parent: %v", err)
	}
	if parent != "long" {
		t.Errorf("subagent parent = %q, want long — a half-resolved chain strands it on a deleted session", parent)
	}
}
