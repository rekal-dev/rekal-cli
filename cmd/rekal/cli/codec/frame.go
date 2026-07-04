package codec

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/klauspost/compress/zstd"
)

// Tool codes for binary encoding.
const (
	ToolWrite   byte = 0x00
	ToolRead    byte = 0x01
	ToolBash    byte = 0x02
	ToolEdit    byte = 0x03
	ToolGlob    byte = 0x04
	ToolGrep    byte = 0x05
	ToolTask    byte = 0x06
	ToolUnknown byte = 0xFF
)

// Path flag values.
const (
	PathDictRef byte = 0x00
	PathInline  byte = 0x01
	PathNull    byte = 0x02
)

// Actor type values.
const (
	ActorHuman byte = 0x00
	ActorAgent byte = 0x01
)

// Role values. RoleHumanSteering marks out-of-band user steering messages
// (queue-operation/enqueue captures) — the highest-intent text in the corpus,
// used to boost recall ranking. It costs nothing on the wire: same byte
// field, one more value.
const (
	RoleHuman         byte = 0x00
	RoleAssistant     byte = 0x01
	RoleHumanSteering byte = 0x02
)

// Change type values (ASCII bytes).
const (
	ChangeAdded    byte = 'A'
	ChangeModified byte = 'M'
	ChangeDeleted  byte = 'D'
	ChangeRenamed  byte = 'R'
)

var (
	sessionMagic    = []byte("RKLS")
	checkpointMagic = []byte("RKLC")
	metaMagic       = []byte("RKLM")
)

// Payload versions. V1 stores the session turn/tool-call counts and the
// checkpoint file count as single bytes — anything above 255 silently wrapped
// mod 256 and corrupted the frame. V2 stores those counts as varints. New
// frames are written as v1 while the counts fit (so binaries that predate v2
// keep reading them) and as v2 — under the FrameSessionV2/FrameCheckpointV2
// envelope types, which old readers skip — only when they don't.
const (
	payloadVersionV1 = 0x01
	payloadVersionV2 = 0x02
)

// SessionFrame is the decoded content of a session frame (0x01).
type SessionFrame struct {
	SessionRef uint64
	CapturedAt time.Time
	EmailRef   uint64
	ActorType  byte
	AgentIDRef uint64 // only valid if ActorType == ActorAgent
	Turns      []TurnRecord
	ToolCalls  []ToolCallRecord
}

// TurnRecord is a single conversation turn.
type TurnRecord struct {
	Role      byte
	TsDelta   uint64 // seconds since previous turn
	BranchRef uint64
	Text      string
}

// ToolCallRecord is a single tool invocation.
type ToolCallRecord struct {
	Tool       byte
	PathFlag   byte
	PathRef    uint64 // valid if PathFlag == PathDictRef
	PathInline string // valid if PathFlag == PathInline
	CmdPrefix  string
}

// CheckpointFrame is the decoded content of a checkpoint frame (0x02).
type CheckpointFrame struct {
	CheckpointRef uint64 // dict ref to checkpoint ULID
	GitSHA        string // 40-char hex
	BranchRef     uint64
	EmailRef      uint64
	Timestamp     time.Time
	ActorType     byte
	AgentIDRef    uint64 // only valid if ActorType == ActorAgent
	SessionRefs   []uint64
	Files         []FileTouchedRecord
}

// FileTouchedRecord is a file changed in a checkpoint.
type FileTouchedRecord struct {
	PathRef    uint64
	ChangeType byte
}

// MetaFrame is the decoded content of a meta frame (0x03).
type MetaFrame struct {
	FormatVersion byte
	EmailRef      uint64
	CheckpointSHA string // 40-char hex
	Timestamp     time.Time
	NSessions     uint32
	NCheckpoints  uint32
	NFrames       uint32
	NDictEntries  uint32
}

