package cli

import (
	"database/sql"
	"encoding/json"
	"os"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

type sessionDiscoveryCache struct {
	records map[string]db.DiscoveryRecord
	pending map[string]db.DiscoveryRecord
}

func loadSessionDiscoveryCache(d *sql.DB) (*sessionDiscoveryCache, error) {
	c := &sessionDiscoveryCache{records: map[string]db.DiscoveryRecord{}, pending: map[string]db.DiscoveryRecord{}}
	rows, err := db.LoadDiscoveryCache(d)
	for _, r := range rows {
		c.records[r.Adapter+"\x00"+r.Path] = r
	}
	return c, err
}

func (c *sessionDiscoveryCache) matcher(adapter string) session.RepoMatcher {
	return func(path, repoPath string, read func(string) ([]string, error)) bool {
		info, err := os.Stat(path)
		if err != nil {
			return false
		}
		key := adapter + "\x00" + path
		r, found := c.records[key]
		var paths []string
		if !found || r.Size != info.Size() || r.MtimeNS != info.ModTime().UnixNano() || json.Unmarshal([]byte(r.RepoPaths), &paths) != nil {
			paths, err = read(path)
			if err != nil {
				return false // never cache an incomplete read as a negative match
			}
			encoded, err := json.Marshal(paths)
			if err != nil {
				return false
			}
			r = db.DiscoveryRecord{Adapter: adapter, Path: path, Size: info.Size(), MtimeNS: info.ModTime().UnixNano(), RepoPaths: string(encoded)}
			c.records[key], c.pending[key] = r, r
		}
		return session.RepoPathsMatch(paths, repoPath)
	}
}

func (c *sessionDiscoveryCache) flush(d *sql.DB) error {
	rows := make([]db.DiscoveryRecord, 0, len(c.pending))
	for _, r := range c.pending {
		rows = append(rows, r)
	}
	return db.SaveDiscoveryCache(d, rows)
}
