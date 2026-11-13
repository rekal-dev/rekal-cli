package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Namespace identifies a section in dict.bin.
type Namespace int

const (
	NSSessions Namespace = iota
	NSBranches
	NSEmails
	NSPaths
)

const (
	dictMagic = "RKDICT"
	// dictVersionV1 packs counts into fixed header fields: n_emails in the
	// single reserved byte (max 255 — and agent IDs are interned in the email
	// namespace, so heavy subagent use crosses that fast), n_sessions and
	// n_branches as u16, and 1-byte length prefixes on branch/email entries.
	// dictVersionV2 stores all counts and entry lengths as varints. Encode
	// writes v1 while everything fits (binaries that predate v2 keep reading
	// it) and v2 only once a limit is crossed; v2 makes old readers fail
	// loudly ("unsupported version") instead of misparsing.
	dictVersionV1 = 0x01
	dictVersionV2 = 0x02
	dictHdrSize   = 12 // v1: 6 magic + 1 version + 1 reserved + 2 n_sessions + 2 n_branches
	dictHdrSizeV2 = 8  // v2: 6 magic + 1 version + 1 reserved; varint counts follow
)

// Dict is the in-memory representation of dict.bin.
// It maps strings to compact integer indices within four namespaces.
type Dict struct {
	Sessions []string
	Branches []string
	Emails   []string
	Paths    []string

	// Reverse lookup maps for O(1) lookup.
	sessIdx   map[string]uint64
	branchIdx map[string]uint64
	emailIdx  map[string]uint64
	pathIdx   map[string]uint64
}

// NewDict creates an empty dictionary.
func NewDict() *Dict {
	return &Dict{
		sessIdx:   make(map[string]uint64),
		branchIdx: make(map[string]uint64),
		emailIdx:  make(map[string]uint64),
		pathIdx:   make(map[string]uint64),
	}
}

// LookupOrAdd returns the index for value in the given namespace,
// adding it if it doesn't exist.
func (d *Dict) LookupOrAdd(ns Namespace, value string) uint64 {
	slice, idx := d.nsRef(ns)
	if i, ok := (*idx)[value]; ok {
		return i
	}
	i := uint64(len(*slice))
	*slice = append(*slice, value)
	(*idx)[value] = i
	return i
}

// Lookup returns the index for value without adding it.
// Returns (index, true) if found, (0, false) if not.
func (d *Dict) Lookup(ns Namespace, value string) (uint64, bool) {
	_, idx := d.nsRef(ns)
	i, ok := (*idx)[value]
	return i, ok
}

// Get returns the string at the given index in the namespace.
func (d *Dict) Get(ns Namespace, index uint64) (string, error) {
	slice, _ := d.nsRef(ns)
	if index >= uint64(len(*slice)) {
		return "", fmt.Errorf("dict: index %d out of range for namespace %d (len %d)", index, ns, len(*slice))
	}
	return (*slice)[index], nil
}

// Len returns the number of entries in a namespace.
func (d *Dict) Len(ns Namespace) int {
	slice, _ := d.nsRef(ns)
	return len(*slice)
}

// TotalEntries returns the total number of entries across all namespaces.
func (d *Dict) TotalEntries() int {
	return len(d.Sessions) + len(d.Branches) + len(d.Emails) + len(d.Paths)
}

func (d *Dict) nsRef(ns Namespace) (*[]string, *map[string]uint64) {
	switch ns {
	case NSSessions:
		return &d.Sessions, &d.sessIdx
	case NSBranches:
		return &d.Branches, &d.branchIdx
	case NSEmails:
		return &d.Emails, &d.emailIdx
	case NSPaths:
		return &d.Paths, &d.pathIdx
	default:
		panic(fmt.Sprintf("dict: unknown namespace %d", ns))
	}
}

// Encode serializes the dictionary to the dict.bin binary format. It writes
// v1 while every count and entry length fits v1's fixed-width fields, and v2
// once any limit is crossed (see the dictVersion constants).
func (d *Dict) Encode() []byte {
	if d.needsV2() {
		return d.encodeV2()
	}
	return d.encodeV1()
}

// needsV2 reports whether any namespace has outgrown v1's fixed-width count
// or length fields.
func (d *Dict) needsV2() bool {
	if len(d.Emails) > 0xFF || len(d.Sessions) > 0xFFFF || len(d.Branches) > 0xFFFF {
		return true
	}
	for _, s := range d.Branches {
		if len(s) > 0xFF {
			return true
		}
	}
	for _, s := range d.Emails {
		if len(s) > 0xFF {
			return true
		}
	}
	for _, s := range d.Paths {
		if len(s) > 0xFFFF {
			return true
		}
	}
	return false
}