// toolNameToCode maps tool name strings to binary codes.
var toolNameToCode = map[string]byte{
	"Write": ToolWrite,
	"Read":  ToolRead,
	"Bash":  ToolBash,
	"Edit":  ToolEdit,
	"Glob":  ToolGlob,
	"Grep":  ToolGrep,
	"Task":  ToolTask,
}

// toolCodeToName maps binary codes back to tool name strings.
var toolCodeToName = map[byte]string{
	ToolWrite:   "Write",
	ToolRead:    "Read",
	ToolBash:    "Bash",
	ToolEdit:    "Edit",
	ToolGlob:    "Glob",
	ToolGrep:    "Grep",
	ToolTask:    "Task",
	ToolUnknown: "Unknown",
}

// ToolCode returns the binary code for a tool name.
func ToolCode(name string) byte {
	if c, ok := toolNameToCode[name]; ok {
		return c
	}
	return ToolUnknown
}

// ToolName returns the string name for a tool code.
func ToolName(code byte) string {
	if n, ok := toolCodeToName[code]; ok {
		return n
	}
	return "Unknown"
}

// Encoder handles frame encoding with zstd compression.
type Encoder struct {
	zw *zstd.Encoder
}

// NewEncoder creates a new frame encoder with zstd preset dictionary support.
func NewEncoder() (*Encoder, error) {
	opts := []zstd.EOption{
		zstd.WithEncoderLevel(zstd.SpeedDefault), // level 3
	}
	if len(presetDict) > 0 {
		opts = append(opts, zstd.WithEncoderDict(presetDict))
	}
	zw, err := zstd.NewWriter(nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("codec: create zstd encoder: %w", err)
	}
	return &Encoder{zw: zw}, nil
}

// Close releases encoder resources.
func (e *Encoder) Close() {
	_ = e.zw.Close()
}

// EncodeSessionFrame encodes a session frame to bytes (envelope + compressed
// payload). Sessions whose turn/tool-call counts fit in a byte are written as
// v1 for compatibility with readers that predate v2; larger sessions use the
// v2 payload under the FrameSessionV2 envelope type, which old readers skip.
func (e *Encoder) EncodeSessionFrame(sf *SessionFrame) ([]byte, error) {
	if sessionNeedsV2(sf) {
		return e.wrapFrame(FrameSessionV2, encodeSessionPayloadV2(sf))
	}
	return e.wrapFrame(FrameSession, encodeSessionPayload(sf))
}

// EncodeCheckpointFrame encodes a checkpoint frame to bytes. Checkpoints
// touching more than 255 files use the v2 payload (varint file count) under
// the FrameCheckpointV2 envelope type; smaller ones stay v1 for compatibility.
func (e *Encoder) EncodeCheckpointFrame(cf *CheckpointFrame) ([]byte, error) {
	if len(cf.Files) > 0xFF {
		return e.wrapFrame(FrameCheckpointV2, encodeCheckpointPayloadV2(cf))
	}
	return e.wrapFrame(FrameCheckpoint, encodeCheckpointPayload(cf))
}

// EncodeMetaFrame encodes a meta frame to bytes.
func (e *Encoder) EncodeMetaFrame(mf *MetaFrame) ([]byte, error) {
	payload := encodeMetaPayload(mf)
	return e.wrapFrame(FrameMeta, payload)
}

// sessionNeedsV2 reports whether sf requires the v2 payload: v1 stores the
// turn and tool-call counts as single bytes.
func sessionNeedsV2(sf *SessionFrame) bool {
	return len(sf.Turns) > 0xFF || len(sf.ToolCalls) > 0xFF
}

func (e *Encoder) wrapFrame(ft FrameType, payload []byte) ([]byte, error) {
	compressed := e.zw.EncodeAll(payload, nil)
	if len(compressed) > maxFrameCompressedLen {
		return nil, fmt.Errorf("codec: compressed frame is %d bytes, exceeds the format's %d-byte limit", len(compressed), maxFrameCompressedLen)
	}
	env := WriteEnvelope(ft, len(compressed), len(payload))
	return append(env, compressed...), nil
}

