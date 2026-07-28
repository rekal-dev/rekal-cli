package cli

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var force bool
	var reExport bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push Rekal data to the remote branch",
		Long: `Export new checkpoints to the wire format and push to the remote orphan branch.

Only YOUR unexported checkpoints are pushed — team data imported via 'rekal sync'
is never re-exported. Each user pushes to their own branch (rekal/<email>).

Only merged work is shared: a checkpoint reaches the wire only when its commit
is an ancestor of the default branch, or its branch's changes landed as a
squash merge (detected by patch equivalence — works with merge-commit, rebase,
and squash workflows alike). Sessions on unmerged branches stay local and are
re-checked on every push, so they ship automatically once the branch merges —
and never if it is abandoned. The local data DB always keeps every branch;
only what crosses the wire is filtered.

Checkpoints contain sessions (conversation turns, tool calls) and file change
metadata anchored to git commits. They are encoded into a compact binary wire
format (rekal.body + dict.bin) using zstd compression and string interning —
a 2-10 MB session compresses to ~300 bytes on the wire.

Use --force to overwrite the remote branch when it has diverged from local
(e.g. after a rebuild or conflict).

Use --re-export to regenerate the branch's wire data from scratch out of the
local data DB and force-push it. This repairs a branch whose wire data was
written by a rekal version with the frame-count bug (sessions with more than
255 turns or tool calls were corrupted on the wire) and drops accumulated
stale meta frames. The merged-only rule applies here too: the rebuilt branch
contains only merged checkpoints. Implies --force.

Normally runs automatically via the pre-push git hook installed by 'rekal init'.
You do not need to run this manually.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}

			if reExport {
				return doReExport(gitRoot, cmd.ErrOrStderr())
			}
			return doPush(gitRoot, cmd.ErrOrStderr(), force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force push (overwrite remote with local data)")
	cmd.Flags().BoolVar(&reExport, "re-export", false, "Rebuild the branch's wire data from the local data DB and force push")
	return cmd
}

// doReExport regenerates the orphan branch's wire data from every checkpoint
// in data.db and force-pushes it. The branch is derived data; data.db is the
// source of truth — this heals wire bytes corrupted by the pre-v2 frame-count
// bug and drops stale meta frames accumulated by past pushes.
func doReExport(gitRoot string, w io.Writer) error {
	body, dict, exportedIDs, err := transport.ExportAllFrames(gitRoot)
	if err != nil {
		return fmt.Errorf("re-export: %w", err)
	}
	if body == nil {
		fmt.Fprintln(w, "rekal: no checkpoints to export")
		return nil
	}

	if err := transport.EnsureOrphanBranch(gitRoot); err != nil {
		return fmt.Errorf("ensure rekal branch: %w", err)
	}
	if _, err := transport.CommitWireFormat(gitRoot, body, dict); err != nil {
		return fmt.Errorf("commit to rekal branch: %w", err)
	}
	if err := markCheckpointsExported(gitRoot, exportedIDs); err != nil {
		return fmt.Errorf("mark checkpoints exported: %w", err)
	}
	fmt.Fprintf(w, "rekal: re-exported %d checkpoint(s)\n", len(exportedIDs))

	branch := gitx.RekalBranchName()
	if err := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin").Run(); err != nil {
		fmt.Fprintln(w, "rekal: no remote 'origin' configured — skipping push")
		return nil
	}
	forceCmd := exec.Command("git", "-C", gitRoot, "push", "--no-verify", "--force", "origin", branch)
	forceCmd.Stdin = nil
	if output, err := forceCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(w, "rekal: force push failed: %s\n", strings.TrimSpace(string(output)))
		return nil
	}
	fmt.Fprintf(w, "rekal: force pushed to origin/%s\n", branch)
	return nil
}

// doPush pushes Rekal data to the remote orphan branch.
// Extracted so sync can call it without a cobra.Command.
func doPush(gitRoot string, w io.Writer, force bool) error {
	branch := gitx.RekalBranchName()

	// Check if local branch exists — if not, nothing to push.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", branch).Run(); err != nil {
		fmt.Fprintln(w, "rekal: no data to push (run 'rekal checkpoint' first)")
		return nil
	}

	// Check if remote is configured.
	if err := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin").Run(); err != nil {
		fmt.Fprintln(w, "rekal: no remote 'origin' configured — skipping push")
		return nil
	}

	// Catch the local branch up to the remote before exporting. The body is
	// built by appending to whatever the local tip holds, so a stale tip would
	// produce a snapshot missing everything another machine on this identity
	// already pushed — and a push that is then rejected with no way forward
	// except a destructive force. Fast-forward only; a real fork is left alone
	// for the rejection path below to report.
	if moved, ffErr := transport.FastForwardOrphanBranch(gitRoot); ffErr != nil {
		fmt.Fprintf(w, "rekal: warning: %v\n", ffErr)
	} else if moved {
		fmt.Fprintf(w, "rekal: caught local %s up to origin\n", branch)
	}

	// Export unexported checkpoints from DuckDB → wire format → orphan branch.
	body, dict, exportedIDs, err := transport.ExportNewFrames(gitRoot)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}
	if body != nil {
		if _, err := transport.CommitWireFormat(gitRoot, body, dict); err != nil {
			return fmt.Errorf("commit to rekal branch: %w", err)
		}
		// Only mark checkpoints exported once the wire format is durably
		// committed to the orphan branch — marking earlier risks a
		// checkpoint being flagged exported but never actually written if
		// transport.CommitWireFormat had failed (or the process died) in between.
		if err := markCheckpointsExported(gitRoot, exportedIDs); err != nil {
			return fmt.Errorf("mark checkpoints exported: %w", err)
		}
	} else {
		fmt.Fprintln(w, "rekal: no new checkpoints to export")
	}

	// Compare local SHA vs remote tracking SHA — skip if identical.
	localSHA, err := exec.Command("git", "-C", gitRoot, "rev-parse", branch).Output()
	if err != nil {
		return nil
	}
	remoteSHA, err := exec.Command("git", "-C", gitRoot, "rev-parse", "origin/"+branch).Output()
	if err == nil && strings.TrimSpace(string(localSHA)) == strings.TrimSpace(string(remoteSHA)) {
		fmt.Fprintln(w, "rekal: already up to date")
		return nil
	}

	if force {
		forceCmd := exec.Command("git", "-C", gitRoot, "push", "--no-verify", "--force", "origin", branch)
		forceCmd.Stdin = nil
		if output, err := forceCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(w, "rekal: force push failed: %s\n", strings.TrimSpace(string(output)))
			return nil
		}
		fmt.Fprintf(w, "rekal: force pushed to origin/%s\n", branch)
		return nil
	}

	// Push with --no-verify to prevent recursive pre-push hook.
	pushCmd := exec.Command("git", "-C", gitRoot, "push", "--no-verify", "origin", branch)
	pushCmd.Stdin = nil // disconnect stdin so git doesn't hang in hook context
	output, err := pushCmd.CombinedOutput()
	if err != nil {
		if isNonFastForward(string(output)) && remoteDiverged(gitRoot, branch) {
			fmt.Fprintf(w, "rekal: push rejected (non-fast-forward) for origin/%s\n", branch)
			fmt.Fprintln(w, "rekal: your remote branch has diverged from local — review and run 'rekal push --force' to overwrite remote with local data")
			return nil
		}
		fmt.Fprintf(w, "rekal: push failed: %s\n", strings.TrimSpace(string(output)))
		return nil
	}

	fmt.Fprintf(w, "rekal: pushed to origin/%s\n", branch)
	return nil
}

// markCheckpointsExported opens data.db and flags the given checkpoint IDs as
// exported. Called only after their wire-format bytes are durably committed
// to the orphan branch (see doPush) — never before.
func markCheckpointsExported(gitRoot string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	return db.MarkCheckpointsExported(dataDB, ids)
}

// isNonFastForward checks if git push output indicates a non-fast-forward rejection.
func isNonFastForward(output string) bool {
	return strings.Contains(output, "non-fast-forward") ||
		strings.Contains(output, "[rejected]") ||
		strings.Contains(output, "fetch first")
}

// remoteDiverged confirms, against the refs themselves, that the remote branch
// really does carry work local does not.
//
// The push output alone cannot be trusted for this. When the transport fails
// mid-push — a proxy answering 403, an expired credential, a branch-protection
// rule — git reports the ref as "[rejected] ... (fetch first)" all the same,
// and that phrasing is indistinguishable from a genuine divergence. Acting on
// the text alone told the user their history had diverged and pointed them at
// 'push --force', which overwrites the shared branch with local data: a
// destructive answer to what was only ever a failed connection, and the wire is
// the team's only copy of conversations already shared.
//
// So ask git instead. A best-effort fetch first, because the remote-tracking
// ref may be stale — if that fetch fails there is nothing to diverge from that
// we can see, and the honest report is the transport error.
func remoteDiverged(gitRoot, branch string) bool {
	fetch := exec.Command("git", "-C", gitRoot, "fetch", "origin", branch)
	fetch.Stdin = nil
	if err := fetch.Run(); err != nil {
		return false
	}
	remote, err := exec.Command("git", "-C", gitRoot, "rev-parse", "FETCH_HEAD").Output()
	if err != nil {
		return false
	}
	// Diverged exactly when the remote tip is not already contained in local.
	return !gitx.IsAncestor(gitRoot, strings.TrimSpace(string(remote)), branch)
}
