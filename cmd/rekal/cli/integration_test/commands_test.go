//go:build integration

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli"
)

// TestEnv provides an isolated git repo for integration testing.
type TestEnv struct {
	T       *testing.T
	RepoDir string
}

// NewTestEnv creates a temp directory with git init.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()
	dir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var).
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	// Set local git user config (required for git commit-tree in ensureOrphanBranch).
	// gc.auto=0/receive.autogc=false: git forks a detached `git gc --auto`
	// after commits/pushes once loose-object thresholds are crossed — see
	// disableAutoGC's doc comment for why that matters for a t.TempDir().
	for _, kv := range [][2]string{
		{"user.email", "test@rekal.dev"},
		{"user.name", "Rekal Test"},
		{"gc.auto", "0"},
		{"receive.autogc", "false"},
	} {
		c := exec.Command("git", "-C", dir, "config", kv[0], kv[1])
		if err := c.Run(); err != nil {
			t.Fatalf("git config %s: %v", kv[0], err)
		}
	}
	return &TestEnv{T: t, RepoDir: dir}
}

// initBareRemote creates a bare git repo at dir (for use as a `git push`
// remote in tests) with automatic gc disabled — see disableAutoGC.
func initBareRemote(t *testing.T, dir string) {
	t.Helper()
	if err := exec.Command("git", "init", "--bare", dir).Run(); err != nil {
		t.Fatalf("git init --bare %s: %v", dir, err)
	}
	disableAutoGC(t, dir)
}

// disableAutoGC turns off git's automatic background maintenance for a repo
// used in a test.
//
// This is the fix for a real, reproduced flake in TestPush_E2E_ExportAndPush:
// receive-pack runs `git gc --auto` after a push once loose-object/pack
// thresholds are crossed, and that gc forks and detaches — control returns to
// the pushing `git push` (and so to RunCLI, and so to the test) before gc
// necessarily finishes repacking/pruning the receiving repo's objects. A test
// that pushes more than once in quick succession (this one does two
// checkpoint→push cycles) is exactly the shape that crosses gc.auto's
// thresholds. t.TempDir()'s cleanup (os.RemoveAll) then races that detached
// process: if gc is still writing/removing pack or loose-object files when
// RemoveAll walks the directory, RemoveAll can fail with ENOTEMPTY
// ("directory not empty") because a file it already listed gets
// removed/replaced out from under it, or a new file appears after it thought
// the directory was empty. That is the exact failure signature CI hit:
// "TempDir RemoveAll cleanup: unlinkat .../002: directory not empty" — path
// "002" being the bare remote directory, the second t.TempDir() call in that
// test.
//
// This is a test-lifecycle issue, not a product bug: a real user's repos are
// never subject to a test harness deleting them out from under a background
// gc a few milliseconds after a push returns. The fix belongs here, not in
// rekal's own git invocations.
func disableAutoGC(t *testing.T, dir string) {
	t.Helper()
	for _, kv := range [][2]string{
		{"gc.auto", "0"},
		{"receive.autogc", "false"},
	} {
		if err := exec.Command("git", "-C", dir, "config", kv[0], kv[1]).Run(); err != nil {
			t.Fatalf("git -C %s config %s %s: %v", dir, kv[0], kv[1], err)
		}
	}
}

// NewTestEnvAt creates a TestEnv pointing at an existing git repo directory.
func NewTestEnvAt(t *testing.T, dir string) *TestEnv {
	t.Helper()
	return &TestEnv{T: t, RepoDir: dir}
}

// RunCLI executes rekal with the given args from the test repo directory.
// Returns stdout, stderr, and error.
func (env *TestEnv) RunCLI(args ...string) (stdout, stderr string, err error) {
	env.T.Helper()
	rootCmd := cli.NewRootCmd()
	rootCmd.SetArgs(args)

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)

	// Change to repo dir; restore after.
	oldDir, _ := os.Getwd()
	_ = os.Chdir(env.RepoDir)
	defer func() { _ = os.Chdir(oldDir) }()

	execErr := rootCmd.Execute()
	return outBuf.String(), errBuf.String(), execErr
}

// Init runs `rekal init` and fails if it errors.
func (env *TestEnv) Init() {
	env.T.Helper()
	_, _, err := env.RunCLI("init")
	if err != nil {
		env.T.Fatalf("rekal init: %v", err)
	}
}

// FileExists checks whether a file exists under the repo dir.
func (env *TestEnv) FileExists(relPath string) bool {
	_, err := os.Stat(filepath.Join(env.RepoDir, relPath))
	return err == nil
}

