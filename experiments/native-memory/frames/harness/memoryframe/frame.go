// Package memoryframe is a prototype contract, not a production Rekal feature.
package memoryframe

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
)

const SnapshotType codec.FrameType = 0x07 // Experimental; not allocated upstream.

type MemoryEnvelope struct {
	Schema       int             `json:"schema"`
	CheckpointID string          `json:"checkpoint_id"`
	Snapshot     json.RawMessage `json:"snapshot"`
}

func DecodeMemoryPayload(payload []byte) (*MemoryEnvelope, error) {
	if err := CheckPayload(payload); err != nil {
		return nil, err
	}
	var envelope MemoryEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	if envelope.CheckpointID == "" {
		return nil, fmt.Errorf("memory checkpoint ID missing")
	}
	if err := CheckPayload(envelope.Snapshot); err != nil {
		return nil, err
	}
	return &envelope, nil
}

func CheckPayload(payload []byte) error {
	var value struct {
		Schema int `json:"schema"`
	}
	if err := json.Unmarshal(payload, &value); err != nil {
		return err
	}
	if value.Schema != 1 {
		return fmt.Errorf("unsupported memory schema %d", value.Schema)
	}
	return nil
}

// AppendCheckpoint can emit a checkpoint with memory and zero session refs.
// Memory has its own typed member; existing session/checkpoint schemas stay intact.
func AppendCheckpoint(body []byte, dict *codec.Dict, checkpointID, gitSHA string, payloads [][]byte, batch bool) ([]byte, error) {
	if len(payloads) == 0 {
		return nil, fmt.Errorf("empty memory update")
	}
	members := make([]codec.BatchMember, 0, len(payloads)+1)
	for _, payload := range payloads {
		if err := CheckPayload(payload); err != nil {
			return nil, err
		}
		var association struct {
			GitSHA string `json:"observed_at_commit"`
		}
		if err := json.Unmarshal(payload, &association); err != nil {
			return nil, err
		}
		if association.GitSHA != "" && association.GitSHA != gitSHA {
			return nil, fmt.Errorf("memory/checkpoint commit mismatch")
		}
		wrapped, err := json.Marshal(MemoryEnvelope{Schema: 1, CheckpointID: checkpointID, Snapshot: payload})
		if err != nil {
			return nil, err
		}
		members = append(members, codec.BatchMember{Type: SnapshotType, Payload: wrapped})
	}
	cp := &codec.CheckpointFrame{
		CheckpointRef: dict.LookupOrAdd(codec.NSSessions, checkpointID),
		GitSHA:        gitSHA, BranchRef: dict.LookupOrAdd(codec.NSBranches, "main"),
		EmailRef:  dict.LookupOrAdd(codec.NSEmails, "synthetic@example.invalid"),
		Timestamp: time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC),
	}
	members = append(members, codec.CheckpointMember(cp))
	enc, err := codec.NewEncoder()
	if err != nil {
		return nil, err
	}
	defer enc.Close()
	out := append([]byte(nil), body...)
	if batch {
		frame, err := enc.EncodeBatch(members)
		if err != nil {
			return nil, err
		}
		return codec.AppendFrame(out, frame), nil
	}
	for _, member := range members {
		frame, err := enc.EncodeMemberFrame(member)
		if err != nil {
			return nil, err
		}
		out = codec.AppendFrame(out, frame)
	}
	return out, nil
}

// Members uses the real upstream zstd codec and envelope/batch parser.
// The wrapper additionally rejects a partial trailing envelope and bad body version.
func Members(body []byte) ([]codec.BatchMember, error) {
	if len(body) < 9 || body[7] != 1 {
		return nil, fmt.Errorf("unsupported body header/version")
	}
	frames, err := codec.ScanFrames(body)
	if err != nil {
		return nil, err
	}
	end := 9
	if len(frames) > 0 {
		last := frames[len(frames)-1]
		end = last.PayloadOffset + last.CompressedLen
	}
	if end != len(body) {
		return nil, fmt.Errorf("partial trailing frame envelope")
	}
	dec, err := codec.NewDecoder()
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	var out []codec.BatchMember
	for _, frame := range frames {
		compressed := codec.ExtractFramePayload(body, frame)
		if frame.Type == codec.FrameBatch {
			members, err := dec.DecodeBatch(compressed)
			if err != nil {
				return nil, err
			}
			out = append(out, members...)
		} else {
			payload, err := dec.ProofDecodePayload(compressed)
			if err != nil {
				return nil, err
			}
			out = append(out, codec.BatchMember{Type: frame.Type, Payload: payload})
		}
	}
	return out, nil
}

