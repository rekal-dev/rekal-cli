package codec

import (
	"testing"
	"time"
)

func TestNewBody_Header(t *testing.T) {
	body := NewBody()
	if len(body) != bodyHdrSize {
		t.Errorf("body header: got %d bytes, want %d", len(body), bodyHdrSize)
	}
	if string(body[0:7]) != "RKLBODY" {
		t.Errorf("magic: got %q", body[0:7])
	}
	if body[7] != bodyVersion {
		t.Errorf("version: got %d, want %d", body[7], bodyVersion)
	}
}

func TestBody_AppendAndScan(t *testing.T) {
	enc, err := NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	body := NewBody()

	// Append a session frame.
	sf := &SessionFrame{
		SessionRef: 0,
		CapturedAt: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		EmailRef:   0,
		ActorType:  ActorHuman,
		Turns: []TurnRecord{
			{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "hello"},
		},
	}
	frame1 := enc.EncodeSessionFrame(sf)
	body = AppendFrame(body, frame1)

	// Append a checkpoint frame.
	cf := &CheckpointFrame{
		GitSHA:      "aaa111bbb222ccc333ddd444eee555fff666aaa1",
		BranchRef:   0,
		EmailRef:    0,
		Timestamp:   time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		ActorType:   ActorHuman,
		SessionRefs: []uint64{0},
		Files: []FileTouchedRecord{
			{PathRef: 0, ChangeType: ChangeModified},
		},
	}
	frame2 := enc.EncodeCheckpointFrame(cf)
	body = AppendFrame(body, frame2)

	// Append a meta frame.
	mf := &MetaFrame{
		FormatVersion: 0x01,
		EmailRef:      0,
		CheckpointSHA: "aaa111bbb222ccc333ddd444eee555fff666aaa1",
		Timestamp:     time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		NSessions:     1,
		NCheckpoints:  1,
		NFrames:       3,
		NDictEntries:  5,
	}
	frame3 := enc.EncodeMetaFrame(mf)
	body = AppendFrame(body, frame3)

	// Scan.
	frames, err := ScanFrames(body)
	if err != nil {
		t.Fatalf("ScanFrames: %v", err)
	}
	if len(frames) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(frames))
	}

	if frames[0].Type != FrameSession {
		t.Errorf("frame 0: got type %x, want %x", frames[0].Type, FrameSession)
	}
	if frames[1].Type != FrameCheckpoint {
		t.Errorf("frame 1: got type %x, want %x", frames[1].Type, FrameCheckpoint)
	}
	if frames[2].Type != FrameMeta {
		t.Errorf("frame 2: got type %x, want %x", frames[2].Type, FrameMeta)
	}

	// Verify we can decode each frame.
	dec, err := NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	payload0 := ExtractFramePayload(body, frames[0])
	decodedSF, err := dec.DecodeSessionFrame(payload0)
	if err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if len(decodedSF.Turns) != 1 || decodedSF.Turns[0].Text != "hello" {
		t.Errorf("session turn text: %v", decodedSF.Turns)
	}

	payload1 := ExtractFramePayload(body, frames[1])
	decodedCF, err := dec.DecodeCheckpointFrame(payload1)
	if err != nil {
		t.Fatalf("decode checkpoint: %v", err)
	}
	if decodedCF.GitSHA != cf.GitSHA {
		t.Errorf("checkpoint git_sha: got %q", decodedCF.GitSHA)
	}

	payload2 := ExtractFramePayload(body, frames[2])
	decodedMF, err := dec.DecodeMetaFrame(payload2)
	if err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if decodedMF.NSessions != 1 {
		t.Errorf("meta n_sessions: got %d", decodedMF.NSessions)
	}
}

