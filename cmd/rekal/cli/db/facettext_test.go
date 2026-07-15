package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPopulateFacetText builds facet documents from the index DB's own
// tables (tool paths + command prefixes + steering text), covering the full
// (all-rows) and incremental (per-session) paths plus the length cap.
func TestPopulateFacetText(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := OpenIndex(dir)
	if err != nil {
		t.Fatalf("OpenIndex: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	if err := InitIndexSchema(d); err != nil {
		t.Fatal(err)
	}

	mustExec := func(q string, args ...interface{}) {
		t.Helper()
		if _, err := d.Exec(q, args...); err != nil {
			t.Fatalf("exec %s: %v", q, err)
		}
	}

	mustExec(`INSERT INTO session_facets (session_id, actor_type, captured_at) VALUES
		('s1', 'human', '2026-01-01T10:00:00Z'),
		('s2', 'human', '2026-01-01T11:00:00Z')`)
	mustExec(`INSERT INTO tool_calls_index (id, session_id, call_order, tool, path, cmd_prefix) VALUES
		('c1', 's1', 0, 'Bash', NULL, 'kubectl apply'),
		('c2', 's1', 1, 'Edit', 'deploy/chart.yaml', NULL),
		('c3', 's1', 2, 'Edit', 'deploy/chart.yaml', NULL)`) // duplicate path — deduped
	mustExec(`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts) VALUES
		('t1', 's1', 0, 'human', 'ordinary turn text stays out of the facet doc', ''),
		('t2', 's1', 1, 'human_steering', 'use rolling restarts not recreate', ''),
		('t3', 's2', 0, 'human', 'unrelated', '')`)

	if err := PopulateFacetText(d); err != nil {
		t.Fatalf("PopulateFacetText: %v", err)
	}

	var facet string
	if err := d.QueryRow(`SELECT facet_text FROM session_facets WHERE session_id = 's1'`).Scan(&facet); err != nil {
		t.Fatalf("read facet_text: %v", err)
	}
	for _, want := range []string{"deploy/chart.yaml", "kubectl apply", "use rolling restarts not recreate"} {
		if !strings.Contains(facet, want) {
			t.Fatalf("facet_text missing %q: %q", want, facet)
		}
	}
	if strings.Contains(facet, "ordinary turn text") {
		t.Fatalf("non-steering turn content leaked into facet doc: %q", facet)
	}
	if strings.Count(facet, "deploy/chart.yaml") != 1 {
		t.Fatalf("duplicate tool path not deduped: %q", facet)
	}

	// Incremental path touches only the named session.
	mustExec(`INSERT INTO tool_calls_index (id, session_id, call_order, tool, path, cmd_prefix)
		VALUES ('c4', 's2', 0, 'Bash', NULL, 'terraform plan')`)
	if err := PopulateFacetText(d, "s2"); err != nil {
		t.Fatalf("PopulateFacetText(s2): %v", err)
	}
	var facet2 string
	if err := d.QueryRow(`SELECT facet_text FROM session_facets WHERE session_id = 's2'`).Scan(&facet2); err != nil {
		t.Fatalf("read s2 facet_text: %v", err)
	}
	if !strings.Contains(facet2, "terraform plan") {
		t.Fatalf("incremental facet_text missing cmd_prefix: %q", facet2)
	}

	// Cap: a giant path list truncates to the bound instead of growing the doc.
	mustExec(`INSERT INTO session_facets (session_id, actor_type, captured_at) VALUES ('s3', 'human', '2026-01-01T12:00:00Z')`)
	mustExec(`INSERT INTO tool_calls_index (id, session_id, call_order, tool, path, cmd_prefix)
		SELECT 'big-' || range::VARCHAR, 's3', range, 'Edit', 'src/' || repeat('x', 60) || range::VARCHAR || '.go', NULL
		FROM range(200)`)
	if err := PopulateFacetText(d, "s3"); err != nil {
		t.Fatalf("PopulateFacetText(s3): %v", err)
	}
	var n int
	if err := d.QueryRow(`SELECT length(facet_text) FROM session_facets WHERE session_id = 's3'`).Scan(&n); err != nil {
		t.Fatalf("read s3 facet length: %v", err)
	}
	if n > 4000 {
		t.Fatalf("facet_text exceeds cap: %d", n)
	}
}
