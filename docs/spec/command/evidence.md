# rekal evidence

Fetch original evidence from several locations in one bounded call. The agent
chooses what to fetch; the command handles expansion, deduplication, citations
and pagination. Data remains local and the ledger is never modified.

```bash
rekal --json -n 5 "why did we change retries?" | rekal evidence --input - -j
rekal evidence --anchor s3:12-15 --anchor s8:42 --radius 2
rekal evidence --input page.json --json
```

Requires an initialized repo and a populated index. Reads the index so own,
synced teammate and locally imported transcripts use the same interface.

| Flag | Default | Meaning |
| --- | ---: | --- |
| `--anchor` | none | `session:turn[-turn][@startByte-endByte]`; repeatable; session may be a short handle or full ID |
| `--input` | none | Recall JSON, an explicit anchors request, or a previous evidence page; `-` reads stdin |
| `--radius` | 2 | Neighboring turns on each side, between 0 and 8 |
| `--max-bytes` | 16000 | Maximum original UTF-8 text bytes, between 128 and 1048576; excludes metadata |
| `--max-turns` | 80 | Maximum delivered slices, between 1 and 512 |
| `-j`, `--json` | false | Structured output with citations and continuation |

`--anchor` and `--input` are mutually exclusive. Input is capped at 4 MiB,
128 distinct anchors and 256 turns per anchor before neighbor expansion.
For example, `s8:42@70000-78000` fetches the located bytes of a long turn;
byte bounds on a multi-turn range apply to its first and last turn.

Matching ranges have budget priority over neighboring context. Overlapping
source bytes are delivered once within a page. Windows are grouped by source
session and ordered by original assertion time where known, then session time;
turns inside each session keep their original order. No timestamp is invented.

JSON returns `windows`, `content_bytes`, `max_bytes`, `max_turns`, `truncated`,
optional `warnings`, and optional `next`. Each window includes its source
`session_id`, short `sid`, turn range, original turns and explicit gaps between
delivered turn indices. Each turn includes role, original timestamp when known,
content, a `session_id:tN` citation, byte bounds and a `partial` marker for a
clipped long turn. Byte ranges are half-open and never split a UTF-8 rune.

The byte budget measures text, not JSON formatting or model tokens. Metadata
is additionally bounded by the turn-slice and input-anchor limits.

When `truncated` is true, `next` contains canonical source IDs, the original
radius, cursor and byte offset. Feeding that output through `--input` resumes
the fetch; neighboring context may overlap across pages. The cursor is bound
to the source-window layout: if append/rebuild changes it, restart the request
instead of silently skipping evidence. A session-wide semantic hit without a
located passage emits a warning and contributes no invented turn-zero anchor.

Text output prints citations and original text. JSON is needed to retain the
machine-readable continuation. Command/runtime errors exit 1; a completed
fetch exits 0, including an empty result accompanied by warnings.
