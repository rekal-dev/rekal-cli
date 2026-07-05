// Package transport encodes checkpoints to the append-only wire format on the
// rekal orphan branch and decodes them back — the git-side sync mechanism that
// keeps data.db thin on the wire. It sits above codec (frame format), db
// (storage), and gitx (git plumbing).
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
)

func ExportNewFrames(gitRoot string) ([]byte, []byte, []string, error) {
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	checkpoints, err := db.QueryUnexportedCheckpoints(dataDB)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query unexported checkpoints: %w", err)
	}
	if len(checkpoints) == 0 {
		return nil, nil, nil, nil
	}

	// Load existing wire format from orphan branch.
	branch := gitx.RekalBranchName()
	bodyData := gitx.ShowFile(gitRoot, branch, codec.BodyFilename)
	dictData := gitx.ShowFile(gitRoot, branch, codec.DictFilename)

	dict := codec.NewDict()
	if len(dictData) > 0 {
		loaded, err := codec.LoadDict(dictData)
		if err == nil {
			dict = loaded
		}
	}
	body := bodyData
	if len(body) == 0 {
		body = codec.NewBody()
	}

	return encodeCheckpointFrames(dataDB, body, dict, checkpoints)
}

// ExportAllFrames re-encodes every checkpoint in data.db into a fresh body
// and dict, ignoring the orphan branch's current contents and exported flags.
// Used by `rekal push --re-export` to regenerate a branch whose wire data is
// corrupt (frames written before the v2 count fix) or bloated (stale meta
// frames) — data.db is the source of truth, the branch is derived.
func ExportAllFrames(gitRoot string) ([]byte, []byte, []string, error) {
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	checkpoints, err := db.QueryAllCheckpoints(dataDB)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query checkpoints: %w", err)
	}
	if len(checkpoints) == 0 {
		return nil, nil, nil, nil
	}

	return encodeCheckpointFrames(dataDB, codec.NewBody(), codec.NewDict(), checkpoints)
}

