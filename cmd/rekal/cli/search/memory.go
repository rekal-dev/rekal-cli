package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
)

type MemoryHit struct {
	Ref              string        `json:"ref"`
	Path             string        `json:"path"`
	Source           memory.Source `json:"source"`
	CheckpointID     string        `json:"checkpoint_id"`
	GitSHA           string        `json:"git_sha"`
	ObservedAt       string        `json:"observed_at"`
	ModifiedAt       string        `json:"modified_at,omitempty"`
	SourceModifiedAt string        `json:"source_modified_at,omitempty"`
	NativeType       string        `json:"native_type,omitempty"`
	Snippet          string        `json:"snippet"`
	MatchedTerms     int           `json:"matched_terms"`
	Anchors          []string      `json:"anchors,omitempty"`
	// SQL uses the existing read-only query command to fetch the exact revision.
	SQL string `json:"sql"`
}

var memoryTerms = regexp.MustCompile(`[\pL\pN_]+`)
var memoryCitations = regexp.MustCompile(`\b([0-9A-HJKMNP-TV-Z]{26}):t([0-9]+)(?::bytes([0-9]+)-([0-9]+))?`)

// SearchMemory is a separate bounded keyword block. It never changes session
// scores or treats a native note as a conversation, instruction, or confidence
// vote. Private source eligibility is applied in SQL before ranking/limiting.
func SearchMemory(d *sql.DB, f Filters, privateSources []string) ([]MemoryHit, error) {
	if f.Actor != "" || (f.LimitExplicit && f.Limit <= 0) {
		return nil, nil
	}
	terms := memoryTerms.FindAllString(strings.ToLower(f.Query), 16)
	if len(terms) == 0 {
		return nil, nil
	}
	var exists int
	if err := d.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_name='memory_documents'`).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, nil
	}
	args := []interface{}{}
	var scores []string
	seen := map[string]bool{}
	for _, term := range terms {
		if seen[term] {
			continue
		}
		seen[term] = true
		scores = append(scores, `CASE WHEN contains(lower(content),?) THEN 1 ELSE 0 END`)
		args = append(args, term)
	}
	eligible := `(json_extract_string(payload,'$.source.scope') IN ('repo','local') OR CAST(json_extract(payload,'$.source.shared') AS BOOLEAN))`
	if len(privateSources) > 0 {
		marks := make([]string, len(privateSources))
		for i, id := range privateSources {
			marks[i] = "?"
			args = append(args, id)
		}
		eligible += " OR source_id IN (" + strings.Join(marks, ",") + ")"
	}
	where := "(" + eligible + ")"
	if f.Author != "" {
		where += ` AND json_extract_string(payload,'$.source.owner') = ?`
		args = append(args, f.Author)
	}
	if f.Commit != "" {
		where += ` AND starts_with(json_extract_string(payload,'$.git_sha'),?)`
		args = append(args, f.Commit)
	}
	if f.File != "" {
		if _, err := regexp.Compile(f.File); err != nil {
			return nil, err
		}
		where += ` AND regexp_matches(path,?)`
		args = append(args, f.File)
	}
	limit := 5
	if f.Limit > 0 && f.Limit < limit {
		limit = f.Limit
	}
	args = append(args, limit)
	documents := "memory_documents"
	var localExists int
	if err := d.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_name='local_memory_documents'`).Scan(&localExists); err != nil {
		return nil, err
	}
	if localExists > 0 {
		documents = "(SELECT * FROM memory_documents UNION ALL SELECT * FROM local_memory_documents)"
	}
	query := `SELECT payload, matched FROM (SELECT payload,id,(` + strings.Join(scores, "+") + `) AS matched FROM ` + documents + ` WHERE ` + where + `) WHERE matched>0 ORDER BY matched DESC,id LIMIT ?`
	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var out []MemoryHit
	for rows.Next() {
		var payload string
		var matched int
		if err := rows.Scan(&payload, &matched); err != nil {
			return nil, err
		}
		var r memory.Revision
		if err := json.Unmarshal([]byte(payload), &r); err != nil {
			return nil, err
		}
		if err := r.Validate(); err != nil {
			return nil, err
		}
		snippet := extractSnippet(r.Content, f.Query)
		if len(snippet) > 384 {
			snippet = snippet[:384]
			for !utf8.ValidString(snippet) {
				snippet = snippet[:len(snippet)-1]
			}
		}
		hit := MemoryHit{Ref: "memory:" + r.ID, Path: r.Path, Source: r.Source, CheckpointID: r.CheckpointID, GitSHA: r.GitSHA, ObservedAt: r.ObservedAt, ModifiedAt: r.ModifiedAt, SourceModifiedAt: r.SourceModifiedAt, NativeType: r.NativeType, Snippet: snippet, MatchedTerms: matched, SQL: fmt.Sprintf("SELECT payload FROM memory_archive WHERE id = '%s'", r.ID)}
		if r.Source.Scope == "local" {
			hit.Ref = "local-memory:" + r.ID
			hit.SQL = fmt.Sprintf("SELECT payload FROM local_memory_documents WHERE id = '%s'", r.ID)
		}
		unique := map[string]bool{}
		for _, m := range memoryCitations.FindAllStringSubmatch(r.Content, 16) {
			anchor := m[1] + ":" + m[2]
			if m[3] != "" {
				anchor += "@" + m[3] + "-" + m[4]
			}
			if !unique[anchor] {
				unique[anchor] = true
				hit.Anchors = append(hit.Anchors, anchor)
			}
			if len(hit.Anchors) == 8 {
				break
			}
		}
		out = append(out, hit)
	}
	return out, rows.Err()
}
