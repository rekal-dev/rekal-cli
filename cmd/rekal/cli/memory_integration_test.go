package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
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
	cfg := Config{MemorySources: []memory.Config{{Name: "project", Adapter: "markdown", Path: notes, Scope: "repo"}}}
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
	global := Config{MemorySources: []memory.Config{{Name: "personal", Path: "/personal", Scope: "repo"}}}
	if len(mergeConfig(global, Config{}).MemorySources) != 0 {
		t.Fatal("global setting authorized capture in every repo")
	}
	local := Config{MemorySources: []memory.Config{{Name: "project", Path: "notes"}}}
	if local.empty() {
		t.Fatal("writing unrelated settings would erase memory intent")
	}
	got := mergeConfig(global, local)
	if len(got.MemorySources) != 1 || got.MemorySources[0].Name != "project" || got.MemorySources[0].Scope == "repo" {
		t.Fatal("local source policy changed")
	}
}

func TestDefaultNativeMemoryLifecycle(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	t.Chdir(repo)
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME"} {
		t.Setenv(key, home)
	}
	t.Setenv("CLAUDE_CODE_PROJECT_DIR_NAME", "")
	for _, args := range [][]string{{"init", "-b", "main"}, {"config", "user.email", "synthetic@example.invalid"}, {"config", "user.name", "Synthetic"}, {"commit", "--allow-empty", "-m", "baseline"}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %s %v", args, out, err)
		}
	}
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
	initialIndex, err := db.OpenIndex(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InitIndexSchema(initialIndex); err != nil {
		t.Fatal(err)
	}
	_ = initialIndex.Close()
	checkpoint := func() {
		t.Helper()
		var output bytes.Buffer
		if err := doCheckpointNow(repo, &output); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output.String(), "memory capture skipped") {
			t.Fatalf("unexpected warning: %s", output.String())
		}
	}
	count := func() int {
		t.Helper()
		d, err := db.OpenData(repo)
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()
		var n int
		if err := d.QueryRow("SELECT count(*) FROM memory_revisions").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// No memory directory is a quiet no-op, with no configuration required.
	checkpoint()
	if count() != 0 {
		t.Fatal("missing source captured")
	}
	notes := filepath.Join(session.FindSessionDir(repo), "memory")
	if err := os.MkdirAll(notes, 0700); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(notes, "MEMORY.md")
	if err := os.WriteFile(note, []byte("Use idempotency for payment retries."), 0600); err != nil {
		t.Fatal(err)
	}
	checkpoint()
	checkpoint()
	if count() != 1 {
		t.Fatal("default capture missing or duplicated")
	}
	index, err := db.OpenIndexReadOnly(repo)
	if err != nil {
		t.Fatal(err)
	}
	source := nativeMemorySources(Config{})[0]
	identity := memory.Source{Name: source.Name, Adapter: source.Adapter, Profile: source.Profile, Owner: "synthetic@example.invalid"}
	hits, err := search.SearchMemory(index, search.Filters{Query: "payment retries"}, []string{identity.ID()})
	if err != nil || len(hits) != 1 || !hits[0].Source.Repository() {
		t.Fatalf("default recall: %+v %v", hits, err)
	}
	hits, err = search.SearchMemory(index, search.Filters{Query: "payment retries"}, nil)
	if err != nil || len(hits) != 1 {
		t.Fatalf("project note unavailable to other source: %+v %v", hits, err)
	}
	_ = index.Close()
	// The opt-out survives serialization and dominates explicit source settings.
	enabled := false
	cfg := Config{Memory: &enabled, MemorySources: []memory.Config{{Name: "explicit", Adapter: "markdown", Path: notes, Scope: "repo"}}}
	if cfg.empty() {
		t.Fatal("opt-out would be erased")
	}
	if err := writeConfig(repo, cfg); err != nil {
		t.Fatal(err)
	}
	cfg, err = readConfig(repo)
	if err != nil || len(nativeMemorySources(cfg)) != 0 {
		t.Fatalf("opt-out lost: %+v %v", cfg, err)
	}
	if err := os.WriteFile(note, []byte("Changed while capture disabled."), 0600); err != nil {
		t.Fatal(err)
	}
	checkpoint()
	if count() != 1 {
		t.Fatal("disabled capture changed history")
	}
	if err := writeConfig(repo, Config{}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(notes); err != nil {
		t.Fatal(err)
	}
	checkpoint()
	if count() != 1 {
		t.Fatal("missing directory created tombstone")
	}
	// A global source must not opt every repository into sharing.
	global := Config{MemorySources: []memory.Config{{Name: "global", Scope: "repo"}}}
	sources := nativeMemorySources(mergeConfig(global, Config{}))
	if len(sources) != 1 || sources[0].Scope != "repo" || sources[0].Name != "claude-project" {
		t.Fatalf("unsafe defaults: %+v", sources)
	}
}

func TestLocalMemoryNeverEntersLedger(t *testing.T) {
	repo, notes := t.TempDir(), t.TempDir()
	t.Chdir(repo)
	t.Setenv("HOME", t.TempDir())
	for _, args := range [][]string{{"init", "-b", "main"}, {"config", "user.email", "synthetic@example.invalid"}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %s %v", out, err)
		}
	}
	if err := os.MkdirAll(db.StoreDir(repo), 0700); err != nil {
		t.Fatal(err)
	}
	d, err := db.OpenData(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := db.InitDataSchema(d); err != nil {
		t.Fatal(err)
	}
	cfg := Config{MemorySources: []memory.Config{{Name: "personal", Adapter: "markdown", Path: notes, Scope: "local"}}}
	if err := writeConfig(repo, cfg); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(notes, "preference.md")
	if err := os.WriteFile(path, []byte("personalmarker prefer concise payment explanations"), 0600); err != nil {
		t.Fatal(err)
	}
	changes, err := captureMemoryChanges(d, repo, "synthetic@example.invalid", "cp", "sha", time.Now())
	if err != nil || len(changes) != 0 {
		t.Fatalf("local source entered capture: %+v %v", changes, err)
	}
	index, err := db.OpenIndex(repo)
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	if err := db.InitIndexSchema(index); err != nil {
		t.Fatal(err)
	}
	read := func() []search.MemoryHit {
		t.Helper()
		if err := refreshLocalMemory(index, repo, cfg); err != nil {
			t.Fatal(err)
		}
		hits, err := search.SearchMemory(index, search.Filters{Query: "personalmarker"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return hits
	}
	hits := read()
	if len(hits) != 1 || hits[0].Source.Scope != "local" || hits[0].CheckpointID != "" {
		t.Fatalf("local search: %+v", hits)
	}
	id := hits[0].Ref
	if again := read(); len(again) != 1 || again[0].Ref != id {
		t.Fatal("unchanged snapshot changed reference")
	}
	var payload string
	if err := index.QueryRow("SELECT payload FROM local_memory_documents").Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var revision memory.Revision
	if err := json.Unmarshal([]byte(payload), &revision); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertMemoryRevisions(d, []memory.Revision{revision}); err == nil {
		t.Fatal("local snapshot accepted into ledger")
	}
	if err := db.ImportMemoryIndex(index, []memory.Revision{revision}); err == nil {
		t.Fatal("local snapshot accepted from team wire")
	}
	for _, table := range []string{"memory_archive", "memory_documents"} {
		var n int
		if err := index.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("local history in %s: %d %v", table, n, err)
		}
	}
	// A disposable-index rebuild returns the same source reference.
	if _, err := index.Exec("DROP TABLE local_memory_documents"); err != nil {
		t.Fatal(err)
	}
	if rebuilt := read(); len(rebuilt) != 1 || rebuilt[0].Ref != id {
		t.Fatal("rebuild changed local reference")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if len(read()) != 0 {
		t.Fatal("deleted local note remained searchable")
	}
	if err := os.WriteFile(path, []byte("personalmarker new note"), 0600); err != nil {
		t.Fatal(err)
	}
	if len(read()) != 1 {
		t.Fatal("local note did not return")
	}
	if err := os.RemoveAll(notes); err != nil {
		t.Fatal(err)
	}
	if err := refreshLocalMemory(index, repo, cfg); err == nil {
		t.Fatal("missing explicit directory not reported")
	}
	remaining, err := search.SearchMemory(index, search.Filters{Query: "personalmarker"}, nil)
	if err != nil || len(remaining) != 0 {
		t.Fatal("failed scan served stale local content")
	}
	var n int
	if err := d.QueryRow("SELECT count(*) FROM memory_revisions").Scan(&n); err != nil || n != 0 {
		t.Fatalf("local ledger not empty: %d %v", n, err)
	}
}
