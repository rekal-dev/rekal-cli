package cli

import (
	"database/sql"
	"fmt"
	"strings"

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
  rekal query --session <session_id> --offset <turn-2> --limit 5

The sweep is complete (no limit); the agent judges class-mapping, set size, and
the other speaker's uptake. Optional role ∈ human|assistant|human_steering|summary.`,
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

func runFind(cmd *cobra.Command, gitRoot, term, role string) error {
	d, err := db.OpenData(gitRoot)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer d.Close()

	query := `SELECT session_id, turn_index, ts, role, content FROM turns
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
			ts      sql.NullTime
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
			ts16 = ts.Time.Format("2006-01-02T15:04")
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
