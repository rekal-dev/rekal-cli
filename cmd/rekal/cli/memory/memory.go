// Package memory captures native notes without taking ownership of agent memory.
package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxFileBytes = 1 << 20
const MaxSourceBytes = 16 << 20
const MaxFiles = 1024

// Config is local capture intent. An absent source is never discovered implicitly.
// Path is local-only and is never part of a revision on the wire.
type Config struct {
	Name    string `json:"name"`
	Adapter string `json:"adapter"`
	Path    string `json:"path,omitempty"`
	Profile string `json:"profile,omitempty"`
	Share   bool   `json:"share,omitempty"`
}

type Source struct {
	Name    string `json:"name"`
	Adapter string `json:"adapter"`
	Owner   string `json:"owner"`
	Profile string `json:"profile"`
	Shared  bool   `json:"shared"`
}

func (s Source) ID() string { b, _ := json.Marshal(s); return Hash(b) }
func Hash(b []byte) string  { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

type File struct {
	Path             string
	Content          string
	ModifiedAt       string
	SourceModifiedAt string
	NativeType       string
}

// Adapter returns a complete scan or an error. Missing/unreadable sources must
// never be interpreted as an empty directory and generate deletion records.
type Adapter interface {
	Read(root string) ([]File, error)
}
type MarkdownAdapter struct{}

func (MarkdownAdapter) Read(dir string) ([]File, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	var files []File
	total := int64(0)
	err = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("memory source contains symlink: %s", name)
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(path.Ext(name), ".md") {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("memory source contains non-regular note: %s", name)
		}
		if len(files) >= MaxFiles {
			return fmt.Errorf("memory source exceeds %d notes", MaxFiles)
		}
		f, err := root.Open(name)
		if err != nil {
			return err
		}
		before, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return err
		}
		if !before.Mode().IsRegular() || before.Size() > MaxFileBytes {
			_ = f.Close()
			return fmt.Errorf("invalid or oversized note: %s", name)
		}
		data, readErr := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
		after, statErr := f.Stat()
		closeErr := f.Close()
		if readErr != nil {
			return readErr
		}
		if statErr != nil {
			return statErr
		}
		if closeErr != nil {
			return closeErr
		}
		if len(data) > MaxFileBytes || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
			return fmt.Errorf("note changed during capture: %s", name)
		}
		total += int64(len(data))
		if total > MaxSourceBytes {
			return fmt.Errorf("memory source exceeds byte budget")
		}
		if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
			return fmt.Errorf("note is not UTF-8 text: %s", name)
		}
		files = append(files, File{Path: name, Content: string(data), ModifiedAt: before.ModTime().UTC().Format(time.RFC3339Nano)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// Revision is immutable and content addressed. Deleted means source membership
// ended, not that earlier contents were removed. ModifiedAt is filesystem time;
// it is neither an authorship claim nor an inferred originating session.
type Revision struct {
	Schema           int    `json:"schema"`
	ID               string `json:"id"`
	CheckpointID     string `json:"checkpoint_id"`
	GitSHA           string `json:"git_sha"`
	ObservedAt       string `json:"observed_at"`
	Source           Source `json:"source"`
	Path             string `json:"path"`
	Content          string `json:"content"`
	ModifiedAt       string `json:"modified_at,omitempty"`
	SourceModifiedAt string `json:"source_modified_at,omitempty"`
	NativeType       string `json:"native_type,omitempty"`
	Deleted          bool   `json:"deleted,omitempty"`
}

func (r Revision) Digest() string { r.ID = ""; b, _ := json.Marshal(r); return Hash(b) }
func (r Revision) Validate() error {
	if r.Schema != 1 {
		return fmt.Errorf("unsupported memory schema %d", r.Schema)
	}
	if r.ID != r.Digest() {
		return fmt.Errorf("memory revision hash mismatch")
	}
	if r.CheckpointID == "" || r.GitSHA == "" {
		return fmt.Errorf("memory checkpoint association missing")
	}
	if observed, err := time.Parse(time.RFC3339Nano, r.ObservedAt); err != nil {
		return fmt.Errorf("invalid memory observation time: %w", err)
	} else if observed.UTC().Format(time.RFC3339Nano) != r.ObservedAt {
		return fmt.Errorf("memory observation time must be canonical UTC")
	}
	if r.ModifiedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, r.ModifiedAt); err != nil {
			return err
		}
	}
	if r.SourceModifiedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, r.SourceModifiedAt); err != nil {
			return err
		}
	}
	if !fs.ValidPath(r.Path) || r.Path == "." || strings.Contains(r.Path, "\\") {
		return fmt.Errorf("invalid memory path")
	}
	for _, s := range []string{r.Path, r.Source.Name, r.Source.Adapter, r.Source.Owner, r.Source.Profile} {
		if s == "" || len(s) > 4096 || !utf8.ValidString(s) || strings.ContainsAny(s, "\x00\r\n\t") {
			return fmt.Errorf("invalid memory source identity or path")
		}
	}
	if !utf8.ValidString(r.Content) || strings.ContainsRune(r.Content, 0) || len(r.Content) > MaxFileBytes || (r.Deleted && r.Content != "") {
		return fmt.Errorf("invalid memory content")
	}
	return nil
}
