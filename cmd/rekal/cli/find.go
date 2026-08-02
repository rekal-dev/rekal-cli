package cli

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/spf13/cobra"
)

// find enumerates every mention of a term across the ledger — a complete,
// time-ordered sweep over turns, not a ranking. It is a port of the skill's
// find.py with byte-identical output, so complete-set questions ("all / every /
// how many") don't need hand-written SQL (a re-derived schema is a Binder error
// waiting to happen). The agent judges class-mapping / set-size / uptake; the
// sweep just surfaces every row.

const findWindowWords = 12 // context words each side of the match (display budget)

var findRoles = []string{"human", "assistant", "human_steering", "summary"}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}

// runeIndexFold returns the rune index of the first case-insensitive occurrence
// of term in s, or -1. Rune-based to mirror Python str.find on unicode; ASCII
// (the corpus norm) is exact.
func runeIndexFold(s, term string) int {
	ls := []rune(strings.ToLower(s))
	lt := []rune(strings.ToLower(term))
	if len(lt) == 0 {
		return 0
	}
	for i := 0; i+len(lt) <= len(ls); i++ {
		match := true
		for j := range lt {
			if ls[i+j] != lt[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// findSnippet is a word-window around the first match, ellipsized — a faithful
// port of find.py's snippet().
func findSnippet(content, term string) string {
	flat := strings.Join(strings.Fields(content), " ")
	pos := runeIndexFold(flat, term)
	flatRunes := []rune(flat)
	if pos < 0 {
		end := findWindowWords * 12
		if len(flatRunes) < end {
			end = len(flatRunes)
		}
		return string(flatRunes[:end])
	}
	prefix := string(flatRunes[:pos])
	suffix := string(flatRunes[pos:])
	before := strings.Fields(prefix)
	after := strings.Fields(suffix)

	termWords := len(strings.Fields(term))
	if termWords < 1 {
		termWords = 1
	}
	afterN := findWindowWords + termWords

	beforeTrunc := before
	if len(beforeTrunc) > findWindowWords {
		beforeTrunc = beforeTrunc[len(beforeTrunc)-findWindowWords:]
	}
	afterTrunc := after
	if len(afterTrunc) > afterN {
		afterTrunc = afterTrunc[:afterN]
	}

	head := ""
	if len(before) > findWindowWords {
		head = "…"
	}
	tail := ""
	if len(after) > len(afterTrunc) {
		tail = "…"
	}

	words := make([]string, 0, len(beforeTrunc)+len(afterTrunc))
	words = append(words, beforeTrunc...)
	words = append(words, afterTrunc...)
	return head + strings.Join(words, " ") + tail
}

func newFindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   `find "<term>" [role]`,
		Short: "Enumerate every ledger mention of a term (complete, time order)",
		Long: `Enumerate every mention of a term across the ledger — the complete, time-ordered
sweep for "all / every / how many / which of the N" questions, where a partial
list is a wrong answer. Output is one compact line per mention:

  <session_id> t<turn> <ts> <role>: …context around the match…

then a total. Drill a mention with:
  rekal query -s <session_id> -o <turn-2> -n 5

The sweep is complete (no limit) and covers the whole ledger — this machine's
capture plus every teammate session pulled in by 'rekal sync'. The agent judges
class-mapping, set size, and the other speaker's uptake.
Optional role ∈ human|assistant|human_steering|summary.

Use find, not recall, when the question is "all / every / how many". Recall
ranks and returns the best few; find returns the set. It also answers the
opposite question: a term with no mentions exits non-zero, which is how you
tell "we never discussed this" from "the ranking buried it".`,
		Example: `  # Every mention, oldest first, then a total
  rekal find "sockaddr_un"
    01KYF3M5VK… t842  2026-08-01T14:22 assistant: …the daemon died on 'bind: invalid
    01KYF3M5VK… t879  2026-08-01T15:03 assistant: …a deep repo would still blow past…
    total 4 mentions in 1 sessions — "sockaddr_un"

  # Only what a human asked for, not what the agent said back
  rekal find "force push" human_steering

  # Only the compaction summaries
  rekal find "merge gate" summary

  # Nothing found — prints a reformulation hint and exits 1
  rekal find "kubernetes"

  # Drill a mention from the sweep
  rekal query -s 01KYF3M5VK… -o 840 -n 5`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			gitRoot, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}
			term := strings.TrimSpace(args[0])
			if term == "" {
				return fmt.Errorf("provide a non-empty <term>")
			}
			role := ""
			if len(args) == 2 {
				role = args[1]
				valid := false
				for _, r := range findRoles {
					if role == r {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("unknown role %q (want %s)", role, strings.Join(findRoles, "|"))
				}
			}
			return runFind(cmd, gitRoot, term, role)
		},
	}
	return cmd
}

// findSource picks the table find sweeps.
//
// The index is the whole ledger: this machine's capture plus every teammate
// session that arrived over the wire (which never touches data.db) minus the
// duplicate captures supersession collapsed. data.db holds only what this
// machine recorded, so sweeping it answered "every mention" with one member's
// share of the corpus — on a two-person store, a third of it. A complete-set
// command that quietly returns a subset is worse than no command, so find
// reads the index and falls back to data.db only when there is no index to
// read (never built, or built empty).
type findSource struct {
	db    *sql.DB
	table string
}

func openFindSource(gitRoot string) (*findSource, error) {
	if d, err := db.OpenIndexReadOnly(gitRoot); err == nil {
		var n int
		if err := d.QueryRow(`SELECT COUNT(*) FROM turns_ft`).Scan(&n); err == nil && n > 0 {
			return &findSource{db: d, table: "turns_ft"}, nil
		}
		d.Close()
	}
	d, err := db.OpenDataReadOnly(gitRoot)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &findSource{db: d, table: "turns"}, nil
}

// findTimestampLayouts covers what the two tables can hold: DuckDB's cast of a
// TIMESTAMP (data.db) and the ISO-8601 strings written into the index.
var findTimestampLayouts = []string{
	"2006-01-02 15:04:05",
	time.RFC3339,
	"2006-01-02T15:04:05",
}

// findTimestamp renders a stored timestamp as the minute-precision stamp the
// sweep prints. An unparseable value is truncated rather than dropped — a
// mention is still a mention when its timestamp is odd.
func findTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	for _, layout := range findTimestampLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02T15:04")
		}
	}
	if len(raw) > 16 {
		return raw[:16]
	}
	return raw
}

