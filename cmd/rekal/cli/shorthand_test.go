package cli

import (
	"testing"

	"github.com/spf13/pflag"
)

// TestFlagShorthands pins every registered short flag.
//
// Shorthands are part of the CLI contract the moment they ship: a script or a
// habit forms around `-n`, and silently moving it to another flag breaks that
// caller in the worst possible way — it still runs, with a different meaning.
// The map is the spec, and docs/commands.md must agree with it.
//
// It also catches the collision cobra cannot: the same letter bound to two
// different concepts across commands reads as one flag to anyone using both.
func TestFlagShorthands(t *testing.T) {
	t.Parallel()

	want := map[string]map[string]string{ // command → shorthand → flag name
		"rekal": {
			"A": "actor", "a": "author", "c": "commit", "e": "explain",
			"j": "json", "n": "limit", "p": "file",
		},
		"query": {
			"F": "full", "i": "index", "j": "json", "n": "limit",
			"o": "offset", "q": "sql", "r": "role", "s": "session",
		},
		"log": {"n": "limit"},
		// sync's --self takes no shorthand: -s belongs to --session,
		// the most-typed flag in the CLI, and one letter means one thing.
		"sync": {},
		// push has no shorthands: --force was removed (the wire format is
		// append-only, so overwriting a branch is a git operation, not a rekal
		// one) and --rebuild is deliberately spelled out.
		"push": {},
	}

	root := NewRootCmd()
	got := map[string]map[string]string{}
	collect := func(name string, fs *pflag.FlagSet) {
		got[name] = map[string]string{}
		fs.VisitAll(func(f *pflag.Flag) {
			if f.Shorthand != "" && f.Name != "help" && f.Name != "version" {
				got[name][f.Shorthand] = f.Name
			}
		})
	}
	collect("rekal", root.Flags())
	for _, sub := range root.Commands() {
		if _, tracked := want[sub.Name()]; tracked {
			collect(sub.Name(), sub.Flags())
		}
	}

	for cmd, shorts := range want {
		for short, flag := range shorts {
			if got[cmd][short] != flag {
				t.Errorf("%s: -%s is %q, want %q", cmd, short, got[cmd][short], flag)
			}
		}
		for short, flag := range got[cmd] {
			if _, ok := shorts[short]; !ok {
				t.Errorf("%s: undeclared shorthand -%s for --%s — add it to this map and to docs/commands.md",
					cmd, short, flag)
			}
		}
	}

	// One letter, one meaning, across every command that binds it.
	meaning := map[string]string{}
	for cmd, shorts := range want {
		for short, flag := range shorts {
			if prev, seen := meaning[short]; seen && prev != flag {
				t.Errorf("-%s means %q elsewhere but %q in %s — same letter, two concepts",
					short, prev, flag, cmd)
			}
			meaning[short] = flag
		}
	}
}

// TestPushRebuildAlias pins that --re-export keeps working after the rename.
//
// It was renamed because "re-export" names the mechanism while --rebuild names
// the effect, and the effect is what someone has to weigh before running the
// most destructive command in the tool. A rename that silently dropped the old
// name would break the one flag people reach for while repairing a branch —
// exactly the wrong moment to hand them an unknown-flag error.
func TestPushRebuildAlias(t *testing.T) {
	t.Parallel()
	cmd := newPushCmd()

	rebuild := cmd.Flags().Lookup("rebuild")
	if rebuild == nil {
		t.Fatal("--rebuild is missing")
	}
	alias := cmd.Flags().Lookup("re-export")
	if alias == nil {
		t.Fatal("--re-export must survive as an alias: it is what existing scripts and muscle memory invoke")
	}
	if alias.Deprecated == "" {
		t.Error("--re-export should be marked deprecated so it stays out of --help while still working")
	}
	if !alias.Hidden {
		t.Error("a deprecated flag should be hidden from --help")
	}
}
