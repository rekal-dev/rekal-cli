package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rekal-dev/cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/cli/cmd/rekal/cli/session"
	"github.com/spf13/cobra"
)

func newCheckpointCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checkpoint",
		Short: "Capture the current session after a commit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true

			gitRoot, err := EnsureGitRoot()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}
			if err := EnsureInitDone(gitRoot); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return NewSilentError(err)
			}

			return runCheckpoint(cmd, gitRoot)
		},
	}
}

func runCheckpoint(cmd *cobra.Command, gitRoot string) error {
	// Find session directory for this repo.
	sessionDir := session.FindSessionDir(gitRoot)
	if sessionDir == "" {
		return nil
	}

	files, err := session.FindSessionFiles(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("find session files: %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	// Open data DB.
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	email := gitConfigValue("user.email")
	entropy := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	newID := func() string {
		return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
	}

	var sessionIDs []string
	var inserted int

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if len(data) == 0 {
			continue
		}

		hash := sha256Hex(data)

		exists, err := db.SessionExistsByHash(dataDB, hash)
		if err != nil {
			return fmt.Errorf("dedup check: %w", err)
		}
		if exists {
			continue
		}

		payload, err := session.ParseTranscript(data)
		if err != nil {
			continue
		}

		if len(payload.Turns) == 0 && len(payload.ToolCalls) == 0 {
			continue
		}

		sessionID := newID()
		capturedAt := time.Now().UTC().Format(time.RFC3339)

		// Insert session row.
		if err := db.InsertSession(
			dataDB, sessionID, "", hash,
			payload.ActorType, payload.AgentID, email, payload.Branch, capturedAt,
		); err != nil {
			return fmt.Errorf("insert session: %w", err)
		}

		// Insert turns.
		for i, t := range payload.Turns {
			ts := ""
			if !t.Timestamp.IsZero() {
				ts = t.Timestamp.UTC().Format(time.RFC3339)
			}
			if err := db.InsertTurn(dataDB, newID(), sessionID, i, t.Role, t.Content, ts); err != nil {
				return fmt.Errorf("insert turn: %w", err)
			}
		}

		// Insert tool calls.
		for i, tc := range payload.ToolCalls {
			if err := db.InsertToolCall(dataDB, newID(), sessionID, i, tc.Tool, tc.Path, tc.CmdPrefix); err != nil {
				return fmt.Errorf("insert tool_call: %w", err)
			}
		}

		sessionIDs = append(sessionIDs, sessionID)
		inserted++
	}

	if inserted == 0 {
		return nil
	}

	// Commit data dump to the local orphan branch.
	// The commit SHA on the orphan branch becomes the checkpoint ID.
	commitSHA, err := commitToOrphanBranch(gitRoot)
	if err != nil {
		return fmt.Errorf("commit to rekal branch: %w", err)
	}

	_ = commitSHA
	_ = sessionIDs
	// TODO: Insert checkpoint row (id = commitSHA), files_touched, checkpoint_sessions.
	// TODO: Incremental index update.

	fmt.Fprintf(cmd.ErrOrStderr(), "rekal: %d session(s) captured\n", inserted)
	return nil
}

// commitToOrphanBranch commits the current data DB dump to the rekal orphan branch.
// Returns the new commit SHA (which is the checkpoint ID).
func commitToOrphanBranch(gitRoot string) (string, error) {
	branch := rekalBranchName()

	// Get the current tip of the orphan branch.
	parentOut, err := exec.Command("git", "-C", gitRoot, "rev-parse", branch).Output()
	if err != nil {
		return "", fmt.Errorf("resolve branch %s: %w", branch, err)
	}
	parent := strings.TrimSpace(string(parentOut))

	// Store the raw .db file as a blob.
	dbPath := fmt.Sprintf("%s/.rekal/data.db", gitRoot)
	blobOut, err := exec.Command("git", "-C", gitRoot, "hash-object", "-w", dbPath).Output()
	if err != nil {
		return "", fmt.Errorf("hash data.db: %w", err)
	}
	blobHash := strings.TrimSpace(string(blobOut))

	// Build a tree with just data.db.
	treeEntry := fmt.Sprintf("100644 blob %s\tdata.db\n", blobHash)
	mktreeCmd := exec.Command("git", "-C", gitRoot, "mktree")
	mktreeCmd.Stdin = strings.NewReader(treeEntry)
	treeOut, err := mktreeCmd.Output()
	if err != nil {
		return "", fmt.Errorf("mktree: %w", err)
	}
	treeHash := strings.TrimSpace(string(treeOut))

	// Create commit on the orphan branch.
	commitOut, err := exec.Command("git", "-C", gitRoot,
		"commit-tree", treeHash, "-p", parent, "-m", "rekal: checkpoint",
	).Output()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	commitSHA := strings.TrimSpace(string(commitOut))

	// Update the branch ref.
	if err := exec.Command("git", "-C", gitRoot, "update-ref", "refs/heads/"+branch, commitSHA).Run(); err != nil {
		return "", fmt.Errorf("update-ref: %w", err)
	}

	return commitSHA, nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
