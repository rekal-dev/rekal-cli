package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// Includes the directory walk and file stats: the optimization removes opens,
// not inventory enumeration. Run with -benchtime=3x for a reproducible smoke.
func BenchmarkSessionDiscoveryCache(b *testing.B) {
	root := b.TempDir()
	b.Setenv("COPILOT_HOME", root)
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0o755); err != nil {
		b.Fatal(err)
	}
	const inventory = 1000
	for i := 0; i < inventory; i++ {
		dir := filepath.Join(root, "session-state", fmt.Sprintf("session-%04d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		body := `{"type":"session.start","data":{"context":{"cwd":"/another/repo"}}}` + "\n"
		if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte(body), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	d, err := db.OpenData(root)
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()
	if err := db.InitDataSchema(d); err != nil {
		b.Fatal(err)
	}
	cache, err := loadSessionDiscoveryCache(d)
	if err != nil {
		b.Fatal(err)
	}
	a := &session.CopilotAdapter{}
	if _, err := a.DiscoverCached("/repo", cache.matcher(a.Name())); err != nil {
		b.Fatal(err)
	}
	if err := cache.flush(d); err != nil {
		b.Fatal(err)
	}
	for _, mode := range []string{"uncached", "warm", "four-changed"} {
		b.Run(mode, func(b *testing.B) {
			opens := 0
			for i := 0; i < b.N; i++ {
				// A new command reloads metadata. Model changed files by removing
				// four cached signatures; the unit test covers real mtime changes.
				c, err := loadSessionDiscoveryCache(d)
				if err != nil {
					b.Fatal(err)
				}
				switch mode {
				case "uncached":
					clear(c.records)
				case "four-changed":
					for j := 0; j < 4; j++ {
						delete(c.records, "copilot\x00"+filepath.Join(root, "session-state", fmt.Sprintf("session-%04d", j), "events.jsonl"))
					}
				}
				match := c.matcher(a.Name())
				_, err = a.DiscoverCached("/repo", func(path, repo string, read func(string) ([]string, error)) bool {
					return match(path, repo, func(path string) ([]string, error) {
						opens++
						return read(path)
					})
				})
				if err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(opens)/float64(b.N), "opens/op")
		})
	}
}
