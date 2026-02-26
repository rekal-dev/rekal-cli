package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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
		Long: `Snapshot the active AI session into the local data DB.

Reads session transcript files (conversation turns, tool calls, file changes)
from the agent's session directory, deduplicates by content hash, and inserts
into .rekal/data.db. Each checkpoint is linked to the current HEAD commit and
records which files were changed.

Normally runs automatically via the post-commit hook installed by 'rekal init'.
Run manually to capture a session without committing.`,
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
	return doCheckpoint(gitRoot, cmd.ErrOrStderr())
}

// doCheckpoint captures the current session after a commit.
// Extracted so sync can call it without a cobra.Command.
func doCheckpoint(gitRoot string, w io.Writer) error {
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

	// Verify DB is healthy by running a simple query.
	if _, err := dataDB.Exec("SELECT 1"); err != nil {
		return fmt.Errorf("data DB is corrupt or unreadable: %w", err)
	}

	email := gitConfigValue("user.email")
	entropy := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	newID := func() string {
		return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
	}

	var sessionIDs []string
	var inserted int
	// Collect unique relative file paths from file-modifying tool_calls across all sessions.
	toolCallPaths := make(map[string]struct{})

	for _, f := range files {
		// Incremental: check checkpoint_state to skip unchanged files.
		info, statErr := os.Stat(f)
		if statErr != nil {
			continue
		}

		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if len(data) == 0 {
			continue
		}

		hash := sha256Hex(data)

		// Check cached state — skip if size + hash match.
		cachedSize, cachedHash, found, csErr := db.GetCheckpointState(dataDB, f)
		if csErr != nil {
			return fmt.Errorf("check checkpoint state: %w", csErr)
		}
		if found && cachedSize == info.Size() && cachedHash == hash {
			continue
		}

		exists, err := db.SessionExistsByHash(dataDB, hash)
		if err != nil {
			return fmt.Errorf("dedup check: %w", err)
		}
		if exists {
			// File changed but session already exists (re-parse produced same hash).
			// Update state cache and skip.
			_ = db.UpsertCheckpointState(dataDB, f, info.Size(), hash)
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
		capturedAt := time.Now().UTC()

		// Insert session into DuckDB.
		if err := db.InsertSession(
			dataDB, sessionID, "", hash,
			payload.ActorType, payload.AgentID, email, payload.Branch, capturedAt.Format(time.RFC3339),
		); err != nil {
			return fmt.Errorf("insert session: %w", err)
		}

		// Insert turns into DuckDB.
		for i, t := range payload.Turns {
			ts := ""
			if !t.Timestamp.IsZero() {
				ts = t.Timestamp.UTC().Format(time.RFC3339)
			}
			if err := db.InsertTurn(dataDB, newID(), sessionID, i, t.Role, t.Content, ts); err != nil {
				return fmt.Errorf("insert turn: %w", err)
			}
		}

		// Insert tool calls into DuckDB.
		for i, tc := range payload.ToolCalls {
			if err := db.InsertToolCall(dataDB, newID(), sessionID, i, tc.Tool, tc.Path, tc.CmdPrefix); err != nil {
				return fmt.Errorf("insert tool_call: %w", err)
			}
		}

		// Collect file-modifying tool_call paths for files_touched supplementation.
		for _, tc := range payload.ToolCalls {
			if tc.Path == "" {
				continue
			}
			switch tc.Tool {
			case "Write", "Edit", "NotebookEdit":
			default:
				continue
			}
			rel := strings.TrimPrefix(tc.Path, gitRoot+"/")
			if rel == tc.Path {
				// Path is not under gitRoot — external file, skip.
				continue
			}
			toolCallPaths[rel] = struct{}{}
		}

		// Update checkpoint state cache.
		_ = db.UpsertCheckpointState(dataDB, f, info.Size(), hash)

		sessionIDs = append(sessionIDs, sessionID)
		inserted++
	}

	if inserted == 0 {
		return nil
	}

	// Get git state for checkpoint.
	gitSHA := gitHeadSHA(gitRoot)
	gitBranch := gitCurrentBranch(gitRoot)
	filesTouched := gitFilesChanged(gitRoot)

	// Generate checkpoint ULID.
	checkpointID := newID()

	// Insert checkpoint into DuckDB (exported = FALSE by default).
	now := time.Now().UTC()
	if err := db.InsertCheckpoint(dataDB, checkpointID, gitSHA, gitBranch, email, now.Format(time.RFC3339), "human", ""); err != nil {
		return fmt.Errorf("insert checkpoint: %w", err)
	}

	// Insert files_touched from git diff.
	gitTouchedSet := make(map[string]struct{})
	for _, ft := range filesTouched {
		parts := strings.SplitN(ft, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		gitTouchedSet[parts[1]] = struct{}{}
		if err := db.InsertFileTouched(dataDB, newID(), checkpointID, parts[1], parts[0]); err != nil {
			return fmt.Errorf("insert file_touched: %w", err)
		}
	}

	// Supplement files_touched with file-modifying tool_call paths not already covered by git diff.
	for p := range toolCallPaths {
		if _, exists := gitTouchedSet[p]; exists {
			continue
		}
		if err := db.InsertFileTouched(dataDB, newID(), checkpointID, p, "T"); err != nil {
			return fmt.Errorf("insert file_touched (tool_call): %w", err)
		}
	}

	// Insert checkpoint_sessions junction rows.
	for _, sid := range sessionIDs {
		if err := db.InsertCheckpointSession(dataDB, checkpointID, sid); err != nil {
			return fmt.Errorf("insert checkpoint_session: %w", err)
		}
	}

	fmt.Fprintf(w, "rekal: %d session(s) captured\n", inserted)
	return nil
}

func gitHeadSHA(gitRoot string) string {
	out, err := exec.Command("git", "-C", gitRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return strings.Repeat("0", 40)
	}
	return strings.TrimSpace(string(out))
}

func gitCurrentBranch(gitRoot string) string {
	out, err := exec.Command("git", "-C", gitRoot, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func gitFilesChanged(gitRoot string) []string {
	out, err := exec.Command("git", "-C", gitRoot, "diff", "--name-status", "HEAD~1", "HEAD").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

// gitShowFile reads a file from a git ref. Returns nil if not found.
func gitShowFile(gitRoot, ref, path string) []byte {
	out, err := exec.Command("git", "-C", gitRoot, "show", ref+":"+path).Output()
	if err != nil {
		return nil
	}
	return out
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