func (d *Dict) encodeV1() []byte {
	// Calculate size.
	size := dictHdrSize
	size += len(d.Sessions) * 26 // fixed-width ULIDs
	for _, s := range d.Branches {
		size += 1 + len(s) // 1-byte length prefix
	}
	for _, s := range d.Emails {
		size += 1 + len(s)
	}
	for _, s := range d.Paths {
		size += 2 + len(s) // 2-byte length prefix
	}

	buf := make([]byte, dictHdrSize, size)
	d.encodeHeader(buf)

	buf = appendSessionEntries(buf, d.Sessions)

	// Branch entries: 1-byte length prefix + UTF-8.
	for _, s := range d.Branches {
		buf = append(buf, byte(len(s)))
		buf = append(buf, []byte(s)...)
	}

	// Email entries: 1-byte length prefix + UTF-8.
	for _, s := range d.Emails {
		buf = append(buf, byte(len(s)))
		buf = append(buf, []byte(s)...)
	}

	// Path entries: 2-byte length prefix (u16 LE) + UTF-8.
	for _, s := range d.Paths {
		buf = binary.LittleEndian.AppendUint16(buf, uint16(len(s)))
		buf = append(buf, []byte(s)...)
	}

	return buf
}

// encodeV2 writes the v2 layout: 8-byte header (magic + version + reserved),
// varint counts for all four namespaces, fixed 26-byte session entries, then
// varint-length-prefixed branch/email/path entries.
func (d *Dict) encodeV2() []byte {
	buf := make([]byte, 0, dictHdrSizeV2+len(d.Sessions)*26)
	buf = append(buf, dictMagic...)
	buf = append(buf, dictVersionV2, 0x00)
	buf = appendUvarint(buf, uint64(len(d.Sessions)))
	buf = appendUvarint(buf, uint64(len(d.Branches)))
	buf = appendUvarint(buf, uint64(len(d.Emails)))
	buf = appendUvarint(buf, uint64(len(d.Paths)))

	buf = appendSessionEntries(buf, d.Sessions)
	for _, ns := range [][]string{d.Branches, d.Emails, d.Paths} {
		for _, s := range ns {
			buf = appendUvarint(buf, uint64(len(s)))
			buf = append(buf, []byte(s)...)
		}
	}
	return buf
}

// appendSessionEntries appends fixed 26-byte ULID strings, zero-padding any
// entry that is not exactly 26 bytes.
func appendSessionEntries(buf []byte, sessions []string) []byte {
	for _, s := range sessions {
		if len(s) != 26 {
			padded := make([]byte, 26)
			copy(padded, s)
			buf = append(buf, padded...)
		} else {
			buf = append(buf, []byte(s)...)
		}
	}
	return buf
}

// LoadDict parses a dict.bin binary blob into a Dict.
func LoadDict(data []byte) (*Dict, error) {
	if len(data) < dictHdrSizeV2 {
		return nil, errors.New("dict: data too short for header")
	}

	magic := string(data[0:6])
	if magic != dictMagic {
		return nil, fmt.Errorf("dict: bad magic %q, want %q", magic, dictMagic)
	}
	switch data[6] {
	case dictVersionV1:
		return loadDictV1(data)
	case dictVersionV2:
		return loadDictV2(data)
	default:
		return nil, fmt.Errorf("dict: unsupported version %d", data[6])
	}
}

