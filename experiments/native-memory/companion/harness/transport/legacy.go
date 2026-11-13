package transport

import ("fmt"; "os/exec"; "strings"; "github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx")

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