// encodeCheckpointFrames appends session + checkpoint frames for the given
// checkpoints to body, interning strings into dict, and finishes with a meta
// frame. Returns the updated body, encoded dict, and the checkpoint IDs
// encoded — which the caller must only mark exported after the bytes are
// durably committed (see ExportNewFrames' doc comment).
func encodeCheckpointFrames(dataDB *sql.DB, body []byte, dict *codec.Dict, checkpoints []db.CheckpointRow) ([]byte, []byte, []string, error) {
	enc, err := codec.NewEncoder()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create encoder: %w", err)
	}
	defer enc.Close()

	var exportedIDs []string

	for _, cp := range checkpoints {
		// Query sessions linked to this checkpoint.
		sessionIDs, err := db.QuerySessionsByCheckpoint(dataDB, cp.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("query sessions for checkpoint %s: %w", cp.ID, err)
		}

		var sessionRefs []uint64

		for _, sid := range sessionIDs {
			sess, err := db.QuerySession(dataDB, sid)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("query session %s: %w", sid, err)
			}
			turns, err := db.QueryTurns(dataDB, sid)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("query turns for %s: %w", sid, err)
			}
			toolCalls, err := db.QueryToolCalls(dataDB, sid)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("query tool_calls for %s: %w", sid, err)
			}

			sessRef := dict.LookupOrAdd(codec.NSSessions, sid)
			emailRef := dict.LookupOrAdd(codec.NSEmails, sess.Email)
			branchRef := uint64(0)
			if sess.Branch != "" {
				branchRef = dict.LookupOrAdd(codec.NSBranches, sess.Branch)
			}

			actorType := codec.ActorHuman
			agentIDRef := uint64(0)
			if sess.ActorType == "agent" {
				actorType = codec.ActorAgent
				if sess.AgentID != "" {
					agentIDRef = dict.LookupOrAdd(codec.NSEmails, sess.AgentID)
				}
			}

			capturedAt, _ := time.Parse(time.RFC3339, sess.CapturedAt)
			sf := &codec.SessionFrame{
				SessionRef: sessRef,
				CapturedAt: capturedAt,
				EmailRef:   emailRef,
				ActorType:  actorType,
				AgentIDRef: agentIDRef,

				// Harness metadata rides the v2 payload; zero values cost
				// nothing on the wire (single flags byte).
				TeamName:     sess.TeamName,
				WorkflowName: sess.WorkflowName,
				AgentType:    sess.AgentType,
				Description:  sess.Description,
				SpawnDepth:   sess.SpawnDepth,
			}
			if sess.ParentSessionID != "" {
				sf.HasParent = true
				sf.ParentRef = dict.LookupOrAdd(codec.NSSessions, sess.ParentSessionID)
			}

			// Build turn records with delta timestamps.
			var prevTs time.Time
			for _, t := range turns {
				role := codec.RoleHuman
				switch t.Role {
				case "assistant":
					role = codec.RoleAssistant
				case "human_steering":
					role = codec.RoleHumanSteering
				}
				var tsDelta uint64
				if t.Ts != "" {
					ts, _ := time.Parse(time.RFC3339, t.Ts)
					if !prevTs.IsZero() && !ts.IsZero() {
						delta := ts.Sub(prevTs)
						if delta > 0 {
							tsDelta = uint64(delta.Seconds())
						}
					}
					prevTs = ts
				}
				sf.Turns = append(sf.Turns, codec.TurnRecord{
					Role:      role,
					TsDelta:   tsDelta,
					BranchRef: branchRef,
					Text:      t.Content,
				})
			}

			// Build tool call records.
			for _, tc := range toolCalls {
				toolCode := codec.ToolCode(tc.Tool)
				tcr := codec.ToolCallRecord{
					Tool: toolCode,
				}
				if tc.Path == "" {
					tcr.PathFlag = codec.PathNull
				} else {
					pathRef := dict.LookupOrAdd(codec.NSPaths, tc.Path)
					tcr.PathFlag = codec.PathDictRef
					tcr.PathRef = pathRef
				}
				tcr.CmdPrefix = tc.CmdPrefix
				sf.ToolCalls = append(sf.ToolCalls, tcr)
			}

			frame, err := enc.EncodeSessionFrame(sf)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("encode session %s: %w", sid, err)
			}
			body = codec.AppendFrame(body, frame)
			sessionRefs = append(sessionRefs, sessRef)
		}

		// Build checkpoint frame.
		cpRef := dict.LookupOrAdd(codec.NSSessions, cp.ID)
		cpBranchRef := dict.LookupOrAdd(codec.NSBranches, cp.GitBranch)
		cpEmailRef := dict.LookupOrAdd(codec.NSEmails, cp.Email)

		cpTs, _ := time.Parse(time.RFC3339, cp.Ts)

		actorType := codec.ActorHuman
		agentIDRef := uint64(0)
		if cp.ActorType == "agent" {
			actorType = codec.ActorAgent
			if cp.AgentID != "" {
				agentIDRef = dict.LookupOrAdd(codec.NSEmails, cp.AgentID)
			}
		}

		// Query files touched.
		filesTouched, err := db.QueryFilesTouched(dataDB, cp.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("query files_touched for %s: %w", cp.ID, err)
		}
		var fileRecords []codec.FileTouchedRecord
		for _, ft := range filesTouched {
			pathRef := dict.LookupOrAdd(codec.NSPaths, ft.Path)
			changeType := byte('M')
			if len(ft.ChangeType) > 0 {
				changeType = ft.ChangeType[0]
			}
			fileRecords = append(fileRecords, codec.FileTouchedRecord{
				PathRef:    pathRef,
				ChangeType: changeType,
			})
		}

		cf := &codec.CheckpointFrame{
			CheckpointRef: cpRef,
			GitSHA:        cp.GitSHA,
			BranchRef:     cpBranchRef,
			EmailRef:      cpEmailRef,
			Timestamp:     cpTs,
			ActorType:     actorType,
			AgentIDRef:    agentIDRef,
			SessionRefs:   sessionRefs,
			Files:         fileRecords,
		}
		frame, err := enc.EncodeCheckpointFrame(cf)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("encode checkpoint %s: %w", cp.ID, err)
		}
		body = codec.AppendFrame(body, frame)

		exportedIDs = append(exportedIDs, cp.ID)
	}

	// Append meta frame.
	existingFrames, _ := codec.ScanFrames(body)
	nFrames := uint32(len(existingFrames))

	email := gitx.ConfigValue("user.email")
	metaEmailRef := dict.LookupOrAdd(codec.NSEmails, email)

	mf := &codec.MetaFrame{
		FormatVersion: 0x01,
		EmailRef:      metaEmailRef,
		CheckpointSHA: strings.Repeat("0", 40), // placeholder
		Timestamp:     time.Now().UTC(),
		NSessions:     uint32(dict.Len(codec.NSSessions)),
		NCheckpoints:  uint32(len(exportedIDs)),
		NFrames:       nFrames + 1, // +1 for this meta frame
		NDictEntries:  uint32(dict.TotalEntries()),
	}
	metaFrame, err := enc.EncodeMetaFrame(mf)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encode meta frame: %w", err)
	}
	body = codec.AppendFrame(body, metaFrame)

	// Checkpoints are NOT marked exported here — see the doc comment above.
	// The caller marks them only once CommitWireFormat has durably committed
	// this body/dict to the orphan branch.
	return body, dict.Encode(), exportedIDs, nil
}

