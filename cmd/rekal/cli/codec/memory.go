package codec

// FrameMemory carries an immutable, versioned native-memory revision. It uses
// full content-addressed IDs, so rebuilding a compression dictionary cannot
// change its identity or checkpoint association.
const FrameMemory FrameType = 0x07

// DecodePayload expands a standalone member with the same decoder and limits
// used by the built-in session/checkpoint members.
func (d *Decoder) DecodePayload(compressed []byte) ([]byte, error) {
	return d.zr.DecodeAll(compressed, nil)
}
