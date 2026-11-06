package search

import (
	"regexp"
	"sort"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
)

// Anchor is a located piece of original evidence. Byte bounds apply only to
// the first/last turn; EndByte=0 means the complete final turn. SessionID is
// explicit because a grouped conversation can match a child transcript.
type Anchor struct {
	SessionID         string  `json:"session_id"`
	Sid               string  `json:"sid,omitempty"`
	StartTurn         int     `json:"start_turn"`
	EndTurn           int     `json:"end_turn"`
	StartByte         int     `json:"start_byte,omitempty"`
	EndByte           int     `json:"end_byte,omitempty"`
	Kind              string  `json:"kind,omitempty"`
	Score             float64 `json:"score,omitempty"`
	snippet           string
	unboundedLongTurn bool
}

const anchorsPerResult = 4

func lexicalAnchor(hit bm25Hit, query string, score float64) Anchor {
	a := Anchor{SessionID: hit.sessionID, StartTurn: hit.turnIndex, EndTurn: hit.turnIndex, Kind: "lexical", Score: score}
	if len(hit.content) <= db.PassageMaxBytes {
		return a
	}
	pos := -1
	for _, term := range lsa.Tokenize(query) {
		// Match on the original string: Unicode case folding can change byte
		// lengths, making an offset into a lowercased copy point elsewhere.
		if loc := regexp.MustCompile("(?i)" + regexp.QuoteMeta(term)).FindStringIndex(hit.content); loc != nil && (pos < 0 || loc[0] < pos) {
			pos = loc[0]
		}
	}
	if pos < 0 {
		a.unboundedLongTurn = true
		return a
	} // stemming may match without an exact substring
	lo := max(0, pos-db.PassageMaxBytes/2)
	for lo > 0 && !utf8.RuneStart(hit.content[lo]) {
		lo--
	}
	hi := min(len(hit.content), lo+db.PassageMaxBytes)
	for hi < len(hit.content) && hi > lo && !utf8.RuneStart(hit.content[hi]) {
		hi--
	}
	a.StartByte, a.EndByte = lo, hi
	return a
}

// MergeAnchors preserves input priority and distinct locations across query
// framings. The cap bounds the seed, not how much evidence an agent may fetch.
func MergeAnchors(groups ...[]Anchor) []Anchor {
	var out []Anchor
	type location struct {
		sid                string
		start, end, lo, hi int
	}
	seen := map[location]bool{}
	for _, group := range groups {
		for _, a := range group {
			key := location{a.SessionID, a.StartTurn, a.EndTurn, a.StartByte, a.EndByte}
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, a)
			if len(out) == anchorsPerResult {
				return out
			}
		}
	}
	return out
}

func rankedAnchors(anchors []Anchor, n int) []Anchor {
	sort.SliceStable(anchors, func(i, j int) bool {
		if anchors[i].Score != anchors[j].Score {
			return anchors[i].Score > anchors[j].Score
		}
		if anchors[i].StartTurn != anchors[j].StartTurn {
			return anchors[i].StartTurn < anchors[j].StartTurn
		}
		return anchors[i].StartByte < anchors[j].StartByte
	})
	if len(anchors) > n {
		anchors = anchors[:n]
	}
	return anchors
}
