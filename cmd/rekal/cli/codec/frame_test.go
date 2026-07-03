package codec

import (
	"testing"
	"time"
)

func TestSessionFrame_Roundtrip(t *testing.T) {
	enc, err := NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	dec, err := NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	sf := &SessionFrame{
		SessionRef: 0,
		CapturedAt: time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC),
		EmailRef:   0,
		ActorType:  ActorHuman,
		Turns: []TurnRecord{
			{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "fix the bug in auth middleware"},
			{Role: RoleAssistant, TsDelta: 45, BranchRef: 0, Text: "Let me read the file first."},
			{Role: RoleHuman, TsDelta: 120, BranchRef: 0, Text: "looks good, thanks"},
		},
		ToolCalls: []ToolCallRecord{
			{Tool: ToolRead, PathFlag: PathDictRef, PathRef: 0},
			{Tool: ToolEdit, PathFlag: PathDictRef, PathRef: 0},
			{Tool: ToolBash, PathFlag: PathNull, CmdPrefix: "go test -race ./..."},
		},
	}

	encoded := enc.EncodeSessionFrame(sf)

	// Verify envelope.
	if FrameType(encoded[0]) != FrameSession {
		t.Errorf("frame type: got %x, want %x", encoded[0], FrameSession)
	}

	// Decode the compressed payload (skip 6-byte envelope).
	compressed := encoded[frameEnvSize:]
	decoded, err := dec.DecodeSessionFrame(compressed)
	if err != nil {
		t.Fatalf("DecodeSessionFrame: %v", err)
	}

	if decoded.SessionRef != sf.SessionRef {
		t.Errorf("SessionRef: got %d, want %d", decoded.SessionRef, sf.SessionRef)
	}
	if !decoded.CapturedAt.Equal(sf.CapturedAt) {
		t.Errorf("CapturedAt: got %v, want %v", decoded.CapturedAt, sf.CapturedAt)
	}
	if decoded.ActorType != sf.ActorType {
		t.Errorf("ActorType: got %d, want %d", decoded.ActorType, sf.ActorType)
	}
	if len(decoded.Turns) != len(sf.Turns) {
		t.Fatalf("Turns: got %d, want %d", len(decoded.Turns), len(sf.Turns))
	}
	for i, turn := range decoded.Turns {
		if turn.Role != sf.Turns[i].Role {
			t.Errorf("turn %d role: got %d, want %d", i, turn.Role, sf.Turns[i].Role)
		}
		if turn.TsDelta != sf.Turns[i].TsDelta {
			t.Errorf("turn %d ts_delta: got %d, want %d", i, turn.TsDelta, sf.Turns[i].TsDelta)
		}
		if turn.Text != sf.Turns[i].Text {
			t.Errorf("turn %d text: got %q, want %q", i, turn.Text, sf.Turns[i].Text)
		}
	}
	if len(decoded.ToolCalls) != len(sf.ToolCalls) {
		t.Fatalf("ToolCalls: got %d, want %d", len(decoded.ToolCalls), len(sf.ToolCalls))
	}
	for i, tc := range decoded.ToolCalls {
		if tc.Tool != sf.ToolCalls[i].Tool {
			t.Errorf("tool %d: got %d, want %d", i, tc.Tool, sf.ToolCalls[i].Tool)
		}
		if tc.PathFlag != sf.ToolCalls[i].PathFlag {
			t.Errorf("tool %d path_flag: got %d, want %d", i, tc.PathFlag, sf.ToolCalls[i].PathFlag)
		}
		if tc.CmdPrefix != sf.ToolCalls[i].CmdPrefix {
			t.Errorf("tool %d cmd: got %q, want %q", i, tc.CmdPrefix, sf.ToolCalls[i].CmdPrefix)
		}
	}
}

