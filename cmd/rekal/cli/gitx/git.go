// Package gitx holds the thin git-plumbing helpers shared by the command and
// transport layers. It sits below both so they can share it without an import
// cycle, and it wraps only os/exec — no rekal domain logic beyond the orphan
// branch name.
package gitx

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// mainWorktreeCache memoizes MainWorktreeRoot per input path. The resolution
// shells out to git, and the .rekal path builders call it several times per
// command run; the mapping is stable for a process's lifetime.
var mainWorktreeCache sync.Map // gitRoot → resolved main worktree root

// MainWorktreeRoot returns the repository's primary (main) worktree root — the
// checkout every linked worktree shares a store with. For a normal repository
// this is gitRoot itself (a no-op, so existing installs need no migration).
// For a `git worktree add` checkout it is the main checkout, so Rekal's
// .rekal/ store resolves to one shared place across all worktrees. Falls back
// to gitRoot on any error.
//
// This is why worktree support needs no relocation into .git and no cutover:
// the store stays a normal gitignored directory in the main checkout, and
// linked worktrees simply resolve to it.
func MainWorktreeRoot(gitRoot string) string {
	if v, ok := mainWorktreeCache.Load(gitRoot); ok {
		return v.(string)
	}
	root := resolveMainWorktreeRoot(gitRoot)
	mainWorktreeCache.Store(gitRoot, root)
	return root
}

func resolveMainWorktreeRoot(gitRoot string) string {
	// `git worktree list --porcelain` lists the main worktree first, then any
	// linked ones. This is robust where the parent-of-git-common-dir heuristic
	// is not (submodules, where the common dir lives under .git/modules/<name>).
	out, err := exec.Command("git", "-C", gitRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return gitRoot
	}
	for _, line := range strings.Split(string(out), "\n") {
		if rest, ok := strings.CutPrefix(line, "worktree "); ok {
			if path := strings.TrimSpace(rest); path != "" {
				return filepath.Clean(path)
			}
		}
	}
	return gitRoot
}

// HeadSHA returns the current HEAD commit SHA, or 40 zeros if it can't be read.
func HeadSHA(gitRoot string) string {
	out, err := exec.Command("git", "-C", gitRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return strings.Repeat("0", 40)
	}
	return strings.TrimSpace(string(out))
}

