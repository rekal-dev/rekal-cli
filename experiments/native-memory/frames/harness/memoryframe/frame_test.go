package memoryframe

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
)

func fixture() []byte {
	return []byte(`{"schema":1,"text":"记忆: use idempotency keys","modified":"2026-04-02T10:00:00Z","observed":"2026-11-07T12:00:00Z"}`)
}
func seed(t *testing.T) ([]byte, *codec.Dict) {
	t.Helper()
	b, d, e := Seed()
	if e != nil {
		t.Fatal(e)
	}
	return b, d
}
func appendMemory(t *testing.T, b []byte, d *codec.Dict, batch bool, payload []byte) []byte {
	t.Helper()
	out, e := AppendCheckpoint(b, d, "checkpoint-1", strings.Repeat("1", 40), [][]byte{payload}, batch)
	if e != nil {
		t.Fatal(e)
	}
	return out
}

func TestStandaloneAndBatchedMemoryRoundTrip(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(map[bool]string{true: "batch", false: "standalone"}[batch], func(t *testing.T) {
			b, d := seed(t)
			before := append([]byte{}, b...)
			out := appendMemory(t, b, d, batch, fixture())
			if !bytes.HasPrefix(out, before) || !bytes.Equal(b, before) {
				t.Fatal("existing body mutated")
			}
			records, e := MemoryRecords(out)
			if e != nil || len(records) != 1 || !bytes.Equal(records[0], fixture()) {
				t.Fatalf("round trip: %v %s", e, records)
			}
			counts, e := LegacyCounts(out)
			if e != nil || counts["sessions"] != 1 || counts["checkpoints"] != 1 || counts["unknown_skipped"] != 1 {
				t.Fatalf("legacy dispatch: %v %v", e, counts)
			}
		})
	}
}

func TestMemoryOnlyCheckpointHasNoFabricatedSession(t *testing.T) {
	out := appendMemory(t, codec.NewBody(), codec.NewDict(), true, fixture())
	members, e := Members(out)
	if e != nil {
		t.Fatal(e)
	}
	if len(members) != 2 || members[0].Type != SnapshotType {
		t.Fatal("member layout")
	}
	envelope, e := DecodeMemoryPayload(members[0].Payload)
	if e != nil {
		t.Fatal(e)
	}
	if envelope.CheckpointID != "checkpoint-1" {
		t.Fatal("missing explicit checkpoint association")
	}
	cp, e := codec.DecodeCheckpointPayload(members[1].Payload)
	if e != nil {
		t.Fatal(e)
	}
	if len(cp.SessionRefs) != 0 {
		t.Fatal("memory was misrepresented as a session")
	}
}

func TestMismatchedMemoryCommitRejected(t *testing.T) {
	b, d := seed(t)
	_, err := AppendCheckpoint(b, d, "checkpoint-1", strings.Repeat("1", 40), [][]byte{[]byte(`{"schema":1,"observed_at_commit":"2222222222222222222222222222222222222222"}`)}, true)
	if err == nil {
		t.Fatal("unrelated checkpoint accepted")
	}
}

func TestOlderBodyWithoutMemoryRemainsReadable(t *testing.T) {
	b, _ := seed(t)
	records, e := MemoryRecords(b)
	if e != nil || len(records) != 0 {
		t.Fatalf("old body: %v", e)
	}
}

func TestLaterSessionAppendRetainsMemory(t *testing.T) {
	b, d := seed(t)
	b = appendMemory(t, b, d, true, fixture())
	enc, e := codec.NewEncoder()
	if e != nil {
		t.Fatal(e)
	}
	defer enc.Close()
	frame, e := enc.EncodeSessionFrame(&codec.SessionFrame{SessionRef: 1, Turns: []codec.TurnRecord{{Text: "later session"}}})
	if e != nil {
		t.Fatal(e)
	}
	out := codec.AppendFrame(append([]byte{}, b...), frame)
	records, e := MemoryRecords(out)
	if e != nil || len(records) != 1 {
		t.Fatalf("memory lost: %v", e)
	}
	counts, e := LegacyCounts(out)
	if e != nil || counts["sessions"] != 2 {
		t.Fatalf("later session: %v %v", e, counts)
	}
}

func TestLegacyReexportModelDropsUnknownMemory(t *testing.T) {
	b, d := seed(t)
	b = appendMemory(t, b, d, true, fixture())
	out, e := ReexportModel(b, false)
	if e != nil {
		t.Fatal(e)
	}
	records, e := MemoryRecords(out)
	if e != nil || len(records) != 0 {
		t.Fatalf("expected demonstrated loss: %v", e)
	}
	counts, e := LegacyCounts(out)
	if e != nil || counts["sessions"] != 1 || counts["checkpoints"] != 1 {
		t.Fatalf("known records: %v %v", e, counts)
	}
}

func TestAwareReexportModelRetainsMemory(t *testing.T) {
	b, d := seed(t)
	b = appendMemory(t, b, d, true, fixture())
	out, e := ReexportModel(b, true)
	if e != nil {
		t.Fatal(e)
	}
	records, e := MemoryRecords(out)
	if e != nil || len(records) != 1 || !bytes.Equal(records[0], fixture()) {
		t.Fatalf("memory lost: %v", e)
	}
}

func TestFutureSchemaRefusedByNewMemoryReaderAndRepair(t *testing.T) {
	enc, e := codec.NewEncoder()
	if e != nil {
		t.Fatal(e)
	}
	defer enc.Close()
	frame, e := enc.EncodeMemberFrame(codec.BatchMember{Type: SnapshotType, Payload: []byte(`{"schema":2}`)})
	if e != nil {
		t.Fatal(e)
	}
	b := codec.AppendFrame(codec.NewBody(), frame)
	if _, e := MemoryRecords(b); e == nil {
		t.Fatal("unknown schema accepted")
	}
	if _, e := ReexportModel(b, true); e == nil {
		t.Fatal("unknown schema repaired destructively")
	}
}

func TestFutureFrameRefusedByAwareRepair(t *testing.T) {
	enc, e := codec.NewEncoder()
	if e != nil {
		t.Fatal(e)
	}
	defer enc.Close()
	frame, e := enc.EncodeMemberFrame(codec.BatchMember{Type: 0x08, Payload: []byte("future")})
	if e != nil {
		t.Fatal(e)
	}
	if _, e := ReexportModel(codec.AppendFrame(codec.NewBody(), frame), true); e == nil {
		t.Fatal("unknown frame silently dropped")
	}
}

func TestTrailingPartialAndTruncatedFramesRejected(t *testing.T) {
	b, d := seed(t)
	b = appendMemory(t, b, d, true, fixture())
	for _, bad := range [][]byte{b[:len(b)-1], append(append([]byte{}, b...), 0x07), append(append([]byte{}, b...), 0x07, 0, 0, 0, 0)} {
		if _, e := Members(bad); e == nil {
			t.Fatal("partial frame accepted")
		}
	}
}

func TestLargeMemoryPayloadBeyondAdvisoryLengthRoundTrips(t *testing.T) {
	payload, e := json.Marshal(map[string]any{"schema": 1, "text": strings.Repeat("long note with paths and citations\n", 5000)})
	if e != nil {
		t.Fatal(e)
	}
	b, d := seed(t)
	b = appendMemory(t, b, d, false, payload)
	out, e := MemoryRecords(b)
	if e != nil || len(out) != 1 || !bytes.Equal(out[0], payload) {
		t.Fatalf("large frame: %v", e)
	}
}
