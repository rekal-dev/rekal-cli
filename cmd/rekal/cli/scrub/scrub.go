package scrub

import (
	"strings"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/session"
)

// Scrub applies secret redaction and path anonymization to a SessionPayload
// in place, then guarantees every stored string is valid UTF-8. Call this
// after session.ParseTranscript and before DB insertion.
func Scrub(payload *session.SessionPayload) {
	if payload == nil {
		return
	}

	for i := range payload.Turns {
		payload.Turns[i].Content = RedactText(payload.Turns[i].Content)
		payload.Turns[i].Content = AnonymizeText(payload.Turns[i].Content)
		payload.Turns[i].Content = SanitizeText(payload.Turns[i].Content)
	}

	for i := range payload.ToolCalls {
		payload.ToolCalls[i].Tool = SanitizeText(payload.ToolCalls[i].Tool)
		payload.ToolCalls[i].Path = AnonymizePath(payload.ToolCalls[i].Path)
		payload.ToolCalls[i].Path = SanitizeText(payload.ToolCalls[i].Path)
		payload.ToolCalls[i].CmdPrefix = RedactText(payload.ToolCalls[i].CmdPrefix)
		payload.ToolCalls[i].CmdPrefix = AnonymizeText(payload.ToolCalls[i].CmdPrefix)
		payload.ToolCalls[i].CmdPrefix = SanitizeText(payload.ToolCalls[i].CmdPrefix)
	}
}

// SanitizeText makes s safe to bind as a DuckDB VARCHAR: it must be valid
// UTF-8 (DuckDB rejects invalid varchars with "could not bind parameter") and
// carry no NUL bytes. Tool commands and file paths can hold arbitrary bytes
// (binary output, truncated multi-byte runes, unusual filenames), so this is
// the last-line guarantee before any insert. Valid, NUL-free strings pass
// through untouched with no allocation.
func SanitizeText(s string) string {
	if s == "" {
		return s
	}
	if strings.IndexByte(s, 0) >= 0 {
		s = strings.ReplaceAll(s, "\x00", "")
	}
	if !utf8.ValidString(s) {
		// Replace each run of invalid bytes with the Unicode replacement
		// character so the loss is visible rather than silent.
		s = strings.ToValidUTF8(s, "�")
	}
	return s
}
