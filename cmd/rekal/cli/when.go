package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// when resolves a relative time phrase to an absolute date, deterministically,
// from an anchor (the date the phrase was said). It is a pure port of the
// skill's when.py — byte-identical output — so the agent can date an event
// ("event time != mention time") without error-prone mental math. No store,
// no network, no tuned constant.

// pyWeekday maps a time to Python's convention: Monday=0 .. Sunday=6.
func pyWeekday(t time.Time) int { return (int(t.Weekday()) + 6) % 7 }

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int) int { return a - floorDiv(a, b)*b }

func dateOnly(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func fmtDate(t time.Time) string {
	return t.Format("2006-01-02") + " (" + t.Weekday().String() + ")"
}

func windowApprox(a, b time.Time) string {
	lo, hi := a, b
	if hi.Before(lo) {
		lo, hi = hi, lo
	}
	return lo.Format("2006-01-02") + ".." + hi.Format("2006-01-02") + " (approx)"
}

func addMonths(d time.Time, n int) time.Time {
	m0 := d.Year()*12 + (int(d.Month()) - 1) + n
	year := floorDiv(m0, 12)
	month := floorMod(m0, 12) + 1
	last := dateOnly(year, month+1, 0).Day()
	day := d.Day()
	if day > last {
		day = last
	}
	return dateOnly(year, month, day)
}

func monthBounds(d time.Time) (time.Time, time.Time) {
	last := dateOnly(d.Year(), int(d.Month())+1, 0).Day()
	return dateOnly(d.Year(), int(d.Month()), 1), dateOnly(d.Year(), int(d.Month()), last)
}

func weekBounds(d time.Time) (time.Time, time.Time) {
	mon := d.AddDate(0, 0, -pyWeekday(d))
	return mon, mon.AddDate(0, 0, 6)
}

func lastWeekday(a time.Time, target int) time.Time {
	delta := floorMod(pyWeekday(a)-target, 7)
	if delta == 0 {
		delta = 7
	}
	return a.AddDate(0, 0, -delta)
}

func nextWeekday(a time.Time, target int) time.Time {
	delta := floorMod(target-pyWeekday(a), 7)
	if delta == 0 {
		delta = 7
	}
	return a.AddDate(0, 0, delta)
}

func nearestPastWeekday(a time.Time, target int) time.Time {
	return a.AddDate(0, 0, -floorMod(pyWeekday(a)-target, 7))
}

var whenWeekdays = map[string]int{
	"monday": 0, "tuesday": 1, "wednesday": 2, "thursday": 3,
	"friday": 4, "saturday": 5, "sunday": 6,
	"mon": 0, "tue": 1, "tues": 1, "wed": 2, "thu": 3, "thur": 3,
	"thurs": 3, "fri": 4, "sat": 5, "sun": 6,
}

var whenNumwords = map[string]int{
	"a": 1, "an": 1, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
	"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10, "couple": 2,
}

func whenCount(word string) (int, bool) {
	if n, err := strconv.Atoi(word); err == nil {
		return n, true
	}
	n, ok := whenNumwords[word]
	return n, ok
}

var (
	whenReRel   = regexp.MustCompile(`^(last|this|next|past) (\w+)$`)
	whenReNUnit = regexp.MustCompile(`^(\w+) (day|week|month|year)s? (ago|from now|later)$`)
)

// fuzzy windows, checked in order; each returns (a-d1 .. a-d2).
var whenFuzzy = []struct {
	re     *regexp.Regexp
	d1, d2 int
}{
	{regexp.MustCompile(`^(a few|several) days? (ago|before|earlier|prior)$`), 5, 1},
	{regexp.MustCompile(`^(a )?couple( of)? days? (ago|before|earlier|prior)$`), 3, 1},
	{regexp.MustCompile(`^(recently|the other day|a while ago)$`), 7, 1},
	{regexp.MustCompile(`^(a few|several) weeks? (ago|before|earlier|prior)$`), 28, 14},
}

var whenRelShift = map[string]int{"last": -1, "past": -1, "this": 0, "next": 1}

// resolveWhen returns the formatted answer and true, or ("", false) when the
// phrase is not recognized.
func resolveWhen(a time.Time, phrase string) (string, bool) {
	p := strings.ToLower(phrase)
	p = strings.Trim(strings.TrimSpace(p), ".")
	p = strings.Join(strings.Fields(p), " ")

	fixed := map[string]int{
		"today": 0, "tonight": 0,
		"yesterday":                -1,
		"tomorrow":                 1,
		"day before yesterday":     -2,
		"the day before yesterday": -2,
		"day after tomorrow":       2,
		"the day after tomorrow":   2,
		"last night":               -1,
		"this morning":             0,
		"this afternoon":           0,
		"this evening":             0,
	}
	if off, ok := fixed[p]; ok {
		return fmtDate(a.AddDate(0, 0, off)), true
	}

	for _, f := range whenFuzzy {
		if f.re.MatchString(p) {
			return windowApprox(a.AddDate(0, 0, -f.d1), a.AddDate(0, 0, -f.d2)), true
		}
	}

	if m := whenReRel.FindStringSubmatch(p); m != nil {
		rel, word := m[1], m[2]
		if wd, ok := whenWeekdays[word]; ok {
			switch rel {
			case "last", "past":
				return fmtDate(lastWeekday(a, wd)), true
			case "next":
				return fmtDate(nextWeekday(a, wd)), true
			default: // "this"
				return fmtDate(nearestPastWeekday(a, wd)), true
			}
		}
		switch word {
		case "week":
			base := a.AddDate(0, 0, 7*whenRelShift[rel])
			lo, hi := weekBounds(base)
			return lo.Format("2006-01-02") + ".." + hi.Format("2006-01-02") + " (week)", true
		case "month":
			lo, hi := monthBounds(addMonths(a, whenRelShift[rel]))
			return lo.Format("2006-01-02") + ".." + hi.Format("2006-01-02") + " (month)", true
		case "year":
			y := a.Year() + whenRelShift[rel]
			return fmt.Sprintf("%04d-01-01..%04d-12-31 (year)", y, y), true
		}
	}

	if wd, ok := whenWeekdays[p]; ok {
		return fmtDate(nearestPastWeekday(a, wd)), true
	}

	if m := whenReNUnit.FindStringSubmatch(p); m != nil {
		n, ok := whenCount(m[1])
		if !ok {
			return "", false
		}
		sign := 1
		if m[3] == "ago" {
			sign = -1
		}
		switch m[2] {
		case "day":
			return fmtDate(a.AddDate(0, 0, sign*n)), true
		case "week":
			return fmtDate(a.AddDate(0, 0, sign*7*n)), true
		case "month":
			return fmtDate(addMonths(a, sign*n)), true
		case "year":
			return fmtDate(addMonths(a, sign*12*n)), true
		}
	}

	return "", false
}

func newWhenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   `when <YYYY-MM-DD> "<relative phrase>"`,
		Short: "Resolve a relative time phrase to an absolute date (pure calendar)",
		Long: `Resolve a relative time phrase to an absolute date from an anchor — the date
the phrase was said. Event time is not mention time: pass the mention's date as
the anchor and this returns when the event actually was, without mental math.
Vague phrases return an honest labelled window rather than fake precision.

  rekal when 2023-05-25 "last Saturday"      -> 2023-05-20 (Saturday)
  rekal when 2023-08-14 "last night"         -> 2023-08-13 (Sunday)
  rekal when 2023-11-22 "a few days before"  -> 2023-11-17..2023-11-21 (approx)

Pure: no store, no network, no tuned constant. The agent picks the anchor and
judges the result.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			anchor, err := time.Parse("2006-01-02", strings.TrimSpace(args[0]))
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "when: bad anchor date %q — want YYYY-MM-DD\n", args[0])
				return NewSilentError(err)
			}
			out, ok := resolveWhen(anchor, args[1])
			if !ok {
				fmt.Fprintf(cmd.ErrOrStderr(), "when: phrase not recognized: %q (resolve it by hand, or drill the turn)\n", args[1])
				return NewSilentError(fmt.Errorf("phrase not recognized"))
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}
}
