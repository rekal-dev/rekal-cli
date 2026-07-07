package codec

import (
	"testing"
	"time"
)

// sampleSession builds a v1 session (small counts, no metadata).
func sampleSession(ref uint64, text string) *SessionFrame {
	return &SessionFrame{
		SessionRef: ref,
		CapturedAt: time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC),
		EmailRef:   1,
		ActorType:  ActorHuman,
		Turns:      []TurnRecord{{Role: RoleHuman, Text: text}},
		ToolCalls:  []ToolCallRecord{{Tool: ToolEdit, PathFlag: PathDictRef, PathRef: 2}},
	}
}

func sampleCheckpoint(ref uint64) *CheckpointFrame {
	return &CheckpointFrame{
		CheckpointRef: ref,
		GitSHA:        "aaa111bbb222ccc333ddd444eee555fff666aaa1",
		Timestamp:     time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC),
		ActorType:     ActorHuman,
		SessionRefs:   []uint64{0, 1},
		Files:         []FileTouchedRecord{{PathRef: 2, ChangeType: ChangeModified}},
	}
}

// TestBatch_Roundtrip: a batch of mixed member types survives encode → append
// → scan → DecodeBatch → per-member parse, byte-for-byte on the fields.
func TestBatch_Roundtrip(t *testing.T) {
	enc, dec := newEncDec(t)

	// A v2 session (metadata present forces v2) alongside a v1 session.
	v2 := sampleSession(5, "reviewing the auth change")
	v2.TeamName = "core"
	v2.SpawnDepth = 2
	members := []BatchMember{
		SessionMember(sampleSession(0, "first session text")),
		SessionMember(v2),
		CheckpointMember(sampleCheckpoint(9)),
		MetaMember(&MetaFrame{FormatVersion: 1, CheckpointSHA: "e7b3a91f2c4d5e6f7890abcdef1234567890abcd", Timestamp: time.Now(), NSessions: 2}),
	}

	batch, err := enc.EncodeBatch(members)
	if err != nil {
		t.Fatalf("EncodeBatch: %v", err)
	}
	body := AppendFrame(NewBody(), batch)

	frames, err := ScanFrames(body)
	if err != nil {
		t.Fatalf("ScanFrames: %v", err)
	}
	if len(frames) != 1 || frames[0].Type != FrameBatch {
		t.Fatalf("expected 1 batch frame, got %d frames (type[0]=%x)", len(frames), frames[0].Type)
	}

	got, err := dec.DecodeBatch(ExtractFramePayload(body, frames[0]))
	if err != nil {
		t.Fatalf("DecodeBatch: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("members: got %d, want 4", len(got))
	}
	if got[0].Type != FrameSession || got[1].Type != FrameSessionV2 ||
		got[2].Type != FrameCheckpoint || got[3].Type != FrameMeta {
		t.Fatalf("member types: %x %x %x %x", got[0].Type, got[1].Type, got[2].Type, got[3].Type)
	}

	s0, err := DecodeSessionPayload(got[0].Payload)
	if err != nil || s0.Turns[0].Text != "first session text" {
		t.Fatalf("member 0 session: %v %+v", err, s0)
	}
	s1, err := DecodeSessionPayload(got[1].Payload)
	if err != nil || s1.TeamName != "core" || s1.SpawnDepth != 2 {
		t.Fatalf("member 1 v2 session metadata lost: %v %+v", err, s1)
	}
	cp, err := DecodeCheckpointPayload(got[2].Payload)
	if err != nil || cp.GitSHA != sampleCheckpoint(9).GitSHA || len(cp.SessionRefs) != 2 {
		t.Fatalf("member 2 checkpoint: %v %+v", err, cp)
	}
	mf, err := DecodeMetaPayload(got[3].Payload)
	if err != nil || mf.NSessions != 2 {
		t.Fatalf("member 3 meta: %v %+v", err, mf)
	}
}

// TestBatch_OldReaderSkips is the core backwards-compat guarantee: a binary
// that predates FrameBatch scans a body containing a batch frame followed by a
// standalone frame, skips the batch it doesn't understand (by envelope
// length), and still finds and decodes the trailing frame it does understand.
func TestBatch_OldReaderSkips(t *testing.T) {
	enc, dec := newEncDec(t)

	batch, err := enc.EncodeBatch([]BatchMember{
		SessionMember(sampleSession(0, "batched session a")),
		SessionMember(sampleSession(1, "batched session b")),
	})
	if err != nil {
		t.Fatalf("EncodeBatch: %v", err)
	}
	trailing, err := enc.EncodeSessionFrame(sampleSession(2, "standalone trailing session"))
	if err != nil {
		t.Fatalf("EncodeSessionFrame: %v", err)
	}
	body := AppendFrame(AppendFrame(NewBody(), batch), trailing)

	// An old reader uses ScanFrames (version-agnostic) and a switch that only
	// knows the pre-batch types. Simulate: collect sessions it can decode.
	frames, err := ScanFrames(body)
	if err != nil {
		t.Fatalf("ScanFrames: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("scan found %d frames, want 2 (batch + trailing)", len(frames))
	}
	var decoded []string
	for _, fs := range frames {
		switch fs.Type {
		case FrameSession, FrameSessionV2:
			sf, err := dec.DecodeSessionFrame(ExtractFramePayload(body, fs))
			if err != nil {
				t.Fatalf("decode standalone: %v", err)
			}
			decoded = append(decoded, sf.Turns[0].Text)
		case FrameBatch:
			// Old reader has no case for this; it falls through and is skipped.
			continue
		default:
			continue
		}
	}
	if len(decoded) != 1 || decoded[0] != "standalone trailing session" {
		t.Fatalf("old reader should see only the trailing frame, got %v", decoded)
	}
}

// TestBatch_MixedBody: a new reader correctly decodes a body that interleaves
// standalone frames and batch frames (the state during gradual rollout).
func TestBatch_MixedBody(t *testing.T) {
	enc, dec := newEncDec(t)

	body := NewBody()
	f1, _ := enc.EncodeSessionFrame(sampleSession(0, "standalone one"))
	body = AppendFrame(body, f1)
	b1, _ := enc.EncodeBatch([]BatchMember{
		SessionMember(sampleSession(1, "batched two")),
		SessionMember(sampleSession(2, "batched three")),
	})
	body = AppendFrame(body, b1)
	f2, _ := enc.EncodeSessionFrame(sampleSession(3, "standalone four"))
	body = AppendFrame(body, f2)

	frames, err := ScanFrames(body)
	if err != nil {
		t.Fatalf("ScanFrames: %v", err)
	}
	var texts []string
	for _, fs := range frames {
		switch fs.Type {
		case FrameSession, FrameSessionV2:
			sf, err := dec.DecodeSessionFrame(ExtractFramePayload(body, fs))
			if err != nil {
				t.Fatal(err)
			}
			texts = append(texts, sf.Turns[0].Text)
		case FrameBatch:
			members, err := dec.DecodeBatch(ExtractFramePayload(body, fs))
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range members {
				sf, err := DecodeSessionPayload(m.Payload)
				if err != nil {
					t.Fatal(err)
				}
				texts = append(texts, sf.Turns[0].Text)
			}
		}
	}
	want := []string{"standalone one", "batched two", "batched three", "standalone four"}
	if len(texts) != len(want) {
		t.Fatalf("got %d texts, want %d: %v", len(texts), len(want), texts)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Errorf("text %d: got %q, want %q", i, texts[i], want[i])
		}
	}
}

// TestBatch_CompressesBetterThanPerFrame locks in the reason the format
// exists: on a redundant multi-session push, one batch is smaller than the sum
// of independently-compressed frames.
func TestBatch_CompressesBetterThanPerFrame(t *testing.T) {
	enc, _ := newEncDec(t)

	var members []BatchMember
	perFrame := 0
	for i := 0; i < 10; i++ {
		sf := sampleSession(uint64(i), "fix the authentication bug in the login handler and refresh the token")
		members = append(members, SessionMember(sf))
		f, err := enc.EncodeSessionFrame(sf)
		if err != nil {
			t.Fatal(err)
		}
		perFrame += len(f)
	}
	batch, err := enc.EncodeBatch(members)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("per-frame total: %d bytes, single batch: %d bytes (%.2fx)", perFrame, len(batch), float64(perFrame)/float64(len(batch)))
	if len(batch) >= perFrame {
		t.Fatalf("batch (%d) should be smaller than per-frame sum (%d)", len(batch), perFrame)
	}
}

// TestBatch_V2MemberInside: a >255-turn session (a v2 member) round-trips
// inside a batch — the batch path must not reintroduce the count-overflow bug.
func TestBatch_V2MemberInside(t *testing.T) {
	enc, dec := newEncDec(t)

	big := &SessionFrame{
		SessionRef: 1,
		CapturedAt: time.Now(),
		ActorType:  ActorHuman,
	}
	for i := 0; i < 300; i++ {
		big.Turns = append(big.Turns, TurnRecord{Role: RoleHuman, Text: "turn"})
	}
	m := SessionMember(big)
	if m.Type != FrameSessionV2 {
		t.Fatalf("300-turn session should be a v2 member, got %x", m.Type)
	}
	batch, err := enc.EncodeBatch([]BatchMember{m})
	if err != nil {
		t.Fatal(err)
	}
	members, err := dec.DecodeBatch(batch[frameEnvSize:])
	if err != nil {
		t.Fatal(err)
	}
	sf, err := DecodeSessionPayload(members[0].Payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(sf.Turns) != 300 {
		t.Fatalf("turns in batched v2 member: got %d, want 300", len(sf.Turns))
	}
}

// TestDecodeBatch_TruncatedNeverPanics feeds every truncation of a valid batch
// frame's compressed bytes to DecodeBatch — corrupt input must error, never
// panic (a batch can arrive corrupt from a hostile teammate push).
func TestDecodeBatch_TruncatedNeverPanics(t *testing.T) {
	enc, dec := newEncDec(t)
	batch, err := enc.EncodeBatch([]BatchMember{
		SessionMember(sampleSession(0, "a")),
		CheckpointMember(sampleCheckpoint(1)),
	})
	if err != nil {
		t.Fatal(err)
	}
	compressed := batch[frameEnvSize:]
	for i := 0; i <= len(compressed); i++ {
		_ = callParse(t, func() error {
			_, perr := dec.DecodeBatch(compressed[:i])
			return perr
		})
	}
}

// TestDecodeBatch_CorruptInnerLengthRejected: a batch whose decompressed inner
// stream declares a member longer than the buffer must be rejected, not slice
// out of range.
func TestDecodeBatch_CorruptInnerLengthRejected(t *testing.T) {
	enc, dec := newEncDec(t)
	// Inner: type byte + a huge varint length + no payload.
	inner := append([]byte{byte(FrameSession)}, appendUvarint(nil, 1<<40)...)
	compressed := enc.zw.EncodeAll(inner, nil)
	err := callParse(t, func() error {
		_, perr := dec.DecodeBatch(compressed)
		return perr
	})
	if err == nil {
		t.Fatal("expected error for oversized member length")
	}
}

// TestEncodeMemberFrame_FallbackReadableAsStandalone: when a batch would
// overflow the envelope's length field, appendBatch (transport) falls back to
// EncodeMemberFrame — each member written as its own standalone frame. Those
// fallback frames must be indistinguishable from normally encoded standalone
// frames: same envelope types, decodable by the standard session/checkpoint
// decoders any reader already has.
func TestEncodeMemberFrame_FallbackReadableAsStandalone(t *testing.T) {
	enc, dec := newEncDec(t)

	sf := sampleSession(3, "fallback path session")
	cf := sampleCheckpoint(4)

	body := NewBody()
	for _, m := range []BatchMember{SessionMember(sf), CheckpointMember(cf)} {
		frame, err := enc.EncodeMemberFrame(m)
		if err != nil {
			t.Fatalf("EncodeMemberFrame: %v", err)
		}
		body = AppendFrame(body, frame)
	}

	frames, err := ScanFrames(body)
	if err != nil {
		t.Fatalf("ScanFrames: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("expected 2 standalone frames, got %d", len(frames))
	}

	gotSF, err := dec.DecodeSessionFrame(ExtractFramePayload(body, frames[0]))
	if err != nil {
		t.Fatalf("DecodeSessionFrame on fallback frame: %v", err)
	}
	if gotSF.SessionRef != sf.SessionRef || len(gotSF.Turns) != 1 || gotSF.Turns[0].Text != sf.Turns[0].Text {
		t.Fatalf("session round-trip mismatch: %+v", gotSF)
	}

	gotCF, err := dec.DecodeCheckpointFrame(ExtractFramePayload(body, frames[1]))
	if err != nil {
		t.Fatalf("DecodeCheckpointFrame on fallback frame: %v", err)
	}
	if gotCF.CheckpointRef != cf.CheckpointRef || gotCF.GitSHA != cf.GitSHA {
		t.Fatalf("checkpoint round-trip mismatch: %+v", gotCF)
	}
}
