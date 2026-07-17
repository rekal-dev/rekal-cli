package session

import (
	"testing"
)

func TestSkipCapture_BenchFingerprint(t *testing.T) {
	t.Parallel()
	p := &SessionPayload{
		Turns: []Turn{{
			Role:    "human",
			Content: "Write ONE natural question they would ask about the how/why of that change",
		}},
	}
	if !SkipCapture(p) {
		t.Fatal("bench prompt fingerprint should skip capture")
	}
}

func TestSkipCapture_NormalSession(t *testing.T) {
	t.Parallel()
	p := &SessionPayload{
		CWD:   "/workspace/app",
		Turns: []Turn{{Role: "human", Content: "fix the auth middleware"}},
	}
	if SkipCapture(p) {
		t.Fatal("normal session must not skip")
	}
}

func TestSkipCapture_BenchCWD(t *testing.T) {
	t.Parallel()
	p := &SessionPayload{
		CWD:   "/tmp/rekal-bench/out",
		Turns: []Turn{{Role: "human", Content: "hello"}},
	}
	if !SkipCapture(p) {
		t.Fatal("bench cwd should skip")
	}
}