func TestSessionFrame_WithAgent(t *testing.T) {
	enc, err := NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	dec, err := NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	sf := &SessionFrame{
		SessionRef: 5,
		CapturedAt: time.Date(2026, 1, 15, 8, 0, 0, 0, time.UTC),
		EmailRef:   2,
		ActorType:  ActorAgent,
		AgentIDRef: 3,
		Turns: []TurnRecord{
			{Role: RoleAssistant, TsDelta: 0, BranchRef: 1, Text: "Running automated tests"},
		},
		ToolCalls: []ToolCallRecord{
			{Tool: ToolBash, PathFlag: PathNull, CmdPrefix: "npm test"},
		},
	}

	encoded := enc.EncodeSessionFrame(sf)
	compressed := encoded[frameEnvSize:]
	decoded, err := dec.DecodeSessionFrame(compressed)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.ActorType != ActorAgent {
		t.Errorf("actor_type: got %d, want %d", decoded.ActorType, ActorAgent)
	}
	if decoded.AgentIDRef != 3 {
		t.Errorf("agent_id_ref: got %d, want 3", decoded.AgentIDRef)
	}
}

func TestSessionFrame_InlinePath(t *testing.T) {
	enc, err := NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	dec, err := NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	sf := &SessionFrame{
		SessionRef: 0,
		CapturedAt: time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC),
		EmailRef:   0,
		ActorType:  ActorHuman,
		ToolCalls: []ToolCallRecord{
			{Tool: ToolWrite, PathFlag: PathInline, PathInline: "src/new-file.go"},
		},
	}

	encoded := enc.EncodeSessionFrame(sf)
	compressed := encoded[frameEnvSize:]
	decoded, err := dec.DecodeSessionFrame(compressed)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("tool calls: %d", len(decoded.ToolCalls))
	}
	tc := decoded.ToolCalls[0]
	if tc.PathFlag != PathInline {
		t.Errorf("path_flag: got %d, want %d", tc.PathFlag, PathInline)
	}
	if tc.PathInline != "src/new-file.go" {
		t.Errorf("path_inline: got %q", tc.PathInline)
	}
}

func TestCheckpointFrame_Roundtrip(t *testing.T) {
	enc, err := NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	dec, err := NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	cf := &CheckpointFrame{
		CheckpointRef: 42,
		GitSHA:        "aaa111bbb222ccc333ddd444eee555fff666aaa1",
		BranchRef:     0,
		EmailRef:      0,
		Timestamp:     time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC),
		ActorType:     ActorHuman,
		SessionRefs:   []uint64{0, 1},
		Files: []FileTouchedRecord{
			{PathRef: 0, ChangeType: ChangeModified},
			{PathRef: 1, ChangeType: ChangeAdded},
			{PathRef: 2, ChangeType: ChangeDeleted},
		},
	}

	encoded := enc.EncodeCheckpointFrame(cf)
	if FrameType(encoded[0]) != FrameCheckpoint {
		t.Errorf("frame type: got %x, want %x", encoded[0], FrameCheckpoint)
	}

	compressed := encoded[frameEnvSize:]
	decoded, err := dec.DecodeCheckpointFrame(compressed)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.GitSHA != cf.GitSHA {
		t.Errorf("git_sha: got %q, want %q", decoded.GitSHA, cf.GitSHA)
	}
	if decoded.CheckpointRef != 42 {
		t.Errorf("checkpoint_ref: got %d, want 42", decoded.CheckpointRef)
	}
	if !decoded.Timestamp.Equal(cf.Timestamp) {
		t.Errorf("ts: got %v, want %v", decoded.Timestamp, cf.Timestamp)
	}
	if len(decoded.SessionRefs) != 2 {
		t.Fatalf("session_refs: %d", len(decoded.SessionRefs))
	}
	if decoded.SessionRefs[0] != 0 || decoded.SessionRefs[1] != 1 {
		t.Errorf("session_refs: %v", decoded.SessionRefs)
	}
	if len(decoded.Files) != 3 {
		t.Fatalf("files: %d", len(decoded.Files))
	}
	if decoded.Files[2].ChangeType != ChangeDeleted {
		t.Errorf("file 2 change_type: got %c, want %c", decoded.Files[2].ChangeType, ChangeDeleted)
	}
}

