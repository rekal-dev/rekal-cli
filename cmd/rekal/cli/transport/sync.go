package transport

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/ids"
)

// FetchRemoteRekalRefs fetches all rekal/* branches from origin.
// Non-fatal: returns nil if no remote or fetch fails.
func FetchRemoteRekalRefs(gitRoot string) error {
	// Check if remote is configured.
	if err := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin").Run(); err != nil {
		return nil // no remote configured
	}

	cmd := exec.Command("git", "-C", gitRoot, "fetch", "origin", "refs/heads/rekal/*:refs/remotes/origin/rekal/*")
	cmd.Stdin = nil
	_ = cmd.Run() // non-fatal
	return nil
}

// ListRemoteRekalBranches returns remote rekal branch refs, excluding the current user's branch.
func ListRemoteRekalBranches(gitRoot string) ([]string, error) {
	out, err := exec.Command("git", "-C", gitRoot,
		"for-each-ref", "--format=%(refname:short)", "refs/remotes/origin/rekal/",
	).Output()
	if err != nil {
		return nil, nil // no remote refs
	}

	selfBranch := "origin/" + gitx.RekalBranchName()

	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == selfBranch {
			continue
		}
		branches = append(branches, line)
	}
	return branches, nil
}

// ImportBranchToIndex decodes wire format from a remote branch and inserts
// sessions and checkpoints directly into the index DB tables.
// Tool calls are skipped for remote data.
// Returns the number of sessions imported.
func ImportBranchToIndex(gitRoot string, indexDB *sql.DB, remoteBranch string) (int, error) {
	bodyData := gitx.ShowFile(gitRoot, remoteBranch, codec.BodyFilename)
	if len(bodyData) <= 9 {
		return 0, nil
	}

	dictData := gitx.ShowFile(gitRoot, remoteBranch, codec.DictFilename)
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

	// Track session → checkpoint mapping for updating facets.
	sessionCheckpoints := make(map[string]*cpInfo)

	var imported int

	handleSession := func(sf *codec.SessionFrame) error {
		did, err := indexSessionFrame(indexDB, dict, sf, newID)
		if err != nil {
			return err
		}
		if did {
			imported++
		}
		return nil
	}
	handleCheckpoint := func(cf *codec.CheckpointFrame) error {
		return indexCheckpointFrame(indexDB, dict, cf, sessionCheckpoints)
	}

	for _, fs := range frames {
		compressed := codec.ExtractFramePayload(bodyData, fs)

		switch fs.Type {
		case codec.FrameSession, codec.FrameSessionV2:
			sf, err := dec.DecodeSessionFrame(compressed)
			if err != nil {
				continue
			}
			if err := handleSession(sf); err != nil {
				return imported, err
			}

		case codec.FrameCheckpoint, codec.FrameCheckpointV2:
			cf, err := dec.DecodeCheckpointFrame(compressed)
			if err != nil {
				continue
			}
			if err := handleCheckpoint(cf); err != nil {
				return imported, err
			}

		case codec.FrameBatch:
			members, err := dec.DecodeBatch(compressed)
			if err != nil {
				continue
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
					if err := handleCheckpoint(cf); err != nil {
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

	// Update session_facets with checkpoint info.
	for sid, cp := range sessionCheckpoints {
		if _, err := indexDB.Exec(
			`UPDATE session_facets SET checkpoint_id = $1, git_sha = $2, file_count = $3
			 WHERE session_id = $4`,
			cp.checkpointID, cp.gitSHA, cp.fileCount, sid,
		); err != nil {
			// Non-fatal: session may not have been imported (already existed).
			continue
		}
	}

	return imported, nil
}

// cpInfo carries the checkpoint fields folded back into session_facets after
// all frames are imported.
type cpInfo struct {
	checkpointID string
	gitSHA       string
	fileCount    int
}

// indexSessionFrame inserts a decoded session's turns and facet row into the
// index DB. Reports whether a session was inserted; a dict miss is skipped
// (false, nil), only DB failures are hard errors. Shared by the standalone and
// batched frame paths.
func indexSessionFrame(indexDB *sql.DB, dict *codec.Dict, sf *codec.SessionFrame, newID func() string) (bool, error) {
	sessionID, err := dict.Get(codec.NSSessions, sf.SessionRef)
	if err != nil {
		return false, nil
	}

	// One conversation spanning several commits is linked to several
	// checkpoints and rides in each one's frame, so the same session arrives
	// more than once; inserting it twice violates session_facets' primary key
	// and aborts the whole import. (Before capture learned to append, every
	// checkpoint carried a distinct session and this could not arise.)
	//
	// Which copy wins matters. Frames from a single export carry identical
	// turns, but frames written by *different* pushes do not: the author
	// checkpointed at one commit, pushed, kept talking, and pushed again. The
	// later frame is longer, and skipping it would leave the reader holding a
	// truncated conversation forever. Keep the longest, never the first.
	var indexedTurns int
	if err := indexDB.QueryRow(
		`SELECT count(*) FROM turns_ft WHERE session_id = $1`, sessionID,
	).Scan(&indexedTurns); err != nil {
		return false, fmt.Errorf("check indexed turns: %w", err)
	}
	if indexedTurns > 0 {
		if len(sf.Turns) <= indexedTurns {
			return false, nil
		}
		for _, tbl := range []string{"turns_ft", "session_facets"} {
			if _, err := indexDB.Exec(
				fmt.Sprintf("DELETE FROM %s WHERE session_id = $1", tbl), sessionID,
			); err != nil {
				return false, fmt.Errorf("replace stale %s: %w", tbl, err)
			}
		}
	}

	email, _ := dict.Get(codec.NSEmails, sf.EmailRef)
	actorType := "human"
	agentID := ""
	if sf.ActorType == codec.ActorAgent {
		actorType = "agent"
		agentID, _ = dict.Get(codec.NSEmails, sf.AgentIDRef)
	}

	branch := ""
	if len(sf.Turns) > 0 {
		branch, _ = dict.Get(codec.NSBranches, sf.Turns[0].BranchRef)
	}

	capturedAt := sf.CapturedAt.UTC().Format(time.RFC3339)

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
		// Frames encoded by binaries that predate RoleSummary carry
		// compaction summaries as plain human turns; reclassify by the same
		// fingerprint the index rebuild uses (see db.SummaryFingerprint).
		// The wire carries no source and every wire import is stored as
		// source 'claude' (InsertSessionMeta's default), so applying the
		// fingerprint unscoped here matches the source-scoped rule used on
		// the data side.
		if role == "human" && strings.HasPrefix(t.Text, db.SummaryFingerprint) {
			role = "summary"
		}
		if _, err := indexDB.Exec(
			`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			newID(), sessionID, i, role, t.Text, "",
		); err != nil {
			return false, fmt.Errorf("insert turn_ft: %w", err)
		}
	}

	// Harness metadata is only present on v2 frames; for v1 frames every field
	// is zero and stored as NULL.
	parentID := ""
	if sf.HasParent {
		parentID, _ = dict.Get(codec.NSSessions, sf.ParentRef)
	}

	if _, err := indexDB.Exec(
		`INSERT INTO session_facets (
			session_id, user_email, git_branch, actor_type, agent_id,
			captured_at, turn_count, tool_call_count, file_count,
			parent_session_id, team_name, workflow_name,
			agent_type, description, spawn_depth
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		sessionID, email, branch, actorType, agentID,
		capturedAt, len(sf.Turns), 0, 0,
		db.NullIfEmpty(parentID), db.NullIfEmpty(sf.TeamName), db.NullIfEmpty(sf.WorkflowName),
		db.NullIfEmpty(sf.AgentType), db.NullIfEmpty(sf.Description), db.NullIfZero(sf.SpawnDepth),
	); err != nil {
		return false, fmt.Errorf("insert session_facet: %w", err)
	}

	return true, nil
}

// indexCheckpointFrame inserts a checkpoint's files_index rows and records the
// checkpoint fields to fold back into session_facets afterward.
func indexCheckpointFrame(indexDB *sql.DB, dict *codec.Dict, cf *codec.CheckpointFrame, sessionCheckpoints map[string]*cpInfo) error {
	checkpointID, err := dict.Get(codec.NSSessions, cf.CheckpointRef)
	if err != nil {
		return nil
	}

	for _, ref := range cf.SessionRefs {
		sid, err := dict.Get(codec.NSSessions, ref)
		if err != nil {
			continue
		}
		for _, f := range cf.Files {
			filePath, _ := dict.Get(codec.NSPaths, f.PathRef)
			changeType := string(f.ChangeType)
			if _, err := indexDB.Exec(
				`INSERT INTO files_index (checkpoint_id, session_id, file_path, change_type)
				 VALUES ($1, $2, $3, $4)`,
				checkpointID, sid, filePath, changeType,
			); err != nil {
				return fmt.Errorf("insert files_index: %w", err)
			}
		}

		sessionCheckpoints[sid] = &cpInfo{
			checkpointID: checkpointID,
			gitSHA:       cf.GitSHA,
			fileCount:    len(cf.Files),
		}
	}
	return nil
}
