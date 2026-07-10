package skill

import (
	"embed"
	"io/fs"
	"path"
	"sort"
)

//go:embed all:skills
var skillsFS embed.FS

// Skill is one Claude Code skill managed by rekal: a directory name and the
// SKILL.md body that init installs to .claude/skills/<Name>/SKILL.md and clean
// removes. Rekal ships a small suite rather than one file — a base search skill
// plus focused skills for provenance, self-reflection, and knowledge distilling
// — so the agent loads only the workflow the task actually needs.
type Skill struct {
	Name    string
	Content string
}

// All returns every rekal-managed skill, "rekal" (the base entry point) first,
// then the rest by name. It reads the embedded skills/ tree, so adding a skill
// is just adding skills/<name>/SKILL.md — install and clean pick it up with no
// further wiring.
func All() []Skill {
	entries, err := fs.ReadDir(skillsFS, "skills")
	if err != nil {
		// The tree is embedded at build time; a read failure is a build bug.
		panic("rekal: embedded skills unreadable: " + err.Error())
	}

	out := make([]Skill, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		body, err := skillsFS.ReadFile(path.Join("skills", e.Name(), "SKILL.md"))
		if err != nil {
			continue // a dir without SKILL.md is not a skill
		}
		out = append(out, Skill{Name: e.Name(), Content: string(body)})
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
