package scrub

import (
	"crypto/sha256"
	"fmt"
	"os/user"
	"regexp"
	"strings"
)

// pathAnonymizer holds the precomputed state for path anonymization.
type pathAnonymizer struct {
	username     string
	hashedUser   string // "user_<8hex>"
	macPattern   *regexp.Regexp
	linuxPattern *regexp.Regexp
	hyphenMac    *regexp.Regexp
	hyphenLinux  *regexp.Regexp
	bareUser     *regexp.Regexp
}

var defaultAnonymizer *pathAnonymizer

func init() {
	u, err := user.Current()
	if err != nil {
		return
	}
	defaultAnonymizer = newAnonymizer(u.Username)
}

func newAnonymizer(username string) *pathAnonymizer {
	if username == "" {
		return nil
	}
	h := sha256.Sum256([]byte(username))
	hashed := fmt.Sprintf("user_%x", h[:4])

	escaped := regexp.QuoteMeta(username)
	return &pathAnonymizer{
		username:     username,
		hashedUser:   hashed,
		macPattern:   regexp.MustCompile(`/Users/` + escaped + `(?:/|$)`),
		linuxPattern: regexp.MustCompile(`/home/` + escaped + `(?:/|$)`),
		hyphenMac:    regexp.MustCompile(`-Users-` + escaped + `-`),
		hyphenLinux:  regexp.MustCompile(`-home-` + escaped + `-`),
		bareUser:     regexp.MustCompile(`(?:^|[/\-])` + escaped + `(?:$|[/\-])`),
	}
}

// AnonymizePath replaces the username in a single file path with the hashed form.
func AnonymizePath(path string) string {
	if defaultAnonymizer == nil {
		return path
	}
	return defaultAnonymizer.anonymizePath(path)
}

func (a *pathAnonymizer) anonymizePath(path string) string {
	// /Users/<username>/... → /Users/<hashed>/...
	path = a.macPattern.ReplaceAllStringFunc(path, func(m string) string {
		return strings.Replace(m, a.username, a.hashedUser, 1)
	})
	// /home/<username>/... → /home/<hashed>/...
	path = a.linuxPattern.ReplaceAllStringFunc(path, func(m string) string {
		return strings.Replace(m, a.username, a.hashedUser, 1)
	})
	// Hyphen-encoded: -Users-<username>- → -Users-<hashed>-
	path = a.hyphenMac.ReplaceAllStringFunc(path, func(m string) string {
		return strings.Replace(m, a.username, a.hashedUser, 1)
	})
	path = a.hyphenLinux.ReplaceAllStringFunc(path, func(m string) string {
		return strings.Replace(m, a.username, a.hashedUser, 1)
	})
	return path
}

// AnonymizeText replaces usernames in paths throughout a block of text.
func AnonymizeText(text string) string {
	if defaultAnonymizer == nil {
		return text
	}
	return defaultAnonymizer.anonymizeText(text)
}

func (a *pathAnonymizer) anonymizeText(text string) string {
	// Replace full paths first (most specific).
	text = a.macPattern.ReplaceAllStringFunc(text, func(m string) string {
		return strings.Replace(m, a.username, a.hashedUser, 1)
	})
	text = a.linuxPattern.ReplaceAllStringFunc(text, func(m string) string {
		return strings.Replace(m, a.username, a.hashedUser, 1)
	})
	// Hyphen-encoded paths.
	text = a.hyphenMac.ReplaceAllStringFunc(text, func(m string) string {
		return strings.Replace(m, a.username, a.hashedUser, 1)
	})
	text = a.hyphenLinux.ReplaceAllStringFunc(text, func(m string) string {
		return strings.Replace(m, a.username, a.hashedUser, 1)
	})
	return text
}