func encodeSessionPayload(sf *SessionFrame) []byte {
	buf := make([]byte, 0, 256)

	// Header: magic + payload_version + dict_flags + n_turns + n_tools (u8 each)
	buf = append(buf, sessionMagic...)
	buf = append(buf, payloadVersionV1)
	buf = append(buf, dictFlagsByte())
	buf = append(buf, byte(len(sf.Turns)))
	buf = append(buf, byte(len(sf.ToolCalls)))

	return appendSessionBody(buf, sf)
}

// encodeSessionPayloadV2 is the v1 layout with varint turn/tool-call counts
// instead of single bytes.
func encodeSessionPayloadV2(sf *SessionFrame) []byte {
	buf := make([]byte, 0, 256)

	// Header: magic + payload_version + dict_flags + n_turns + n_tools (varints)
	buf = append(buf, sessionMagic...)
	buf = append(buf, payloadVersionV2)
	buf = append(buf, dictFlagsByte())
	buf = appendUvarint(buf, uint64(len(sf.Turns)))
	buf = appendUvarint(buf, uint64(len(sf.ToolCalls)))

	return appendSessionBody(buf, sf)
}

func dictFlagsByte() byte {
	if len(presetDict) > 0 {
		return 0x01
	}
	return 0x00
}

// appendSessionBody appends the session meta, turn records, and tool-call
// records — identical between payload v1 and v2.
func appendSessionBody(buf []byte, sf *SessionFrame) []byte {
	// Session meta.
	buf = appendUvarint(buf, sf.SessionRef)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(sf.CapturedAt.Unix()))
	buf = appendUvarint(buf, sf.EmailRef)
	buf = append(buf, sf.ActorType)
	if sf.ActorType == ActorAgent {
		buf = appendUvarint(buf, sf.AgentIDRef)
	}

	// Turns.
	for _, t := range sf.Turns {
		buf = append(buf, t.Role)
		buf = appendUvarint(buf, t.TsDelta)
		buf = appendUvarint(buf, t.BranchRef)
		buf = appendUvarint(buf, uint64(len(t.Text)))
		buf = append(buf, []byte(t.Text)...)
	}

	// Tool calls.
	for _, tc := range sf.ToolCalls {
		buf = append(buf, tc.Tool)
		buf = append(buf, tc.PathFlag)
		switch tc.PathFlag {
		case PathDictRef:
			buf = appendUvarint(buf, tc.PathRef)
		case PathInline:
			buf = appendUvarint(buf, uint64(len(tc.PathInline)))
			buf = append(buf, []byte(tc.PathInline)...)
		case PathNull:
			// no additional bytes
		}
		cmdBytes := []byte(tc.CmdPrefix)
		buf = appendUvarint(buf, uint64(len(cmdBytes)))
		if len(cmdBytes) > 0 {
			buf = append(buf, cmdBytes...)
		}
	}

	return buf
}

func encodeCheckpointPayload(cf *CheckpointFrame) []byte {
	buf := make([]byte, 0, 128)

	// Header: magic + payload_version + n_files (u8)
	buf = append(buf, checkpointMagic...)
	buf = append(buf, payloadVersionV1)
	buf = append(buf, byte(len(cf.Files)))

	return appendCheckpointBody(buf, cf)
}

// encodeCheckpointPayloadV2 is the v1 layout with a varint file count instead
// of a single byte.
func encodeCheckpointPayloadV2(cf *CheckpointFrame) []byte {
	buf := make([]byte, 0, 128)

	// Header: magic + payload_version + n_files (varint)
	buf = append(buf, checkpointMagic...)
	buf = append(buf, payloadVersionV2)
	buf = appendUvarint(buf, uint64(len(cf.Files)))

	return appendCheckpointBody(buf, cf)
}

