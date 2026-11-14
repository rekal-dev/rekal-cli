package search

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type LiteralHit struct {
	Ref       string `json:"ref"`
	SessionID string `json:"session_id"`
	Turn      int    `json:"turn"`
	Role      string `json:"role"`
	Timestamp string `json:"timestamp"`
	Snippet   string `json:"snippet"`
}

type LiteralPage struct {
	Scope  string       `json:"scope"`
	Query  string       `json:"query"`
	Total  int          `json:"total"`
	Offset int          `json:"offset"`
	Hits   []LiteralHit `json:"hits"`
	Next   string       `json:"next,omitempty"`
}

type literalCursor struct {
	Version  int    `json:"v"`
	Query    string `json:"q"`
	File     string `json:"f,omitempty"`
	Commit   string `json:"c,omitempty"`
	Author   string `json:"a,omitempty"`
	Actor    string `json:"r,omitempty"`
	Offset   int    `json:"o"`
	Limit    int    `json:"n"`
	Snapshot string `json:"s"`
}

// LiteralMatches enumerates case-insensitive literal substrings in indexed
// conversation turns. Related/ranked hits remain a separate, non-exhaustive view.
func LiteralMatches(d *sql.DB, f Filters, cursor string) (*LiteralPage, error) {
	state := literalCursor{Version: 1, Query: strings.TrimSpace(f.Query), File: f.File, Commit: f.Commit, Author: f.Author, Actor: f.Actor, Limit: DefaultLimit}
	if f.LimitExplicit {
		state.Limit = min(f.Limit, 100)
	}
	if cursor != "" {
		if len(cursor) > 16384 {
			return nil, fmt.Errorf("search continuation too large")
		}
		b, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid search continuation")
		}
		if err := json.Unmarshal(b, &state); err != nil {
			return nil, fmt.Errorf("invalid search continuation")
		}
		if state.Version != 1 || state.Offset < 0 || state.Limit < 1 || state.Limit > 100 || state.Snapshot == "" || strings.TrimSpace(state.Query) == "" {
			return nil, fmt.Errorf("invalid search continuation")
		}
	}
	if state.Query == "" {
		return nil, nil
	}
	if state.Limit < 0 {
		return nil, fmt.Errorf("invalid literal limit")
	}
	if len(state.Query) > 4096 {
		return nil, fmt.Errorf("literal query exceeds 4096 bytes")
	}
	if state.File != "" {
		if _, err := regexp.Compile(state.File); err != nil {
			return nil, err
		}
	}
	where := `contains(lower(t.content),lower(?))`
	args := []interface{}{state.Query}
	if state.Author != "" {
		where += ` AND f.user_email=?`
		args = append(args, state.Author)
	}
	if state.Actor != "" {
		where += ` AND f.actor_type=?`
		args = append(args, state.Actor)
	}
	if state.Commit != "" {
		where += ` AND starts_with(f.git_sha,?)`
		args = append(args, state.Commit)
	}
	if state.File != "" {
		where += ` AND EXISTS (SELECT 1 FROM files_index p WHERE p.session_id=t.session_id AND regexp_matches(p.file_path,?))`
		args = append(args, state.File)
	}
	from := ` FROM turns_ft t JOIN session_facets f ON f.session_id=t.session_id WHERE ` + where
	// Both summary and page use one database snapshot. The fingerprint detects
	// changes between pages; a continuation never silently skips shifted rows.
	tx, err := d.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck
	var total int
	var fingerprint string
	if err := tx.QueryRow(`SELECT count(*),COALESCE(CAST(bit_xor(hash(t.session_id,t.turn_index,t.content,t.role,t.ts)) AS VARCHAR),'empty')`+from, args...).Scan(&total, &fingerprint); err != nil {
		return nil, err
	}
	snapshot := fmt.Sprintf("%d:%s", total, fingerprint)
	if cursor != "" && state.Snapshot != snapshot {
		return nil, fmt.Errorf("literal results changed; repeat the original search")
	}
	page := &LiteralPage{Scope: "indexed conversation turns; case-insensitive literal substring", Query: state.Query, Total: total, Offset: state.Offset, Hits: []LiteralHit{}}
	if state.Limit == 0 {
		return page, nil
	}
	if state.Offset > total {
		return nil, fmt.Errorf("search continuation is beyond the result set")
	}
	rows, err := tx.Query(`SELECT t.session_id,t.turn_index,t.role,COALESCE(CAST(t.ts AS VARCHAR),''),t.content`+from+` ORDER BY t.session_id,t.turn_index LIMIT ? OFFSET ?`, append(args, state.Limit, state.Offset)...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var h LiteralHit
		var content string
		if err := rows.Scan(&h.SessionID, &h.Turn, &h.Role, &h.Timestamp, &content); err != nil {
			_ = rows.Close()
			return nil, err
		}
		h.Snippet = extractSnippet(content, state.Query)
		anchor := Anchor{SessionID: h.SessionID, StartTurn: h.Turn, EndTurn: h.Turn, Kind: "literal"}
		if len(content) > 8192 {
			if loc := regexp.MustCompile("(?i)" + regexp.QuoteMeta(state.Query)).FindStringIndex(content); loc != nil {
				lo := max(0, loc[0]-1024)
				for lo > 0 && !utf8.RuneStart(content[lo]) {
					lo--
				}
				hi := min(len(content), max(lo+8192, loc[1]))
				for hi < len(content) && !utf8.RuneStart(content[hi]) {
					hi--
				}
				anchor.StartByte, anchor.EndByte = lo, hi
			}
		}
		h.Ref = AnchorReference(anchor)
		page.Hits = append(page.Hits, h)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if state.Offset+len(page.Hits) < total {
		state.Offset += len(page.Hits)
		state.Snapshot = snapshot
		b, _ := json.Marshal(state)
		page.Next = base64.RawURLEncoding.EncodeToString(b)
	}
	return page, nil
}

func AnchorReference(a Anchor) string {
	ref := fmt.Sprintf("passage:%s:%d-%d", a.SessionID, a.StartTurn, a.EndTurn)
	if a.StartByte > 0 || a.EndByte > 0 {
		ref += fmt.Sprintf("@%d-%d", a.StartByte, a.EndByte)
	}
	return ref
}

// AttachReadReferences keeps permanent source IDs behind compact display handles.
func AttachReadReferences(results []Result) {
	for i := range results {
		results[i].Ref = "session:" + results[i].SessionID
		if len(results[i].Anchors) > 0 {
			results[i].Ref = AnchorReference(results[i].Anchors[0])
		}
		AttachReadReferences(results[i].Children)
	}
}
