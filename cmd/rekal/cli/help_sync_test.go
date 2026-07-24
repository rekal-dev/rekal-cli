package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// publicCommands are user-facing subcommands (excludes hidden _nomic-daemon).
// Root recall is tested separately as "" / root.
var publicCommands = []string{
	"init", "clean", "version",
	"checkpoint", "push", "sync", "log",
	"query", "find", "index", "embed",
}

// commandsWithSpec must have docs/spec/command/<name>.md (version is too thin).
var commandsWithSpec = []string{
	"init", "clean", "checkpoint", "push", "sync", "log",
	"query", "find", "index", "embed",
	// root recall is documented as recall.md
}

// TestHelpSync_OutputDefaults guards the agent-first defaults that docs and
// --help must stay aligned with. If a default flips, update Long/flag help
// and docs/spec/command/ in the same change (CLAUDE.md rule).
func TestHelpSync_OutputDefaults(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	cases := []struct {
		name     string
		cmdPath  []string // empty = root
		mustHave []string
		mustNot  []string
	}{
		{
			name: "root",
			mustHave: []string{
				"seed digest",
				"rekal find",
				"rekal embed",
				"--json",
			},
			mustNot: []string{"rekal when", "--also", "rekal show"},
		},
		{
			name:    "query",
			cmdPath: []string{"query"},
			mustHave: []string{
				"readable turn transcript",
				"TSV by default",
				"NDJSON with --json",
				"JSON instead of the default text/TSV",
			},
			mustNot: []string{
				"returns the full conversation as JSON",
				"SQL rows are already NDJSON",
				"output is one JSON object per row (NDJSON)",
			},
		},
		{
			name:     "index",
			cmdPath:  []string{"index"},
			mustHave: []string{"atomically replaces", "temporary file"},
			mustNot:  []string{"Drop and rebuild the index DB"},
		},
		{
			name:     "find",
			cmdPath:  []string{"find"},
			mustHave: []string{"complete, time-ordered", "rekal query --session"},
		},
		{
			name:     "embed",
			cmdPath:  []string{"embed"},
			mustHave: []string{"budgeted bites", "rekal index"},
		},
		{
			name:     "push",
			cmdPath:  []string{"push"},
			mustHave: []string{"merged", "rekal/<email>"},
		},
		{
			name:     "sync",
			cmdPath:  []string{"sync"},
			mustHave: []string{"rekal branches", "--self"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := lookupCmd(t, root, tc.cmdPath)
			text := helpText(cmd)
			for _, want := range tc.mustHave {
				if !strings.Contains(text, want) {
					t.Errorf("help missing %q\n---\n%s", want, text)
				}
			}
			for _, ban := range tc.mustNot {
				if strings.Contains(text, ban) {
					t.Errorf("help still contains stale %q", ban)
				}
			}
		})
	}
}

// TestHelpSync_PublicCommandInventory evidences every public cobra command
// is registered, and every documented command (except version) has a spec file.
func TestHelpSync_PublicCommandInventory(t *testing.T) {
	t.Parallel()
	root := NewRootCmd()

	registered := map[string]bool{}
	for _, c := range root.Commands() {
		if c.Hidden || c.Name() == "help" {
			continue
		}
		registered[c.Name()] = true
	}
	for _, name := range publicCommands {
		if !registered[name] {
			t.Errorf("public command %q missing from cobra registration", name)
		}
	}
	for name := range registered {
		if name == "completion" {
			continue
		}
		found := false
		for _, want := range publicCommands {
			if want == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected public command %q — add to publicCommands + docs if intentional", name)
		}
	}

	specDir := filepath.Join(repoRoot(t), "docs", "spec", "command")
	for _, name := range commandsWithSpec {
		path := filepath.Join(specDir, name+".md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing spec for %s: %v", name, err)
		}
	}
	// Root recall is recall.md, not rekal.md
	if _, err := os.Stat(filepath.Join(specDir, "recall.md")); err != nil {
		t.Errorf("missing root recall spec: %v", err)
	}

	// Spec index must list find + embed (regression for the gap that started this sync).
	indexBody, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "spec", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"command/find.md", "command/embed.md", "Output defaults"} {
		if !strings.Contains(string(indexBody), needle) {
			t.Errorf("docs/spec/README.md missing %q", needle)
		}
	}
}

