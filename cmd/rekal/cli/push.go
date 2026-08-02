package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var rebuild bool
	var reExportAlias bool
	var remote string
	var strict bool
	var progress bool
	var timeout time.Duration

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

There is no force flag. The wire format is append-only — no byte is ever
modified after it is written — and that is a structural guarantee, not a
policy: 'rekal push' appends checkpoints and can do nothing else. Overwriting a
ref is a git operation, not a memory operation, so if you truly mean to discard
what a branch holds, say so in git: git push --force origin rekal/<email>.

When a push is rejected because the branch genuinely diverged, the fix is to
push from the machine that holds the missing checkpoints; it will append to the
branch and this one will fast-forward onto it.

Use --rebuild to regenerate the branch's wire data from scratch out of the
local data DB and push it. This repairs a branch whose wire data was
written by a rekal version with the frame-count bug (sessions with more than
255 turns or tool calls were corrupted on the wire) and drops accumulated
stale meta frames. The merged-only rule applies here too: the rebuilt branch
contains only merged checkpoints.

--rebuild is the one exception, and it is bounded: it refuses unless the
branch's current body is already contained in what this machine would write, so
it can only ever produce a superset. It repairs derived bytes from data.db,
which is the source of truth — it never edits the ledger.

Normally runs automatically via the pre-push git hook installed by 'rekal init'.
You do not need to run this manually.

Options (no short forms: every one of these changes what leaves your machine,
so they are spelled out):

  --remote <name>   Publish to this remote instead of origin. The pre-push
                    hook passes whichever remote git is pushing to, so a push
                    to a fork no longer sends memory to origin.
  --strict          Exit non-zero when publication fails. Default is to warn
                    and exit 0, because the hook runs on every git push and a
                    diverged memory branch must not abort your code push.
  --timeout <dur>   Deadline for each git network call (default 2m). Raise it
                    on a slow remote; lower it to fail fast.
  --progress        Print timed stages, for when a push feels slow.
  --rebuild         Re-encode the branch's wire data from the local data DB.
                    A repair, not a routine step: it refuses unless the
                    branch's current body is already a prefix of what this
                    machine would write, so it can only ever add.

There is deliberately no --force.`,
		Example: `  # What the pre-push hook runs for you
  rekal push
    rekal: pushed to origin/rekal/you@example.com

  # Nothing merged since last time — normal, not an error
  rekal push
    rekal: no new checkpoints to export

  # Publish to a fork instead of origin
  rekal push --remote upstream

  # Fail the command if publication fails (default is to warn and exit 0,
  # so a memory push never aborts your git push)
  rekal push --strict

  # Slow or wedged remote: bound every git call
  rekal push --timeout 30s

  # See where the time goes
  rekal push --progress

  # Repair a branch's wire bytes from the local data DB (superset-only)
  rekal push --rebuild

  # Check what is still held back, and why
  rekal query --sql "SELECT git_sha, git_branch, exported FROM checkpoints WHERE NOT exported"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}

			if rebuild || reExportAlias {
				return doReExport(gitRoot, cmd.ErrOrStderr())
			}
			return doPush(gitRoot, cmd.ErrOrStderr(), pushOptions{
				Remote:   remote,
				Strict:   strict,
				Timeout:  timeout,
				Progress: progress,
			})
		},
	}

	cmd.Flags().BoolVar(&rebuild, "rebuild", false, "Re-encode the branch's wire data from the local data DB (superset-only)")
	// --re-export was the original name. It described the mechanism rather than
	// the effect, and the effect is what a user has to reason about before
	// running something this destructive. Kept as a deprecated alias so nothing
	// that already invokes it breaks.
	cmd.Flags().BoolVar(&reExportAlias, "re-export", false, "Deprecated alias for --rebuild")
	_ = cmd.Flags().MarkDeprecated("re-export", "use --rebuild")
	cmd.Flags().StringVar(&remote, "remote", "origin", "Remote to publish to (the pre-push hook passes the remote git is pushing to)")
	cmd.Flags().BoolVar(&strict, "strict", false, "Exit non-zero when publication fails (default: warn and exit 0, so a memory push never fails your git push)")
	// Accepted and ignored: the installed hook passes it, and a hook written by
	// a newer rekal must keep working against an older binary and the reverse.
	var bestEffortCompat bool
	cmd.Flags().BoolVar(&bestEffortCompat, "best-effort", false, "No-op; warning-on-failure is the default")
	_ = cmd.Flags().MarkHidden("best-effort")
	cmd.Flags().BoolVar(&progress, "progress", false, "Print timed stages")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultGitTimeout, "Deadline for each git network call")
	return cmd
}

