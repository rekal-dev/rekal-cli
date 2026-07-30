//go:build integration

package integration

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestSync_TeammateSessionKeepsCommitAnchorAndMessage covers two things that
// were broken together.
//
// The anchor: ImportBranchToIndex captured each imported session's checkpoint
// and wrote it back with an UPDATE, but session_facets.session_id is a PRIMARY
// KEY and DuckDB raises a spurious duplicate-key error updating a row on an
// indexed table. Every one of those writes failed and the error was swallowed
// as "the session may not have been imported", so no synced session ever
// carried a git_sha — `--commit` filtering and everything keyed on the commit
// silently saw only this machine's own sessions.
//
// The signal: with the anchor present, the commit's full message — subject and
// body — is resolved from the reader's own clone and folded into the facet
// document. It is never stored in data.db and never crosses the wire, because
// every clone already has it; the wire ships the SHA it always shipped.
func TestSync_TeammateSessionKeepsCommitAnchorAndMessage(t *testing.T) {
	origin := NewTestEnv(t)
	origin.Init()

	bare := t.TempDir()
	bare, _ = filepath.EvalSymlinks(bare)
	initBareRemote(t, bare)
	gitRun(t, origin.RepoDir, "remote", "add", "origin", bare)
	if err := os.WriteFile(filepath.Join(origin.RepoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, origin.RepoDir, "initial")
	pushMain(t, origin.RepoDir)

	// A teammate commits with reasoning in the *body* — the part a subject
	// line never carries, and the part worth recalling.
	peer := newPeer(t, bare, "author@rekal.dev", "author")
	sessionDir := sessionDirFor(t, peer.dir)
	if err := os.WriteFile(filepath.Join(sessionDir, "s.jsonl"),
		[]byte(transcript("looking at the retry path", "changed it")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(peer.dir, "retry.go"), []byte("package main // retry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, peer.dir, "add", "-A")
	gitRun(t, peer.dir, "commit", "-q", "-m",
		"fix: stop the retry stampede\n\nA fixed delay makes every client retry in lockstep, so the\ndownstream is re-flooded the moment it recovers. Exponential\nbackoff with jitter instead. The word quokka appears only here.")
	pushMain(t, peer.dir)
	if _, stderr, err := peer.env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("peer checkpoint: %v\n%s", err, stderr)
	}
	if _, stderr, err := peer.env.RunCLI("push"); err != nil {
		t.Fatalf("peer push: %v\n%s", err, stderr)
	}

	// A reader on a different identity syncs the team.
	reader := newPeer(t, bare, "reader@rekal.dev", "reader")
	pullMain(t, reader.dir)
	if _, stderr, err := reader.env.RunCLI("sync"); err != nil {
		t.Fatalf("reader sync: %v\n%s", err, stderr)
	}

	idx, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "index.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader index: %v", err)
	}
	defer idx.Close() //nolint:errcheck

	var anchored int
	if err := idx.QueryRow(
		`SELECT count(*) FROM session_facets WHERE git_sha IS NOT NULL AND git_sha <> ''`,
	).Scan(&anchored); err != nil {
		t.Fatalf("count anchored: %v", err)
	}
	if anchored == 0 {
		t.Error("no synced session carries a git_sha — the checkpoint anchor was dropped on import, so --commit and everything keyed on the commit is blind to teammate sessions")
	}

	var withMsg int
	if err := idx.QueryRow(
		`SELECT count(*) FROM session_facets WHERE commit_message LIKE '%quokka%'`,
	).Scan(&withMsg); err != nil {
		t.Fatalf("count commit_message: %v", err)
	}
	if withMsg == 0 {
		t.Error("the commit body never reached session_facets — the reasoning in the message is exactly what should be recallable")
	}

	// And it must be in the facet document, which is what recall searches.
	var inFacet int
	if err := idx.QueryRow(
		`SELECT count(*) FROM session_facets WHERE facet_text LIKE '%quokka%'`,
	).Scan(&inFacet); err != nil {
		t.Fatalf("count facet_text: %v", err)
	}
	if inFacet == 0 {
		t.Error("the commit message is stored but not folded into facet_text, so it contributes nothing to ranking")
	}

	// The message must never enter data.db: every clone already has it, and
	// SOUL.md's thin-wire rule is to strip what git already carries.
	data, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "data.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader data.db: %v", err)
	}
	defer data.Close() //nolint:errcheck
	var cols int
	if err := data.QueryRow(
		`SELECT count(*) FROM information_schema.columns
		 WHERE table_name = 'checkpoints' AND column_name LIKE '%message%'`,
	).Scan(&cols); err != nil {
		t.Fatalf("inspect data.db schema: %v", err)
	}
	if cols != 0 {
		t.Error("a commit message column exists in data.db — it would ride the wire, duplicating bytes every clone already has")
	}
}