// ReadFile reads a file from the repo dir.
func (env *TestEnv) ReadFile(relPath string) string {
	data, err := os.ReadFile(filepath.Join(env.RepoDir, relPath))
	if err != nil {
		env.T.Fatalf("read %s: %v", relPath, err)
	}
	return string(data)
}

// --- Init command tests ---

func TestInit_CreatesRekalDir(t *testing.T) {
	env := NewTestEnv(t)
	stdout, _, err := env.RunCLI("init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if !strings.Contains(stdout, "Rekal initialized.") {
		t.Errorf("expected success message, got: %q", stdout)
	}
	if !env.FileExists(".rekal") {
		t.Error(".rekal/ should exist after init")
	}
	if !env.FileExists(".rekal/data.db") {
		t.Error("data.db should exist after init")
	}
	if !env.FileExists(".rekal/index.db") {
		t.Error("index.db should exist after init")
	}
}

func TestInit_GitignoreEntry(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()
	content := env.ReadFile(".gitignore")
	if !strings.Contains(content, ".rekal/") {
		t.Error(".gitignore should contain .rekal/")
	}
}

func TestInit_InstallsHooks(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()
	postCommit := env.ReadFile(".git/hooks/post-commit")
	if !strings.Contains(postCommit, "# managed by rekal") {
		t.Error("post-commit should contain rekal marker")
	}
	if !strings.Contains(postCommit, "rekal checkpoint") {
		t.Error("post-commit should call rekal checkpoint")
	}
}

func TestInit_Reinit(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	stdout, _, err := env.RunCLI("init")
	if err != nil {
		t.Fatalf("reinit: %v", err)
	}
	if !strings.Contains(stdout, "already initialized") {
		t.Errorf("reinit should say already initialized, got: %q", stdout)
	}
}

// TestInit_ReinitRefreshesSkills is the upgrade path: after Rekal is already
// initialized, a second `rekal init` must re-install the version-managed skill
// tree (tip + scripts + references) without disturbing the store.
func TestInit_ReinitRefreshesSkills(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	base := filepath.Join(".claude", "skills", "rekal", "SKILL.md")
	script := filepath.Join(".claude", "skills", "rekal", "scripts", "route.py")
	if !env.FileExists(base) {
		t.Fatalf("base skill missing after init: %s", base)
	}
	if !env.FileExists(script) {
		t.Fatalf("skill script missing after init: %s", script)
	}

	// Simulate a stale tree: delete a gate script; re-init must restore it.
	if err := os.Remove(filepath.Join(env.RepoDir, script)); err != nil {
		t.Fatalf("remove skill script: %v", err)
	}
	if env.FileExists(script) {
		t.Fatalf("skill script should be gone before refresh")
	}

	// Re-init must not rebuild the store, but must restore the missing script.
	if env.FileExists(".rekal/data.db") == false {
		t.Fatalf("data.db missing before reinit")
	}
	stdout, _, err := env.RunCLI("init")
	if err != nil {
		t.Fatalf("reinit: %v", err)
	}
	if !strings.Contains(stdout, "Refreshed skills and hooks") {
		t.Errorf("reinit should report a refresh, got: %q", stdout)
	}
	if !env.FileExists(script) {
		t.Errorf("reinit should have restored the missing skill script %s", script)
	}
	if !env.FileExists(".rekal/data.db") {
		t.Errorf("reinit must not disturb the store")
	}
}

func TestInit_NotGitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rootCmd := cli.NewRootCmd()
	rootCmd.SetArgs([]string{"init"})
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)

	oldDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(oldDir) }()

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("init outside git repo should fail")
	}
	if !strings.Contains(errBuf.String(), "not a git repository") {
		t.Errorf("expected git repo error, got: %q", errBuf.String())
	}
}

// --- Clean command tests ---

func TestClean_RemovesRekalDir(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	stdout, _, err := env.RunCLI("clean")
	if err != nil {
		t.Fatalf("clean: %v", err)
	}
	if !strings.Contains(stdout, "Rekal cleaned.") {
		t.Errorf("expected clean message, got: %q", stdout)
	}
	if env.FileExists(".rekal") {
		t.Error(".rekal/ should not exist after clean")
	}
}

func TestClean_RemovesHooks(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	env.RunCLI("clean")
	if env.FileExists(".git/hooks/post-commit") {
		t.Error("post-commit hook should be removed after clean")
	}
}

