package codec

import (
	"testing"
)

func TestNewDict_Empty(t *testing.T) {
	d := NewDict()
	if d.Len(NSSessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", d.Len(NSSessions))
	}
	if d.TotalEntries() != 0 {
		t.Errorf("expected 0 total, got %d", d.TotalEntries())
	}
}

func TestDict_LookupOrAdd(t *testing.T) {
	d := NewDict()

	// First add returns 0.
	idx := d.LookupOrAdd(NSBranches, "main")
	if idx != 0 {
		t.Errorf("first add: expected 0, got %d", idx)
	}

	// Same value returns same index.
	idx2 := d.LookupOrAdd(NSBranches, "main")
	if idx2 != 0 {
		t.Errorf("duplicate add: expected 0, got %d", idx2)
	}

	// Different value gets next index.
	idx3 := d.LookupOrAdd(NSBranches, "feature/auth")
	if idx3 != 1 {
		t.Errorf("second add: expected 1, got %d", idx3)
	}

	if d.Len(NSBranches) != 2 {
		t.Errorf("expected 2 branches, got %d", d.Len(NSBranches))
	}
}

func TestDict_Lookup(t *testing.T) {
	d := NewDict()
	d.LookupOrAdd(NSEmails, "alice@example.com")

	idx, ok := d.Lookup(NSEmails, "alice@example.com")
	if !ok || idx != 0 {
		t.Errorf("lookup existing: got (%d, %v), want (0, true)", idx, ok)
	}

	_, ok = d.Lookup(NSEmails, "bob@example.com")
	if ok {
		t.Error("lookup missing: expected false")
	}
}

func TestDict_Get(t *testing.T) {
	d := NewDict()
	d.LookupOrAdd(NSPaths, "src/main.go")
	d.LookupOrAdd(NSPaths, "src/auth/handler.go")

	v, err := d.Get(NSPaths, 0)
	if err != nil || v != "src/main.go" {
		t.Errorf("get 0: got (%q, %v)", v, err)
	}

	v, err = d.Get(NSPaths, 1)
	if err != nil || v != "src/auth/handler.go" {
		t.Errorf("get 1: got (%q, %v)", v, err)
	}

	_, err = d.Get(NSPaths, 99)
	if err == nil {
		t.Error("get out of range: expected error")
	}
}

func TestDict_NamespaceIsolation(t *testing.T) {
	d := NewDict()

	// Same string in different namespaces should be independent.
	d.LookupOrAdd(NSBranches, "main")
	d.LookupOrAdd(NSEmails, "main")
	d.LookupOrAdd(NSPaths, "main")

	if d.Len(NSBranches) != 1 {
		t.Errorf("branches: %d", d.Len(NSBranches))
	}
	if d.Len(NSEmails) != 1 {
		t.Errorf("emails: %d", d.Len(NSEmails))
	}
	if d.Len(NSPaths) != 1 {
		t.Errorf("paths: %d", d.Len(NSPaths))
	}
	if d.TotalEntries() != 3 {
		t.Errorf("total: %d", d.TotalEntries())
	}
}

func TestDict_EncodeLoadRoundtrip(t *testing.T) {
	d := NewDict()

	// Add entries to all namespaces.
	d.LookupOrAdd(NSSessions, "01JMXD1234567890ABCDEFGH")
	d.LookupOrAdd(NSSessions, "01JMXE1234567890ABCDEFGH")
	d.LookupOrAdd(NSBranches, "main")
	d.LookupOrAdd(NSBranches, "feature/auth")
	d.LookupOrAdd(NSEmails, "alice@example.com")
	d.LookupOrAdd(NSEmails, "bob@example.com")
	d.LookupOrAdd(NSPaths, "src/auth/handler.go")
	d.LookupOrAdd(NSPaths, "cmd/server/main.go")
	d.LookupOrAdd(NSPaths, "internal/config/config.go")

	encoded := d.Encode()

	// Verify header magic.
	if string(encoded[0:6]) != "RKDICT" {
		t.Fatalf("bad magic: %q", encoded[0:6])
	}

	// Reload.
	d2, err := LoadDict(encoded)
	if err != nil {
		t.Fatalf("LoadDict: %v", err)
	}

	// Verify all entries survived.
	if d2.Len(NSSessions) != 2 {
		t.Errorf("sessions: got %d, want 2", d2.Len(NSSessions))
	}
	if d2.Len(NSBranches) != 2 {
		t.Errorf("branches: got %d, want 2", d2.Len(NSBranches))
	}
	if d2.Len(NSEmails) != 2 {
		t.Errorf("emails: got %d, want 2", d2.Len(NSEmails))
	}
	if d2.Len(NSPaths) != 3 {
		t.Errorf("paths: got %d, want 3", d2.Len(NSPaths))
	}

	// Verify lookups work.
	idx, ok := d2.Lookup(NSBranches, "feature/auth")
	if !ok || idx != 1 {
		t.Errorf("branch lookup: (%d, %v)", idx, ok)
	}

	v, err := d2.Get(NSPaths, 2)
	if err != nil || v != "internal/config/config.go" {
		t.Errorf("path get: (%q, %v)", v, err)
	}
}

func TestDict_EmptyEncode(t *testing.T) {
	d := NewDict()
	encoded := d.Encode()
	if len(encoded) != dictHdrSize {
		t.Errorf("empty dict should be %d bytes, got %d", dictHdrSize, len(encoded))
	}

	d2, err := LoadDict(encoded)
	if err != nil {
		t.Fatalf("LoadDict empty: %v", err)
	}
	if d2.TotalEntries() != 0 {
		t.Errorf("expected 0 entries, got %d", d2.TotalEntries())
	}
}

func TestLoadDict_BadMagic(t *testing.T) {
	data := []byte("BADMAG\x01\x00\x00\x00\x00\x00")
	_, err := LoadDict(data)
	if err == nil {
		t.Error("expected error for bad magic")
	}
}

func TestLoadDict_TooShort(t *testing.T) {
	_, err := LoadDict([]byte("RKDI"))
	if err == nil {
		t.Error("expected error for short data")
	}
}

func BenchmarkDictEncode(b *testing.B) {
	d := NewDict()
	for i := 0; i < 100; i++ {
		d.LookupOrAdd(NSSessions, "01JMXD1234567890ABCDEFG"+string(rune('A'+i%26)))
	}
	for i := 0; i < 10; i++ {
		d.LookupOrAdd(NSBranches, "branch-"+string(rune('a'+i)))
	}
	for i := 0; i < 5; i++ {
		d.LookupOrAdd(NSEmails, "user"+string(rune('0'+i))+"@example.com")
	}
	for i := 0; i < 50; i++ {
		d.LookupOrAdd(NSPaths, "src/pkg"+string(rune('a'+i%26))+"/file.go")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Encode()
	}
}

func BenchmarkDictLoad(b *testing.B) {
	d := NewDict()
	for i := 0; i < 100; i++ {
		d.LookupOrAdd(NSSessions, "01JMXD1234567890ABCDEFG"+string(rune('A'+i%26)))
	}
	for i := 0; i < 10; i++ {
		d.LookupOrAdd(NSBranches, "branch-"+string(rune('a'+i)))
	}
	encoded := d.Encode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadDict(encoded)
	}
}