// TestHelpSync_SpecPhrasesMatchDefaults evidences authoritative specs still
// state the live output/rebuild/budget contracts (not historical JSON-default).
func TestHelpSync_SpecPhrasesMatchDefaults(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	checks := []struct {
		rel      string
		mustHave []string
		mustNot  []string
	}{
		{
			rel: "docs/spec/command/recall.md",
			mustHave: []string{
				"seed digest",
				"rekal query --session",
				"`--json`",
			},
			mustNot: []string{"rekal show"},
		},
		{
			rel: "docs/spec/command/query.md",
			mustHave: []string{
				"TSV rows by default",
				"Readable turn transcript by default",
			},
			mustNot: []string{"SQL mode already emits compact NDJSON"},
		},
		{
			rel: "docs/spec/command/index.md",
			mustHave: []string{
				"index.db.rebuilding",
				"atomically",
			},
			mustNot: []string{"Drop and recreate all index tables"},
		},
		{
			rel: "docs/spec/command/embed.md",
			mustHave: []string{
				"up to 16",
				"sessionEmbedBudget",
				"knowledgeEmbedBudget",
			},
			mustNot: []string{"up to 256"},
		},
		{
			rel:      "docs/spec/command/find.md",
			mustHave: []string{"rekal find", "complete", "time-ordered"},
		},
		{
			rel: "README.md",
			mustHave: []string{
				"`rekal find",
				"`rekal embed`",
				"[--json]",
			},
		},
	}
	for _, c := range checks {
		t.Run(c.rel, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(root, c.rel))
			if err != nil {
				t.Fatal(err)
			}
			s := string(body)
			for _, want := range c.mustHave {
				if !strings.Contains(s, want) {
					t.Errorf("%s missing %q", c.rel, want)
				}
			}
			for _, ban := range c.mustNot {
				if strings.Contains(s, ban) {
					t.Errorf("%s still contains stale %q", c.rel, ban)
				}
			}
		})
	}
}

// TestHelpSync_EmbedBudgets evidences code constants match the documented bite size.
func TestHelpSync_EmbedBudgets(t *testing.T) {
	t.Parallel()
	if sessionEmbedBudget != 16 {
		t.Errorf("sessionEmbedBudget = %d, docs claim 16", sessionEmbedBudget)
	}
	if knowledgeEmbedBudget != 16 {
		t.Errorf("knowledgeEmbedBudget = %d, docs claim 16", knowledgeEmbedBudget)
	}
}

// TestHelpSync_QueryFlags evidences the query flag surface stays complete.
func TestHelpSync_QueryFlags(t *testing.T) {
	t.Parallel()
	cmd := lookupCmd(t, NewRootCmd(), []string{"query"})
	want := []string{"sql", "index", "session", "full", "offset", "limit", "role", "json"}
	for _, name := range want {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("query missing flag --%s", name)
		}
	}
}

// TestHelpSync_RecallFlags evidences root recall flags stay complete.
func TestHelpSync_RecallFlags(t *testing.T) {
	t.Parallel()
	cmd := NewRootCmd()
	want := []string{"file", "commit", "author", "actor", "limit", "explain", "json"}
	for _, name := range want {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("root missing flag --%s", name)
		}
	}
}

func lookupCmd(t *testing.T, root *cobra.Command, path []string) *cobra.Command {
	t.Helper()
	cmd := root
	for _, name := range path {
		var next *cobra.Command
		for _, c := range cmd.Commands() {
			if c.Name() == name {
				next = c
				break
			}
		}
		if next == nil {
			t.Fatalf("command %q not found under %v", name, path)
		}
		cmd = next
	}
	return cmd
}

func helpText(cmd *cobra.Command) string {
	var b strings.Builder
	b.WriteString(cmd.Long)
	b.WriteByte('\n')
	b.WriteString(cmd.Short)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		b.WriteByte('\n')
		b.WriteString(f.Name)
		b.WriteByte(' ')
		b.WriteString(f.Usage)
	})
	return b.String()
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// cmd/rekal/cli/help_sync_test.go → repo root is ../../..
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