func TestBody_IncrementalAppend(t *testing.T) {
	enc, err := NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	body := NewBody()
	originalLen := len(body)

	// First checkpoint: 1 session + 1 checkpoint + 1 meta = 3 frames.
	sf1 := enc.EncodeSessionFrame(&SessionFrame{
		SessionRef: 0,
		CapturedAt: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		EmailRef:   0,
		ActorType:  ActorHuman,
		Turns:      []TurnRecord{{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "first session"}},
	})
	body = AppendFrame(body, sf1)
	body = AppendFrame(body, enc.EncodeCheckpointFrame(&CheckpointFrame{
		GitSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", BranchRef: 0, EmailRef: 0,
		Timestamp: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC), ActorType: ActorHuman,
		SessionRefs: []uint64{0},
	}))
	body = AppendFrame(body, enc.EncodeMetaFrame(&MetaFrame{
		FormatVersion: 1, EmailRef: 0,
		CheckpointSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Timestamp:     time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		NSessions:     1, NCheckpoints: 1, NFrames: 3, NDictEntries: 3,
	}))

	afterFirst := len(body)
	t.Logf("After first checkpoint: %d bytes (header: %d, data: %d)", afterFirst, originalLen, afterFirst-originalLen)

	// Second checkpoint: append more frames. Verify prefix is unchanged.
	prefixSnapshot := make([]byte, afterFirst)
	copy(prefixSnapshot, body)

	sf2 := enc.EncodeSessionFrame(&SessionFrame{
		SessionRef: 1,
		CapturedAt: time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC),
		EmailRef:   0,
		ActorType:  ActorHuman,
		Turns:      []TurnRecord{{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "second session"}},
	})
	body = AppendFrame(body, sf2)
	body = AppendFrame(body, enc.EncodeCheckpointFrame(&CheckpointFrame{
		GitSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", BranchRef: 0, EmailRef: 0,
		Timestamp: time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC), ActorType: ActorHuman,
		SessionRefs: []uint64{1},
	}))
	body = AppendFrame(body, enc.EncodeMetaFrame(&MetaFrame{
		FormatVersion: 1, EmailRef: 0,
		CheckpointSHA: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Timestamp:     time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC),
		NSessions:     2, NCheckpoints: 2, NFrames: 6, NDictEntries: 4,
	}))

	afterSecond := len(body)
	t.Logf("After second checkpoint: %d bytes (delta: %d)", afterSecond, afterSecond-afterFirst)

	// Verify prefix is byte-identical (append-only property).
	for i := 0; i < afterFirst; i++ {
		if body[i] != prefixSnapshot[i] {
			t.Fatalf("prefix changed at byte %d: was %x, now %x", i, prefixSnapshot[i], body[i])
		}
	}

	// Scan should find 6 frames.
	frames, err := ScanFrames(body)
	if err != nil {
		t.Fatalf("ScanFrames: %v", err)
	}
	if len(frames) != 6 {
		t.Fatalf("expected 6 frames, got %d", len(frames))
	}

	// Last meta frame should have updated counts.
	dec, err := NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	lastMeta := frames[5]
	payload := ExtractFramePayload(body, lastMeta)
	mf, err := dec.DecodeMetaFrame(payload)
	if err != nil {
		t.Fatalf("decode last meta: %v", err)
	}
	if mf.NSessions != 2 {
		t.Errorf("last meta n_sessions: got %d, want 2", mf.NSessions)
	}
	if mf.CheckpointSHA != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Errorf("last meta checkpoint_sha: got %q", mf.CheckpointSHA)
	}
}

func TestScanFrames_EmptyBody(t *testing.T) {
	body := NewBody()
	frames, err := ScanFrames(body)
	if err != nil {
		t.Fatalf("ScanFrames empty: %v", err)
	}
	if len(frames) != 0 {
		t.Errorf("expected 0 frames, got %d", len(frames))
	}
}

func TestScanFrames_BadMagic(t *testing.T) {
	body := []byte("BADMAGIC\x00")
	_, err := ScanFrames(body)
	if err == nil {
		t.Error("expected error for bad magic")
	}
}

func BenchmarkBodyAppendAndScan(b *testing.B) {
	enc, err := NewEncoder()
	if err != nil {
		b.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	sf := &SessionFrame{
		SessionRef: 0,
		CapturedAt: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		EmailRef:   0,
		ActorType:  ActorHuman,
		Turns: []TurnRecord{
			{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "fix the bug"},
			{Role: RoleAssistant, TsDelta: 30, BranchRef: 0, Text: "Done."},
		},
	}
	frame := enc.EncodeSessionFrame(sf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := NewBody()
		for j := 0; j < 100; j++ {
			body = AppendFrame(body, frame)
		}
		_, _ = ScanFrames(body)
	}
}
