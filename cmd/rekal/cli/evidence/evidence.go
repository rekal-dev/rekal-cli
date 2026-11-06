package evidence

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

type Request struct {
	Anchors    []search.Anchor `json:"anchors"`
	Cursor     int             `json:"cursor,omitempty"`
	ByteOffset int             `json:"byte_offset,omitempty"`
	Radius     *int            `json:"radius,omitempty"`
	PlanHash   string          `json:"plan_hash,omitempty"`
}

type Turn struct {
	Index     int    `json:"index"`
	Role      string `json:"role"`
	Ts        string `json:"ts,omitempty"`
	Content   string `json:"content"`
	Citation  string `json:"citation"`
	StartByte int    `json:"start_byte"`
	EndByte   int    `json:"end_byte"`
	Partial   bool   `json:"partial,omitempty"`
}

type Gap struct {
	StartTurn int `json:"start_turn"`
	EndTurn   int `json:"end_turn"`
}

type Window struct {
	SessionID  string `json:"session_id"`
	Sid        string `json:"sid,omitempty"`
	CapturedAt string `json:"captured_at,omitempty"`
	StartTurn  int    `json:"start_turn"`
	EndTurn    int    `json:"end_turn"`
	Turns      []Turn `json:"turns"`
	Gaps       []Gap  `json:"gaps,omitempty"`
}

