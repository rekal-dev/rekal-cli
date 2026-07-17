package cli

import (
	"testing"
)

func TestLimitStringMap(t *testing.T) {
	t.Parallel()
	m := map[string]string{"c": "3", "a": "1", "b": "2"}
	if got := limitStringMap(m, 0); len(got) != 3 {
		t.Fatalf("budget 0 = %d, want all 3", len(got))
	}
	got := limitStringMap(m, 2)
	if len(got) != 2 {
		t.Fatalf("budget 2 = %d, want 2", len(got))
	}
	// Stable order: lowest keys first.
	if _, ok := got["a"]; !ok {
		t.Fatal("expected key a")
	}
	if _, ok := got["b"]; !ok {
		t.Fatal("expected key b")
	}
	if _, ok := got["c"]; ok {
		t.Fatal("c should be deferred past budget")
	}
}
