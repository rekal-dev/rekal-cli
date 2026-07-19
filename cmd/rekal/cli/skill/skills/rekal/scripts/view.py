#!/usr/bin/env python3
"""Compress rekal query JSON for agent ingest.

Every skill-facing `rekal query` should pipe here. Recall stays on `route.py`
(gate + digest). This script strips JSON chrome from session drills and SQL
rows so the agent reads dialogue / values, not structure.

  rekal query --session <id> --offset N --limit 5 | python3 view.py
  rekal query --session <id> --full                 | python3 view.py
  rekal query "SELECT …"                            | python3 view.py
  rekal query --index "SELECT …"                    | python3 view.py

Exit: 0 ok, 1 empty/no usable content, 2 parse/usage error.
"""
from __future__ import annotations

import json
import sys


def _load(raw: str):
    raw = raw.strip()
    if not raw:
        return None
    # SQL mode: JSON array.
    if raw[0] == "[":
        return json.loads(raw)
    # Session drill is one JSON object (often pretty-printed). SQL mode is
    # NDJSON (one object per line). Try a single value first; on trailing
    # data, fall back to line-wise.
    if raw[0] == "{":
        try:
            return json.loads(raw)
        except json.JSONDecodeError:
            pass
        rows = []
        for line in raw.splitlines():
            line = line.strip()
            if not line:
                continue
            rows.append(json.loads(line))
        return rows
    rows = []
    for line in raw.splitlines():
        line = line.strip()
        if not line:
            continue
        rows.append(json.loads(line))
    return rows


# Role abbreviations — one letter where stable; keep rare roles readable.
_ROLE_ABBR = {
    "human": "h",
    "assistant": "a",
    "human_steering": "hs",
    "summary": "sum",
    "system": "sys",
    "tool": "tool",
}


def _role_abbr(role: str) -> str:
    if not role:
        return ""
    return _ROLE_ABBR.get(role, role)


def _split_ts(ts: str) -> tuple[str, str]:
    """Split into (date, time) for shortening. Handles `YYYY-MM-DD HH:MM…` and ISO `T`."""
    ts = ts.strip()
    if "T" in ts and " " not in ts.split("T", 1)[0]:
        date, _, rest = ts.partition("T")
        return date, rest
    if " " in ts:
        date, _, rest = ts.partition(" ")
        return date, rest
    return ts, ""


def _short_ts(ts: str, prev_ts: str) -> str:
    """Omit repeated date (or whole stamp) when same as the previous turn."""
    if not ts:
        return ""
    if not prev_ts:
        return ts
    if ts == prev_ts:
        return "…"
    date, time = _split_ts(ts)
    prev_date, _ = _split_ts(prev_ts)
    if date and time and date == prev_date:
        return f"…{time}"
    return ts


def view_session(data: dict) -> str:
    sid = data.get("session_id") or "?"
    turns = data.get("turns") or []
    if not turns:
        return f"{sid} (no turns)"
    idxs = [t.get("index") for t in turns if isinstance(t, dict) and t.get("index") is not None]
    span = f"t{min(idxs)}-{max(idxs)}" if idxs else f"{len(turns)} turns"
    # Legend once — avoid repeating "human"/"assistant" on every line.
    lines = [f"{sid} {span}  (h=human a=assistant hs=steering sum=summary)"]
    prev_ts = ""
    for t in turns:
        if not isinstance(t, dict):
            continue
        idx = t.get("index")
        role = _role_abbr(t.get("role") or "")
        ts_raw = (t.get("ts") or "").strip()
        ts = _short_ts(ts_raw, prev_ts)
        if ts_raw:
            prev_ts = ts_raw
        content = (t.get("content") or "").rstrip()
        # Keep timestamp — agents need when, not just what. Same-date → …time.
        head = []
        if idx is not None:
            head.append(f"t{idx}")
        if ts:
            head.append(ts)
        if role:
            head.append(role)
        prefix = " ".join(head)
        lines.append(f"{prefix}: {content}" if prefix else content)
    # Optional --full extras, compacted.
    tools = data.get("tool_calls") or []
    if tools:
        lines.append("tools:")
        for tc in tools[:40]:
            if not isinstance(tc, dict):
                continue
            tool = tc.get("tool") or "?"
            path = tc.get("path") or ""
            lines.append(f"  {tool} {path}".rstrip())
        if len(tools) > 40:
            lines.append(f"  (+{len(tools) - 40} more tools)")
    files = data.get("files_touched") or data.get("files") or []
    if files:
        lines.append("files: " + " ".join(str(f) for f in files[:30]))
        if len(files) > 30:
            lines.append(f"  (+{len(files) - 30} more files)")
    children = data.get("child_session_ids") or []
    if children:
        lines.append("children: " + " ".join(str(c) for c in children[:20]))
    if data.get("has_more"):
        lines.append("(has_more — raise --limit / --offset)")
    return "\n".join(lines)


def _fmt_cell(v) -> str:
    if v is None:
        return ""
    if isinstance(v, (dict, list)):
        return json.dumps(v, separators=(",", ":"), ensure_ascii=False)
    s = str(v).replace("\n", "\\n")
    return s


def view_rows(rows: list) -> str:
    if not rows:
        return "(no rows)"
    # Stable column order from first row keys, then extras.
    keys: list[str] = []
    seen = set()
    for r in rows:
        if not isinstance(r, dict):
            continue
        for k in r:
            if k not in seen:
                seen.add(k)
                keys.append(k)
    if not keys:
        return "\n".join(str(r) for r in rows)
    out = ["\t".join(keys)]
    for r in rows:
        if not isinstance(r, dict):
            out.append(str(r))
            continue
        out.append("\t".join(_fmt_cell(r.get(k)) for k in keys))
    return "\n".join(out)


def main() -> int:
    if len(sys.argv) > 2 or (len(sys.argv) == 2 and sys.argv[1] in ("-h", "--help")):
        print(
            "usage: view.py [query.json]  # else stdin\n"
            "  Compress rekal query JSON (session drill or SQL rows).\n"
            "  Recall → use route.py instead.",
            file=sys.stderr,
        )
        return 2
    try:
        raw = open(sys.argv[1], encoding="utf-8").read() if len(sys.argv) == 2 else sys.stdin.read()
        data = _load(raw)
    except (OSError, json.JSONDecodeError) as e:
        print(f"view: parse_error:{e}", file=sys.stderr)
        return 2

    if data is None:
        print("(empty)")
        return 1

    # Mis-piped recall JSON — don't silently dump structure; point at route.py.
    if isinstance(data, dict) and ("results" in data or "knowledge" in data) and "turns" not in data:
        print(
            "view: this looks like recall JSON — pipe through route.py instead",
            file=sys.stderr,
        )
        return 2

    if isinstance(data, dict) and "turns" in data:
        text = view_session(data)
        print(text)
        return 0 if (data.get("turns") or data.get("tool_calls") or data.get("files")) else 1

    if isinstance(data, list):
        text = view_rows(data)
        print(text)
        return 0 if data else 1

    if isinstance(data, dict):
        # Single SQL row object (rare) or unknown dict — compact as one row.
        text = view_rows([data])
        print(text)
        return 0

    print(str(data))
    return 0


if __name__ == "__main__":
    sys.exit(main())
