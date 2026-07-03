//go:build ignore

// Command gen-session-fixtures scrubs real Claude Code session transcripts
// found on this machine into small, structurally faithful fixtures for the
// session package's real-data regression tests
// (cmd/rekal/cli/session/real_fixtures_test.go).
//
// The synthetic-fixture-vs-real-data gap has bitten this project twice
// (migration and checkpoint bugs shipped despite green suites, both only
// caught by manually validating against real captured transcripts). This
// tool makes that validation step repeatable and checked-in instead of a
// one-off manual pass: it copies real transcript *structure* (JSONL field
// shapes, subagent directory layout, meta.json sidecar format) while
// stripping anything sensitive, so the fixtures can live in the repo.
//
// It walks the source directory recursively, preserving relative paths (so
// the copied tree still looks like <session>.jsonl plus
// <session>/subagents/agent-*.jsonl plus *.meta.json sidecars — the exact
// layout session.FindSessionDir/discoverSessionRefs expect), and for every
// JSON value in every .jsonl line and every .meta.json file, applies the
// same secret-redaction and path/username-anonymization the production
// capture path applies (scrub.RedactText, scrub.AnonymizeText). Structural
// fields (type, role, operation, agentType, ...) are short strings that
// never match a secret pattern, so they pass through unchanged.
//
// Usage:
//
//	go run scripts/gen-session-fixtures.go <src-session-dir> <dest-dir> [maxLinesPerFile]
//
// To refresh the checked-in fixtures from this machine's own Claude Code
// history for this repo:
//
//	go run scripts/gen-session-fixtures.go \
//	  ~/.claude/projects/-Users-<you>-path-to-rekal-cli \
//	  cmd/rekal/cli/session/testdata/real \
//	  200
//
// Review the diff before committing — this tool redacts known secret/path
// patterns, it does not guarantee every possible sensitive value is caught.
// Only run it against transcripts you're comfortable checking into git
// history, and skim the output for anything project-specific that shouldn't
// be public (this repo's fixtures were reviewed by hand before committing).
package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/scrub"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: gen-session-fixtures <src-session-dir> <dest-dir> [maxLinesPerFile]")
		os.Exit(1)
	}
	src := os.Args[1]
	dest := os.Args[2]
	maxLines := 200
	if len(os.Args) > 3 {
		n, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid maxLinesPerFile %q: %v\n", os.Args[3], err)
			os.Exit(1)
		}
		maxLines = n
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		exitErr(err)
	}

	copied := 0
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, rel)

		switch {
		case strings.HasSuffix(path, ".jsonl"):
			if err := scrubJSONLFile(path, destPath, maxLines); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			copied++
		case strings.HasSuffix(path, ".meta.json"):
			if err := scrubMetaFile(path, destPath); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			copied++
		}
		return nil
	})
	if err != nil {
		exitErr(err)
	}
	fmt.Printf("scrubbed %d file(s) from %s into %s\n", copied, src, dest)
}

func scrubJSONLFile(src, dest string, maxLines int) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var out []string
	for _, line := range lines {
		if len(out) >= maxLines {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		scrubbed, err := scrubJSONLine(line)
		if err != nil {
			// A real transcript can contain a malformed/partial trailing
			// line (e.g. captured mid-write) — skip it rather than fail
			// the whole fixture, same tolerance the parser itself has.
			continue
		}
		out = append(out, scrubbed)
	}
	return os.WriteFile(dest, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func scrubMetaFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	scrubbed, err := scrubJSONLine(strings.TrimSpace(string(data)))
	if err != nil {
		return err
	}
	return os.WriteFile(dest, []byte(scrubbed+"\n"), 0o644)
}

func scrubJSONLine(line string) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(line), &v); err != nil {
		return "", err
	}
	scrubValue(&v)
	out, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// maxStringLen caps individual string values after scrubbing. Real turns can
// carry huge tool_result blobs (file contents, command output); fixtures
// only need enough text to exercise the parser's field/shape handling, not
// the full payload, so long values are truncated to keep the checked-in
// fixture set small.
const maxStringLen = 400

// scrubValue walks a decoded JSON value in place, replacing every string leaf
// with its redacted+anonymized, length-capped form. Structural fields (type,
// role, uuid, ...) pass through unchanged because the redaction patterns
// require long, high-entropy, or clearly secret-shaped matches, and are well
// under the truncation length — see cmd/rekal/cli/scrub.
func scrubValue(v *interface{}) {
	switch val := (*v).(type) {
	case string:
		s := scrub.AnonymizeText(scrub.RedactText(val))
		if len(s) > maxStringLen {
			s = s[:maxStringLen] + "...[truncated for fixture]"
		}
		*v = s
	case map[string]interface{}:
		for k, sub := range val {
			scrubValue(&sub)
			val[k] = sub
		}
	case []interface{}:
		for i := range val {
			scrubValue(&val[i])
		}
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