// appendCheckpointBody appends the checkpoint meta, session refs, and
// file-touched records — identical between payload v1 and v2.
func appendCheckpointBody(buf []byte, cf *CheckpointFrame) []byte {
	// Checkpoint ULID dict ref (before GitSHA).
	buf = appendUvarint(buf, cf.CheckpointRef)

	// Checkpoint meta.
	sha := []byte(cf.GitSHA)
	if len(sha) < 40 {
		padded := make([]byte, 40)
		copy(padded, sha)
		sha = padded
	}
	buf = append(buf, sha[:40]...)
	buf = appendUvarint(buf, cf.BranchRef)
	buf = appendUvarint(buf, cf.EmailRef)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(cf.Timestamp.Unix()))
	buf = append(buf, cf.ActorType)
	if cf.ActorType == ActorAgent {
		buf = appendUvarint(buf, cf.AgentIDRef)
	}
	buf = appendUvarint(buf, uint64(len(cf.SessionRefs)))
	for _, ref := range cf.SessionRefs {
		buf = appendUvarint(buf, ref)
	}

	// Files touched.
	for _, f := range cf.Files {
		buf = appendUvarint(buf, f.PathRef)
		buf = append(buf, f.ChangeType)
	}

	return buf
}

func encodeMetaPayload(mf *MetaFrame) []byte {
	buf := make([]byte, 0, 64)

	// Header: magic + payload_version
	buf = append(buf, metaMagic...)
	buf = append(buf, payloadVersionV1)

	// Meta fields.
	buf = append(buf, mf.FormatVersion)
	buf = appendUvarint(buf, mf.EmailRef)
	sha := []byte(mf.CheckpointSHA)
	if len(sha) < 40 {
		padded := make([]byte, 40)
		copy(padded, sha)
		sha = padded
	}
	buf = append(buf, sha[:40]...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(mf.Timestamp.Unix()))
	buf = binary.LittleEndian.AppendUint32(buf, mf.NSessions)
	buf = binary.LittleEndian.AppendUint32(buf, mf.NCheckpoints)
	buf = binary.LittleEndian.AppendUint32(buf, mf.NFrames)
	buf = binary.LittleEndian.AppendUint32(buf, mf.NDictEntries)

	return buf
}

// Decoder handles frame decoding with zstd decompression.
type Decoder struct {
	zr *zstd.Decoder
}

// NewDecoder creates a new frame decoder.
func NewDecoder() (*Decoder, error) {
	opts := []zstd.DOption{}
	if len(presetDict) > 0 {
		opts = append(opts, zstd.WithDecoderDicts(presetDict))
	}
	zr, err := zstd.NewReader(nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("codec: create zstd decoder: %w", err)
	}
	return &Decoder{zr: zr}, nil
}

// Close releases decoder resources.
func (d *Decoder) Close() {
	d.zr.Close()
}

// DecodeSessionFrame decodes a compressed session frame payload.
func (d *Decoder) DecodeSessionFrame(compressed []byte) (*SessionFrame, error) {
	payload, err := d.zr.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("decode session: zstd: %w", err)
	}
	return parseSessionPayload(payload)
}

// DecodeCheckpointFrame decodes a compressed checkpoint frame payload.
func (d *Decoder) DecodeCheckpointFrame(compressed []byte) (*CheckpointFrame, error) {
	payload, err := d.zr.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("decode checkpoint: zstd: %w", err)
	}
	return parseCheckpointPayload(payload)
}

// DecodeMetaFrame decodes a compressed meta frame payload.
func (d *Decoder) DecodeMetaFrame(compressed []byte) (*MetaFrame, error) {
	payload, err := d.zr.DecodeAll(compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("decode meta: zstd: %w", err)
	}
	return parseMetaPayload(payload)
}