// CommitWireFormat commits rekal.body and dict.bin to the orphan branch.
// Returns the new commit SHA.
func CommitWireFormat(gitRoot string, bodyData, dictData []byte) (string, error) {
	branch := gitx.RekalBranchName()

	// Get the current tip of the orphan branch.
	parentOut, err := exec.Command("git", "-C", gitRoot, "rev-parse", branch).Output()
	if err != nil {
		return "", fmt.Errorf("resolve branch %s: %w", branch, err)
	}
	parent := strings.TrimSpace(string(parentOut))

	bodyHash, err := gitx.HashObject(gitRoot, bodyData)
	if err != nil {
		return "", fmt.Errorf("hash rekal.body: %w", err)
	}
	dictHash, err := gitx.HashObject(gitRoot, dictData)
	if err != nil {
		return "", fmt.Errorf("hash dict.bin: %w", err)
	}

	treeEntry := fmt.Sprintf("100644 blob %s\tdict.bin\n100644 blob %s\trekal.body\n", dictHash, bodyHash)
	mktreeCmd := exec.Command("git", "-C", gitRoot, "mktree")
	mktreeCmd.Stdin = strings.NewReader(treeEntry)
	treeOut, err := mktreeCmd.Output()
	if err != nil {
		return "", fmt.Errorf("mktree: %w", err)
	}
	treeHash := strings.TrimSpace(string(treeOut))

	// Use the HEAD commit message from the main branch.
	msg := "rekal: checkpoint"
	if headMsg, err := exec.Command("git", "-C", gitRoot, "log", "-1", "--format=%s", "HEAD").Output(); err == nil {
		if m := strings.TrimSpace(string(headMsg)); m != "" {
			msg = m
		}
	}

	commitOut, err := exec.Command("git", "-C", gitRoot,
		"commit-tree", treeHash, "-p", parent, "-m", msg,
	).Output()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	commitSHA := strings.TrimSpace(string(commitOut))

	if err := exec.Command("git", "-C", gitRoot, "update-ref", "refs/heads/"+branch, commitSHA).Run(); err != nil {
		return "", fmt.Errorf("update-ref: %w", err)
	}

	return commitSHA, nil
}

// EnsureOrphanBranch creates or fetches the local rekal orphan branch.
// If the branch exists locally, it's left as-is.
// If it exists on the remote, it's fetched.
// Otherwise, a new orphan branch is created with empty rekal.body and dict.bin.
func EnsureOrphanBranch(gitRoot string) error {
	branch := gitx.RekalBranchName()

	// Check if local branch already exists.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", branch).Run(); err == nil {
		return nil // already exists locally
	}

	// Check if remote branch exists and fetch it.
	remote := "origin"
	remoteBranch := remote + "/" + branch
	// Fetch the specific branch (ignore errors — remote may not exist or branch may not exist).
	_ = exec.Command("git", "-C", gitRoot, "fetch", remote, branch).Run()

	// If remote branch now exists locally as a remote-tracking branch, create local from it.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", remoteBranch).Run(); err == nil {
		return exec.Command("git", "-C", gitRoot, "branch", branch, remoteBranch).Run()
	}

	// Create new orphan branch with initial wire format files.
	bodyData := codec.NewBody()
	dictData := codec.NewDict().Encode()

	bodyHash, err := gitx.HashObject(gitRoot, bodyData)
	if err != nil {
		return fmt.Errorf("hash rekal.body: %w", err)
	}
	dictHash, err := gitx.HashObject(gitRoot, dictData)
	if err != nil {
		return fmt.Errorf("hash dict.bin: %w", err)
	}

	treeEntry := fmt.Sprintf("100644 blob %s\tdict.bin\n100644 blob %s\trekal.body\n", dictHash, bodyHash)
	mktreeCmd := exec.Command("git", "-C", gitRoot, "mktree")
	mktreeCmd.Stdin = strings.NewReader(treeEntry)
	treeOut, err := mktreeCmd.Output()
	if err != nil {
		return fmt.Errorf("mktree: %w", err)
	}
	treeHash := strings.TrimSpace(string(treeOut))

	commitOut, err := exec.Command("git", "-C", gitRoot,
		"commit-tree", treeHash, "-m", "rekal: initialize checkpoint branch",
	).Output()
	if err != nil {
		return fmt.Errorf("create initial commit: %w", err)
	}
	commitHash := strings.TrimSpace(string(commitOut))

	return exec.Command("git", "-C", gitRoot, "update-ref", "refs/heads/"+branch, commitHash).Run()
}
