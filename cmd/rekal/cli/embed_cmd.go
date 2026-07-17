package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/spf13/cobra"
)

// sessionEmbedBudget caps how many *uncached* session vectors one embed bite
// will request from the model. Cache hits are free and do not count. Matches
// the knowledge-chunk budget order of magnitude so one background pass
// releases the DuckDB write lock between bites (recall/checkpoint can
// interleave) without stalling a large Cohere/HTTP corpus for minutes.
const sessionEmbedBudget = 256

func newEmbedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "embed",
		Short: "Fill missing semantic embeddings (resumable)",
		Long: `Fill missing deep-semantic vectors for sessions and knowledge chunks.

Structural indexing (FTS, LSA, knowledge chunks) stays on 'rekal index' /
'rekal sync'. Semantic vectors are network-bound and already soft-fail —
this command fills them in budgeted bites, releasing the index lock between
passes so recall can run meanwhile. Safe to interrupt and re-run.

'rekal index' and 'rekal sync' start this in the background after the
structural rebuild finishes; you can also run it by hand.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}
			return runEmbed(cmd.ErrOrStderr(), gitRoot)
		},
	}
}

// runEmbed fills missing session + knowledge semantic vectors in budgeted
// bites. Each bite opens the index, embeds up to the budget, then closes —
// DuckDB is single-writer, so releasing between bites lets recall/checkpoint
// interleave. Idempotent and resumable.
func runEmbed(w io.Writer, gitRoot string) error {
	unlock, err := tryEmbedLock(gitRoot)
	if err != nil {
		fmt.Fprintf(w, "rekal: embed already running (or lock busy) — %v\n", err)
		return nil
	}
	defer unlock()

	for pass := 1; ; pass++ {
		more, err := embedBite(w, gitRoot, pass)
		if err != nil {
			return err
		}
		if !more {
			fmt.Fprintln(w, "rekal: semantic embeddings up to date")
			return nil
		}
		// Brief yield so a waiting recall/checkpoint can grab the lock.
		time.Sleep(50 * time.Millisecond)
	}
}

// embedBite runs one budgeted session + knowledge embed pass. more is true
// when either layer still has uncached work remaining after this bite.
func embedBite(w io.Writer, gitRoot string, pass int) (more bool, err error) {
	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		return false, err
	}
	defer indexDB.Close() //nolint:errcheck

	if err := db.LoadFTSExtension(indexDB); err != nil {
		return false, err
	}
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		return false, err
	}

	sessionContent, err := db.QuerySessionContent(indexDB)
	if err != nil {
		return false, fmt.Errorf("query session content: %w", err)
	}

	sessionMore := false
	if len(sessionContent) > 0 {
		remaining, err := buildSemanticEmbeddings(indexDB, sessionContent, w, gitRoot, sessionEmbedBudget)
		if err != nil {
			fmt.Fprintf(w, "rekal: warning: session embeddings pass %d: %v\n", pass, err)
		} else {
			sessionMore = remaining
		}
	}

	if err := embedKnowledgeChunks(w, indexDB, gitRoot, knowledgeEmbedBudget); err != nil {
		fmt.Fprintf(w, "rekal: warning: knowledge embeddings pass %d: %v\n", pass, err)
	}
	// Ask whether knowledge still has missing vectors under the intended model.
	knowledgeMore := false
	if model := intendedEmbedModel(gitRoot); model != "" {
		missing, merr := db.QueryKnowledgeChunksMissingEmbeddings(indexDB, model, 1)
		if merr == nil && len(missing) > 0 {
			knowledgeMore = true
		}
	}

	return sessionMore || knowledgeMore, nil
}

// startBackgroundEmbed detaches 'rekal embed' after a structural index/sync
// rebuild. Failures to spawn are warnings only — the index is already
// searchable via keyword/LSA/facets; vectors converge on the next embed,
// checkpoint, or manual 'rekal embed'.
func startBackgroundEmbed(w io.Writer, gitRoot string) {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(w, "rekal: warning: could not start background embed: %v\n", err)
		return
	}
	logPath := filepath.Join(RekalDir(gitRoot), "embed.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(w, "rekal: warning: could not open %s: %v\n", logPath, err)
		return
	}

	cmd := exec.Command(exe, "embed") //nolint:gosec // self-invocation
	cmd.Dir = gitRoot
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = detachSysProcAttr()
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		fmt.Fprintf(w, "rekal: warning: background embed failed to start: %v\n", err)
		return
	}
	// Child owns the log fd after Start; we must not wait on it.
	_ = logFile.Close()
	go func() { _ = cmd.Wait() }() // reap zombie; ignore exit status
	fmt.Fprintf(w, "rekal: semantic embeddings continuing in background (%s)\n", logPath)
}

// tryEmbedLock acquires an exclusive non-blocking lock so only one embed
// worker runs per store. unlock releases it.
func tryEmbedLock(gitRoot string) (unlock func(), err error) {
	path := filepath.Join(RekalDir(gitRoot), "embed.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := flockExclusiveNB(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		_ = flockUnlock(f)
		_ = f.Close()
	}, nil
}

// limitStringMap returns at most n entries of m with stable key order. n <= 0
// means no limit.
func limitStringMap(m map[string]string, n int) map[string]string {
	if n <= 0 || len(m) <= n {
		return m
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]string, n)
	for _, k := range keys[:n] {
		out[k] = m[k]
	}
	return out
}
