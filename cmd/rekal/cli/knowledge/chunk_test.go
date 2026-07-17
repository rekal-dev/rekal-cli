package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestIsProseFile(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"docs/auth.md", true},
		{"README.markdown", true},
		{"notes.TXT", true},
		{"guide.rst", true},
		{"handbook.adoc", true},
		{"main.go", false},
		{"config.yaml", false},
		{"Makefile", false},
	} {
		if got := IsProseFile(tc.path); got != tc.want {
			t.Errorf("IsProseFile(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestChunkMarkdown_HeadingSectionsAndBreadcrumbs(t *testing.T) {
	t.Parallel()
	doc := `Intro paragraph before any heading.

# Auth Guide

Guide overview text.

## Token handling

Tokens are issued per session.

### Refresh rotation

Refresh tokens rotate on every use.

## Failure modes

Expired tokens return 401.
`
	chunks := ChunkFile("docs/auth.md", []byte(doc))
	if len(chunks) != 5 {
		t.Fatalf("want 5 chunks, got %d: %+v", len(chunks), chunks)
	}

	// Preamble: no anchor, path breadcrumb.
	if chunks[0].Anchor != "" || chunks[0].Breadcrumb != "docs/auth.md" {
		t.Fatalf("preamble chunk wrong: %+v", chunks[0])
	}
	if chunks[0].StartLine != 1 || chunks[0].EndLine != 1 {
		t.Fatalf("preamble lines = %d-%d, want 1-1", chunks[0].StartLine, chunks[0].EndLine)
	}

	// Nested section carries the full trail.
	refresh := chunks[3]
	if refresh.Anchor != "### Refresh rotation" {
		t.Fatalf("anchor = %q", refresh.Anchor)
	}
	if refresh.Breadcrumb != "Auth Guide > Token handling > Refresh rotation" {
		t.Fatalf("breadcrumb = %q", refresh.Breadcrumb)
	}
	if !strings.HasPrefix(refresh.Content, refresh.Breadcrumb+"\n\n") {
		t.Fatalf("content must start with breadcrumb: %q", refresh.Content)
	}
	// Line range includes the heading line (Read shows the title).
	if refresh.StartLine != 11 || refresh.EndLine != 13 {
		t.Fatalf("refresh lines = %d-%d, want 11-13", refresh.StartLine, refresh.EndLine)
	}

	// Sibling heading pops the deeper trail entry.
	failure := chunks[4]
	if failure.Breadcrumb != "Auth Guide > Failure modes" {
		t.Fatalf("breadcrumb = %q", failure.Breadcrumb)
	}
}

func TestChunkMarkdown_FencedCodeIsNotAHeading(t *testing.T) {
	t.Parallel()
	doc := "# Real\n\nbody\n\n```bash\n# not a heading\necho hi\n```\n\nmore body\n"
	chunks := ChunkFile("a.md", []byte(doc))
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk, got %d: %+v", len(chunks), chunks)
	}
	if !strings.Contains(chunks[0].Text, "# not a heading") {
		t.Fatalf("fenced content missing from body: %q", chunks[0].Text)
	}
}

func TestChunkMarkdown_EmptySectionSkipped(t *testing.T) {
	t.Parallel()
	doc := "# Empty\n\n# Full\n\ncontent here\n"
	chunks := ChunkFile("a.md", []byte(doc))
	if len(chunks) != 1 || chunks[0].Anchor != "# Full" {
		t.Fatalf("want only the Full section, got %+v", chunks)
	}
}

func TestChunkPlainText_ParagraphsSplitUnderCap(t *testing.T) {
	t.Parallel()
	para := strings.Repeat("word ", 600) // ~3000 chars
	doc := para + "\n\n" + para + "\n\n" + para + "\n"
	chunks := ChunkFile("notes.txt", []byte(doc))
	if len(chunks) < 2 {
		t.Fatalf("oversized plain text should split, got %d chunk(s)", len(chunks))
	}
	for _, c := range chunks {
		if c.Anchor != "" || c.Breadcrumb != "notes.txt" {
			t.Fatalf("plain-text chunk wrong: anchor=%q breadcrumb=%q", c.Anchor, c.Breadcrumb)
		}
		if len(c.Text) > maxChunkChars+1000 {
			t.Fatalf("chunk exceeds cap: %d chars", len(c.Text))
		}
	}
	// Line ranges must be disjoint and increasing.
	for i := 1; i < len(chunks); i++ {
		if chunks[i].StartLine <= chunks[i-1].EndLine {
			t.Fatalf("overlapping ranges: %d-%d then %d-%d",
				chunks[i-1].StartLine, chunks[i-1].EndLine, chunks[i].StartLine, chunks[i].EndLine)
		}
	}
}

func TestChunkFile_HashStableAndContentKeyed(t *testing.T) {
	t.Parallel()
	doc := "# T\n\nsame body\n"
	a := ChunkFile("a.md", []byte(doc))
	b := ChunkFile("a.md", []byte(doc))
	if a[0].Hash != b[0].Hash {
		t.Fatal("hash must be deterministic")
	}
	c := ChunkFile("a.md", []byte("# T\n\ndifferent body\n"))
	if a[0].Hash == c[0].Hash {
		t.Fatal("different content must hash differently")
	}
}

func TestChunkFile_SkipsOversizedAndNonProse(t *testing.T) {
	t.Parallel()
	big := make([]byte, MaxFileBytes+1)
	if got := ChunkFile("big.md", big); got != nil {
		t.Fatal("oversized file must be skipped")
	}
	if got := ChunkFile("main.go", []byte("package main")); got != nil {
		t.Fatal("non-prose file must be skipped")
	}
	if got := ChunkFile("empty.md", nil); got != nil {
		t.Fatal("empty file must be skipped")
	}
}

// TestChunkFile_SanitizesInvalidUTF8: prose dumps (incident logs, etc.) can
// carry truncated runes — chunk Content/Hash must be DuckDB-safe and matched.
func TestChunkFile_SanitizesInvalidUTF8(t *testing.T) {
	t.Parallel()
	data := []byte("line one\nincident \xc3 log\n")
	chunks := ChunkFile("tmncs_incident_log.txt", data)
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk, got %d", len(chunks))
	}
	if !utf8.ValidString(chunks[0].Content) {
		t.Fatalf("Content not valid UTF-8: %q", chunks[0].Content)
	}
	if chunks[0].Hash == "" {
		t.Fatal("Hash empty")
	}
	sum := sha256.Sum256([]byte(chunks[0].Content))
	want := hex.EncodeToString(sum[:])
	if chunks[0].Hash != want {
		t.Fatalf("Hash = %s, want sha256(Content) %s", chunks[0].Hash, want)
	}
}
