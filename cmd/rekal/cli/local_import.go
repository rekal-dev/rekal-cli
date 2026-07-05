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

// importLocalSessions folds the developer's cross-repo Claude Code history into
// this repo's index, honoring the persisted preference (pref). It reads local
// session transcripts straight from ~/.claude/projects and inserts them into
// the index DB only — turns_ft and session_facets — and NEVER into data.db.
//
// This index-only path is the whole basis for the feature being safe: data.db
// is the only thing `push`/`export` reads, so a session that exists only in the
// index is structurally incapable of reaching the team wire. Cross-repo and
// shell history can be recalled locally but can never be shared.
//
// Sessions whose content hash already appears in data.db (this repo's own
// captured history, plus anything synced) are skipped, so a re-run or an
// overlap with the current repo does not double-index.
//
// Returns the number of sessions and distinct project directories imported.
func importLocalSessions(indexDB *sql.DB, gitRoot string, pref localPref, w io.Writer) (sessions, projects int, err error) {
	if !pref.enabled() {
		return 0, 0, nil
	}

	roots, err := pref.roots()
	if err != nil {
		return 0, 0, fmt.Errorf("resolve local roots: %w", err)
	}
	if len(roots) == 0 {
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
	// than one root (e.g. --include of a repo that --include-all also covers).
	seen := make(map[string]bool)
	newID := ids.NewULIDFunc()
	adapter := &session.ClaudeAdapter{}

	for _, projectDir := range roots {
		refs, derr := session.DiscoverSessionRefsInDir(projectDir)
		if derr != nil {
			fmt.Fprintf(w, "rekal: warning: skipping %s: %v\n", projectDir, derr)
			continue
		}

		importedFromDir := 0
		for _, ref := range refs {
			if ref.Path == "" {
				continue
			}
			data, rerr := os.ReadFile(ref.Path)
			if rerr != nil || len(data) == 0 {
				continue
			}
			hash := sha256Hex(data)
			if knownHashes[hash] || seen[hash] {
				continue
			}
			seen[hash] = true

			payload, perr := adapter.Parse(ref)
			if perr != nil || payload == nil {
				continue
			}
			scrub.Scrub(payload)
			if len(payload.Turns) == 0 {
				continue
			}

			origin := originLabel(payload.CWD, projectDir)
			if err := insertLocalSession(indexDB, newID, payload, origin); err != nil {
				return sessions, projects, err
			}
			sessions++
			importedFromDir++
		}
		if importedFromDir > 0 {
			projects++
		}
	}

	return sessions, projects, nil
}

// originLabel describes where a cross-repo session came from, for display in
// recall. Prefers the transcript's recorded cwd; falls back to the project
// directory when the transcript carried no cwd. A cwd that is itself a git
// repository is labeled repo:, everything else shell:.
func originLabel(cwd, projectDir string) string {
	path := cwd
	if path == "" {
		return "local:" + projectDir
	}
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		return "repo:" + path
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