func parseSessionPayload(data []byte) (*SessionFrame, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("session payload too short: %d bytes", len(data))
	}
	if string(data[0:4]) != string(sessionMagic) {
		return nil, fmt.Errorf("session payload bad magic: %x", data[0:4])
	}
	// data[5] = dict_flags

	switch data[4] {
	case payloadVersionV1:
		return parseSessionBody(data, 8, int(data[6]), int(data[7]))
	case payloadVersionV2:
		pos := 6
		nTurns, n, err := readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("session payload: n_turns: %w", err)
		}
		pos += n
		nTools, n, err := readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("session payload: n_tools: %w", err)
		}
		pos += n
		// Each turn and tool call costs multiple bytes on the wire, so counts
		// exceeding the remaining bytes are definitely corrupt — reject them
		// (individually, so their sum cannot overflow) before trusting a
		// corruption-controlled count for slice capacities.
		remaining := uint64(len(data) - pos)
		if nTurns > remaining || nTools > remaining {
			return nil, fmt.Errorf("session payload: counts (%d turns, %d tools) exceed remaining data", nTurns, nTools)
		}
		return parseSessionBody(data, pos, int(nTurns), int(nTools))
	default:
		return nil, fmt.Errorf("session payload: unsupported version %d", data[4])
	}
}

// parseSessionBody parses the session meta, turn records, and tool-call
// records starting at pos — identical between payload v1 and v2.
func parseSessionBody(data []byte, pos, nTurns, nTools int) (*SessionFrame, error) {
	sf := &SessionFrame{}

	var n int
	var err error
	sf.SessionRef, n, err = readUvarint(data[pos:])
	if err != nil {
		return nil, fmt.Errorf("session payload: session_ref: %w", err)
	}
	pos += n
	if pos+4 > len(data) {
		return nil, fmt.Errorf("session payload truncated at captured_at")
	}
	sf.CapturedAt = time.Unix(int64(binary.LittleEndian.Uint32(data[pos:pos+4])), 0).UTC()
	pos += 4
	sf.EmailRef, n, err = readUvarint(data[pos:])
	if err != nil {
		return nil, fmt.Errorf("session payload: email_ref: %w", err)
	}
	pos += n
	if pos >= len(data) {
		return nil, fmt.Errorf("session payload truncated at actor_type")
	}
	sf.ActorType = data[pos]
	pos++
	if sf.ActorType == ActorAgent {
		sf.AgentIDRef, n, err = readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("session payload: agent_id_ref: %w", err)
		}
		pos += n
	}

	// Turns.
	sf.Turns = make([]TurnRecord, 0, nTurns)
	for i := 0; i < nTurns; i++ {
		if pos >= len(data) {
			return nil, fmt.Errorf("session payload truncated at turn %d", i)
		}
		var t TurnRecord
		t.Role = data[pos]
		pos++
		t.TsDelta, n, err = readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("session payload: turn %d ts_delta: %w", i, err)
		}
		pos += n
		t.BranchRef, n, err = readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("session payload: turn %d branch_ref: %w", i, err)
		}
		pos += n
		textLen, n2, err := readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("session payload: turn %d text_len: %w", i, err)
		}
		pos += n2
		if textLen > uint64(len(data)-pos) {
			return nil, fmt.Errorf("session payload truncated at turn %d text", i)
		}
		t.Text = string(data[pos : pos+int(textLen)])
		pos += int(textLen)
		sf.Turns = append(sf.Turns, t)
	}

	// Tool calls.
	sf.ToolCalls = make([]ToolCallRecord, 0, nTools)
	for i := 0; i < nTools; i++ {
		if pos+2 > len(data) {
			return nil, fmt.Errorf("session payload truncated at tool %d", i)
		}
		var tc ToolCallRecord
		tc.Tool = data[pos]
		pos++
		tc.PathFlag = data[pos]
		pos++
		switch tc.PathFlag {
		case PathDictRef:
			tc.PathRef, n, err = readUvarint(data[pos:])
			if err != nil {
				return nil, fmt.Errorf("session payload: tool %d path_ref: %w", i, err)
			}
			pos += n
		case PathInline:
			pathLen, n2, err := readUvarint(data[pos:])
			if err != nil {
				return nil, fmt.Errorf("session payload: tool %d path_len: %w", i, err)
			}
			pos += n2
			if pathLen > uint64(len(data)-pos) {
				return nil, fmt.Errorf("session payload truncated at tool %d inline path", i)
			}
			tc.PathInline = string(data[pos : pos+int(pathLen)])
			pos += int(pathLen)
		case PathNull:
			// no additional bytes
		}
		cmdLen, n2, err := readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("session payload: tool %d cmd_len: %w", i, err)
		}
		pos += n2
		if cmdLen > 0 {
			if cmdLen > uint64(len(data)-pos) {
				return nil, fmt.Errorf("session payload truncated at tool %d cmd", i)
			}
			tc.CmdPrefix = string(data[pos : pos+int(cmdLen)])
			pos += int(cmdLen)
		}
		sf.ToolCalls = append(sf.ToolCalls, tc)
	}

	return sf, nil
}

