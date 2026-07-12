package transport

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/ids"
)

// ImportBranch decodes wire format from an orphan branch and imports
// sessions + checkpoints into DuckDB. Returns the number of sessions imported.
// Deduplicates by session ID and checkpoint ID.
func ImportBranch(gitRoot string, dataDB *sql.DB, branch string) (int, error) {
	bodyData := gitx.ShowFile(gitRoot, branch, codec.BodyFilename)
	if len(bodyData) <= 9 {
		return 0, nil // empty body (header only)
	}

	dictData := gitx.ShowFile(gitRoot, branch, codec.DictFilename)
	if len(dictData) == 0 {
		return 0, nil
	}

	dict, err := codec.LoadDict(dictData)
	if err != nil {
		return 0, fmt.Errorf("load dict: %w", err)
	}

	frames, err := codec.ScanFrames(bodyData)
	if err != nil {
		return 0, fmt.Errorf("scan frames: %w", err)
	}

	dec, err := codec.NewDecoder()
	if err != nil {
		return 0, fmt.Errorf("create decoder: %w", err)
	}
	defer dec.Close()

	newID := ids.NewULIDFunc()
	var imported int

	// handleSession/handleCheckpoint work off already-parsed frames so both the
	// standalone-frame path and the batch-member path share the same insert
	// logic. handleSession reports whether a new session was inserted.
	handleSession := func(sf *codec.SessionFrame) error {
		did, err := importSessionFrame(dataDB, dict, sf, newID)
		if err != nil {
			return err
		}
		if did {
			imported++
		}
		return nil
	}

	for _, fs := range frames {
		compressed := codec.ExtractFramePayload(bodyData, fs)

		switch fs.Type {
		case codec.FrameSession, codec.FrameSessionV2:
			sf, err := dec.DecodeSessionFrame(compressed)
			if err != nil {
				continue // skip malformed frames
			}
			if err := handleSession(sf); err != nil {
				return imported, err
			}

		case codec.FrameCheckpoint, codec.FrameCheckpointV2:
			cf, err := dec.DecodeCheckpointFrame(compressed)
			if err != nil {
				continue
			}
			if err := importCheckpointFrame(dataDB, dict, cf, newID); err != nil {
				return imported, err
			}

		case codec.FrameBatch:
			members, err := dec.DecodeBatch(compressed)
			if err != nil {
				continue // a corrupt batch is skipped whole, like any bad frame
			}
			for _, m := range members {
				switch m.Type {
				case codec.FrameSession, codec.FrameSessionV2:
					sf, err := codec.DecodeSessionPayload(m.Payload)
					if err != nil {
						continue
					}
					if err := handleSession(sf); err != nil {
						return imported, err
					}
				case codec.FrameCheckpoint, codec.FrameCheckpointV2:
					cf, err := codec.DecodeCheckpointPayload(m.Payload)
					if err != nil {
						continue
					}
					if err := importCheckpointFrame(dataDB, dict, cf, newID); err != nil {
						return imported, err
					}
				case codec.FrameMeta:
					continue
				}
			}

		case codec.FrameMeta:
			continue
		}
	}

	return imported, nil
}

