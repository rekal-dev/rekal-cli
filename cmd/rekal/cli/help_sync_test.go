package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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