func parseCheckpointPayload(data []byte) (*CheckpointFrame, error) {
	if len(data) < 6 {
		return nil, fmt.Errorf("checkpoint payload too short: %d bytes", len(data))
	}
	if string(data[0:4]) != string(checkpointMagic) {
		return nil, fmt.Errorf("checkpoint payload bad magic: %x", data[0:4])
	}

	switch data[4] {
	case payloadVersionV1:
		return parseCheckpointBody(data, 6, int(data[5]))
	case payloadVersionV2:
		pos := 5
		nFiles, n, err := readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("checkpoint payload: n_files: %w", err)
		}
		pos += n
		// Each file record costs at least 2 bytes on the wire — reject a
		// corrupt count before trusting it for a slice capacity.
		if nFiles > uint64(len(data)-pos) {
			return nil, fmt.Errorf("checkpoint payload: n_files %d exceeds remaining data", nFiles)
		}
		return parseCheckpointBody(data, pos, int(nFiles))
	default:
		return nil, fmt.Errorf("checkpoint payload: unsupported version %d", data[4])
	}
}

// parseCheckpointBody parses the checkpoint meta, session refs, and
// file-touched records starting at pos — identical between payload v1 and v2.
func parseCheckpointBody(data []byte, pos, nFiles int) (*CheckpointFrame, error) {
	cf := &CheckpointFrame{}

	// Checkpoint ULID dict ref.
	var n int
	var err error
	cf.CheckpointRef, n, err = readUvarint(data[pos:])
	if err != nil {
		return nil, fmt.Errorf("checkpoint payload: checkpoint_ref: %w", err)
	}
	pos += n

	if pos+40 > len(data) {
		return nil, fmt.Errorf("checkpoint payload truncated at git_sha")
	}
	cf.GitSHA = string(data[pos : pos+40])
	pos += 40
	cf.BranchRef, n, err = readUvarint(data[pos:])
	if err != nil {
		return nil, fmt.Errorf("checkpoint payload: branch_ref: %w", err)
	}
	pos += n
	cf.EmailRef, n, err = readUvarint(data[pos:])
	if err != nil {
		return nil, fmt.Errorf("checkpoint payload: email_ref: %w", err)
	}
	pos += n
	if pos+4 > len(data) {
		return nil, fmt.Errorf("checkpoint payload truncated at ts")
	}
	cf.Timestamp = time.Unix(int64(binary.LittleEndian.Uint32(data[pos:pos+4])), 0).UTC()
	pos += 4
	if pos >= len(data) {
		return nil, fmt.Errorf("checkpoint payload truncated at actor_type")
	}
	cf.ActorType = data[pos]
	pos++
	if cf.ActorType == ActorAgent {
		cf.AgentIDRef, n, err = readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("checkpoint payload: agent_id_ref: %w", err)
		}
		pos += n
	}

	nSess, n2, err := readUvarint(data[pos:])
	if err != nil {
		return nil, fmt.Errorf("checkpoint payload: n_sessions: %w", err)
	}
	pos += n2
	// Each session ref costs at least 1 byte on the wire, so a count that
	// exceeds the remaining bytes is definitely corrupt — reject it before
	// allocating, rather than trusting an attacker/corruption-controlled
	// count for a slice capacity.
	if nSess > uint64(len(data)-pos) {
		return nil, fmt.Errorf("checkpoint payload: n_sessions %d exceeds remaining data", nSess)
	}
	cf.SessionRefs = make([]uint64, 0, nSess)
	for i := uint64(0); i < nSess; i++ {
		ref, n3, err := readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("checkpoint payload: session_ref %d: %w", i, err)
		}
		pos += n3
		cf.SessionRefs = append(cf.SessionRefs, ref)
	}

	// Files touched. nFiles is bounded by the caller: [0,255] for v1 (single
	// byte), checked against the remaining data for v2.
	cf.Files = make([]FileTouchedRecord, 0, nFiles)
	for i := 0; i < nFiles; i++ {
		var f FileTouchedRecord
		f.PathRef, n, err = readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("checkpoint payload: file %d path_ref: %w", i, err)
		}
		pos += n
		if pos >= len(data) {
			return nil, fmt.Errorf("checkpoint payload truncated at file %d change_type", i)
		}
		f.ChangeType = data[pos]
		pos++
		cf.Files = append(cf.Files, f)
	}

	return cf, nil
}

