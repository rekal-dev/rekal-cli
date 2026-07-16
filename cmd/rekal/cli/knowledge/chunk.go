// Package knowledge chunks the repository's tracked prose files (markdown
// and plain text) into heading-anchored sections for the index's knowledge
// layer (docs/design/knowledge-layer.md). Pure functions over bytes — no
// git, no DB — so the chunking rules are unit-testable in isolation.
//
// The unit of indexing is the heading section: everything under one markdown
// heading until the next heading. Each chunk carries a breadcrumb trail
// (prepended to the searchable content so a lone paragraph embeds as what it
// means), a 1-indexed line range directly usable by an agent's Read tool,
// and a content hash — the identity that makes reindexing surgical and keys
// the embedding cache.
package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

const (
	// MaxFileBytes caps how large a prose file the chunker will process —
	// anything bigger is skipped outright (generated dumps, concatenated
	// logs with a .md name).
	MaxFileBytes = 1 << 20 // 1 MiB

	// maxChunkChars caps a single chunk's body. Oversized sections (walls of
	// text without headings) split at paragraph boundaries under this cap.
	// Matches the facet-document cap order of magnitude: enough signal,
	// cheap FTS.
	maxChunkChars = 4000
)

// Chunk is one heading section of a prose file.
type Chunk struct {
	// Anchor is the raw heading line ("## Token refresh"), or "" for a
	// file preamble / plain-text chunk.
	Anchor string
	// Breadcrumb is the heading trail from the document root to this
	// section ("Auth Guide > Token handling > Refresh rotation"); the file
	// path for chunks with no heading context.
	Breadcrumb string
	// StartLine/EndLine are 1-indexed inclusive and directly Read-able.
	StartLine int
	EndLine   int
	// Text is the section body (heading line excluded — Anchor carries it).
	Text string
	// Content is the searchable/embeddable document: breadcrumb + body.
	Content string
	// Hash is the sha256 of Content — the chunk's identity for surgical
	// reindexing and embedding-cache keys.
	Hash string
}

// markdownExts parse as markdown (heading sections); proseExts chunk as
// plain text (paragraph groups). Tracked files outside both sets are not
// knowledge material — code belongs to the tree substrate (grep), not here.
var markdownExts = map[string]bool{
	".md": true, ".markdown": true, ".mdx": true,
}

var proseExts = map[string]bool{
	".txt": true, ".text": true, ".rst": true, ".adoc": true,
}

// IsProseFile reports whether path is knowledge material for the index.
func IsProseFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return markdownExts[ext] || proseExts[ext]
}

// ChunkFile splits a prose file into chunks. Returns nil for oversized
// files, non-prose paths, and files with no chunkable text.
func ChunkFile(path string, data []byte) []Chunk {
	if len(data) == 0 || len(data) > MaxFileBytes {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case markdownExts[ext]:
		return chunkMarkdown(path, data)
	case proseExts[ext]:
		return chunkPlainText(path, data)
	default:
		return nil
	}
}

// section is an in-flight heading section during the markdown walk.
type section struct {
	anchor     string
	breadcrumb string
	startLine  int // 1-indexed line of the heading (or first body line)
	bodyStart  int // 1-indexed first body line
	body       []string
}

func chunkMarkdown(path string, data []byte) []Chunk {
	lines := splitLines(data)

	type crumb struct {
		level int
		title string
	}
	var trail []crumb
	breadcrumbOf := func() string {
		if len(trail) == 0 {
			return path
		}
		parts := make([]string, len(trail))
		for i, c := range trail {
			parts[i] = c.title
		}
		return strings.Join(parts, " > ")
	}

	var chunks []Chunk
	cur := &section{anchor: "", breadcrumb: path, startLine: 1, bodyStart: 1}
	flush := func() {
		chunks = append(chunks, finishSection(path, cur)...)
	}

	inFence := false
	fenceMarker := ""
	for i, line := range lines {
		lineNo := i + 1
		trimmed := strings.TrimLeft(line, " \t")

		// Fenced code blocks: a # inside ``` is code, not a heading.
		if marker := fenceStart(trimmed); marker != "" {
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if strings.HasPrefix(trimmed, fenceMarker) {
				inFence = false
			}
			cur.body = append(cur.body, line)
			continue
		}

		level, title := headingOf(line)
		if inFence || level == 0 {
			cur.body = append(cur.body, line)
			continue
		}

		// New heading: close the running section, update the trail.
		flush()
		for len(trail) > 0 && trail[len(trail)-1].level >= level {
			trail = trail[:len(trail)-1]
		}
		trail = append(trail, crumb{level: level, title: title})
		cur = &section{
			anchor:     strings.TrimRight(line, " \t"),
			breadcrumb: breadcrumbOf(),
			startLine:  lineNo,
			bodyStart:  lineNo + 1,
		}
	}
	flush()
	return chunks
}

