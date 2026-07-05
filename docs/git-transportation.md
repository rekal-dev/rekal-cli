# Git Transportation Layer

## Problem

Rekal captures AI coding sessions and stores them in a local DuckDB database. To share session data across machines and team members, this data needs to live in git — specifically on a local orphan branch (`rekal/<email>`) that doesn't pollute the working tree.

The naive approach — committing the raw DuckDB file — fails for several reasons:

1. **DuckDB files are opaque binaries.** Git can't delta-compress them effectively. A 50MB database that gains 1KB of data produces a 50MB delta.
2. **No incremental append.** Every checkpoint replaces the entire blob.
3. **Merging is impossible.** Two developers can't merge DuckDB files.

## Solution: Append-Only Binary Wire Format

Two files on the orphan branch:

```
rekal.body    Append-only sequence of compressed frames.
dict.bin      Append-only string dictionary.
```

### Why two files, not one?

`dict.bin` is a separate file because it needs random-access lookup during decode — a frame payload contains varint indices into the dictionary, not raw strings. Keeping the dictionary separate means a reader can load it once and decode any frame. It also keeps `rekal.body` strictly append-only (the dictionary grows but its existing entries never change).

### Why not TSV/JSON?

We considered TSV files for checkpoint metadata and file-touched records (human-readable, easy to debug). We rejected this because:

- It adds serialization complexity for marginal debuggability gain.
- The DuckDB database already serves as the human-readable query interface (`rekal query`).
- Binary frames with a consistent envelope are simpler to implement and maintain.
- Fewer files means fewer git objects per checkpoint.

## Format Details

### rekal.body

```
Header (9 bytes):
  "RKLBODY" (7 bytes magic)
  version   (u8, currently 0x01)
  flags     (u8, bit 0 = preset zstd dictionary available)

Frame sequence (repeated):
  Envelope (6 bytes, uncompressed):
    type            (u8: 0x01=session, 0x02=checkpoint, 0x03=meta,
                         0x04=session v2, 0x05=checkpoint v2, 0x06=batch)
    compressed_len  (u24 little-endian)
    uncompressed_len (u16 little-endian, advisory only — saturates at 0xFFFF;
                      decoding never relies on it)
  Payload (compressed_len bytes, zstd-compressed)
```

A reader that encounters an unknown frame type skips it using the envelope's
`compressed_len` — this is how binaries that predate v2 frames handle them:
they silently skip the sessions/checkpoints they cannot parse instead of
misreading them, and continue with the frames they understand.

The 6-byte envelope is always uncompressed. This allows scanning all frame offsets without decompressing any payload — useful for seeking to a specific frame or counting frames.

### dict.bin

Four namespaces, each append-only:

| Namespace | Entry format (v1) | Typical values |
|-----------|-------------|----------------|
| Sessions  | Fixed 26-byte ULID | `01KJ9KSM...` |
| Branches  | 1-byte length + UTF-8 | `main`, `feature/auth` |
| Emails    | 1-byte length + UTF-8 | `dev@example.com` |
| Paths     | 2-byte length (u16 LE) + UTF-8 | `src/auth/handler.go` |

Frame payloads reference strings by namespace + varint index. For index < 128, this costs 1 byte instead of the full string.

The v1 header packs the counts into fixed-width fields: n_emails in a single
byte, n_sessions/n_branches as u16 (paths run to EOF). Because agent IDs are
interned in the email namespace, the 255-email cap is easy to cross. Dict v2
(version byte 0x02) stores all four counts and all entry lengths as varints.
The encoder writes v1 while everything fits and switches to v2 only once a
limit is crossed; readers that predate v2 reject a v2 dict loudly
("unsupported version") instead of misparsing it.

### Frame types

**Session (0x01):** One captured AI session — turns (role + text + timestamp delta) and tool calls (tool code + path ref + command prefix). Payload v1: turn and tool-call counts are single bytes (max 255 each).