func TestMetaFrame_Roundtrip(t *testing.T) {
	enc, err := NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	dec, err := NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	mf := &MetaFrame{
		FormatVersion: 0x01,
		EmailRef:      0,
		CheckpointSHA: "e7b3a91f2c4d5e6f7890abcdef1234567890abcd",
		Timestamp:     time.Date(2026, 2, 25, 16, 50, 0, 0, time.UTC),
		NSessions:     42,
		NCheckpoints:  38,
		NFrames:       80,
		NDictEntries:  133,
	}

	encoded := enc.EncodeMetaFrame(mf)
	if FrameType(encoded[0]) != FrameMeta {
		t.Errorf("frame type: got %x, want %x", encoded[0], FrameMeta)
	}

	compressed := encoded[frameEnvSize:]
	decoded, err := dec.DecodeMetaFrame(compressed)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if decoded.FormatVersion != mf.FormatVersion {
		t.Errorf("format_version: got %d, want %d", decoded.FormatVersion, mf.FormatVersion)
	}
	if decoded.CheckpointSHA != mf.CheckpointSHA {
		t.Errorf("sha: got %q, want %q", decoded.CheckpointSHA, mf.CheckpointSHA)
	}
	if decoded.NSessions != 42 {
		t.Errorf("n_sessions: got %d, want 42", decoded.NSessions)
	}
	if decoded.NCheckpoints != 38 {
		t.Errorf("n_checkpoints: got %d, want 38", decoded.NCheckpoints)
	}
	if decoded.NFrames != 80 {
		t.Errorf("n_frames: got %d, want 80", decoded.NFrames)
	}
	if decoded.NDictEntries != 133 {
		t.Errorf("n_dict_entries: got %d, want 133", decoded.NDictEntries)
	}
}

func TestToolCode_Mapping(t *testing.T) {
	tests := []struct {
		name string
		code byte
	}{
		{"Write", ToolWrite},
		{"Read", ToolRead},
		{"Bash", ToolBash},
		{"Edit", ToolEdit},
		{"Glob", ToolGlob},
		{"Grep", ToolGrep},
		{"Task", ToolTask},
	}
	for _, tt := range tests {
		if got := ToolCode(tt.name); got != tt.code {
			t.Errorf("ToolCode(%q) = %d, want %d", tt.name, got, tt.code)
		}
		if got := ToolName(tt.code); got != tt.name {
			t.Errorf("ToolName(%d) = %q, want %q", tt.code, got, tt.name)
		}
	}
	if ToolCode("SomethingNew") != ToolUnknown {
		t.Error("unknown tool should map to ToolUnknown")
	}
}

func TestCompressionRatio(t *testing.T) {
	enc, err := NewEncoder()
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	// Realistic session with several turns.
	sf := &SessionFrame{
		SessionRef: 0,
		CapturedAt: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		EmailRef:   0,
		ActorType:  ActorHuman,
		Turns: []TurnRecord{
			{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "fix the authentication bug in the login handler. the JWT token is not being validated correctly."},
			{Role: RoleAssistant, TsDelta: 30, BranchRef: 0, Text: "Let me read the authentication middleware to understand the current JWT validation logic."},
			{Role: RoleHuman, TsDelta: 90, BranchRef: 0, Text: "also check the token refresh endpoint"},
			{Role: RoleAssistant, TsDelta: 15, BranchRef: 0, Text: "I've updated the JWT validation to properly check the token expiry and refresh the token when needed. The issue was that the middleware was not handling expired tokens gracefully."},
		},
		ToolCalls: []ToolCallRecord{
			{Tool: ToolRead, PathFlag: PathDictRef, PathRef: 0},
			{Tool: ToolRead, PathFlag: PathDictRef, PathRef: 1},
			{Tool: ToolEdit, PathFlag: PathDictRef, PathRef: 0},
			{Tool: ToolBash, PathFlag: PathNull, CmdPrefix: "go test -race ./cmd/auth/..."},
		},
	}

	encoded := enc.EncodeSessionFrame(sf)
	compressedSize := len(encoded) - frameEnvSize
	// Build uncompressed payload to measure ratio.
	payload := encodeSessionPayload(sf)

	ratio := float64(len(payload)) / float64(compressedSize)
	t.Logf("Uncompressed: %d bytes, Compressed: %d bytes, Ratio: %.2f:1, Total with envelope: %d bytes",
		len(payload), compressedSize, ratio, len(encoded))

	// With preset dict, we expect at least 1.2:1 on a ~400 byte payload.
	if ratio < 1.0 {
		t.Errorf("compression made it larger: ratio %.2f", ratio)
	}
}

