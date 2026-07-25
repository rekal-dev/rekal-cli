package cli

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/ids"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/scrub"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// importLocalSessions folds the developer's cross-repo agent history into this
// repo's index, honoring the persisted preference (pref). It discovers sessions
// from every registered agent adapter (Claude, Cursor, Codex, Gemini, OpenCode,
// Copilot, Kiro) and inserts them into the index DB only — turns_ft and
// session_facets — and NEVER into data.db.
//
// This index-only path is the whole basis for the feature being safe: data.db
// is the only thing `push`/`export` reads, so a session that exists only in the
// index is structurally incapable of reaching the team wire. Cross-repo and
// shell history can be recalled locally but can never be shared.
//
// Sessions whose content hash already appears in data.db (this repo's own
// captured history, plus anything synced) are skipped, so a re-run or an
// overlap with the current repo does not double-index. The hash matches
// checkpoint: file bytes for Path refs, sha256("adapter:DBID") for DB refs.
//
// Returns the number of sessions and distinct origin roots imported.
func importLocalSessions(indexDB *sql.DB, gitRoot string, pref localPref, w io.Writer) (sessions, projects int, err error) {
	if !pref.enabled() {
		return 0, 0, nil
	}

	// Dedup against everything already in data.db (own + synced history).
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("open data db: %w", err)
	}
	knownHashes, err := db.QueryAllSessionHashes(dataDB)
	_ = dataDB.Close()
	if err != nil {
		return 0, 0, fmt.Errorf("load known hashes: %w", err)
	}

	// seen guards against the same transcript being reachable through more
	// than one adapter path or overlapping --include roots.
	seen := make(map[string]bool)
	origins := make(map[string]bool)
	newID := ids.NewULIDFunc()

	type work struct {
		adapter  session.Adapter
		ref      session.SessionRef
		fallback string // origin fallback when Parse leaves CWD empty
	}
	var jobs []work

	if pref.All {
		for _, adapter := range session.Adapters {
			refs, derr := session.DiscoverAllSessions(adapter)
			if derr != nil {
				fmt.Fprintf(w, "rekal: warning: skipping %s all-discover: %v\n", adapter.Name(), derr)
				continue
			}
			for _, ref := range refs {
				jobs = append(jobs, work{adapter: adapter, ref: ref})
			}
		}
	} else {
		for _, repo := range pref.Repos {
			for _, adapter := range session.Adapters {
				refs, derr := adapter.Discover(repo)
				if derr != nil {
					fmt.Fprintf(w, "rekal: warning: skipping %s in %s: %v\n", adapter.Name(), repo, derr)
					continue
				}
				for _, ref := range refs {
					jobs = append(jobs, work{adapter: adapter, ref: ref, fallback: repo})
				}
			}
		}
	}

	for _, job := range jobs {
		var fileData []byte
		if job.ref.Path != "" {
			data, rerr := os.ReadFile(job.ref.Path)
			if rerr != nil || len(data) == 0 {
				continue
			}
			fileData = data
		} else if job.ref.DBID == "" {
			continue
		}

		hash := session.ContentHash(job.adapter, job.ref, fileData)
		if knownHashes[hash] || seen[hash] {
			continue
		}
		seen[hash] = true

		payload, perr := job.adapter.Parse(job.ref)
		if perr != nil || payload == nil {
			continue
		}
		scrub.Scrub(payload)
		if len(payload.Turns) == 0 {
			continue
		}
		if session.SkipCapture(payload) {
			continue
		}

		origin := originLabel(payload.CWD, job.fallback)
		if err := insertLocalSession(indexDB, newID, payload, origin); err != nil {
			return sessions, projects, err
		}
		sessions++
		if origin != "" {
			origins[origin] = true
		}
	}

	return sessions, len(origins), nil
}

// originLabel describes where a cross-repo session came from, for display in
// recall. Prefers the transcript's recorded cwd; falls back to the --include
// repo path. A path that is itself a git repository is labeled repo:; a real
// cwd that isn't is shell:; an include-only fallback with no .git is local:.
func originLabel(cwd, fallback string) string {
	path := cwd
	if path == "" {
		path = fallback
	}
	if path == "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return "repo:" + path
	}
	if cwd == "" {
		return "local:" + path
	}
	return "shell:" + path
}

// insertLocalSession writes one parsed session into the index DB (turns_ft +
// session_facets) with its origin label. Index-only by construction — there is
// deliberately no data.db write path here.
func insertLocalSession(indexDB *sql.DB, newID func() string, payload *session.SessionPayload, origin string) error {
	sessionID := newID()

	for i, t := range payload.Turns {
		ts := ""
		if !t.Timestamp.IsZero() {
			ts = t.Timestamp.UTC().Format(time.RFC3339)
		}
		if _, err := indexDB.Exec(
			`INSERT INTO turns_ft (id, session_id, turn_index, role, content, ts)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			newID(), sessionID, i, t.Role, t.Content, ts,
		); err != nil {
			return fmt.Errorf("insert turn_ft: %w", err)
		}
	}

	actorType := payload.ActorType
	if actorType == "" {
		actorType = "human"
	}
	capturedAt := payload.CapturedAt.UTC().Format(time.RFC3339)

	if _, err := indexDB.Exec(
		`INSERT INTO session_facets (
			session_id, user_email, git_branch, actor_type, agent_id,
			captured_at, turn_count, tool_call_count, file_count,
			parent_session_id, team_name, workflow_name,
			agent_type, description, spawn_depth, origin
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		sessionID, db.NullIfEmpty(""), db.NullIfEmpty(payload.Branch), actorType, db.NullIfEmpty(payload.AgentID),
		capturedAt, len(payload.Turns), 0, 0,
		db.NullIfEmpty(""), db.NullIfEmpty(payload.TeamName), db.NullIfEmpty(payload.WorkflowName),
		db.NullIfEmpty(payload.AgentType), db.NullIfEmpty(payload.Description), db.NullIfZero(payload.SpawnDepth),
		origin,
	); err != nil {
		return fmt.Errorf("insert session_facet: %w", err)
	}

	return nil
}
