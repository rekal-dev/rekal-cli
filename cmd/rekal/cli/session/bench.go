package session

import (
	"os"
	"strings"
)

// Bench harness prompts (scripts/bench/gen_queries.py) leave distinctive
// fingerprints in captured Claude sessions. When those sessions are
// checkpointed into a real repo's data.db they pollute recall (BUG 6).

var benchFingerprints = []string{
	"Write ONE natural question they would ask",
	"Write ONE natural question a developer would later ask",
	"Write ONE natural question of the form \"have we tried",
	"Write ONE natural developer question whose answer genuinely requires facts from BOTH sessions",
	"set BENCH_LLM",
	"RekalBench",
}

// SkipCapture reports whether this payload should not be written to data.db.
// True when the operator set REKAL_BENCH / REKAL_SKIP_CHECKPOINT, the
// transcript cwd looks like a bench tree, or turn text matches harness prompts.
func SkipCapture(p *SessionPayload) bool {
	if os.Getenv("REKAL_BENCH") != "" || os.Getenv("REKAL_SKIP_CHECKPOINT") != "" {
		return true
	}
	if p == nil {
		return false
	}
	cwd := strings.ToLower(p.CWD)
	if strings.Contains(cwd, "/scripts/bench") ||
		strings.Contains(cwd, "rekal-bench") ||
		strings.Contains(cwd, "/.rekal-bench") {
		return true
	}
	for _, t := range p.Turns {
		if t.Role != "human" && t.Role != "human_steering" {
			continue
		}
		for _, fp := range benchFingerprints {
			if strings.Contains(t.Content, fp) {
				return true
			}
		}
	}
	return false
}
