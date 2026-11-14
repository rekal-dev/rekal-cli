package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/scrub"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// nativeMemorySources gives capture and recall the same source identities.
// Configuration overrides discovery; it never implicitly grants sharing.
func nativeMemorySources(cfg Config) []memory.Config {
	if cfg.Memory != nil && !*cfg.Memory {
		return nil
	}
	if len(cfg.MemorySources) != 0 {
		return cfg.MemorySources
	}
	return []memory.Config{{Name: "claude-project", Adapter: "claude", Profile: "default", Scope: "repo"}}
}

func captureMemoryChanges(d *sql.DB, gitRoot, owner, checkpoint, gitSHA string, now time.Time) ([]memory.Revision, error) {
	// Capture intent is per-repository and local, never inherited from global
	// defaults or imported from a teammate's wire data.
	cfg, err := readConfig(gitRoot)
	if err != nil {
		return nil, err
	}
	var changes []memory.Revision
	seen := map[string]bool{}
	automatic := len(cfg.MemorySources) == 0
	for _, c := range nativeMemorySources(cfg) {
		if c.Scope != "" && c.Scope != "repo" && c.Scope != "local" {
			return nil, fmt.Errorf("invalid memory scope %q", c.Scope)
		}
		if c.Scope != "repo" {
			continue
		}
		source, files, present, err := readNativeMemorySource(c, gitRoot, owner, automatic)
		if err != nil {
			return nil, err
		}
		if !present {
			continue
		}
		if seen[source.ID()] {
			return nil, fmt.Errorf("duplicate memory source %q", c.Name)
		}
		seen[source.ID()] = true
		previous, err := db.LatestMemory(d, source.ID(), source.LegacyRepositoryID())
		if err != nil {
			return nil, err
		}
		changes = append(changes, db.MemoryChanges(previous, source, files, checkpoint, gitSHA, now.UTC().Format(time.RFC3339Nano))...)
	}
	return changes, nil
}

func readNativeMemorySource(c memory.Config, gitRoot, owner string, automatic bool) (memory.Source, []memory.File, bool, error) {
	if c.Scope == "" {
		c.Scope = "local"
	}
	if c.Scope != "local" && c.Scope != "repo" {
		return memory.Source{}, nil, false, fmt.Errorf("invalid memory scope %q", c.Scope)
	}
	if c.Profile == "" {
		c.Profile = "default"
	}
	if c.Adapter == "" {
		c.Adapter = "markdown"
	}
	if c.Adapter != "markdown" && c.Adapter != "claude" {
		return memory.Source{}, nil, false, fmt.Errorf("unsupported memory adapter %q", c.Adapter)
	}
	dir := c.Path
	if dir == "" && c.Adapter == "claude" {
		base := session.FindSessionDir(gitx.MainWorktreeRoot(gitRoot))
		if project := os.Getenv("CLAUDE_CODE_PROJECT_DIR_NAME"); project != "" && os.Getenv("CLAUDE_CONFIG_DIR") != "" {
			if project == "." || project == ".." || strings.ContainsAny(project, "/\\") {
				return memory.Source{}, nil, false, fmt.Errorf("invalid Claude project directory name")
			}
			base = filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), "projects", project)
		}
		if base != "" {
			dir = filepath.Join(base, "memory")
		}
	}
	if dir == "" && automatic {
		return memory.Source{}, nil, false, nil
	}
	if dir == "" {
		return memory.Source{}, nil, false, fmt.Errorf("memory source %q needs a path", c.Name)
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(gitx.MainWorktreeRoot(gitRoot), dir)
	}
	// Absence is normal for agents without native memory, including when a
	// previously observed directory disappears. It never creates tombstones.
	if automatic {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return memory.Source{}, nil, false, nil
		}
	}
	if c.Name == "" || owner == "" {
		return memory.Source{}, nil, false, fmt.Errorf("memory source name and git user.email are required")
	}
	source := memory.Source{Name: c.Name, Adapter: c.Adapter, Owner: owner, Profile: c.Profile, Scope: c.Scope}
	var adapter memory.Adapter = memory.MarkdownAdapter{}
	if c.Adapter == "claude" {
		adapter = memory.ClaudeAdapter{}
	}
	files, err := adapter.Read(dir)
	if err != nil {
		return memory.Source{}, nil, false, fmt.Errorf("memory source %q: %w", c.Name, err)
	}
	// Use the same redaction boundary as transcripts. Native files are never
	// modified; archived text is explicitly the scrubbed observation.
	for i := range files {
		files[i].Content = scrub.SanitizeText(scrub.AnonymizeText(scrub.RedactText(files[i].Content)))
	}
	return source, files, true, nil
}

// Local sources are disposable snapshots. They never touch data.db or export.
func refreshLocalMemory(index *sql.DB, gitRoot string, cfg Config) error {
	var revisions []memory.Revision
	var scanErr error
	seen := map[string]bool{}
	for _, c := range nativeMemorySources(cfg) {
		if c.Scope == "repo" {
			continue
		}
		source, files, _, err := readNativeMemorySource(c, gitRoot, gitx.ConfigValue("user.email"), false)
		if err != nil {
			scanErr = err
			break
		}
		if seen[source.ID()] {
			scanErr = fmt.Errorf("duplicate local memory source %q", c.Name)
			break
		}
		seen[source.ID()] = true
		for _, f := range files {
			r := memory.Revision{Schema: 2, Source: source, Path: f.Path, Content: f.Content, ModifiedAt: f.ModifiedAt, SourceModifiedAt: f.SourceModifiedAt, NativeType: f.NativeType, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}
			r.ID = r.Digest()
			revisions = append(revisions, r)
		}
	}
	// Failed scans cannot serve stale personal content. Retry on the next recall.
	if scanErr != nil {
		revisions = nil
	}
	if err := db.ReplaceLocalMemory(index, revisions); err != nil {
		return err
	}
	return scanErr
}
