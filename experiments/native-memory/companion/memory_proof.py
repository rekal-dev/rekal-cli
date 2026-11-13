"""Experimental separate memory.bin; Python stdlib only, NOT a Rekal format.

An immutable snapshot is a complete source directory listing. Only new content
blobs are added. Explicit citations remain in the original Markdown; reverse
links are a disposable projection. No agent writes or network requests occur.
"""
import base64
import hashlib
import json
from pathlib import PurePosixPath
import re
import struct
import zlib

MAGIC = b"RKMEM\x01"
MAX_FRAME = 8 * 1024 * 1024
MAX_DECODED = 32 * 1024 * 1024
CITATION = re.compile(r"\b([0-9A-HJKMNP-TV-Z]{26}):t(\d+)(?::bytes(\d+)-(\d+))?")


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()


def digest(data):
    return hashlib.sha256(data).hexdigest()


def source_key(source):
    return digest(canonical(source))


def native_modified(content):
    """Fixture-level parser only. Production adapter needs proper YAML parsing."""
    text = content.decode("utf-8")
    if not text.startswith("---\n"):
        return None
    end = text.find("\n---", 4)
    if end < 0:
        return None
    match = re.search(r"^modified:\s*(\S+)\s*$", text[4:end], re.MULTILINE)
    return match.group(1).strip("\"'") if match else None


def decode(data):
    if not data.startswith(MAGIC):
        raise ValueError("unsupported memory header/version")
    pos, records = len(MAGIC), []
    while pos < len(data):
        if len(data) - pos < 36:
            raise ValueError("truncated frame header")
        length = struct.unpack_from(">I", data, pos)[0]
        expected = data[pos+4:pos+36]
        pos += 36
        if length > MAX_FRAME or length > len(data) - pos:
            raise ValueError("invalid/truncated compressed frame")
        decoder = zlib.decompressobj()
        raw = decoder.decompress(data[pos:pos+length], MAX_DECODED + 1)
        if len(raw) > MAX_DECODED or decoder.unconsumed_tail or decoder.unused_data or not decoder.eof:
            raise ValueError("invalid/oversized compressed payload")
        if hashlib.sha256(raw).digest() != expected:
            raise ValueError("frame checksum mismatch")
        record = json.loads(raw)
        if record.get("schema") != 1:
            raise ValueError("unsupported record schema")
        records.append(record)
        pos += length
    return records


def validated_state(data):
    records, blobs, latest = decode(data), {}, {}
    for record in records:
        for key, encoded in record["new_blobs"].items():
            value = base64.b64decode(encoded, validate=True)
            if digest(value) != key:
                raise ValueError("blob hash mismatch")
            blobs[key] = value
        manifest = {"source": record["source"], "files": record["files"]}
        if digest(canonical(manifest)) != record["snapshot_id"]:
            raise ValueError("snapshot hash mismatch")
        seen = set()
        for file in record["files"]:
            path = file["path"]
            validate_path(path)
            if path in seen or file["blob"] not in blobs:
                raise ValueError("duplicate path or missing content")
            seen.add(path)
        latest[source_key(record["source"])] = record
    return records, blobs, latest


def validate_path(path):
    parsed = PurePosixPath(path)
    if not path or parsed.is_absolute() or ".." in parsed.parts or "\\" in path or str(parsed) != path:
        raise ValueError("path must be a normalized relative path")


def capture(data, source, files, observed_at, code_commit, *, complete=True, allowed_scopes=("repo_shared",)):
    """Append one observed version; repeated identical captures are no-ops.

    Source classification is explicit fixture input, NOT inferred from content.
    A rejected private source is not serialized at all. For production, local
    capture and export need independent policies; this is a team-export proof.
    """
    if source["scope"] not in allowed_scopes:
        raise PermissionError("source is outside this archive's sharing scope")
    if not complete:
        raise ValueError("incomplete scan cannot replace source membership")
    _, blobs, latest = validated_state(data)
    members, new_blobs = [], {}
    for path, content in sorted(files.items()):
        validate_path(path)
        key = digest(content)
        members.append({"path": path, "blob": key, "source_modified_at": native_modified(content)})
        if key not in blobs:
            new_blobs[key] = base64.b64encode(content).decode()
    manifest = {"source": source, "files": members}
    snapshot_id = digest(canonical(manifest))
    prior = latest.get(source_key(source))
    if prior and prior["snapshot_id"] == snapshot_id:
        return data
    record = {
        "schema": 1, **manifest, "snapshot_id": snapshot_id,
        "observed_at": observed_at, "observed_at_commit": code_commit,
        "new_blobs": new_blobs,
    }
    raw = canonical(record)
    compressed = zlib.compress(raw, 6)
    if len(compressed) > MAX_FRAME or len(raw) > MAX_DECODED:
        raise ValueError("frame too large")
    return data + struct.pack(">I", len(compressed)) + hashlib.sha256(raw).digest() + compressed


def rebuild(data):
    """Rebuild current-per-source rows and explicit citation backlinks.

    No embeddings, relevance ranking, citation verification, or native-memory
    write-back is implemented. The saved record stream is the only input.
    """
    _, blobs, latest = validated_state(data)
    rows, reverse = [], []
    for namespace, record in sorted(latest.items()):
        for file in record["files"]:
            content = blobs[file["blob"]].decode("utf-8")
            identity = digest(canonical([namespace, file["path"], file["blob"]]))
            row = {
                "id": identity, "source": record["source"], "path": file["path"],
                "snapshot_id": record["snapshot_id"], "content": content,
                "source_modified_at": file["source_modified_at"],
                "observed_at": record["observed_at"],
                "observed_at_commit": record["observed_at_commit"],
            }
            rows.append(row)
            for match in CITATION.finditer(content):
                reverse.append({
                    "memory_id": identity, "session_id": match.group(1),
                    "turn": int(match.group(2)),
                    "byte_start": int(match.group(3)) if match.group(3) else None,
                    "byte_end": int(match.group(4)) if match.group(4) else None,
                    "relationship": "explicit_citation_not_verified_support",
                })
    return {"memories": sorted(rows, key=lambda x: x["id"]),
            "backlinks": sorted(reverse, key=lambda x: (x["session_id"], x["turn"], x["memory_id"]))}