**Checkpoint (0x02):** Git state at capture time — HEAD SHA, branch, files changed (path ref + change type A/M/D/R), and references to the session frames included in this checkpoint. Payload v1: file count is a single byte (max 255).

**Meta (0x03):** Summary counters — total sessions, checkpoints, frames, dictionary entries. Written last in each checkpoint batch.

**Session v2 (0x04) / Checkpoint v2 (0x05):** Same layout as v1 except the counts above are varints, and the session payload carries an optional harness-metadata block (flags byte + parent session ref, team name, workflow name, agent type, description, spawn depth — see `docs/agent-metadata.md`) between the session meta and the turn records. The encoder writes v1 whenever the counts fit in a byte and no metadata is present (so binaries that predate v2 keep reading those frames) and v2 only when needed. V2 fixed a v1 format bug where a 300-turn session silently wrapped its count mod 256 and corrupted the frame.

**Batch (0x06):** One checkpoint's session frames plus its checkpoint frame, grouped and compressed **together** as a single zstd stream. The decompressed payload is a sequence of members, each `type (u8) + length (varint) + payload`, where `payload` is exactly the bytes a standalone frame of that type would carry (its own magic + version + body). This is the key compression lever: sessions in one commit share paths, author, ULIDs, and phrasing, so compressing them as one stream is far tighter than one zstd frame each (the per-frame design paid framing overhead per frame and could not back-reference across frames). Members are decoded with the ordinary per-type parsers after the batch is decompressed. Old readers skip a `0x06` frame by its envelope length, exactly like any unknown type, so a batched push is invisible to a pre-batch binary rather than misread — the same forward-compatibility contract the v2 frames use. The encoder falls back to standalone frames for the (implausible) case a single checkpoint's batch would overflow the envelope's u24 compressed-length field.

## Why This Works With Git

### Append-only = good deltas

`rekal.body` only grows. Existing bytes never change. When git computes the delta between two versions of `rekal.body`, it sees the old content as a prefix of the new content. The delta is just the appended bytes.

Verified empirically: after a second checkpoint that added 291 bytes, the SHA-256 of the body prefix (first N bytes) was identical to the previous version.

### Zstd with preset dictionary

Each frame payload is independently zstd-compressed using a 16KB preset dictionary embedded in the binary. The dictionary is trained on patterns common in AI coding sessions:

- Assistant phrases ("Let me read the file", "I've updated the")
- Tool names and paths (Read, Edit, Bash, `src/`, `.go`, `.ts`)
- Programming keywords and structures

This achieves ~2:1 compression on typical session frames. Independent compression per frame means any frame can be decoded without context from other frames.

### Dictionary never rewrites

`dict.bin` entries are only appended. Existing indices are stable. A session captured today that references path index 42 will always find the same string at index 42. This means `dict.bin` also benefits from git delta compression.

## Data Flow

```
Claude Code session (.jsonl)
  → rekal checkpoint
    → Parse transcript (session/parse.go)
    → Dedup by SHA-256 hash
    → Insert into DuckDB (local queryable copy)
    → Encode session frame (codec package)
    → Encode checkpoint frame with git state
    → Encode meta frame with counters
    → Append frames to rekal.body
    → Update dict.bin
    → Commit both files to orphan branch
```

The DuckDB database and the wire format contain the same data. DuckDB is the query interface; the wire format is the transport/sync mechanism.

## Trade-offs

| Decision | Chose | Alternative | Reason |
|----------|-------|-------------|--------|
| All binary vs TSV+binary | All binary | TSV for metadata | Simpler, fewer files, DuckDB handles querying |
| 1 body file vs N shards | 1 file | Shard by date/size | Simpler, git handles large files fine, append-only gives good deltas |
| Preset zstd dict | Yes, 16KB | No dictionary | ~2x better compression for small payloads at negligible binary size cost |
| String dictionary | Separate file | Inline in frames | Enables varint refs (1 byte vs full string), random-access lookup |
| Frame envelope uncompressed | Yes | Compress everything | Enables frame scanning without decompression |