// importSessionFrame inserts a decoded session (and its turns and tool calls)
// into data.db, deduplicating by session ID. Returns whether a new session was
// inserted; dict misses and already-present sessions are skipped (false, nil),
// only DB failures are hard errors.
func importSessionFrame(dataDB *sql.DB, dict *codec.Dict, sf *codec.SessionFrame, newID func() string) (bool, error) {
	sessionID, err := dict.Get(codec.NSSessions, sf.SessionRef)
	if err != nil {
		return false, nil
	}

	exists, err := db.SessionExistsByID(dataDB, sessionID)
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}
	if exists {
		return false, nil
	}

	email, _ := dict.Get(codec.NSEmails, sf.EmailRef)
	actorType := "human"
	agentID := ""
	if sf.ActorType == codec.ActorAgent {
		actorType = "agent"
		agentID, _ = dict.Get(codec.NSEmails, sf.AgentIDRef)
	}

	// Determine branch from first turn's BranchRef (all turns share the same branch).
	branch := ""
	if len(sf.Turns) > 0 {
		branch, _ = dict.Get(codec.NSBranches, sf.Turns[0].BranchRef)
	}

	sessionHash := "wire:" + sessionID
	capturedAt := sf.CapturedAt.UTC().Format(time.RFC3339)

	// Harness metadata is only present on v2 frames; for v1 frames every field
	// is zero and stored as NULL.
	parentID := ""
	if sf.HasParent {
		parentID, _ = dict.Get(codec.NSSessions, sf.ParentRef)
	}
	if err := db.InsertSessionMeta(dataDB, sessionID, parentID, sessionHash, actorType, agentID, email, branch, capturedAt, "", db.SessionMetaFields{
		TeamName:     sf.TeamName,
		WorkflowName: sf.WorkflowName,
		AgentType:    sf.AgentType,
		Description:  sf.Description,
		SpawnDepth:   sf.SpawnDepth,
	}); err != nil {
		return false, fmt.Errorf("insert session: %w", err)
	}

	for i, t := range sf.Turns {
		role := "human"
		switch t.Role {
		case codec.RoleAssistant:
			role = "assistant"
		case codec.RoleHumanSteering:
			role = "human_steering"
		case codec.RoleSummary:
			role = "summary"
		}
		if err := db.InsertTurn(dataDB, newID(), sessionID, i, role, t.Text, ""); err != nil {
			return false, fmt.Errorf("insert turn: %w", err)
		}
	}

	for i, tc := range sf.ToolCalls {
		toolName := codec.ToolName(tc.Tool)
		path := ""
		switch tc.PathFlag {
		case codec.PathDictRef:
			path, _ = dict.Get(codec.NSPaths, tc.PathRef)
		case codec.PathInline:
			path = tc.PathInline
		}
		if err := db.InsertToolCall(dataDB, newID(), sessionID, i, toolName, path, tc.CmdPrefix); err != nil {
			return false, fmt.Errorf("insert tool_call: %w", err)
		}
	}

	return true, nil
}

// importCheckpointFrame inserts a decoded checkpoint (and its files_touched and
// checkpoint_sessions rows) into data.db, deduplicating by checkpoint ID.
func importCheckpointFrame(dataDB *sql.DB, dict *codec.Dict, cf *codec.CheckpointFrame, newID func() string) error {
	checkpointID, err := dict.Get(codec.NSSessions, cf.CheckpointRef)
	if err != nil {
		return nil
	}

	exists, err := db.CheckpointExists(dataDB, checkpointID)
	if err != nil {
		return fmt.Errorf("check checkpoint: %w", err)
	}
	if exists {
		return nil
	}

	branchName, _ := dict.Get(codec.NSBranches, cf.BranchRef)
	email, _ := dict.Get(codec.NSEmails, cf.EmailRef)
	actorType := "human"
	agentID := ""
	if cf.ActorType == codec.ActorAgent {
		actorType = "agent"
		agentID, _ = dict.Get(codec.NSEmails, cf.AgentIDRef)
	}

	ts := cf.Timestamp.UTC().Format(time.RFC3339)
	if err := db.InsertCheckpoint(dataDB, checkpointID, cf.GitSHA, branchName, email, ts, actorType, agentID); err != nil {
		return fmt.Errorf("insert checkpoint: %w", err)
	}

	for _, f := range cf.Files {
		filePath, _ := dict.Get(codec.NSPaths, f.PathRef)
		changeType := string(f.ChangeType)
		if err := db.InsertFileTouched(dataDB, newID(), checkpointID, filePath, changeType); err != nil {
			return fmt.Errorf("insert file_touched: %w", err)
		}
	}

	for _, ref := range cf.SessionRefs {
		sessionID, err := dict.Get(codec.NSSessions, ref)
		if err != nil {
			continue
		}
		// Only link if the session exists in DB.
		exists, _ := db.SessionExistsByID(dataDB, sessionID)
		if exists {
			if err := db.InsertCheckpointSession(dataDB, checkpointID, sessionID); err != nil {
				return fmt.Errorf("insert checkpoint_session: %w", err)
			}
		}
	}

	// Mark as exported since it came from wire format.
	_ = db.MarkCheckpointsExported(dataDB, []string{checkpointID})
	return nil
}
