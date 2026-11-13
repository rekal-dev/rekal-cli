package transport

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
)

// readMemoryFrames validates before callers mutate their ledger or overwrite a
// wire body. Unsupported members must not silently disappear in a re-export.
func readMemoryFrames(body []byte) ([]memory.Revision, error) {
	if len(body) == 0 {
		return nil, nil
	}
	if len(body) < 9 || body[7] != 1 {
		return nil, fmt.Errorf("unsupported memory body version")
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
		return nil, fmt.Errorf("truncated wire envelope")
	}
	dec, err := codec.NewDecoder()
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	var result []memory.Revision
	check := func(m codec.BatchMember) error {
		switch m.Type {
		case codec.FrameMemory:
			if len(m.Payload) > 8*memory.MaxFileBytes {
				return fmt.Errorf("oversized memory frame")
			}
			var r memory.Revision
			if err := json.Unmarshal(m.Payload, &r); err != nil {
				return err
			}
			if err := r.Validate(); err != nil {
				return err
			}
			if !r.Source.Shared {
				return fmt.Errorf("private memory revision on shared wire")
			}
			result = append(result, r)
		case codec.FrameSession, codec.FrameSessionV2, codec.FrameCheckpoint, codec.FrameCheckpointV2, codec.FrameMeta:
		default:
			return fmt.Errorf("unsupported wire member 0x%02x; upgrade before importing or exporting", m.Type)
		}
		return nil
	}
	for _, f := range frames {
		compressed := codec.ExtractFramePayload(body, f)
		if f.Type == codec.FrameBatch {
			members, err := dec.DecodeBatch(compressed)
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				if err := check(m); err != nil {
					return nil, err
				}
			}
			continue
		}
		var payload []byte
		if f.Type == codec.FrameMemory {
			payload, err = dec.DecodePayload(compressed)
			if err != nil {
				return nil, err
			}
		}
		if err := check(codec.BatchMember{Type: f.Type, Payload: payload}); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func guardMemoryReexport(d *sql.DB, body []byte) error {
	revisions, err := readMemoryFrames(body)
	if err != nil {
		return err
	}
	for _, r := range revisions {
		var n int
		if err := d.QueryRow(`SELECT count(*) FROM memory_revisions m JOIN checkpoints c ON c.id=m.checkpoint_id WHERE m.id=?`, r.ID).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("memory revision %s is not in the local ledger; sync before re-export", r.ID)
		}
	}
	return nil
}

// validateMemoryAssociations prevents a valid hash from asserting a checkpoint
// or code-commit link that the surrounding wire history does not contain.
func validateMemoryAssociations(body []byte, dict *codec.Dict, revisions []memory.Revision) error {
	if len(revisions) == 0 {
		return nil
	}
	frames, err := codec.ScanFrames(body)
	if err != nil {
		return err
	}
	dec, err := codec.NewDecoder()
	if err != nil {
		return err
	}
	defer dec.Close()
	type association struct{ sha, owner string }
	checkpoints := map[string]association{}
	add := func(cf *codec.CheckpointFrame) error {
		id, err := dict.Get(codec.NSSessions, cf.CheckpointRef)
		if err != nil {
			return err
		}
		owner, err := dict.Get(codec.NSEmails, cf.EmailRef)
		if err != nil {
			return err
		}
		value := association{cf.GitSHA, owner}
		if old, ok := checkpoints[id]; ok && old != value {
			return fmt.Errorf("conflicting memory checkpoint %s", id)
		}
		checkpoints[id] = value
		return nil
	}
	for _, f := range frames {
		compressed := codec.ExtractFramePayload(body, f)
		switch f.Type {
		case codec.FrameCheckpoint, codec.FrameCheckpointV2:
			cf, err := dec.DecodeCheckpointFrame(compressed)
			if err != nil {
				return err
			}
			if err := add(cf); err != nil {
				return err
			}
		case codec.FrameBatch:
			members, err := dec.DecodeBatch(compressed)
			if err != nil {
				return err
			}
			for _, m := range members {
				if m.Type == codec.FrameCheckpoint || m.Type == codec.FrameCheckpointV2 {
					cf, err := codec.DecodeCheckpointPayload(m.Payload)
					if err != nil {
						return err
					}
					if err := add(cf); err != nil {
						return err
					}
				}
			}
		}
	}
	for _, r := range revisions {
		a, ok := checkpoints[r.CheckpointID]
		if !ok || a.sha != r.GitSHA || a.owner != r.Source.Owner {
			return fmt.Errorf("memory checkpoint association mismatch for %s", r.ID)
		}
	}
	return nil
}