func runFind(cmd *cobra.Command, gitRoot, term, role string) error {
	src, err := openFindSource(gitRoot)
	if err != nil {
		return err
	}
	d := src.db
	defer d.Close()

	// turns.ts is a TIMESTAMP and turns_ft.ts is a VARCHAR, so the sweep casts
	// and normalizes rather than scanning a type that depends on which table it
	// landed on.
	query := `SELECT session_id, turn_index, CAST(ts AS VARCHAR), role, content FROM ` + src.table + `
		WHERE content ILIKE ? ESCAPE '\'`
	params := []interface{}{"%" + escapeLike(term) + "%"}
	if role != "" {
		query += ` AND role = ?`
		params = append(params, role)
	}
	query += ` ORDER BY ts, session_id, turn_index`

	rows, err := d.Query(query, params...)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := cmd.OutOrStdout()
	total := 0
	sessions := map[string]struct{}{}
	var lines []string
	for rows.Next() {
		var (
			sid     sql.NullString
			turn    sql.NullInt64
			ts      sql.NullString
			rrole   sql.NullString
			content sql.NullString
		)
		if err := rows.Scan(&sid, &turn, &ts, &rrole, &content); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		total++
		s := sid.String
		if s == "" {
			s = "?"
		}
		sessions[s] = struct{}{}
		ts16 := ""
		if ts.Valid {
			ts16 = findTimestamp(ts.String)
		}
		rl := rrole.String
		if rl == "" {
			rl = "?"
		}
		lines = append(lines, fmt.Sprintf("%s t%d %s %s: %s", s, turn.Int64, ts16, rl, findSnippet(content.String, term)))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	if total == 0 {
		roleNote := ""
		if role != "" {
			roleNote = " role=" + role
		}
		fmt.Fprintf(out, "(no mentions of %q%s — reformulate: synonyms, entity names, the other speaker's uptake)\n", term, roleNote)
		return NewSilentError(fmt.Errorf("no mentions"))
	}

	for _, l := range lines {
		fmt.Fprintln(out, l)
	}
	fmt.Fprintf(out, "total %d mentions in %d sessions — %q\n", total, len(sessions), term)
	return nil
}
