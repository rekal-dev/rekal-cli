"""Render a terminal walkthrough using synthetic project context."""
from html import escape
from pathlib import Path

lines = [
    ("# New feature: export invoices as CSV", "#a6a6b4"),
    ('$ rekal "invoice export conventions"', "#22d3ee"),
    ("", ""),
    ("  Memory       exports.md", "#a79bff"),
    ("  Stream rows. Keep large exports out of memory.", "#e8e8ec"),
    ("  Conversation Invoice export design", "#a79bff"),
    ("  Use UTC timestamps. Exclude internal account IDs.", "#e8e8ec"),
    ("  ref: passage:demo-export:12-14", "#9292a3"),
    ("", ""),
    ("$ rekal query --ref passage:demo-export:12-14", "#22d3ee"),
    ("", ""),
    ("  [12] Export timestamps in UTC for reconciliation.", "#e8e8ec"),
    ("  [13] Keep internal account IDs out of the CSV.", "#e8e8ec"),
    ("  [14] Stream rows so large exports stay bounded.", "#e8e8ec"),
    ("", ""),
    ("# Build the feature with that context. Run the tests.", "#a6a6b4"),
    ('$ git commit -m "Add invoice CSV export"', "#22d3ee"),
    ("# Rekal checkpoints supported sessions and project notes.", "#a6a6b4"),
]
parts = [
    '<svg xmlns="http://www.w3.org/2000/svg" width="1000" height="720" viewBox="0 0 1000 720" role="img" aria-labelledby="title desc">',
    '<title id="title">New feature. Context ready.</title>',
    '<desc id="desc">A terminal walkthrough: search existing project memory before adding invoice CSV export, read an original source, then build and commit. Synthetic content and abbreviated output.</desc>',
    '<rect x="1" y="1" width="998" height="718" rx="18" fill="#0c0c12" stroke="#2d2d38"/>',
    '<path d="M1 57H999" stroke="#2d2d38"/>',
]
for x, color in [(30, "#fb7185"), (54, "#fbbf24"), (78, "#34d399")]:
    parts.append(f'<circle cx="{x}" cy="29" r="6" fill="{color}"/>')

def text(x, y, value, size=21, color="#e8e8ec"):
    parts.append(f'<text x="{x}" y="{y}" fill="{color}" font-size="{size}" font-family="DejaVu Sans Mono,monospace">{escape(value)}</text>')

text(118, 36, "rekal / your-project", 17, "#9292a3")
text(32, 100, "New feature. Context ready.", 26)
for i, (line, color) in enumerate(lines):
    if line:
        text(32, 147 + i * 28, line, 21, color)
text(32, 692, "Illustrative terminal output · synthetic context · already captured history", 15, "#9292a3")
parts.append("</svg>")
Path("docs/assets/demo.svg").write_text("\n".join(parts) + "\n")
