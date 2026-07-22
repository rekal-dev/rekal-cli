package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// view renders query output as compact agent-readable text — the in-binary
// port of the skill's view.py. Session drills become dialogue (no JSON chrome);
// SQL rows become TSV. It is the default query output; --json gives raw JSON.

var viewRoleAbbr = map[string]string{
	"human":          "h",
	"assistant":      "a",
	"human_steering": "hs",
	"summary":        "sum",
	"system":         "sys",
	"tool":           "tool",
}

func viewRole(role string) string {
	if role == "" {
		return ""
	}
	if a, ok := viewRoleAbbr[role]; ok {
		return a
	}
	return role
}

// viewSplitTs splits a timestamp into (date, time) for shortening. Handles
// "YYYY-MM-DD HH:MM…" and ISO "T".
func viewSplitTs(ts string) (string, string) {
	ts = strings.TrimSpace(ts)
	if i := strings.IndexByte(ts, 'T'); i >= 0 && !strings.ContainsRune(ts[:i], ' ') {
		return ts[:i], ts[i+1:]
	}
	if i := strings.IndexByte(ts, ' '); i >= 0 {
		return ts[:i], ts[i+1:]
	}
	return ts, ""
}

// viewShortTs omits a repeated date (or the whole stamp) vs the previous turn.
func viewShortTs(ts, prevTs string) string {
	if ts == "" {
		return ""
	}
	if prevTs == "" {
		return ts
	}
	if ts == prevTs {
		return "…"
	}
	date, tm := viewSplitTs(ts)
	prevDate, _ := viewSplitTs(prevTs)
	if date != "" && tm != "" && date == prevDate {
		return "…" + tm
	}
	return ts
}

func viewSession(s *sessionOutput) string {
	sid := s.Sid
	if sid == "" {
		sid = s.SessionID
	}
	if sid == "" {
		sid = "?"
	}
	if len(s.Turns) == 0 {
		return fmt.Sprintf("%s (no turns)", sid)
	}
	lo, hi := s.Turns[0].Index, s.Turns[0].Index
	for _, t := range s.Turns {
		if t.Index < lo {
			lo = t.Index
		}
		if t.Index > hi {
			hi = t.Index
		}
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("%s t%d-%d  (h=human a=assistant hs=steering sum=summary)", sid, lo, hi))
	prevTs := ""
	for _, t := range s.Turns {
		tsRaw := strings.TrimSpace(t.Ts)
		ts := viewShortTs(tsRaw, prevTs)
		if tsRaw != "" {
			prevTs = tsRaw
		}
		content := strings.TrimRight(t.Content, " \t\n\r\f\v")
		head := []string{fmt.Sprintf("t%d", t.Index)}
		if ts != "" {
			head = append(head, ts)
		}
		if r := viewRole(t.Role); r != "" {
			head = append(head, r)
		}
		lines = append(lines, strings.Join(head, " ")+": "+content)
	}
	if len(s.ToolCalls) > 0 {
		lines = append(lines, "tools:")
		for i, tc := range s.ToolCalls {
			if i >= 40 {
				break
			}
			lines = append(lines, strings.TrimRight(fmt.Sprintf("  %s %s", tc.Tool, tc.Path), " "))
		}
		if len(s.ToolCalls) > 40 {
			lines = append(lines, fmt.Sprintf("  (+%d more tools)", len(s.ToolCalls)-40))
		}
	}
	if len(s.Files) > 0 {
		n := len(s.Files)
		if n > 30 {
			n = 30
		}
		lines = append(lines, "files: "+strings.Join(s.Files[:n], " "))
		if len(s.Files) > 30 {
			lines = append(lines, fmt.Sprintf("  (+%d more files)", len(s.Files)-30))
		}
	}
	if len(s.ChildSessionIDs) > 0 {
		n := len(s.ChildSessionIDs)
		if n > 20 {
			n = 20
		}
		lines = append(lines, "children: "+strings.Join(s.ChildSessionIDs[:n], " "))
	}
	if s.HasMore {
		lines = append(lines, "(has_more — raise --limit / --offset)")
	}
	return strings.Join(lines, "\n")
}

func viewCell(v interface{}) string {
	if v == nil {
		return ""
	}
	switch v.(type) {
	case map[string]interface{}, []interface{}:
		b, _ := json.Marshal(v)
		return string(b)
	}
	return strings.ReplaceAll(fmt.Sprintf("%v", v), "\n", "\\n")
}

// viewRows renders SQL rows as TSV in the query's own column order (an
// improvement over view.py, whose columns were alphabetical — an artifact of
// Go's map-key sorting in the old JSON pipe).
func viewRows(cols []string, rows []map[string]interface{}) string {
	if len(rows) == 0 {
		return "(no rows)"
	}
	out := []string{strings.Join(cols, "\t")}
	for _, r := range rows {
		cells := make([]string, len(cols))
		for i, c := range cols {
			cells[i] = viewCell(r[c])
		}
		out = append(out, strings.Join(cells, "\t"))
	}
	return strings.Join(out, "\n")
}