// doReExport regenerates the orphan branch's wire data from every checkpoint in
// data.db and pushes it. The branch is derived data; data.db is the source of
// truth — this heals wire bytes corrupted by the pre-v2 frame-count bug and
// drops stale meta frames accumulated by past pushes.
//
// It is the only path that rewrites the body rather than appending to it, which
// is why it is bounded by wouldDiscardRemoteFrames: it may only ever produce a
// superset of what the branch already holds. Repairing derived bytes from the
// ledger is not the same as editing the ledger.
func doReExport(gitRoot string, w io.Writer) error {
	// Catch up first, so the rebuild commit lands on top of the branch rather
	// than beside it. Without this a machine that is merely behind looks like a
	// fork to the guard below and the repair is refused for no real reason.
	if _, ffErr := transport.FastForwardOrphanBranch(gitRoot); ffErr != nil {
		fmt.Fprintf(w, "rekal: warning: %v\n", ffErr)
	}
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
	// A rebuild starts from an empty body, so it can only ever carry what this
	// machine's data.db holds. Anything another machine put on the branch is
	// not in there and could not be reconstructed by anyone.
	if lost, ok := wouldDiscardRemoteFrames(gitRoot, "origin", branch); ok && lost != "" {
		fmt.Fprintf(w, "rekal: refusing to rebuild origin/%s\n", branch)
		fmt.Fprintf(w, "rekal: the remote has %s this machine does not have, and a rebuild carries only this machine's data.db\n", lost)
		fmt.Fprintln(w, "rekal: push from the machine that holds them first, then rebuild from there")
		return nil
	}
	// A plain push, like every other path. The rebuild commit was parented on
	// the branch tip this run fast-forwarded to, so it is a descendant and the
	// push is an ordinary fast-forward. Nothing in rekal force-pushes: that is
	// what makes the append-only wire format structural rather than a rule the
	// code keeps only when it feels like it.
	ctx, cancel := context.WithTimeout(context.Background(), defaultGitTimeout)
	defer cancel()
	if output, err := runGitDeadline(ctx, gitRoot, "push", "origin", branch); err != nil {
		fmt.Fprintf(w, "rekal: rebuild push failed: %s\n", strings.TrimSpace(string(output)))
		fmt.Fprintln(w, "rekal: the branch moved while rebuilding — re-run, or push from the machine holding the newer checkpoints")
		return nil
	}
	fmt.Fprintf(w, "rekal: rebuilt origin/%s\n", branch)
	return nil
}

// pushOptions carries what the caller knows that doPush cannot infer.
type pushOptions struct {
	// Remote is the git remote to publish to. The pre-push hook passes the
	// remote git is actually pushing to; everything else defaults to origin.
	Remote string
	// Strict makes a publication failure a non-zero exit. Off by default, and
	// that asymmetry is deliberate: every pre-push hook already installed in
	// the wild invokes a bare `rekal push`, so a failure that exits non-zero
	// there aborts the user's git push. Someone upgrading the binary without
	// re-running `rekal init` would find a diverged memory branch blocking
	// their code — memory failing must never stop work from leaving the
	// machine. --strict opts into the louder behaviour for a script that wants
	// publication treated as required.
	Strict bool
	// Timeout bounds each git network call.
	Timeout time.Duration
	// Progress prints timed stages.
	Progress bool
}

func (o pushOptions) remote() string {
	if o.Remote == "" {
		return "origin"
	}
	return o.Remote
}

