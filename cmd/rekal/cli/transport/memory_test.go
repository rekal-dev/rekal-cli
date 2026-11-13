package transport

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
)

func TestMemoryPrivateExportAndGuards(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(db.StoreDir(root), 0700); err != nil {
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
	r := memory.Revision{Schema: 1, CheckpointID: "cp", GitSHA: "abc", ObservedAt: "2026-11-14T00:00:00Z", Source: memory.Source{Name: "private", Adapter: "markdown", Owner: "alice", Profile: "personal"}, Path: "note.md", Content: "private content"}
	r.ID = r.Digest()
	if err := db.InsertCheckpoint(d, "cp", "abc", "main", "alice", r.ObservedAt, "human", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertMemoryRevisions(d, []memory.Revision{r}); err != nil {
		t.Fatal(err)
	}
	body, _, _, err := encodeCheckpointFrames(d, codec.NewBody(), codec.NewDict(), []db.CheckpointRow{{ID: "cp", GitSHA: "abc", Email: "alice", Ts: r.ObservedAt}})
	if err != nil {
		t.Fatal(err)
	}
	records, err := readMemoryFrames(body)
	if err != nil || len(records) != 0 {
		t.Fatalf("private revision exported: %+v %v", records, err)
	}
	enc, err := codec.NewEncoder()
	if err != nil {
		t.Fatal(err)
	}
	defer enc.Close()
	makeBody := func(m codec.BatchMember) []byte {
		t.Helper()
		frame, err := enc.EncodeBatch([]codec.BatchMember{m})
		if err != nil {
			t.Fatal(err)
		}
		return codec.AppendFrame(codec.NewBody(), frame)
	}
	payload, _ := json.Marshal(r)
	if _, err := readMemoryFrames(makeBody(codec.BatchMember{Type: codec.FrameMemory, Payload: payload})); err == nil {
		t.Fatal("private wire accepted")
	}
	if _, err := readMemoryFrames(makeBody(codec.BatchMember{Type: 0x42, Payload: []byte("future")})); err == nil {
		t.Fatal("unknown member dropped")
	}
	if _, err := readMemoryFrames(append(codec.NewBody(), 1)); err == nil {
		t.Fatal("partial envelope ignored")
	}
	r.Source.Shared = true
	r.ID = r.Digest()
	payload, _ = json.Marshal(r)
	wire := makeBody(codec.BatchMember{Type: codec.FrameMemory, Payload: payload})
	if err := guardMemoryReexport(d, wire); err == nil {
		t.Fatal("unimported revision would be lost by re-export")
	}
	if err := validateMemoryAssociations(wire, codec.NewDict(), []memory.Revision{r}); err == nil {
		t.Fatal("invented checkpoint association accepted")
	}
	dict := codec.NewDict()
	cf := &codec.CheckpointFrame{CheckpointRef: dict.LookupOrAdd(codec.NSSessions, "cp"), EmailRef: dict.LookupOrAdd(codec.NSEmails, "alice"), GitSHA: "wrong", Timestamp: time.Now()}
	frame, err := enc.EncodeBatch([]codec.BatchMember{{Type: codec.FrameMemory, Payload: payload}, codec.CheckpointMember(cf)})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryAssociations(codec.AppendFrame(codec.NewBody(), frame), dict, []memory.Revision{r}); err == nil {
		t.Fatal("wrong code commit accepted")
	}
}
