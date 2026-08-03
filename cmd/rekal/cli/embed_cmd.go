package cli

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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
structural rebuild finishes; you can also run it by hand.

Recall works without these vectors — it falls back to keyword plus LSA and
says so. Filling them improves ranking; it is never required for an answer.`,
		Example: `  # Finish or resume the vectors
  rekal embed
    building deep semantic embeddings (nomic-v1.5-c8k-d1)...
    stored 36 semantic embeddings (12 cached, 24 embedded)
    knowledge embeddings: 16 stored (0 cached, 16 embedded)

  # Already complete
  rekal embed
    rekal: semantic embeddings up to date

  # Another embed already holds the lock — safe, just wait
  rekal embed
    rekal: embed already running (or lock busy)

  # Check coverage yourself
  rekal query -i -q "SELECT model, count(*) FROM session_embeddings GROUP BY 1"
  rekal query -i -q "SELECT count(*) FROM knowledge_embeddings"

  # When it stalls, the reason is in the log
  cat .rekal/embed.log
  cat .rekal/nomic/daemon.log`,
		Args: rejectExtraArgs,
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
	// index write lock. For nomic this waits for the daemon to load the model
	// (~15s) here, lock-free, and every bite reuses this one client — so no bite
	// reloads the model under the lock, and a concurrent recall (common right
	// after init/sync) is never stuck behind a model load.
	emb, err := semanticEmbedder(gitRoot, true)
	if err != nil {
		// The daemon never came up (crash / incompatible build). Embedding is
		// this command's only job, so stop cleanly rather than spin retrying —
		// the keyword layer is already searchable; vectors fill on a later run.
		fmt.Fprintf(w, "rekal: semantic backend unavailable (%v) — skipping vectors\n", err)
		return nil
	}
	if emb == nil {
		// No semantic backend on this platform/config — nothing to embed.
		return nil
	}
	defer emb.Close()

	for pass := 1; ; pass++ {
		more, wrote, err := embedBite(w, gitRoot, pass, emb)
		if err != nil {
			return err
		}
		if !more {
			fmt.Fprintln(w, "rekal: semantic embeddings up to date")
			return nil
		}
		// A pass that wrote nothing while work remains is a stall, not
		// progress. Both halves of a bite warn and swallow their errors, and
		// knowledgeMore is recomputed from the database — so a backend that
		// fails every call leaves "vectors still missing" true forever and the
		// loop runs until something kills it. Measured at 353 identical
		// failures in one run, each printing a warning, none embedding
		// anything; index and sync spawn this in the background, so it burns a
		// core indefinitely on a broken setup. Nothing about the next pass
		// differs from this one, so stop and say where to look.
		if !wrote {
			fmt.Fprintln(w, "rekal: embed stalled — vectors are still missing and this pass wrote none")
			fmt.Fprintf(w, "rekal: see %s\n", filepath.Join(gitRoot, ".rekal", "nomic", "daemon.log"))
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

// embedBite runs one budgeted session + knowledge pass. more is true when
// either layer still has uncached work remaining after this bite; emb is the
// shared embedder from runEmbed (nil to construct per-layer as a fallback). wrote reports whether the pass actually
// stored any vector, which is how the caller tells progress from a stall — the
// two halves below warn and swallow their errors, so neither `more` nor the
// error return distinguishes them.
func embedBite(w io.Writer, gitRoot string, pass int, emb sessionEmbedder) (more bool, wrote bool, err error) {
	indexDB, err := db.OpenIndex(gitRoot)
	if err != nil {
		return false, false, err
	}
	defer indexDB.Close() //nolint:errcheck

	if err := db.LoadFTSExtension(indexDB); err != nil {
		return false, false, err
	}
	if err := db.EnsureKnowledgeSchema(indexDB); err != nil {
		return false, false, err
	}

	before := storedVectorCount(indexDB)

	sessionContent, err := db.QuerySessionContent(indexDB)
	if err != nil {
		return false, false, fmt.Errorf("query session content: %w", err)
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

	return sessionMore || knowledgeMore, storedVectorCount(indexDB) > before, nil
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
	// Never self-invoke from a test binary. `go test` ignores the unrecognised
	// "embed" argument and runs the whole suite again, and each of those runs
	// reaches this line and spawns more — a fork bomb that made the integration
	// suite roughly forty times slower and, before the timeout was raised, got it
	// killed mid-run with a goroutine dump that read like a hang. nomic's
	// spawnDaemon has carried the same guard for the same reason.
	if strings.HasSuffix(exe, ".test") || strings.Contains(exe, "/_test/") {
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

// storedVectorCount is the total number of semantic vectors in the index,
// across both layers. It is the only signal that separates a bite that did
// work from one that failed: the bite's two halves warn and swallow.
// Best-effort — a count that cannot be read reports 0, which at worst makes a
// healthy pass look like a stall and stops early rather than spinning.
func storedVectorCount(indexDB *sql.DB) int64 {
	var n int64
	if err := indexDB.QueryRow(`
		SELECT (SELECT count(*) FROM session_embeddings)
		     + (SELECT count(*) FROM knowledge_embeddings)`).Scan(&n); err != nil {
		return 0
	}
	return n
}