func (o pushOptions) timeout() time.Duration {
	if o.Timeout <= 0 {
		return defaultGitTimeout
	}
	return o.Timeout
}

// defaultGitTimeout bounds a single git network call. A push that hangs on an
// unreachable remote used to hang the commit that triggered it, with nothing
// printed to say which stage was stuck.
const defaultGitTimeout = 2 * time.Minute

// internalPushEnv marks the git push rekal makes itself, so the pre-push hook
// can decline to recurse. This replaces --no-verify, which suppressed *every*
// pre-push hook in the repository — other tools' included — to solve rekal's
// own recursion.
const internalPushEnv = "REKAL_INTERNAL_PUSH"

// stage prints a timed progress line. Publishing is several seconds of work in
// stages that fail differently; without naming them a stall is indistinguishable
// from a hang.
func stage(w io.Writer, on bool, start time.Time, format string, args ...any) {
	if !on {
		return
	}
	fmt.Fprintf(w, "rekal: [%4.1fs] %s\n", time.Since(start).Seconds(), fmt.Sprintf(format, args...))
}

// runGitDeadline runs a git command under a deadline, returning its combined
// output. The environment carries internalPushEnv so rekal's own pushes do not
// re-enter the pre-push hook.
func runGitDeadline(ctx context.Context, gitRoot string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", gitRoot}, args...)...)
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(), internalPushEnv+"=1")
	// WaitDelay is what actually makes the deadline bite. Killing git on
	// context expiry is not enough: git spawns its own children — a remote
	// helper, ssh, a credential prompt — and CombinedOutput blocks until every
	// writer closes the pipe, so a stuck grandchild kept the caller hanging
	// long after the parent was dead. WaitDelay bounds that wait and closes the
	// pipes, which is the difference between a bounded failure and a hang.
	cmd.WaitDelay = 5 * time.Second
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return out, fmt.Errorf("git %s timed out after %s: %w", args[0], defaultGitTimeout, ctx.Err())
	}
	return out, err
}

// doPush pushes Rekal data to the remote orphan branch.
// Extracted so sync can call it without a cobra.Command.
func doPush(gitRoot string, w io.Writer, opts pushOptions) error {
	branch := gitx.RekalBranchName()
	remote := opts.remote()
	started := time.Now()

	// Check if local branch exists — if not, nothing to push.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", branch).Run(); err != nil {
		fmt.Fprintln(w, "rekal: no data to push (run 'rekal checkpoint' first)")
		return nil
	}

	// Check the target remote is configured.
	if err := exec.Command("git", "-C", gitRoot, "remote", "get-url", remote).Run(); err != nil {
		fmt.Fprintf(w, "rekal: no remote %q configured — skipping push\n", remote)
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

	stage(w, opts.Progress, started, "resolving %s against %s", branch, remote)
	// Export unexported checkpoints from DuckDB → wire format → orphan branch.
	stage(w, opts.Progress, started, "selecting and encoding checkpoints")
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
		stage(w, opts.Progress, started, "encoded %d checkpoint(s)", len(exportedIDs))
	} else {
		fmt.Fprintln(w, "rekal: no new checkpoints to export")
	}

	// Compare local SHA vs remote tracking SHA — skip if identical.
	localSHA, err := exec.Command("git", "-C", gitRoot, "rev-parse", branch).Output()
	if err != nil {
		return nil
	}
	remoteSHA, err := exec.Command("git", "-C", gitRoot, "rev-parse", remote+"/"+branch).Output()
	if err == nil && strings.TrimSpace(string(localSHA)) == strings.TrimSpace(string(remoteSHA)) {
		fmt.Fprintln(w, "rekal: already up to date")
		return nil
	}

	// Recursion is prevented by internalPushEnv rather than --no-verify: the
	// flag suppressed every pre-push hook in the repository, other tools'
	// included, to solve a problem that is only rekal's.
	stage(w, opts.Progress, started, "publishing %s to %s", branch, remote)
	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout())
	defer cancel()
	output, err := runGitDeadline(ctx, gitRoot, "push", remote, branch)
	if err != nil {
		if isNonFastForward(string(output)) && remoteDiverged(gitRoot, remote, branch) {
			fmt.Fprintf(w, "rekal: push rejected (non-fast-forward) for %s/%s\n", remote, branch)
			fmt.Fprintln(w, "rekal: the branch diverged — push from the machine holding the missing checkpoints; it will append and this one will fast-forward onto it")
			return pushFailure(opts, fmt.Errorf("push rejected: %s/%s diverged", remote, branch))
		}
		fmt.Fprintf(w, "rekal: push failed: %s\n", strings.TrimSpace(string(output)))
		return pushFailure(opts, fmt.Errorf("push to %s/%s failed: %w", remote, branch, err))
	}

	stage(w, opts.Progress, started, "published")
	fmt.Fprintf(w, "rekal: pushed to %s/%s\n", remote, branch)
	return nil
}

