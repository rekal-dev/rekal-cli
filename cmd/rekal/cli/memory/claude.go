package memory

import (
	"strings"
	"time"
)

// ClaudeAdapter preserves the complete Markdown and recognizes only the simple
// scalar metadata Claude emits. Unknown/complex YAML remains in the archived
// text; it does not become an inferred timestamp or a ranking signal.
type ClaudeAdapter struct{}

func (ClaudeAdapter) Read(root string) ([]File, error) {
	files, err := (MarkdownAdapter{}).Read(root)
	if err != nil {
		return nil, err
	}
	for i := range files {
		files[i].SourceModifiedAt, files[i].NativeType = claudeMetadata(files[i].Content)
	}
	return files, nil
}
func claudeMetadata(content string) (string, string) {
	text := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return "", ""
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return "", ""
	}
	values := map[string]string{}
	seen := map[string]bool{}
	for _, line := range strings.Split(text[4:4+end], "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || (key != "modified" && key != "type") {
			continue
		}
		if seen[key] {
			values[key] = ""
			continue
		}
		seen[key] = true
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	modified := ""
	if t, err := time.Parse(time.RFC3339Nano, values["modified"]); err == nil {
		modified = t.UTC().Format(time.RFC3339Nano)
	}
	kind := values["type"]
	switch kind {
	case "user", "feedback", "project", "reference":
	default:
		kind = ""
	}
	return modified, kind
}