func loadDictV1(data []byte) (*Dict, error) {
	if len(data) < dictHdrSize {
		return nil, errors.New("dict: data too short for header")
	}
	// data[7] = reserved (n_emails, read below)

	nSessions := int(binary.LittleEndian.Uint16(data[8:10]))
	nBranches := int(binary.LittleEndian.Uint16(data[10:12]))

	d := NewDict()
	pos := dictHdrSize

	// Session entries: fixed 26 bytes each.
	for i := 0; i < nSessions; i++ {
		if pos+26 > len(data) {
			return nil, fmt.Errorf("dict: truncated at session entry %d", i)
		}
		s := string(data[pos : pos+26])
		d.Sessions = append(d.Sessions, s)
		d.sessIdx[s] = uint64(i)
		pos += 26
	}

	// Branch entries: 1-byte length prefix.
	for i := 0; i < nBranches; i++ {
		if pos >= len(data) {
			return nil, fmt.Errorf("dict: truncated at branch entry %d", i)
		}
		n := int(data[pos])
		pos++
		if pos+n > len(data) {
			return nil, fmt.Errorf("dict: truncated at branch entry %d data", i)
		}
		s := string(data[pos : pos+n])
		d.Branches = append(d.Branches, s)
		d.branchIdx[s] = uint64(i)
		pos += n
	}

	// Email entries: 1-byte length prefix.
	// We don't know the count from the header, so read until we hit paths or EOF.
	// Detect transition: emails use 1-byte prefix, paths use 2-byte prefix.
	// We need to know email count. The header only stores n_sessions and n_branches.
	//
	// To determine email/path boundary: we read all remaining variable-length entries.
	// Since we can't distinguish email from path by format alone (both are length-prefixed strings),
	// we need to store email count in the header too.
	//
	// WORKAROUND: Store email count in the reserved byte (data[7]).
	// This is a design decision — the reserved byte becomes n_emails (u8, max 255).

	nEmails := int(data[7]) // Use reserved byte as n_emails.

	for i := 0; i < nEmails; i++ {
		if pos >= len(data) {
			return nil, fmt.Errorf("dict: truncated at email entry %d", i)
		}
		n := int(data[pos])
		pos++
		if pos+n > len(data) {
			return nil, fmt.Errorf("dict: truncated at email entry %d data", i)
		}
		s := string(data[pos : pos+n])
		d.Emails = append(d.Emails, s)
		d.emailIdx[s] = uint64(i)
		pos += n
	}

	// Path entries: 2-byte length prefix (u16 LE). Read remaining data.
	for i := 0; pos+2 <= len(data); i++ {
		n := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
		pos += 2
		if pos+n > len(data) {
			return nil, fmt.Errorf("dict: truncated at path entry %d data", i)
		}
		s := string(data[pos : pos+n])
		d.Paths = append(d.Paths, s)
		d.pathIdx[s] = uint64(i)
		pos += n
	}

	return d, nil
}

func loadDictV2(data []byte) (*Dict, error) {
	pos := dictHdrSizeV2

	counts := make([]int, 4) // sessions, branches, emails, paths
	for i := range counts {
		v, n, err := readUvarint(data[pos:])
		if err != nil {
			return nil, fmt.Errorf("dict: count %d: %w", i, err)
		}
		pos += n
		// Every entry costs at least 1 byte on the wire — reject counts that
		// exceed the remaining bytes before trusting them for allocations.
		if v > uint64(len(data)-pos) {
			return nil, fmt.Errorf("dict: count %d (%d) exceeds remaining data", i, v)
		}
		counts[i] = int(v)
	}
	nSessions, nBranches, nEmails, nPaths := counts[0], counts[1], counts[2], counts[3]

	d := NewDict()

	// Session entries: fixed 26 bytes each.
	for i := 0; i < nSessions; i++ {
		if pos+26 > len(data) {
			return nil, fmt.Errorf("dict: truncated at session entry %d", i)
		}
		s := string(data[pos : pos+26])
		d.Sessions = append(d.Sessions, s)
		d.sessIdx[s] = uint64(i)
		pos += 26
	}

	// Branch, email, and path entries: varint length prefix + UTF-8.
	for _, ns := range []struct {
		name    string
		count   int
		entries *[]string
		idx     *map[string]uint64
	}{
		{"branch", nBranches, &d.Branches, &d.branchIdx},
		{"email", nEmails, &d.Emails, &d.emailIdx},
		{"path", nPaths, &d.Paths, &d.pathIdx},
	} {
		for i := 0; i < ns.count; i++ {
			l, n, err := readUvarint(data[pos:])
			if err != nil {
				return nil, fmt.Errorf("dict: truncated at %s entry %d", ns.name, i)
			}
			pos += n
			if l > uint64(len(data)-pos) {
				return nil, fmt.Errorf("dict: truncated at %s entry %d data", ns.name, i)
			}
			s := string(data[pos : pos+int(l)])
			*ns.entries = append(*ns.entries, s)
			(*ns.idx)[s] = uint64(i)
			pos += int(l)
		}
	}

	return d, nil
}

// encodeHeader writes the 12-byte v1 header.
func (d *Dict) encodeHeader(buf []byte) {
	copy(buf[0:6], dictMagic)
	buf[6] = dictVersionV1
	buf[7] = byte(len(d.Emails)) // reserved byte = n_emails
	binary.LittleEndian.PutUint16(buf[8:10], uint16(len(d.Sessions)))
	binary.LittleEndian.PutUint16(buf[10:12], uint16(len(d.Branches)))
}