func TestClean_Idempotent(t *testing.T) {
	env := NewTestEnv(t)
	stdout, _, err := env.RunCLI("clean")
	if err != nil {
		t.Fatalf("clean (no init): %v", err)
	}
	if !strings.Contains(stdout, "Rekal cleaned.") {
		t.Error("expected clean message")
	}
}

// --- Stub command tests ---

func TestStubCommands_RequirePreconditions(t *testing.T) {
	commands := []string{"checkpoint", "push", "index", "log", "sync"}

	for _, name := range commands {
		name := name
		t.Run(name+"_not_git_repo", func(t *testing.T) {
			dir := t.TempDir()
			rootCmd := cli.NewRootCmd()
			rootCmd.SetArgs([]string{name})
			errBuf := &bytes.Buffer{}
			rootCmd.SetErr(errBuf)

			oldDir, _ := os.Getwd()
			_ = os.Chdir(dir)
			defer func() { _ = os.Chdir(oldDir) }()

			err := rootCmd.Execute()
			if err == nil {
				t.Fatalf("%s should fail outside git repo", name)
			}
			if !strings.Contains(errBuf.String(), "not a git repository") {
				t.Errorf("%s: expected git repo error, got: %q", name, errBuf.String())
			}
		})
	}
}

func TestStubCommands_RequireInit(t *testing.T) {
	commands := []string{"checkpoint", "push", "index", "log", "sync"}

	for _, name := range commands {
		name := name
		t.Run(name+"_not_initialized", func(t *testing.T) {
			env := NewTestEnv(t)
			_, stderr, err := env.RunCLI(name)
			if err == nil {
				t.Fatalf("%s should fail without init", name)
			}
			if !strings.Contains(stderr, "rekal not initialized") {
				t.Errorf("%s: expected init error, got: %q", name, stderr)
			}
		})
	}
}

// TestStubCommands_NotYetImplemented was removed — all commands are now implemented.

func TestQuery_RequiresArg(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	_, _, err := env.RunCLI("query")
	if err == nil {
		t.Fatal("query without SQL arg should fail")
	}
}

func TestQuery_ExecutesSQL(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	stdout, _, err := env.RunCLI("query", "SELECT 1 AS val")
	if err != nil {
		t.Fatalf("query should succeed: %v", err)
	}
	if !strings.Contains(stdout, "val") {
		t.Errorf("expected query result with 'val', got: %q", stdout)
	}
}

func TestQuery_ExplicitSQLFlag(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	// Explicit SQL mode must produce the same result as the positional shorthand.
	stdout, _, err := env.RunCLI("query", "--sql", "SELECT 1 AS val")
	if err != nil {
		t.Fatalf("query --sql should succeed: %v", err)
	}
	if !strings.Contains(stdout, "val") {
		t.Errorf("expected --sql result with 'val', got: %q", stdout)
	}

	// --sql and a positional statement conflict.
	if _, _, err := env.RunCLI("query", "--sql", "SELECT 1", "SELECT 2"); err == nil {
		t.Error("--sql plus a positional statement should be rejected")
	}
	// --sql and --session conflict.
	if _, _, err := env.RunCLI("query", "--sql", "SELECT 1", "--session", "s1"); err == nil {
		t.Error("--sql plus --session should be rejected")
	}
}

func TestQuery_SchemaHelpDocumentsFullColumns(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	// The queryable columns that drifted in past — --help must name them so an
	// agent building raw SQL can discover them.
	help, _, err := env.RunCLI("query", "--help")
	if err != nil {
		t.Fatalf("query --help: %v", err)
	}
	for _, col := range []string{"agent_type", "description", "spawn_depth", "origin", "facet_text", "knowledge_chunks", "knowledge_embeddings"} {
		if !strings.Contains(help, col) {
			t.Errorf("query --help must document %q in the schema", col)
		}
	}
}

func TestRecall_ProducesJSON(t *testing.T) {
	env := NewTestEnv(t)
	env.Init()

	stdout, _, err := env.RunCLI("--file", "foo", "JWT")
	if err != nil {
		t.Fatalf("recall should succeed: %v", err)
	}
	if !strings.Contains(stdout, `"results"`) {
		t.Errorf("expected JSON output with 'results', got: %q", stdout)
	}
}

func TestRecall_NoArgsShowsHelp(t *testing.T) {
	env := NewTestEnv(t)
	stdout, _, err := env.RunCLI()
	if err != nil {
		t.Fatalf("root with no args: %v", err)
	}
	if !strings.Contains(stdout, "Rekal") {
		t.Errorf("expected help output, got: %q", stdout)
	}
}