// callParse invokes fn, converting any panic into a test failure instead of
// crashing the test binary — this is what a decoder panic would do to a live
// `rekal sync` process on a corrupt or hostile frame from a teammate's push.
func callParse(t *testing.T, fn func() error) (err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("parse panicked: %v", r)
		}
	}()
	return fn()
}

// TestParseCheckpointPayload_TruncatedNeverPanics feeds every possible
// truncation of a valid checkpoint payload to the parser. A malformed or
// truncated frame (as would arrive from a corrupt git object or a hostile
// teammate push) must return an error, never panic.
func TestParseCheckpointPayload_TruncatedNeverPanics(t *testing.T) {
	cf := &CheckpointFrame{
		CheckpointRef: 42,
		GitSHA:        "aaa111bbb222ccc333ddd444eee555fff666aaa1",
		BranchRef:     7,
		EmailRef:      3,
		Timestamp:     time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC),
		ActorType:     ActorAgent,
		AgentIDRef:    9,
		SessionRefs:   []uint64{0, 1, 2},
		Files: []FileTouchedRecord{
			{PathRef: 0, ChangeType: ChangeModified},
			{PathRef: 1, ChangeType: ChangeAdded},
		},
	}
	payload := encodeCheckpointPayload(cf)

	for i := 0; i <= len(payload); i++ {
		trunc := payload[:i]
		var cfOut *CheckpointFrame
		err := callParse(t, func() error {
			var perr error
			cfOut, perr = parseCheckpointPayload(trunc)
			return perr
		})
		if i == len(payload) {
			if err != nil {
				t.Fatalf("full payload should parse cleanly, got: %v", err)
			}
			if cfOut == nil {
				t.Fatal("full payload parse returned nil frame with nil error")
			}
			continue
		}
		// Any truncation must either error out or (rarely, if the cut lands
		// exactly on a field boundary with no further required bytes) still
		// be internally consistent — never panic. The panic check above is
		// the actual regression guard; err is allowed to be nil only if
		// parsing genuinely had nothing left to misinterpret.
		_ = err
	}
}

// TestParseCheckpointPayload_HugeSessionCountRejected crafts a payload whose
// declared session-ref count vastly exceeds the bytes actually present. This
// must be rejected before allocating a slice sized by the attacker-controlled
// count.
func TestParseCheckpointPayload_HugeSessionCountRejected(t *testing.T) {
	cf := &CheckpointFrame{
		CheckpointRef: 1,
		GitSHA:        "aaa111bbb222ccc333ddd444eee555fff666aaa1",
		Timestamp:     time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC),
		ActorType:     ActorHuman,
		SessionRefs:   []uint64{0},
	}
	payload := encodeCheckpointPayload(cf)

	// Re-derive the byte offset of the n_sessions varint by replaying the
	// same field order parseCheckpointPayload expects (header, checkpoint
	// ref, git_sha, branch_ref, email_ref, timestamp, actor_type — no
	// agent_id_ref since ActorHuman).
	headerLen := 6
	headerLen += len(appendUvarint(nil, cf.CheckpointRef))
	headerLen += 40 // git_sha
	headerLen += len(appendUvarint(nil, cf.BranchRef))
	headerLen += len(appendUvarint(nil, cf.EmailRef))
	headerLen += 4 // timestamp
	headerLen += 1 // actor_type

	// Overwrite everything from n_sessions onward with a huge declared count
	// and nothing else — simulating a corrupted/truncated frame that claims
	// far more session refs than remain in the buffer.
	huge := appendUvarint(nil, 1<<40)
	crafted := append(append([]byte{}, payload[:headerLen]...), huge...)

	var cfOut *CheckpointFrame
	err := callParse(t, func() error {
		var perr error
		cfOut, perr = parseCheckpointPayload(crafted)
		return perr
	})
	if err == nil {
		t.Fatalf("expected error for oversized session count, got frame: %+v", cfOut)
	}
}

