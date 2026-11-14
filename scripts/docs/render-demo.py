"""Render an animated, synthetic agent handoff; no captured user data."""
import json
import textwrap
from html import escape
from pathlib import Path

story = json.loads(Path(__file__).with_name("demo-story.json").read_text())
parts = ['<svg xmlns="http://www.w3.org/2000/svg" width="1100" height="640" viewBox="0 0 1100 640" role="img" aria-labelledby="title desc">',
'<title id="title">Carry the context forward.</title>',
'<desc id="desc">Alex builds invoice CSV export with Claude Code and shares eligible context through Git. Later Sam asks Codex to Rekal that context and add date filters. Animated illustrative sessions, not a recorded run.</desc>',
'''<style>
.row{animation:reveal .2s both}
.typed{animation:typing 1s steps(38,end) both}
@keyframes reveal{from{opacity:0}to{opacity:1}}
@keyframes typing{from{clip-path:inset(0 100% 0 0)}to{clip-path:inset(0 0 0 0)}}
@media(prefers-reduced-motion:reduce){.row,.typed{animation:none!important;opacity:1;clip-path:none}}
</style>''',
'<rect width="1100" height="640" rx="16" fill="#08080a"/>']

def text(x, y, value, color="#e8e8ec", size=17, attrs=""):
    parts.append(f'<text x="{x}" y="{y}" fill="{color}" font-size="{size}" font-family="DejaVu Sans Mono,monospace" {attrs}>{escape(value)}</text>')

text(28, 40, "Carry the context forward.", size=25)
text(28, 72, "Two teammates. Different agents. Shared context.", "#9292a3", 16)
for col, person in enumerate(story):
    x = 20 + col * 540
    parts.append(f'<rect x="{x}" y="100" width="520" height="482" rx="12" fill="#0c0c12" stroke="#2d2d38"/>')
    for n, color in enumerate(["#fb7185", "#fbbf24", "#34d399"]):
        parts.append(f'<circle cx="{x+20+n*17}" cy="124" r="4" fill="{color}"/>')
    text(x+78, 130, f'{person["name"]} / {person["agent"]}', size=16)
    parts.append(f'<path d="M{x} 148h520" stroke="#2d2d38"/>')
    text(x+20, 181, person["label"].upper(), "#a79bff", 14)
    y = 218
    for row in person["rows"]:
        color = {"prompt":"#22d3ee", "answer":"#e8e8ec", "tool":"#a79bff", "status":"#6ee7b7"}[row["kind"]]
        value = ("> " if row["kind"] == "prompt" else "") + row["text"]
        parts.append(f'<g class="row" style="animation-delay:{row["at"]}s">')
        for line_no, line in enumerate(textwrap.wrap(value, width=45)):
            attrs = f'class="typed" style="animation-delay:{row["at"]+line_no*.75}s"' if row["kind"] == "prompt" else ""
            text(x+20, y, line, color, 17, attrs)
            y += 24
        parts.append("</g>")
        y += 20
    assert y < 585, y
text(28, 614, "Illustrative agent sessions · setup first · capture → merge/push → sync", "#9292a3", 14)
parts.append("</svg>")
Path("docs/assets/demo.svg").write_text("\n".join(parts)+"\n")
