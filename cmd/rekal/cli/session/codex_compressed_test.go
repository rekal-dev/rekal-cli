package session

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestCodexAdapter_CompressedRollout(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	plain := filepath.Join(root, "rollout.jsonl")
	compressed := plain + ".zst"
	w, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close() //nolint:errcheck
	encoded := w.EncodeAll([]byte(codexFixtureJSONL), nil)
	for path, body := range map[string][]byte{plain: []byte(codexFixtureJSONL), compressed: encoded} {
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := &CodexAdapter{}
	p, err := a.Parse(SessionRef{Path: plain})
	if err != nil {
		t.Fatal(err)
	}
	c, err := a.Parse(SessionRef{Path: compressed})
	if err != nil {
		t.Fatal(err)
	}
	c.CapturedAt = p.CapturedAt // ingestion wall clock, not transcript data
	if !reflect.DeepEqual(p, c) {
		t.Fatalf("compressed capture differs: plain=%+v compressed=%+v", p, c)
	}
	if !codexSessionMatchesRepo(compressed, "/tmp/repo") || codexSessionMatchesRepo(compressed, "/tmp/other") {
		t.Fatal("compressed repo discovery incorrect")
	}
	if ContentHash(a, SessionRef{Path: plain}, []byte(codexFixtureJSONL)) != ContentHash(a, SessionRef{Path: compressed}, encoded) {
		t.Fatal("compression changed the cross-import dedup hash")
	}
	if err := os.WriteFile(compressed, []byte("not zstd"), 0o644); err != nil {
		t.Fatal(err)
	}
	if payload, err := a.Parse(SessionRef{Path: compressed}); err == nil || payload != nil {
		t.Fatal("corrupt compressed rollout returned a successful partial capture")
	}
}

func TestCodexAdapter_DiscoversCompressedActiveAndArchived(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir()) // env changes cannot run in parallel
	w, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close() //nolint:errcheck
	for _, dir := range codexSessionDirs() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "rollout.jsonl.zst"), w.EncodeAll([]byte(codexFixtureJSONL), nil), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := &CodexAdapter{}
	refs, err := a.Discover("/tmp/repo")
	if err != nil || len(refs) != 2 {
		t.Fatalf("Discover = %v, %v; want both compressed rollouts", refs, err)
	}
	all, err := a.DiscoverAll()
	if err != nil || len(all) != 2 {
		t.Fatalf("DiscoverAll = %v, %v; want both compressed rollouts", all, err)
	}
}