// pushFailure reports a publication failure. It is a warning by default and an
// error only under --strict, because the default is what every already-installed
// hook invokes: a memory push must never fail somebody's git push.
func pushFailure(opts pushOptions, err error) error {
	if !opts.Strict {
		return nil
	}
	return NewSilentError(err)
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

// wouldDiscardRemoteFrames reports what replacing the remote branch's body
// would delete: wire data it holds that this machine does not. lost is a human count
// ("3 commit(s)"), empty when nothing would be lost. ok is false when the
// question could not be answered — offline, no such remote branch — in which
// case the caller must not treat silence as permission.
//
// This is the difference between the two forces. Overwriting a branch you are
// merely ahead of costs nothing; overwriting one that carries another machine's
// checkpoints destroys them, because sync imports other branches into the index
// and never into data.db, so no local re-export can reproduce them. Git can
// answer this in one question and the flag never asked it.
func wouldDiscardRemoteFrames(gitRoot, remote, branch string) (lost string, ok bool) {
	fetch := exec.Command("git", "-C", gitRoot, "fetch", remote, branch)
	fetch.Stdin = nil
	if err := fetch.Run(); err != nil {
		return "", false
	}
	head, err := exec.Command("git", "-C", gitRoot, "rev-parse", "FETCH_HEAD").Output()
	if err != nil {
		return "", false
	}
	remoteSHA := strings.TrimSpace(string(head))
	// Contained in local: a force here only replaces commits local already has.
	if gitx.IsAncestor(gitRoot, remoteSHA, branch) {
		return "", true
	}
	// Diverged by commit — but commits are not the thing worth protecting, the
	// frames inside them are. A push appends to the body it found on the
	// branch, so along any branch the body only ever grows: if the remote's
	// body is a prefix of this machine's, every frame it holds is already here
	// and the force costs nothing. Rewriting a commit (an amend, a re-anchored
	// tip) diverges the history while leaving the body byte-identical, and
	// refusing there would block a repair that loses nothing.
	remoteBody := gitx.ShowFile(gitRoot, remoteSHA, codec.BodyFilename)
	if len(remoteBody) == 0 {
		return "", true // nothing on the branch to lose
	}
	if bytes.HasPrefix(gitx.ShowFile(gitRoot, branch, codec.BodyFilename), remoteBody) {
		return "", true
	}
	out, err := exec.Command("git", "-C", gitRoot,
		"rev-list", "--count", branch+".."+remoteSHA).Output()
	if err != nil {
		return "", false
	}
	n := strings.TrimSpace(string(out))
	if n == "" || n == "0" {
		return "", true
	}
	return n + " commit(s) of wire data", true
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
func remoteDiverged(gitRoot, remote, branch string) bool {
	fetch := exec.Command("git", "-C", gitRoot, "fetch", remote, branch)
	fetch.Stdin = nil
	if err := fetch.Run(); err != nil {
		return false
	}
	head, err := exec.Command("git", "-C", gitRoot, "rev-parse", "FETCH_HEAD").Output()
	if err != nil {
		return false
	}
	// Diverged exactly when the remote tip is not already contained in local.
	return !gitx.IsAncestor(gitRoot, strings.TrimSpace(string(head)), branch)
}