func parseMetaPayload(data []byte) (*MetaFrame, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("meta payload too short: %d bytes", len(data))
	}
	if string(data[0:4]) != string(metaMagic) {
		return nil, fmt.Errorf("meta payload bad magic: %x", data[0:4])
	}
	// data[4] = payload_version

	pos := 5
	mf := &MetaFrame{}

	if pos >= len(data) {
		return nil, fmt.Errorf("meta payload truncated at format_version")
	}
	mf.FormatVersion = data[pos]
	pos++

	var n int
	var err error
	mf.EmailRef, n, err = readUvarint(data[pos:])
	if err != nil {
		return nil, fmt.Errorf("meta payload: email_ref: %w", err)
	}
	pos += n

	if pos+40 > len(data) {
		return nil, fmt.Errorf("meta payload truncated at checkpoint_sha")
	}
	mf.CheckpointSHA = string(data[pos : pos+40])
	pos += 40

	if pos+4+4*4 > len(data) {
		return nil, fmt.Errorf("meta payload truncated at counts")
	}
	mf.Timestamp = time.Unix(int64(binary.LittleEndian.Uint32(data[pos:pos+4])), 0).UTC()
	pos += 4
	mf.NSessions = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	mf.NCheckpoints = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	mf.NFrames = binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4
	mf.NDictEntries = binary.LittleEndian.Uint32(data[pos : pos+4])

	return mf, nil
}

// appendUvarint appends an unsigned LEB128 varint to buf.
func appendUvarint(buf []byte, x uint64) []byte {
	var tmp [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(tmp[:], x)
	return append(buf, tmp[:n]...)
}

// readUvarint reads an unsigned LEB128 varint from data.
// Returns the value and the number of bytes consumed, or an error if data is
// empty or does not contain a complete, valid varint. Callers must check the
// error before trusting the returned count — a malformed or truncated frame
// (e.g. from a corrupt or hostile git push) must produce an error here, not
// silently advance past the end of data and panic on the next slice.
func readUvarint(data []byte) (uint64, int, error) {
	if len(data) == 0 {
		return 0, 0, fmt.Errorf("unexpected end of data reading varint")
	}
	v, n := binary.Uvarint(data)
	if n == 0 {
		return 0, 0, fmt.Errorf("incomplete varint")
	}
	if n < 0 {
		return 0, 0, fmt.Errorf("varint overflows uint64")
	}
	return v, n, nil
}
