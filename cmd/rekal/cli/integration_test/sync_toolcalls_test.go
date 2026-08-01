//go:build integration

package integration

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSync_TeammateToolCallsReachTheIndex pins that a teammate's tool calls
// survive the trip.
//
// The wire has always carried them — tool, path, and command prefix are part of
// the session frame and cost bytes on every push. The --self path stores them.
// The team path decoded them and dropped them, so tool_calls_index held only
// this machine's own sessions, and everything built from that table was blind to
// the team: PopulateFacetText reads it for each session's file paths and command
// prefixes, and file_cooccurrence derives related-file signal from it.
//
// A teammate who edited src/auth/token.go must be findable by that path, not
// only by whatever they happened to say about it in prose.
func TestSync_TeammateToolCallsReachTheIndex(t *testing.T) {
	bare := newSharedRemote(t)

	peer := newPeer(t, bare, "tools@rekal.dev", "tools")
	sessionDir := sessionDirFor(t, peer.dir)
	if err := os.WriteFile(filepath.Join(sessionDir, "s.jsonl"),
		[]byte(transcriptWithToolCalls()), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.WriteFile(filepath.Join(peer.dir, "tools.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, peer.dir, "peer work")
	pushMain(t, peer.dir)
	if _, stderr, err := peer.env.RunCLI("checkpoint"); err != nil {
		t.Fatalf("peer checkpoint: %v\n%s", err, stderr)
	}
	if _, stderr, err := peer.env.RunCLI("push"); err != nil {
		t.Fatalf("peer push: %v\n%s", err, stderr)
	}

	// The peer's own store must have them, or the test proves nothing about the
	// reader — it would only be re-measuring an empty transcript.
	peerData := openStore(t, peer.dir)
	defer peerData.Close() //nolint:errcheck
	if got := countRows(t, peerData, `SELECT count(*) FROM tool_calls WHERE path LIKE '%token.go%'`); got == 0 {
		t.Fatal("peer's own data.db has no tool call for token.go — the fixture never captured one")
	}

	reader := newPeer(t, bare, "reader4@rekal.dev", "reader4")
	if _, stderr, err := reader.env.RunCLI("sync"); err != nil {
		t.Fatalf("reader sync: %v\n%s", err, stderr)
	}

	idx, err := sql.Open("duckdb", filepath.Join(reader.dir, ".rekal", "index.db")+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open reader index: %v", err)
	}
	defer idx.Close() //nolint:errcheck

	if got := countRows(t, idx, `SELECT count(*) FROM tool_calls_index WHERE path LIKE '%token.go%'`); got == 0 {
		t.Error("reader's tool_calls_index has no row for the teammate's token.go edit — the frame carried it and the import dropped it")
	}
	if got := countRows(t, idx, `SELECT count(*) FROM tool_calls_index WHERE cmd_prefix LIKE '%rotate-keys%'`); got == 0 {
		t.Error("reader's tool_calls_index has no row for the teammate's command — command prefixes ride the wire too")
	}
	if got := countRows(t, idx, `SELECT count(*) FROM session_facets WHERE tool_call_count > 0`); got == 0 {
		t.Error("every synced session reports tool_call_count 0 — the count was hardcoded, not derived from the frame")
	}
	// The whole reason the table matters: the facet document is built from it,
	// so a path a teammate touched has to be searchable text on their session.
	if got := countRows(t, idx, `SELECT count(*) FROM session_facets WHERE facet_text LIKE '%token.go%'`); got == 0 {
		t.Error("teammate's facet document does not mention token.go — the facet layer stays blind to what the team actually touched")
	}
}

// transcriptWithToolCalls builds a Claude transcript whose assistant turns carry
// tool_use blocks, so capture records real tool calls rather than prose only.
func transcriptWithToolCalls() string {
	var b strings.Builder
	b.WriteString(`{"type":"summary","sessionId":"tools-001","totalCost":0.01,"totalDuration":10}` + "\n")
	b.WriteString(turnLine("the refresh token expiry is wrong", "2026-02-25T10:00:00Z"))
	b.WriteString(`{"type":"assistant","parentMessageId":"","isSidechain":false,"message":{"role":"assistant","content":[` +
		`{"type":"text","text":"Fixing the expiry."},` +
		`{"type":"tool_use","id":"t1","name":"Edit","input":{"file_path":"src/auth/token.go","old_string":"a","new_string":"b"}},` +
		`{"type":"tool_use","id":"t2","name":"Bash","input":{"command":"./scripts/rotate-keys.sh --dry-run"}}` +
		`]},"timestamp":"2026-02-25T10:01:00Z","gitBranch":"main"}` + "\n")
	return b.String()
}
