package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Wire-format filenames on the rekal orphan branch: the append-only frame
// body and the string dictionary it references.
const (
	BodyFilename = "rekal.body"
	DictFilename = "dict.bin"
)

const (
	bodyMagic    = "RKLBODY"
	bodyVersion  = 0x01
	bodyHdrSize  = 9 // 7 magic + 1 version + 1 flags
	frameEnvSize = 6 // 1 type + 3 compressed_len + 2 uncompressed_len
)

// FrameType identifies the kind of frame.
type FrameType byte

const (
	FrameSession    FrameType = 0x01
	FrameCheckpoint FrameType = 0x02
	FrameMeta       FrameType = 0x03
	// V2 frames carry counts as varints instead of single bytes (see
	// payloadVersionV2 in frame.go). They use distinct envelope types so
	// binaries that predate them skip the frames they can't parse instead of
	// misreading them — unknown types fall through every decode switch.
	FrameSessionV2    FrameType = 0x04
	FrameCheckpointV2 FrameType = 0x05
	// FrameBatch groups several member frames (sessions + a checkpoint) into
	// one zstd stream so redundancy shared across them — repeated paths,
	// emails, ULIDs, phrasing — compresses together instead of once per
	// frame. Its decompressed payload is a sequence of length-prefixed member
	// frames (see EncodeBatch/DecodeBatch). Old readers skip it by envelope
	// length like any unknown type, so a batched push is invisible to a
	// pre-batch binary rather than misread.
	FrameBatch     FrameType = 0x06
	FrameTombstone FrameType = 0xFF
)

// maxFrameCompressedLen is the largest compressed payload the 3-byte
// envelope length field can describe. The encoder refuses to emit anything
// larger rather than silently truncating the length.
const maxFrameCompressedLen = 1<<24 - 1

// FrameSlice describes a frame's location in the body.
type FrameSlice struct {
	Type            FrameType
	Offset          int // byte offset from start of body (includes envelope)
	CompressedLen   int
	UncompressedLen int
	PayloadOffset   int // byte offset of compressed payload (Offset + frameEnvSize)
}

// NewBody returns a 9-byte rekal.body file header.
func NewBody() []byte {
	buf := make([]byte, bodyHdrSize)
	copy(buf[0:7], bodyMagic)
	buf[7] = bodyVersion
	buf[8] = 0x01 // flags: bit 0 = preset dict available
	return buf
}

// AppendFrame appends an encoded frame (envelope + compressed payload) to the body.
func AppendFrame(body, frame []byte) []byte {
	return append(body, frame...)
}

// WriteEnvelope writes a 6-byte frame envelope. The uncompressed_len field is
// advisory (decoding never relies on it — zstd frames carry their own size);
// it saturates at 0xFFFF for payloads larger than the u16 can describe.
func WriteEnvelope(frameType FrameType, compressedLen, uncompressedLen int) []byte {
	env := make([]byte, frameEnvSize)
	env[0] = byte(frameType)
	// compressed_len as u24 LE.
	env[1] = byte(compressedLen)
	env[2] = byte(compressedLen >> 8)
	env[3] = byte(compressedLen >> 16)
	// uncompressed_len as u16 LE.
	if uncompressedLen > 0xFFFF {
		uncompressedLen = 0xFFFF
	}
	binary.LittleEndian.PutUint16(env[4:6], uint16(uncompressedLen))
	return env
}

// ScanFrames scans the body and returns metadata for each frame without decompressing.
func ScanFrames(body []byte) ([]FrameSlice, error) {
	if len(body) < bodyHdrSize {
		return nil, errors.New("body: data too short for header")
	}

	magic := string(body[0:7])
	if magic != bodyMagic {
		return nil, fmt.Errorf("body: bad magic %q, want %q", magic, bodyMagic)
	}

	var frames []FrameSlice
	pos := bodyHdrSize

	for pos+frameEnvSize <= len(body) {
		ft := FrameType(body[pos])
		compLen := int(body[pos+1]) | int(body[pos+2])<<8 | int(body[pos+3])<<16
		uncompLen := int(binary.LittleEndian.Uint16(body[pos+4 : pos+6]))

		payloadStart := pos + frameEnvSize
		if payloadStart+compLen > len(body) {
			return nil, fmt.Errorf("body: frame at offset %d truncated (need %d bytes, have %d)",
				pos, compLen, len(body)-payloadStart)
		}

		frames = append(frames, FrameSlice{
			Type:            ft,
			Offset:          pos,
			CompressedLen:   compLen,
			UncompressedLen: uncompLen,
			PayloadOffset:   payloadStart,
		})

		pos = payloadStart + compLen
	}

	return frames, nil
}

// ExtractFramePayload returns the compressed payload bytes for a frame slice.
func ExtractFramePayload(body []byte, fs FrameSlice) []byte {
	return body[fs.PayloadOffset : fs.PayloadOffset+fs.CompressedLen]
}