// CurrentBranch returns the current branch name, or "unknown" on error.
func CurrentBranch(gitRoot string) string {
	out, err := exec.Command("git", "-C", gitRoot, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// FilesChanged returns the name-status lines of the HEAD~1..HEAD diff.
func FilesChanged(gitRoot string) []string {
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

// ShowFile reads a file from a git ref (ref:path). Returns nil if not found.
func ShowFile(gitRoot, ref, path string) []byte {
	out, err := exec.Command("git", "-C", gitRoot, "show", ref+":"+path).Output()
	if err != nil {
		return nil
	}
	return out
}

// TrackedBlobs returns path → blob SHA for every tracked file at ref
// (`git ls-tree -r <ref>`). Git content-addresses every tracked file, so this
// is a free, exact change detector: the knowledge layer compares it against
// the blob SHAs it indexed to find files needing a re-chunk. Returns nil on
// error (e.g. a repo with no commits).
func TrackedBlobs(gitRoot, ref string) map[string]string {
	// -z: NUL-separated entries with raw (unquoted) paths — without it git
	// C-quotes paths containing spaces/UTF-8, and a quoted path would never
	// match the blob store, leaving those files permanently unindexed.
	out, err := exec.Command("git", "-C", gitRoot, "ls-tree", "-r", "-z", ref).Output()
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	for _, entry := range strings.Split(string(out), "\x00") {
		// Format: "<mode> blob <sha>\t<path>"
		tab := strings.IndexByte(entry, '\t')
		if tab < 0 {
			continue
		}
		meta := strings.Fields(entry[:tab])
		if len(meta) != 3 || meta[1] != "blob" {
			continue
		}
		result[entry[tab+1:]] = meta[2]
	}
	return result
}

// BlobContents returns blob SHA → content for the given SHAs through one
// `git cat-file --batch` process — the bulk read for the knowledge layer's
// full build, where per-file `git show` would cost one process spawn per
// prose file (an Obsidian-vault-sized repo has thousands). Missing SHAs are
// simply absent from the result; nil on process error.
func BlobContents(gitRoot string, shas []string) map[string][]byte {
	if len(shas) == 0 {
		return map[string][]byte{}
	}
	cmd := exec.Command("git", "-C", gitRoot, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(strings.Join(shas, "\n") + "\n")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	result := make(map[string][]byte, len(shas))
	rest := out
	for len(rest) > 0 {
		nl := bytes.IndexByte(rest, '\n')
		if nl < 0 {
			break
		}
		header := string(rest[:nl])
		rest = rest[nl+1:]
		// Header: "<sha> <type> <size>" or "<sha> missing".
		fields := strings.Fields(header)
		if len(fields) < 3 {
			continue // "missing" (or malformed) — no payload follows
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size > len(rest) {
			break
		}
		if fields[1] == "blob" {
			result[fields[0]] = rest[:size]
		}
		rest = rest[size:]
		if len(rest) > 0 && rest[0] == '\n' {
			rest = rest[1:]
		}
	}
	return result
}

// LastCommitShort returns the abbreviated SHA of the last commit touching
// path, or "" when unknown.
func LastCommitShort(gitRoot, path string) string {
	out, err := exec.Command("git", "-C", gitRoot, "log", "-1", "--format=%h", "--", path).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ConfigValue reads a git config value, or "" if unset.
func ConfigValue(key string) string {
	out, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// HashObject writes data to the git object store and returns its blob SHA.
func HashObject(gitRoot string, data []byte) (string, error) {
	cmd := exec.Command("git", "-C", gitRoot, "hash-object", "-w", "--stdin")
	cmd.Stdin = strings.NewReader(string(data))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RekalBranchName returns the per-user orphan branch name, rekal/<email>,
// falling back to rekal/local when no git user.email is configured.
func RekalBranchName() string {
	email := strings.TrimSpace(ConfigValue("user.email"))
	if email == "" {
		email = "local"
	}
	return "rekal/" + email
}

// DefaultBranch resolves the ref that represents merged mainline history — the
// reference the merged-only export filter tests against. It prefers the
// remote's declared default (origin/HEAD → origin/main → origin/master), so
// "merged" means "landed on the shared remote", then falls back to a local
// main/master for repos without a remote. Returns "" when nothing resolves
// (e.g. a repo with no commits), which the caller treats as "share nothing".
func DefaultBranch(gitRoot string) string {
	if out, err := exec.Command("git", "-C", gitRoot,
		"symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output(); err == nil {
		if ref := strings.TrimSpace(string(out)); ref != "" {
			return ref
		}
	}
	for _, ref := range []string{"origin/main", "origin/master", "main", "master"} {
		if exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", "--quiet", ref).Run() == nil {
			return ref
		}
	}
	return ""
}

// IsAncestor reports whether commit sha is an ancestor of (reachable from) ref.
// Because a merge-commit or rebase workflow lands a branch's commits into the
// mainline's history, this is an exact "did this session's code merge?" test
// for those workflows. It is deliberately conservative: a squash-merge rewrites
// the branch into a new commit, so the original sha is not an ancestor and the
// session is held back (fail-closed) rather than risk sharing unmerged work.
func IsAncestor(gitRoot, sha, ref string) bool {
	if sha == "" || ref == "" {
		return false
	}
	return exec.Command("git", "-C", gitRoot, "merge-base", "--is-ancestor", sha, ref).Run() == nil
}

// BranchTip returns the commit sha of a local branch, or "" when the branch
// doesn't exist (e.g. deleted after merge). The merged-only export gate probes
// a held-back checkpoint's surviving branch tip as a squash candidate, so
// mid-branch checkpoints release even when no checkpoint exists at the tip.
func BranchTip(gitRoot, branch string) string {
	if branch == "" {
		return ""
	}
	out, err := exec.Command("git", "-C", gitRoot,
		"rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// IsSquashMergedInto reports whether the cumulative change of tip (relative to
// its merge-base with ref) landed on ref as a patch-equivalent commit — i.e.
// the branch was squash-merged. This is the second signal of the merged-only
// export gate, covering the workflow IsAncestor cannot: a squash merge rewrites
// the branch into one new commit, so the original commits never become
// ancestors of the mainline.
//
// Technique (the standard squashed-branch detection, git-only): synthesize an
// unreferenced commit carrying tip's tree parented on the merge-base — its
// diff is the branch's whole cumulative change — then ask `git cherry` whether
// ref contains a commit with the same patch-id. Patch-ids are stable across
// line offsets, so the match survives the mainline having advanced before the
// squash landed.
//
// Fail-closed by construction: an abandoned or never-merged branch has no
// patch-equivalent commit on ref, and a tip whose tree equals the merge-base
// (empty cumulative diff — e.g. only --allow-empty commits) is rejected
// outright rather than allowed to false-match another empty commit. The
// synthetic commit is never referenced by any ref; git gc removes it.
func IsSquashMergedInto(gitRoot, tip, ref string) bool {
	if tip == "" || ref == "" {
		return false
	}
	mbOut, err := exec.Command("git", "-C", gitRoot, "merge-base", tip, ref).Output()
	if err != nil {
		return false
	}
	mergeBase := strings.TrimSpace(string(mbOut))

	// Empty cumulative diff can never prove a squash landed — fail closed.
	if exec.Command("git", "-C", gitRoot, "diff", "--quiet", mergeBase, tip).Run() == nil {
		return false
	}

	// Fixed probe identity: commit-tree requires one, and the identity only
	// affects the throwaway commit's own sha — patch-ids ignore it.
	synOut, err := exec.Command("git", "-C", gitRoot,
		"-c", "user.name=rekal", "-c", "user.email=rekal@probe",
		"commit-tree", tip+"^{tree}", "-p", mergeBase, "-m", "rekal: squash-merge probe").Output()
	if err != nil {
		return false
	}
	synthetic := strings.TrimSpace(string(synOut))

	cherryOut, err := exec.Command("git", "-C", gitRoot, "cherry", ref, synthetic).Output()
	if err != nil {
		return false
	}
	// `git cherry` prefixes "-" when an equivalent change exists upstream.
	line := strings.TrimSpace(string(cherryOut))
	return strings.HasPrefix(line, "-")
}
