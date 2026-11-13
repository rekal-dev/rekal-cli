package codec

// ProofDecodePayload exposes opaque decompression for the experimental memory
// frame reader. The upstream codec files in this harness are unchanged.
func (d *Decoder) ProofDecodePayload(compressed []byte) ([]byte, error) {
	return d.zr.DecodeAll(compressed, nil)
}