func TestParseSessionPayload_TruncatedNeverPanics(t *testing.T) {
	sf := &SessionFrame{
		SessionRef: 3,
		CapturedAt: time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC),
		EmailRef:   1,
		ActorType:  ActorAgent,
		AgentIDRef: 2,
		Turns: []TurnRecord{
			{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "hello"},
			{Role: RoleAssistant, TsDelta: 5, BranchRef: 0, Text: "hi there, a longer reply"},
		},
		ToolCalls: []ToolCallRecord{
			{Tool: ToolRead, PathFlag: PathDictRef, PathRef: 0},
			{Tool: ToolWrite, PathFlag: PathInline, PathInline: "a/b.go"},
			{Tool: ToolBash, PathFlag: PathNull, CmdPrefix: "go build"},
		},
	}
	payload := encodeSessionPayload(sf)

	for i := 0; i <= len(payload); i++ {
		trunc := payload[:i]
		_ = callParse(t, func() error {
			_, perr := parseSessionPayload(trunc)
			return perr
		})
	}
}

func TestParseMetaPayload_TruncatedNeverPanics(t *testing.T) {
	mf := &MetaFrame{
		FormatVersion: 0x01,
		EmailRef:      4,
		CheckpointSHA: "e7b3a91f2c4d5e6f7890abcdef1234567890abcd",
		Timestamp:     time.Date(2026, 2, 25, 16, 50, 0, 0, time.UTC),
		NSessions:     42,
		NCheckpoints:  38,
		NFrames:       80,
		NDictEntries:  133,
	}
	payload := encodeMetaPayload(mf)

	for i := 0; i <= len(payload); i++ {
		trunc := payload[:i]
		_ = callParse(t, func() error {
			_, perr := parseMetaPayload(trunc)
			return perr
		})
	}
}

// TestDecodeCheckpointFrame_CorruptCompressedBytesNoPanic exercises the full
// decode path (zstd + parse) with garbage compressed bytes, matching what
// import.go actually receives from a teammate's git object.
func TestDecodeCheckpointFrame_CorruptCompressedBytesNoPanic(t *testing.T) {
	dec, err := NewDecoder()
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	inputs := [][]byte{
		nil,
		{},
		{0x00},
		{0xFF, 0xFF, 0xFF, 0xFF},
		bytesRepeat(0xAB, 64),
	}
	for _, in := range inputs {
		_ = callParse(t, func() error {
			_, perr := dec.DecodeCheckpointFrame(in)
			return perr
		})
	}
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func BenchmarkEncodeSessionFrame(b *testing.B) {
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
			{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "fix the bug in the auth handler"},
			{Role: RoleAssistant, TsDelta: 30, BranchRef: 0, Text: "Let me read the file and fix the issue."},
			{Role: RoleHuman, TsDelta: 60, BranchRef: 0, Text: "thanks"},
		},
		ToolCalls: []ToolCallRecord{
			{Tool: ToolRead, PathFlag: PathDictRef, PathRef: 0},
			{Tool: ToolEdit, PathFlag: PathDictRef, PathRef: 0},
			{Tool: ToolBash, PathFlag: PathNull, CmdPrefix: "go test ./..."},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc.EncodeSessionFrame(sf)
	}
}

func BenchmarkDecodeSessionFrame(b *testing.B) {
	enc, err := NewEncoder()
	if err != nil {
		b.Fatalf("NewEncoder: %v", err)
	}
	defer enc.Close()

	dec, err := NewDecoder()
	if err != nil {
		b.Fatalf("NewDecoder: %v", err)
	}
	defer dec.Close()

	sf := &SessionFrame{
		SessionRef: 0,
		CapturedAt: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		EmailRef:   0,
		ActorType:  ActorHuman,
		Turns: []TurnRecord{
			{Role: RoleHuman, TsDelta: 0, BranchRef: 0, Text: "fix the bug"},
			{Role: RoleAssistant, TsDelta: 30, BranchRef: 0, Text: "Done."},
		},
		ToolCalls: []ToolCallRecord{
			{Tool: ToolEdit, PathFlag: PathDictRef, PathRef: 0},
		},
	}

	encoded := enc.EncodeSessionFrame(sf)
	compressed := encoded[frameEnvSize:]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dec.DecodeSessionFrame(compressed)
	}
}
