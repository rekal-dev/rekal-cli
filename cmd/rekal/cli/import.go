package cli

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
)

// importBranch decodes wire format from an orphan branch and imports
// sessions + checkpoints into DuckDB. Returns the number of sessions imported.
// Deduplicates by session ID and checkpoint ID.
func importBranch(gitRoot string, dataDB *sql.DB, branch string) (int, error) {
	bodyData := gitShowFile(gitRoot, branch, "rekal.body")
	if len(bodyData) <= 9 {
		return 0, nil // empty body (header only)
	}

	dictData := gitShowFile(gitRoot, branch, "dict.bin")
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

	entropy := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	newID := func() string {
		return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
	}

	var imported int

	for _, fs := range frames {
		compressed := codec.ExtractFramePayload(bodyData, fs)

		switch fs.Type {
		case codec.FrameSession, codec.FrameSessionV2:
			sf, err := dec.DecodeSessionFrame(compressed)
			if err != nil {
				continue // skip malformed frames
			}

			sessionID, err := dict.Get(codec.NSSessions, sf.SessionRef)
			if err != nil {
				continue
			}

			// Dedup by session ID.
			exists, err := db.SessionExistsByID(dataDB, sessionID)
			if err != nil {
				return imported, fmt.Errorf("check session: %w", err)
			}
			if exists {
				continue
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

			// Harness metadata is only present on v2 frames; for v1 frames
			// every field is zero and stored as NULL.
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
				return imported, fmt.Errorf("insert session: %w", err)
			}

			// Insert turns.
			for i, t := range sf.Turns {
				role := "human"
				switch t.Role {
				case codec.RoleAssistant:
					role = "assistant"
				case codec.RoleHumanSteering:
					role = "human_steering"
				}
				if err := db.InsertTurn(dataDB, newID(), sessionID, i, role, t.Text, ""); err != nil {
					return imported, fmt.Errorf("insert turn: %w", err)
				}
			}

			// Insert tool calls.
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
					return imported, fmt.Errorf("insert tool_call: %w", err)
				}
			}

			imported++

		case codec.FrameCheckpoint, codec.FrameCheckpointV2:
			cf, err := dec.DecodeCheckpointFrame(compressed)
			if err != nil {
				continue
			}

			checkpointID, err := dict.Get(codec.NSSessions, cf.CheckpointRef)
			if err != nil {
				continue
			}

			// Dedup by checkpoint ID.
			exists, err := db.CheckpointExists(dataDB, checkpointID)
			if err != nil {
				return imported, fmt.Errorf("check checkpoint: %w", err)
			}
			if exists {
				continue
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
				return imported, fmt.Errorf("insert checkpoint: %w", err)
			}

			// Insert files_touched.
			for _, f := range cf.Files {
				filePath, _ := dict.Get(codec.NSPaths, f.PathRef)
				changeType := string(f.ChangeType)
				if err := db.InsertFileTouched(dataDB, newID(), checkpointID, filePath, changeType); err != nil {
					return imported, fmt.Errorf("insert file_touched: %w", err)
				}
			}

			// Insert checkpoint_sessions junction rows.
			for _, ref := range cf.SessionRefs {
				sessionID, err := dict.Get(codec.NSSessions, ref)
				if err != nil {
					continue
				}
				// Only link if the session exists in DB.
				exists, _ := db.SessionExistsByID(dataDB, sessionID)
				if exists {
					if err := db.InsertCheckpointSession(dataDB, checkpointID, sessionID); err != nil {
						return imported, fmt.Errorf("insert checkpoint_session: %w", err)
					}
				}
			}

			// Mark as exported since it came from wire format.
			_ = db.MarkCheckpointsExported(dataDB, []string{checkpointID})

		case codec.FrameMeta:
			// Skip meta frames during import.
			continue
		}
	}

	return imported, nil
}