func chunkPlainText(path string, data []byte) []Chunk {
	cur := &section{anchor: "", breadcrumb: path, startLine: 1, bodyStart: 1, body: splitLines(data)}
	return finishSection(path, cur)
}

// finishSection turns a completed section into zero or more chunks,
// splitting oversized bodies at paragraph (blank-line) boundaries.
func finishSection(path string, s *section) []Chunk {
	// Trim trailing blank lines so EndLine points at real text.
	body := s.body
	for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
		body = body[:len(body)-1]
	}
	if strings.TrimSpace(strings.Join(body, "")) == "" {
		return nil // heading with no body, or an all-blank preamble — noise
	}

	// Split into parts under the cap, only ever at paragraph boundaries: a
	// part closes when the next paragraph would push it past the cap. A
	// single paragraph longer than the cap stays whole — chunk boundaries
	// never cut mid-paragraph.
	type part struct {
		startLine int
		lines     []string
	}
	var parts []part
	var cur part
	curLen := 0
	lastBlank := false
	for i, line := range body {
		lineNo := s.bodyStart + i
		isBlank := strings.TrimSpace(line) == ""
		if !isBlank && lastBlank && curLen > 0 && curLen+len(line) > maxChunkChars {
			parts = append(parts, cur)
			cur = part{}
			curLen = 0
		}
		if len(cur.lines) == 0 {
			cur.startLine = lineNo
		}
		cur.lines = append(cur.lines, line)
		curLen += len(line) + 1
		lastBlank = isBlank
	}
	if len(cur.lines) > 0 {
		parts = append(parts, cur)
	}

	chunks := make([]Chunk, 0, len(parts))
	for i, p := range parts {
		// Drop leading/trailing blank lines within a part.
		lines := p.lines
		start := p.startLine
		for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
			lines = lines[1:]
			start++
		}
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		if len(lines) == 0 {
			continue
		}
		text := strings.Join(lines, "\n")
		content := s.breadcrumb + "\n\n" + text
		startLine := start
		if i == 0 && s.anchor != "" {
			// The first part's range includes the heading line, so a Read
			// of the pointer shows the section title.
			startLine = s.startLine
		}
		chunks = append(chunks, Chunk{
			Anchor:     s.anchor,
			Breadcrumb: s.breadcrumb,
			StartLine:  startLine,
			EndLine:    start + len(lines) - 1,
			Text:       text,
			Content:    content,
			Hash:       sha256Hex(content),
		})
	}
	return chunks
}

// headingOf parses an ATX heading line, returning (level, title) or (0, "").
func headingOf(line string) (int, string) {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 || i >= len(line) || (line[i] != ' ' && line[i] != '\t') {
		return 0, ""
	}
	title := strings.TrimSpace(line[i:])
	title = strings.TrimRight(title, "#")
	return i, strings.TrimSpace(title)
}

// fenceStart returns the fence marker ("```" or "~~~") when a trimmed line
// opens or closes a fenced code block, else "".
func fenceStart(trimmed string) string {
	if strings.HasPrefix(trimmed, "```") {
		return "```"
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return "~~~"
	}
	return ""
}

func splitLines(data []byte) []string {
	s := strings.ReplaceAll(string(data), "\r\n", "\n")
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
