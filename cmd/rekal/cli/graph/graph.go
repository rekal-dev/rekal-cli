// Package graph is the write-ahead spool for the L1 recall citation graph.
//
// When an agent recalls or drills, rekal appends one edge per reached session
// to a lock-free NDJSON spool in the store (Append). The spool exists purely so
// the hot recall path never grabs data.db's single writer — capture stays a
// small file append. At checkpoint (which already holds the data.db writer) the
// spool is drained (Drain) into data.db.recall_edges, the permanent record, and
// the derived index.db.session_reach aggregate is refreshed.
//
// The spool is transient, not the store. See docs/design/recall-graph.md.
package graph

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// spoolName is the write-ahead log file inside the store dir. It is under
// .rekal/, which is gitignored, so it never reaches git.
const spoolName = "recall-log.ndjson"

// Edge is one recall-graph event: a session reached (recalled or drilled) while
// working. It carries no id — the id is assigned when it lands in
// data.db.recall_edges at drain time.
type Edge struct {
	TS     time.Time `json:"ts"`
	Kind   string    `json:"kind"`  // "recall" | "drill"
	Query  string    `json:"query"` // omitted for drills
	Target string    `json:"target"`
}

func spoolPath(gitRoot string) string {
	return filepath.Join(db.StoreDir(gitRoot), spoolName)
}

// Append writes edges to the spool, one JSON line each. It is a no-op under the
// bench env (so benchmarks never pollute a store's graph) and best-effort: any
// error is returned for the caller to ignore, never propagated onto the hot
// recall path. Small lines under PIPE_BUF append atomically, so concurrent
// rekal processes interleave whole lines safely.
func Append(gitRoot string, edges []Edge) error {
	if len(edges) == 0 || session.BenchEnv() {
		return nil
	}
	f, err := os.OpenFile(spoolPath(gitRoot), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open recall spool: %w", err)
	}
	defer f.Close() //nolint:errcheck
	w := bufio.NewWriter(f)
	for _, e := range edges {
		line, err := json.Marshal(e)
		if err != nil {
			continue // skip an unmarshalable edge, keep the rest
		}
		if _, err := w.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("write recall spool: %w", err)
		}
	}
	return w.Flush()
}

// Drain atomically claims the current spool (rename-to-tmp so concurrent
// appenders start a fresh file) and returns its edges. A partial trailing line
// (an append caught mid-write) is tolerated — it is skipped, not fatal. When
// the spool does not exist, Drain returns nil with no error.
func Drain(gitRoot string) ([]Edge, error) {
	path := spoolPath(gitRoot)
	tmp := path + ".draining"
	if err := os.Rename(path, tmp); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim recall spool: %w", err)
	}
	f, err := os.Open(tmp)
	if err != nil {
		return nil, fmt.Errorf("open drained spool: %w", err)
	}
	var edges []Edge
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Edge
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue // partial or corrupt line — skip it
		}
		if e.Target == "" {
			continue
		}
		edges = append(edges, e)
	}
	f.Close()          //nolint:errcheck
	_ = os.Remove(tmp) //nolint:errcheck // best-effort cleanup
	if err := sc.Err(); err != nil {
		return edges, fmt.Errorf("read drained spool: %w", err)
	}
	return edges, nil
}
