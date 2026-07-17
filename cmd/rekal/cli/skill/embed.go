package skill

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed all:skills
var skillsFS embed.FS

// Skill is one Claude Code skill managed by rekal: a directory under
// skills/<Name>/ containing SKILL.md plus optional scripts/ and
// references/ (progressive disclosure — tip always loads; modules and
// scripts load on demand).
type Skill struct {
	Name string
	// Content is the SKILL.md body (tip).
	Content string
	// Files maps paths relative to the skill dir → file bytes
	// (SKILL.md, scripts/*, references/*).
	Files map[string][]byte
}

// LegacyNames are skill dirs from older rekal versions that must be removed
// on init refresh / clean so the collapsed suite leaves no residue.
var LegacyNames = []string{
	"rekal-provenance",
	"rekal-reflect",
	"rekal-distill",
	"rekal-census",
	"rekal-wiki",
}

// All returns every rekal-managed skill ("rekal" first). Adding a skill is
// adding skills/<name>/SKILL.md (plus optional scripts/ and references/);
// install and clean pick up the whole tree with no further wiring.
func All() []Skill {
	entries, err := fs.ReadDir(skillsFS, "skills")
	if err != nil {
		panic("rekal: embedded skills unreadable: " + err.Error())
	}

	out := make([]Skill, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		body, err := skillsFS.ReadFile(path.Join("skills", name, "SKILL.md"))
		if err != nil {
			continue // a dir without SKILL.md is not a skill
		}
		files := map[string][]byte{"SKILL.md": body}
		_ = fs.WalkDir(skillsFS, path.Join("skills", name), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := path.Rel(path.Join("skills", name), p)
			if err != nil || rel == "SKILL.md" {
				return err
			}
			data, err := skillsFS.ReadFile(p)
			if err != nil {
				return err
			}
			files[rel] = data
			return nil
		})
		out = append(out, Skill{Name: name, Content: string(body), Files: files})
	}

	sort.Slice(out, func(i, j int) bool {
		switch {
		case out[i].Name == "rekal":
			return true
		case out[j].Name == "rekal":
			return false
		default:
			return out[i].Name < out[j].Name
		}
	})
	return out
}

// IsScript reports whether a skill-relative path should be installed executable.
func IsScript(rel string) bool {
	return strings.HasPrefix(rel, "scripts/")
}