type Output struct {
	Windows      []Window `json:"windows"`
	ContentBytes int      `json:"content_bytes"`
	MaxBytes     int      `json:"max_bytes"`
	MaxTurns     int      `json:"max_turns"`
	Truncated    bool     `json:"truncated"`
	Next         *Request `json:"next,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type evidencePlan struct {
	search.Anchor
	index, priority, distance, order int
	turnID                           string
}
type byteInterval struct{ lo, hi int }

func Fetch(d *sql.DB, req Request, radius, maxBytes, maxTurns int) (Output, error) {
	out := Output{Windows: []Window{}, MaxBytes: maxBytes, MaxTurns: maxTurns}
	if maxBytes < 4 || maxBytes > 1024*1024 || maxTurns < 1 || maxTurns > 512 {
		return out, fmt.Errorf("invalid evidence budget")
	}
	if len(req.Anchors) > 128 {
		return out, fmt.Errorf("at most 128 evidence anchors per request")
	}
	if req.Cursor < 0 || req.ByteOffset < 0 {
		return out, fmt.Errorf("negative evidence cursor")
	}
	if req.Radius != nil {
		radius = *req.Radius
	}
	if radius < 0 || radius > 8 {
		return out, fmt.Errorf("evidence radius must be between 0 and 8")
	}
	req.Radius = &radius
	m, err := db.LoadSessionSIDMap(d)
	if err != nil {
		return out, err
	}
	var plans []evidencePlan
	for i := range req.Anchors {
		a := &req.Anchors[i]
		if a.StartTurn < 0 || a.EndTurn < a.StartTurn || a.EndTurn-a.StartTurn > 255 || a.StartByte < 0 || a.EndByte < 0 {
			return out, fmt.Errorf("invalid evidence anchor: ranges must span 1..256 turns with nonnegative byte bounds")
		}
		a.SessionID, err = m.Resolve(a.SessionID)
		if err != nil {
			return out, err
		}
		rows, e := d.Query(`SELECT id,turn_index FROM turns_ft WHERE session_id=$1 AND turn_index BETWEEN $2 AND $3 ORDER BY turn_index,id`, a.SessionID, max(0, a.StartTurn-radius), a.EndTurn+radius)
		if e != nil {
			return out, e
		}
		var local []evidencePlan
		located := false
		for rows.Next() {
			var idx int
			var turnID string
			if e = rows.Scan(&turnID, &idx); e != nil {
				_ = rows.Close()
				return out, e
			}
			p := evidencePlan{Anchor: *a, index: idx, order: i, turnID: turnID}
			if idx < a.StartTurn {
				p.priority = 1
				p.distance = a.StartTurn - idx
			} else if idx > a.EndTurn {
				p.priority = 1
				p.distance = idx - a.EndTurn
			}
			if p.priority == 0 {
				located = true
			}
			local = append(local, p)
		}
		e = rows.Err()
		_ = rows.Close()
		if e != nil {
			return out, e
		}
		if !located {
			out.Warnings = append(out.Warnings, fmt.Sprintf("no source turn found for %s:%d-%d", a.SessionID, a.StartTurn, a.EndTurn))
			continue
		}
		plans = append(plans, local...)
	}
	sort.SliceStable(plans, func(i, j int) bool {
		a, b := plans[i], plans[j]
		if a.priority != b.priority {
			return a.priority < b.priority
		}
		if a.order != b.order {
			return a.order < b.order
		}
		if a.distance != b.distance {
			return a.distance < b.distance
		}
		return a.index < b.index
	})
	h := sha256.New()
	for _, p := range plans {
		fmt.Fprintf(h, "%q:%q:%d:%d:%d:%d:%d;", p.SessionID, p.turnID, p.index, p.priority, p.StartByte, p.EndByte, p.order)
	}
	planHash := fmt.Sprintf("%x", h.Sum(nil))
	if req.PlanHash != "" && req.PlanHash != planHash {
		return out, fmt.Errorf("source windows changed; restart the evidence fetch")
	}
	req.PlanHash = planHash
	if req.Cursor > len(plans) {
		return out, fmt.Errorf("evidence cursor is outside the requested windows")
	}
	windows := map[string]*Window{}
	covered := map[string][]byteInterval{}
	delivered := 0
	for pos := req.Cursor; pos < len(plans); pos++ {
		p := plans[pos]
		if delivered >= maxTurns || out.ContentBytes >= maxBytes {
			out.Next = &Request{Anchors: req.Anchors, Cursor: pos, Radius: req.Radius, PlanHash: req.PlanHash}
			break
		}
		var content, role, ts string
		if err := d.QueryRow(`SELECT content,role,COALESCE(ts,'') FROM turns_ft WHERE session_id=$1 AND turn_index=$2 ORDER BY id LIMIT 1`, p.SessionID, p.index).Scan(&content, &role, &ts); err != nil {
			return out, err
		}
		lo, hi := 0, len(content)
		if p.priority == 0 {
			if p.index == p.StartTurn {
				lo = p.StartByte
			}
			if p.index == p.EndTurn && p.EndByte > 0 {
				hi = p.EndByte
			}
		}
		if pos == req.Cursor && req.ByteOffset > lo {
			lo = req.ByteOffset
		}
		if lo > len(content) || hi > len(content) || lo > hi {
			return out, fmt.Errorf("byte bounds do not fit %s:t%d", p.SessionID, p.index)
		}
		for lo < hi && !utf8.RuneStart(content[lo]) {
			lo++
		}
		for hi < len(content) && hi > lo && !utf8.RuneStart(content[hi]) {
			hi--
		}
		key := fmt.Sprintf("%s:%d", p.SessionID, p.index)
		parts := subtractIntervals(byteInterval{lo, hi}, covered[key])
		for _, part := range parts {
			if delivered >= maxTurns || out.ContentBytes >= maxBytes {
				out.Next = &Request{Anchors: req.Anchors, Cursor: pos, ByteOffset: part.lo, Radius: req.Radius, PlanHash: req.PlanHash}
				break
			}
			end := min(part.hi, part.lo+maxBytes-out.ContentBytes)
			for end < part.hi && end > part.lo && !utf8.RuneStart(content[end]) {
				end--
			}
			if end == part.lo {
				out.Next = &Request{Anchors: req.Anchors, Cursor: pos, ByteOffset: part.lo, Radius: req.Radius, PlanHash: req.PlanHash}
				break
			}
			w := windows[p.SessionID]
			if w == nil {
				w = &Window{SessionID: p.SessionID, Sid: m.SID(p.SessionID), StartTurn: p.index, EndTurn: p.index}
				_ = d.QueryRow(`SELECT COALESCE(CAST(captured_at AS VARCHAR),'') FROM session_facets WHERE session_id=$1`, p.SessionID).Scan(&w.CapturedAt)
				windows[p.SessionID] = w
			}
			citation := fmt.Sprintf("%s:t%d", p.SessionID, p.index)
			partial := part.lo > 0 || end < len(content)
			if partial {
				citation += fmt.Sprintf(":bytes%d-%d", part.lo, end)
			}
			w.Turns = append(w.Turns, Turn{Index: p.index, Role: role, Ts: ts, Content: strings.Clone(content[part.lo:end]), Citation: citation, StartByte: part.lo, EndByte: end, Partial: partial})
			w.StartTurn = min(w.StartTurn, p.index)
			w.EndTurn = max(w.EndTurn, p.index)
			covered[key] = append(covered[key], byteInterval{part.lo, end})
			out.ContentBytes += end - part.lo
			delivered++
			if end < part.hi {
				out.Next = &Request{Anchors: req.Anchors, Cursor: pos, ByteOffset: end, Radius: req.Radius, PlanHash: req.PlanHash}
				break
			}
		}
		if out.Next != nil {
			break
		}
	}
	for _, w := range windows {
		sort.Slice(w.Turns, func(i, j int) bool {
			if w.Turns[i].Index != w.Turns[j].Index {
				return w.Turns[i].Index < w.Turns[j].Index
			}
			return w.Turns[i].StartByte < w.Turns[j].StartByte
		})
		for i := 1; i < len(w.Turns); i++ {
			prev, next := w.Turns[i-1].Index, w.Turns[i].Index
			if next > prev+1 {
				w.Gaps = append(w.Gaps, Gap{prev + 1, next - 1})
			}
		}
		out.Windows = append(out.Windows, *w)
	}
	sort.Slice(out.Windows, func(i, j int) bool {
		a, b := out.Windows[i], out.Windows[j]
		at, bt := windowTime(a), windowTime(b)
		if !at.Equal(bt) {
			return at.Before(bt)
		}
		return a.SessionID < b.SessionID
	})
	out.Truncated = out.Next != nil
	return out, nil
}

func windowTime(w Window) time.Time {
	values := []string{}
	for _, t := range w.Turns {
		if t.Ts != "" {
			values = append(values, t.Ts)
		}
	}
	values = append(values, w.CapturedAt)
	for _, v := range values {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999"} {
			if t, err := time.Parse(layout, v); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func subtractIntervals(part byteInterval, covered []byteInterval) []byteInterval {
	parts := []byteInterval{part}
	for _, c := range covered {
		var next []byteInterval
		for _, p := range parts {
			if c.hi <= p.lo || c.lo >= p.hi {
				next = append(next, p)
				continue
			}
			if c.lo > p.lo {
				next = append(next, byteInterval{p.lo, c.lo})
			}
			if c.hi < p.hi {
				next = append(next, byteInterval{c.hi, p.hi})
			}
		}
		parts = next
	}
	return parts
}
