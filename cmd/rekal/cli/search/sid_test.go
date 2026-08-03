package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttachShortIDs(t *testing.T) {
	t.Parallel()

	gitRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gitRoot, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	indexDB := openTempIndexDB(t)
	// Lex order: a < b < c → s1, s2, s3
	seedFacet(t, indexDB, "a-ulid", "a@example.com", "human")
	seedFacet(t, indexDB, "c-ulid", "c@example.com", "human")
	seedFacet(t, indexDB, "b-ulid", "b@example.com", "agent")

	results := []Result{
		{SessionID: "c-ulid", Related: []Related{{SessionID: "a-ulid"}}},
		{
			SessionID: "b-ulid",
			Children:  []Result{{SessionID: "a-ulid"}},
		},
	}
	attachShortIDs(indexDB, gitRoot, results)

	if results[0].Sid != "s3" {
		t.Errorf("c-ulid sid = %q, want s3", results[0].Sid)
	}
	if results[0].Related[0].Sid != "s1" {
		t.Errorf("related a-ulid sid = %q, want s1", results[0].Related[0].Sid)
	}
	if results[1].Sid != "s2" {
		t.Errorf("b-ulid sid = %q, want s2", results[1].Sid)
	}
	if results[1].Children[0].Sid != "s1" {
		t.Errorf("child a-ulid sid = %q, want s1", results[1].Children[0].Sid)
	}
}
