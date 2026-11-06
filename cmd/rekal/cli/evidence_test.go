package cli

import "testing"

func TestParseEvidenceAnchor_ByteCoordinates(t *testing.T) {
	t.Parallel()
	a, err := parseEvidenceAnchor("s3:12-15@4096-120")
	if err != nil || a.StartTurn != 12 || a.EndTurn != 15 || a.StartByte != 4096 || a.EndByte != 120 {
		t.Fatalf("byte range lost: %+v %v", a, err)
	}
	local, err := parseEvidenceAnchor("local:sourcehash:12@4096-8192")
	if err != nil || local.SessionID != "local:sourcehash" || local.StartTurn != 12 {
		t.Fatalf("cross-repo source id lost: %+v %v", local, err)
	}
}

func TestDecodeEvidence_SkipsUnlocatedSessionAndKeepsChildSource(t *testing.T) {
	t.Parallel()
	data := []byte(`{"results":[{"session_id":"parent","snippet":"child evidence","snippet_turn_index":7,"snippet_session_id":"child"},{"session_id":"semantic","snippet_turn_index":-1,"match_scope":"session"}]}`)
	req, warnings, err := decodeEvidenceRequest(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Anchors) != 1 || req.Anchors[0].SessionID != "child" || len(warnings) != 1 {
		t.Fatalf("bad evidence provenance: %+v %v", req, warnings)
	}
}
