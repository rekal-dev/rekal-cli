package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
)

func TestMemoryCheckpointWireAndRetrieval(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	t.Chdir(repo)
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME"} {
		t.Setenv(key, home)
	}
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-b", "main")
	git("config", "user.email", "synthetic@example.invalid")
	git("config", "user.name", "Synthetic")
	git("commit", "--allow-empty", "-m", "synthetic baseline")
	if err := os.MkdirAll(db.StoreDir(repo), 0700); err != nil {
		t.Fatal(err)
	}
	d, err := db.OpenData(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitDataSchema(d); err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	notes := filepath.Join(home, "notes")
	if err := os.MkdirAll(notes, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(notes, "retry.md")
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("Use idempotency keys for payment retries.\nSource: 01J00000000000000000000000:t42:bytes0-30\n")
	cfg := Config{MemorySources: []memory.Config{{Name: "project", Adapter: "markdown", Path: notes, Share: true}}}
	if err := writeConfig(repo, cfg); err != nil {
		t.Fatal(err)
	}
	capture := func() {
		t.Helper()
		var log bytes.Buffer
		if err := doCheckpointNow(repo, &log); err != nil {
			t.Fatal(err)
		}
	}
	count := func(query string) int {
		t.Helper()
		d, err := db.OpenData(repo)
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()
		var n int
		if err := d.QueryRow(query).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	capture()
	capture()
	if n := count("SELECT count(*) FROM memory_revisions"); n != 1 {
		t.Fatalf("unchanged capture: %d", n)
	}
	if n := count("SELECT count(*) FROM sessions"); n != 0 {
		t.Fatalf("notes masqueraded as sessions: %d", n)
	}
	if n := count("SELECT count(*) FROM checkpoints"); n != 1 {
		t.Fatalf("memory-only checkpoints: %d", n)
	}
	if err := transport.EnsureOrphanBranch(repo); err != nil {
		t.Fatal(err)
	}
	body, dict, _, err := transport.ExportNewFrames(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) <= 9 {
		t.Fatal("memory-only checkpoint not exported")
	}
	if _, err := transport.CommitWireFormat(repo, body, dict); err != nil {
		t.Fatal(err)
	}
	revisionOneBody := append([]byte(nil), body...)
	sourceTip := git("rev-parse", "rekal/synthetic@example.invalid")

	// Fresh teammate index receives shared notes through the existing team path.
	index, err := db.OpenIndexAt(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	if err := db.InitIndexSchema(index); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.ImportBranchToIndex(repo, index, sourceTip); err != nil {
		t.Fatal(err)
	}
	hits, err := search.SearchMemory(index, search.Filters{Query: "payment retries"}, nil)
	if err != nil || len(hits) != 1 {
		t.Fatalf("team search: %+v %v", hits, err)
	}
	if len(hits[0].Anchors) != 1 {
		t.Fatalf("citation bridge: %+v", hits[0])
	}
	var archived string
	if err := index.QueryRow(hits[0].SQL).Scan(&archived); err != nil {
		t.Fatal(err)
	}
	var original memory.Revision
	if err := json.Unmarshal([]byte(archived), &original); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(original.Content, "idempotency") {
		t.Fatal("historical drill lost content")
	}
	if got, _ := formatDigest(&search.Output{Memories: hits}); !strings.Contains(got, "MEMORY") || strings.Contains(got, "SILENCE") {
		t.Fatalf("digest %s", got)
	}
	// Mutation is a new revision; deletion removes current membership, not history.
	write("Payment retries use a stable request key.")
	capture()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	capture()
	if n := count("SELECT count(*) FROM memory_revisions"); n != 3 {
		t.Fatalf("history count: %d", n)
	}
	body, dict, _, err = transport.ExportNewFrames(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(body, revisionOneBody) {
		t.Fatal("incremental export changed existing bytes")
	}
	if _, err := transport.CommitWireFormat(repo, body, dict); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.ImportBranchToIndex(repo, index, "rekal/synthetic@example.invalid"); err != nil {
		t.Fatal(err)
	}
	hits, err = search.SearchMemory(index, search.Filters{Query: "payment"}, nil)
	if err != nil || len(hits) != 0 {
		t.Fatalf("removed note remains current: %+v %v", hits, err)
	}
	var history int
	if err := index.QueryRow("SELECT count(*) FROM memory_archive").Scan(&history); err != nil || history != 3 {
		t.Fatalf("team history %d %v", history, err)
	}
	// Full export and self import must retain all three revisions idempotently.
	rebuilt, rebuiltDict, _, err := transport.ExportAllFrames(repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.CommitWireFormat(repo, rebuilt, rebuiltDict); err != nil {
		t.Fatal(err)
	}
	freshRoot := t.TempDir()
	if err := os.MkdirAll(db.StoreDir(freshRoot), 0700); err != nil {
		t.Fatal(err)
	}
	fresh, err := db.OpenData(freshRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	if err := db.InitDataSchema(fresh); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := transport.ImportBranch(repo, fresh, "rekal/synthetic@example.invalid"); err != nil {
			t.Fatal(err)
		}
	}
	if err := fresh.QueryRow("SELECT count(*) FROM memory_revisions").Scan(&history); err != nil || history != 3 {
		t.Fatalf("self rebuild: %d %v", history, err)
	}
	// An incomplete source scan cannot turn the source into a deletion.
	if err := os.Remove(notes); err != nil {
		t.Fatal(err)
	}
	var warning bytes.Buffer
	if err := doCheckpointNow(repo, &warning); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warning.String(), "memory capture skipped") {
		t.Fatal("missing source must be reported")
	}
	if n := count("SELECT count(*) FROM memory_revisions"); n != 3 {
		t.Fatalf("failed scan changed history: %d", n)
	}
	// Unknown memory schemas must fail visibly before a team index can be altered.
	enc, err := codec.NewEncoder()
	if err != nil {
		t.Fatal(err)
	}
	defer enc.Close()
	bad, err := enc.EncodeBatch([]codec.BatchMember{{Type: codec.FrameMemory, Payload: []byte(`{"schema":99}`)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.CommitWireFormat(repo, codec.AppendFrame(codec.NewBody(), bad), dict); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.ImportBranchToIndex(repo, index, "rekal/synthetic@example.invalid"); err == nil {
		t.Fatal("unknown schema was dropped")
	}
}

func TestMemoryConfigStaysLocal(t *testing.T) {
	global := Config{MemorySources: []memory.Config{{Name: "personal", Path: "/personal", Share: true}}}
	if len(mergeConfig(global, Config{}).MemorySources) != 0 {
		t.Fatal("global setting authorized capture in every repo")
	}
	local := Config{MemorySources: []memory.Config{{Name: "project", Path: "notes"}}}
	if local.empty() {
		t.Fatal("writing unrelated settings would erase memory intent")
	}
	got := mergeConfig(global, local)
	if len(got.MemorySources) != 1 || got.MemorySources[0].Name != "project" || got.MemorySources[0].Share {
		t.Fatal("local source policy changed")
	}
}
