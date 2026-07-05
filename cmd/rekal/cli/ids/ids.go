// Package ids mints the ULID identifiers rekal assigns to sessions, turns,
// tool calls, checkpoints, and file-touched rows.
package ids

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// NewULIDFunc returns a generator that produces ULID strings from a single
// shared entropy source. The checkpoint and import loops mint many IDs in a
// row and reuse one generator, which is the pattern this preserves.
func NewULIDFunc() func() string {
	entropy := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	return func() string {
		return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
	}
}
