package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

func TestSessionDiscoveryCache_PersistsPositiveAndNegativeMatches(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := db.OpenData(root)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := db.InitDataSchema(d); err != nil {
		t.Fatal(err)
	}
	positive, negative := filepath.Join(root, "positive.jsonl"), filepath.Join(root, "negative.jsonl")
	for path, body := range map[string]string{positive: "/repo", negative: "/skip"} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reads := 0
	read := func(path string) ([]string, error) {
		reads++
		b, err := os.ReadFile(path)
		return []string{string(b)}, err
	}
	c, err := loadSessionDiscoveryCache(d)
	if err != nil {
		t.Fatal(err)
	}
	if !c.matcher("copilot")(positive, "/repo", read) || c.matcher("copilot")(negative, "/repo", read) {
		t.Fatal("cold matches incorrect")
	}
	if reads != 2 {
		t.Fatalf("cold reads = %d", reads)
	}
	if err := c.flush(d); err != nil {
		t.Fatal(err)
	}

	// A new command loads all metadata once, then opens neither unchanged file,
	// including the file belonging to another repo.
	c, err = loadSessionDiscoveryCache(d)
	if err != nil {
		t.Fatal(err)
	}
	if !c.matcher("copilot")(positive, "/repo", read) || c.matcher("copilot")(negative, "/repo", read) {
		t.Fatal("warm matches incorrect")
	}
	if reads != 2 {
		t.Fatalf("warm discovery reopened unchanged transcripts: %d reads", reads)
	}

	// Existing-file writes do not update the parent directory. Equal-length
	// replacement still invalidates through the file's own nanosecond mtime.
	dirBefore, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(negative, []byte("/repo"), 0o644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(time.Second)
	if err := os.Chtimes(negative, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	dirAfter, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if !dirBefore.ModTime().Equal(dirAfter.ModTime()) {
		t.Fatal("fixture unexpectedly changed the parent directory mtime")
	}
	if !c.matcher("copilot")(negative, "/repo", read) || reads != 3 {
		t.Fatal("changed file was not rediscovered")
	}
}

func TestSessionDiscoveryCache_DoesNotCacheReadFailures(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "partial.jsonl")
	if err := os.WriteFile(path, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &sessionDiscoveryCache{records: map[string]db.DiscoveryRecord{}, pending: map[string]db.DiscoveryRecord{}}
	reads := 0
	read := func(string) ([]string, error) { reads++; return nil, io.ErrUnexpectedEOF }
	for i := 0; i < 2; i++ {
		if c.matcher("codex")(path, "/repo", read) {
			t.Fatal("partial read matched")
		}
	}
	if reads != 2 || len(c.pending) != 0 {
		t.Fatal("a failed read became a cached negative result")
	}
}
