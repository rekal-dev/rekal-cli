package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHookScript_PrePushContract pins what the pre-push hook must do.
//
// Two defects lived here. It ignored git's arguments and published to origin
// regardless, so anyone pushing to a fork or a second remote sent their memory
// somewhere they did not ask for. And recursion was prevented with
// --no-verify, which suppresses *every* pre-push hook in the repository —
// other tools' included — to solve a problem that is only rekal's.
func TestHookScript_PrePushContract(t *testing.T) {
	t.Parallel()
	got := hookScript("push")

	if !strings.Contains(got, `--remote "${1:-origin}"`) {
		t.Errorf("the hook must forward git's remote ($1), not assume origin:\n%s", got)
	}
	if !strings.Contains(got, "--best-effort") {
		t.Errorf("the hook must run best-effort — a memory push must never fail somebody's git push:\n%s", got)
	}
	if !strings.Contains(got, internalPushEnv) {
		t.Errorf("recursion must be guarded by %s so unrelated pre-push hooks still run:\n%s", internalPushEnv, got)
	}
	if strings.Contains(got, "--no-verify") {
		t.Errorf("--no-verify disables every hook in the repo, not just rekal's:\n%s", got)
	}

	// post-commit takes none of this: it is not a publish and git passes it
	// no remote.
	commitHook := hookScript("checkpoint")
	for _, unwanted := range []string{"--remote", "--best-effort", internalPushEnv} {
		if strings.Contains(commitHook, unwanted) {
			t.Errorf("post-commit hook should not carry %q:\n%s", unwanted, commitHook)
		}
	}
}

// TestPushFailure_BestEffort covers the exit-code split. A hook must not fail
// the user's git push; a person running it by hand must not be told nothing
// happened when publication failed.
func TestPushFailure_BestEffort(t *testing.T) {
	t.Parallel()
	boom := os.ErrClosed
	if err := pushFailure(pushOptions{BestEffort: true}, boom); err != nil {
		t.Errorf("best-effort must swallow a publication failure, got %v", err)
	}
	if err := pushFailure(pushOptions{}, boom); err == nil {
		t.Error("a hand-run push must exit non-zero when publication fails, or the failure is invisible")
	}
}

// TestRunGitDeadline_TimesOut covers the hang. A push to an unreachable remote
// used to block the commit that triggered it with nothing printed, so the
// failure was indistinguishable from rekal being slow.
func TestRunGitDeadline_TimesOut(t *testing.T) {
	// No t.Parallel: this test shadows git via PATH.
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	// A remote helper that never returns: git blocks, and the deadline is the
	// only thing that ends it.
	blocker := filepath.Join(dir, "git-remote-stall")
	if err := os.WriteFile(blocker, []byte("#!/bin/sh\nsleep 60\n"), 0o755); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := runGitDeadline(ctx, dir, "ls-remote", "stall://nowhere")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("a blocked git command must fail, not hang")
	}
	if elapsed > 15*time.Second {
		t.Errorf("took %s — the deadline did not apply", elapsed)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("the error must name the timeout so the stuck stage is identifiable, got: %v", err)
	}
}

// TestRunGitDeadline_MarksInternalPush observes the guard from the child's
// side. The hook only knows to stand down because rekal sets this on its own
// pushes; asserting it any other way would assert nothing.
func TestRunGitDeadline_MarksInternalPush(t *testing.T) {
	// No t.Parallel: this test shadows git via PATH.
	dir := t.TempDir()
	fake := filepath.Join(dir, "git")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nprintf '%s' \"$"+internalPushEnv+`"
`), 0o755); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := runGitDeadline(ctx, dir, "anything")
	if err != nil {
		t.Fatalf("fake git failed: %v (%s)", err, out)
	}
	if strings.TrimSpace(string(out)) != "1" {
		t.Errorf("%s reached the child as %q, want \"1\" — without it the pre-push hook re-enters rekal", internalPushEnv, out)
	}
}
