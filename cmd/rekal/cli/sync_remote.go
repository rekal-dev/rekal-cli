package cli

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rekal-dev/cli/cmd/rekal/cli/codec"
)

// fetchRemoteRekalRefs fetches all rekal/* branches from origin.
// Non-fatal: returns nil if no remote or fetch fails.
func fetchRemoteRekalRefs(gitRoot string) error {
	// Check if remote is configured.
	if err := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin").Run(); err != nil {
		return nil // no remote configured
	}

	cmd := exec.Command("git", "-C", gitRoot, "fetch", "origin", "refs/heads/rekal/*:refs/remotes/origin/rekal/*")
	cmd.Stdin = nil
	_ = cmd.Run() // non-fatal
	return nil
}

// listRemoteRekalBranches returns remote rekal branch refs, excluding the current user's branch.
func listRemoteRekalBranches(gitRoot string) ([]string, error) {
	out, err := exec.Command("git", "-C", gitRoot,
		"for-each-ref", "--format=%(refname:short)", "refs/remotes/origin/rekal/",
	).Output()
	if err != nil {
		return nil, nil // no remote refs
	}

	selfBranch := "origin/" + rekalBranchName()

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

// importBranchToIndex decodes wire format from a remote branch and inserts
// sessions and checkpoints directly into the index DB tables.
// Tool calls are skipped for remote data.
// Returns the number of sessions imported.
func importBranchToIndex(gitRoot string, indexDB *sql.DB, remoteBranch string) (int, error) {
	bodyData := gitShowFile(gitRoot, remoteBranch, "rekal.body")
	if len(bodyData) <= 9 {
		return 0, nil
	}

	dictData := gitShowFile(gitRoot, remoteBranch, "dict.bin")
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

	// Track session → checkpoint mapping for updating facets.
	type cpInfo struct {
		checkpointID string
		gitSHA       string
		fileCount    int
	}
	sessionCheckpoints := make(map[string]*cpInfo)

	var imported int

	for _, fs := range frames {
		compressed := codec.ExtractFramePayload(bodyData, fs)

		switch fs.Type {
		case codec.FrameSession:
			sf, err := dec.DecodeSessionFrame(compressed)
			if err != nil {
				continue
			}

			sessionID, err := dict.Get(codec.NSSessions, sf.SessionRef)
			if err != nil {
				continue
			}

			email, _ := dict.Get(codec.NSEmails, sf.EmailRef)
			actorType := "human"
			if sf.ActorType == codec.ActorAgent {
				actorType = "agent"
			}

			branch := ""
			if len(sf.Turns) > 0 {
				branch, _ = dict.Get(codec.NSBranches, sf.Turns[0].BranchRef)
			}

			capturedAt := sf.CapturedAt.UTC().Format(time.RFC3339)

			// Insert turns into turns_ft.
			for i, t := range sf.Turns {
				role := "human"
				if t.Role == codec.RoleAssistant {
					role = "assistant"
				}
				if _, err := indexDB.Exec(
					`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
					 VALUES ($1, $2, $3, $4, $5, $6)`,
					newID(), sessionID, i, role, t.Text, "",
				); err != nil {
					return imported, fmt.Errorf("insert turn_ft: %w", err)
				}
			}

			// Insert session_facets.
			if _, err := indexDB.Exec(
				`INSERT INTO session_facets (
					session_id, user_email, git_branch, actor_type, agent_id,
					captured_at, turn_count, tool_call_count, file_count
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				sessionID, email, branch, actorType, "",
				capturedAt, len(sf.Turns), 0, 0,
			); err != nil {
				return imported, fmt.Errorf("insert session_facet: %w", err)
			}

			imported++

		case codec.FrameCheckpoint:
			cf, err := dec.DecodeCheckpointFrame(compressed)
			if err != nil {
				continue
			}

			checkpointID, err := dict.Get(codec.NSSessions, cf.CheckpointRef)
			if err != nil {
				continue
			}

			// Insert files_index.
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
						return imported, fmt.Errorf("insert files_index: %w", err)
					}
				}

				sessionCheckpoints[sid] = &cpInfo{
					checkpointID: checkpointID,
					gitSHA:       cf.GitSHA,
					fileCount:    len(cf.Files),
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
