package cli

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rekal-dev/cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/cli/cmd/rekal/cli/db"
)

// exportNewFrames reads existing wire format from the orphan branch, appends
// frames for any unexported checkpoints from DuckDB, and returns the updated
// body + dict. Returns (nil, nil, nil) if there are no unexported checkpoints.
func exportNewFrames(gitRoot string) ([]byte, []byte, error) {
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	checkpoints, err := db.QueryUnexportedCheckpoints(dataDB)
	if err != nil {
		return nil, nil, fmt.Errorf("query unexported checkpoints: %w", err)
	}
	if len(checkpoints) == 0 {
		return nil, nil, nil
	}

	// Load existing wire format from orphan branch.
	branch := rekalBranchName()
	bodyData := gitShowFile(gitRoot, branch, "rekal.body")
	dictData := gitShowFile(gitRoot, branch, "dict.bin")

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

	enc, err := codec.NewEncoder()
	if err != nil {
		return nil, nil, fmt.Errorf("create encoder: %w", err)
	}
	defer enc.Close()

	var exportedIDs []string

	for _, cp := range checkpoints {
		// Query sessions linked to this checkpoint.
		sessionIDs, err := db.QuerySessionsByCheckpoint(dataDB, cp.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("query sessions for checkpoint %s: %w", cp.ID, err)
		}

		var sessionRefs []uint64

		for _, sid := range sessionIDs {
			sess, err := db.QuerySession(dataDB, sid)
			if err != nil {
				return nil, nil, fmt.Errorf("query session %s: %w", sid, err)
			}
			turns, err := db.QueryTurns(dataDB, sid)
			if err != nil {
				return nil, nil, fmt.Errorf("query turns for %s: %w", sid, err)
			}
			toolCalls, err := db.QueryToolCalls(dataDB, sid)
			if err != nil {
				return nil, nil, fmt.Errorf("query tool_calls for %s: %w", sid, err)
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
			}

			// Build turn records with delta timestamps.
			var prevTs time.Time
			for _, t := range turns {
				role := codec.RoleHuman
				if t.Role == "assistant" {
					role = codec.RoleAssistant
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

			body = codec.AppendFrame(body, enc.EncodeSessionFrame(sf))
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
			return nil, nil, fmt.Errorf("query files_touched for %s: %w", cp.ID, err)
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
		body = codec.AppendFrame(body, enc.EncodeCheckpointFrame(cf))

		exportedIDs = append(exportedIDs, cp.ID)
	}

	// Append meta frame.
	existingFrames, _ := codec.ScanFrames(body)
	nFrames := uint32(len(existingFrames))

	email := gitConfigValue("user.email")
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
	body = codec.AppendFrame(body, enc.EncodeMetaFrame(mf))

	// Mark checkpoints as exported.
	if err := db.MarkCheckpointsExported(dataDB, exportedIDs); err != nil {
		return nil, nil, fmt.Errorf("mark exported: %w", err)
	}

	return body, dict.Encode(), nil
}

// commitWireFormat commits rekal.body and dict.bin to the orphan branch.
// Returns the new commit SHA.
func commitWireFormat(gitRoot string, bodyData, dictData []byte) (string, error) {
	branch := rekalBranchName()

	// Get the current tip of the orphan branch.
	parentOut, err := exec.Command("git", "-C", gitRoot, "rev-parse", branch).Output()
	if err != nil {
		return "", fmt.Errorf("resolve branch %s: %w", branch, err)
	}
	parent := strings.TrimSpace(string(parentOut))

	bodyHash, err := gitHashObject(gitRoot, bodyData)
	if err != nil {
		return "", fmt.Errorf("hash rekal.body: %w", err)
	}
	dictHash, err := gitHashObject(gitRoot, dictData)
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
