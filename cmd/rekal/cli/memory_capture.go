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

func captureMemoryChanges(d *sql.DB, gitRoot, owner, checkpoint, gitSHA string, now time.Time) ([]memory.Revision, error) {
	// Capture intent is per-repository and local, never inherited from global
	// defaults or imported from a teammate's wire data.
	cfg, err := readConfig(gitRoot)
	if err != nil {
		return nil, err
	}
	var changes []memory.Revision
	seen := map[string]bool{}
	for _, c := range cfg.MemorySources {
		if c.Name == "" || owner == "" {
			return nil, fmt.Errorf("memory source name and git user.email are required")
		}
		if c.Profile == "" {
			c.Profile = "default"
		}
		if c.Adapter == "" {
			c.Adapter = "markdown"
		}
		if c.Adapter != "markdown" && c.Adapter != "claude" {
			return nil, fmt.Errorf("unsupported memory adapter %q", c.Adapter)
		}
		dir := c.Path
		if dir == "" && c.Adapter == "claude" {
			base := session.FindSessionDir(gitx.MainWorktreeRoot(gitRoot))
			if project := os.Getenv("CLAUDE_CODE_PROJECT_DIR_NAME"); project != "" && os.Getenv("CLAUDE_CONFIG_DIR") != "" {
				if project == "." || project == ".." || strings.ContainsAny(project, "/\\") {
					return nil, fmt.Errorf("invalid Claude project directory name")
				}
				base = filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), "projects", project)
			}
			if base != "" {
				dir = filepath.Join(base, "memory")
			}
		}
		if dir == "" {
			return nil, fmt.Errorf("memory source %q needs a path", c.Name)
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(gitx.MainWorktreeRoot(gitRoot), dir)
		}
		source := memory.Source{Name: c.Name, Adapter: c.Adapter, Owner: owner, Profile: c.Profile, Shared: c.Share}
		if seen[source.ID()] {
			return nil, fmt.Errorf("duplicate memory source %q", c.Name)
		}
		seen[source.ID()] = true
		var adapter memory.Adapter = memory.MarkdownAdapter{}
		if c.Adapter == "claude" {
			adapter = memory.ClaudeAdapter{}
		}
		files, err := adapter.Read(dir)
		if err != nil {
			return nil, fmt.Errorf("memory source %q: %w", c.Name, err)
		}
		// Use the same redaction boundary as transcripts. Native files are never
		// modified; archived text is explicitly the scrubbed observation.
		for i := range files {
			files[i].Content = scrub.SanitizeText(scrub.AnonymizeText(scrub.RedactText(files[i].Content)))
		}
		previous, err := db.LatestMemory(d, source.ID())
		if err != nil {
			return nil, err
		}
		changes = append(changes, db.MemoryChanges(previous, source, files, checkpoint, gitSHA, now.UTC().Format(time.RFC3339Nano))...)
	}
	return changes, nil
}