func MemoryRecords(body []byte) ([]json.RawMessage, error) {
	members, err := Members(body)
	if err != nil {
		return nil, err
	}
	out := []json.RawMessage{}
	for _, member := range members {
		if member.Type != SnapshotType {
			continue
		}
		envelope, err := DecodeMemoryPayload(member.Payload)
		if err != nil {
			return nil, err
		}
		out = append(out, envelope.Snapshot)
	}
	return out, nil
}

// LegacyCounts models the known-type dispatch in current ImportBranch using
// actual upstream decoders. It is not the full DuckDB importer.
func LegacyCounts(body []byte) (map[string]int, error) {
	members, err := Members(body)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{"sessions": 0, "checkpoints": 0, "unknown_skipped": 0}
	for _, m := range members {
		switch m.Type {
		case codec.FrameSession, codec.FrameSessionV2:
			if _, err := codec.DecodeSessionPayload(m.Payload); err != nil {
				return nil, err
			}
			counts["sessions"]++
		case codec.FrameCheckpoint, codec.FrameCheckpointV2:
			if _, err := codec.DecodeCheckpointPayload(m.Payload); err != nil {
				return nil, err
			}
			counts["checkpoints"]++
		case codec.FrameMeta:
			if _, err := codec.DecodeMetaPayload(m.Payload); err != nil {
				return nil, err
			}
		default:
			counts["unknown_skipped"]++
		}
	}
	return counts, nil
}

// ReexportModel illustrates fresh-body re-export from known records. It does
// not invoke real ExportAllFrames or its DB queries/merge gate. Memory-aware
// mode rejects unsupported types, so an updated repair cannot silently drop them.
func ReexportModel(body []byte, aware bool) ([]byte, error) {
	members, err := Members(body)
	if err != nil {
		return nil, err
	}
	enc, err := codec.NewEncoder()
	if err != nil {
		return nil, err
	}
	defer enc.Close()
	out := codec.NewBody()
	for _, m := range members {
		known := m.Type == codec.FrameSession || m.Type == codec.FrameSessionV2 ||
			m.Type == codec.FrameCheckpoint || m.Type == codec.FrameCheckpointV2 || m.Type == codec.FrameMeta
		if m.Type == SnapshotType && aware {
			if _, err := DecodeMemoryPayload(m.Payload); err != nil {
				return nil, err
			}
			known = true
		}
		if !known {
			if aware {
				return nil, fmt.Errorf("refuse re-export: unsupported frame 0x%02x", m.Type)
			}
			continue
		}
		frame, err := enc.EncodeMemberFrame(m)
		if err != nil {
			return nil, err
		}
		out = codec.AppendFrame(out, frame)
	}
	return out, nil
}

func Seed() ([]byte, *codec.Dict, error) {
	dict := codec.NewDict()
	sf := &codec.SessionFrame{SessionRef: dict.LookupOrAdd(codec.NSSessions, "01J"+strings.Repeat("0", 23)),
		EmailRef:   dict.LookupOrAdd(codec.NSEmails, "synthetic@example.invalid"),
		CapturedAt: time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC),
		Turns:      []codec.TurnRecord{{Role: codec.RoleHuman, Text: "Use idempotency keys for payment retries."}},
	}
	enc, err := codec.NewEncoder()
	if err != nil {
		return nil, nil, err
	}
	defer enc.Close()
	frame, err := enc.EncodeSessionFrame(sf)
	if err != nil {
		return nil, nil, err
	}
	return codec.AppendFrame(codec.NewBody(), frame), dict, nil
}
