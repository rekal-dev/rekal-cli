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
// will request from the model. Cache hits are free and do not count. Kept small
// so a bite holds the DuckDB write lock only briefly: on the embedded CPU-nomic
// path each vector costs tens of ms, so a large bite would pin the lock for
// minutes and a concurrent recall (common right after init/sync) would wait out
// a whole bite. A small bite bounds that wait to a couple of seconds; recall's
// open path retries across the between-bite yield (see db.open, embedYield).
const sessionEmbedBudget = 16

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

	// Construct the embedding backend once, before the bite loop ever takes the
	// index write lock. For nomic this loads the model (~15s) here, lock-free,
	// and every bite reuses this one client — so no bite reloads the model under
	// the lock, and a concurrent recall (common right after init/sync) is never
	// stuck behind a model load. nil is fine (unsupported platform): bites fall
	// back to constructing their own, as before.
	emb, err := semanticEmbedder(gitRoot)
	if err != nil {
		fmt.Fprintf(w, "rekal: warning: embedding backend: %v\n", err)
		emb = nil
	}
	if emb != nil {
		defer emb.Close()
	}

	for pass := 1; ; pass++ {
		more, err := embedBite(w, gitRoot, pass, emb)
		if err != nil {
			return err
		}
		if !more {
			fmt.Fprintln(w, "rekal: semantic embeddings up to date")
			return nil
		}
		// Yield the lock between bites, longer than a reader's max open-retry
		// interval (db.openLockRetryMax) so a waiting recall/checkpoint reliably
		// wins this window instead of racing the next bite's re-acquire.
		time.Sleep(embedYield)
	}
}

// embedYield is the gap between bites during which the index write lock is free.
// It exceeds db.openLockRetryMax so a retrying reader is guaranteed at least one
// open attempt while the lock is released.
const embedYield = 300 * time.Millisecond

// embedBite runs one budgeted session + knowledge embed pass. more is true
// when either layer still has uncached work remaining after this bite. emb is
// the shared embedder from runEmbed (nil to construct per-layer as a fallback).
func embedBite(w io.Writer, gitRoot string, pass int, emb sessionEmbedder) (more bool, err error) {
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
		remaining, err := buildSemanticEmbeddings(indexDB, sessionContent, w, gitRoot, sessionEmbedBudget, emb)
		if err != nil {
			fmt.Fprintf(w, "rekal: warning: session embeddings pass %d: %v\n", pass, err)
		} else {
			sessionMore = remaining
		}
	}

	if err := embedKnowledgeChunks(w, indexDB, gitRoot, knowledgeEmbedBudget, emb); err != nil {
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
