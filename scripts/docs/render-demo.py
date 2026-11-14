"""Render the synthetic README walkthrough; no captured user data."""
from html import escape
from pathlib import Path
steps = [
 ('01  KEEP THE DECISION', 'Dana / Claude Code / yesterday', 'git commit -m "Add jitter to webhook retries"', 'Fixed delays synchronized failing deliveries.', 'Capture the discussion and project note at checkpoint.'),
 ('02  FIND IT AGAIN', 'Sam / Codex / today', 'rekal "why do webhook retries use jitter?"', 'After rekal sync: find conversation leads and matching notes.', 'Choose a source reference returned by search.'),
 ('03  READ BEFORE CHANGING', 'Sam / Codex / same task', 'rekal query --ref <returned-reference>', 'Read why jitter was chosen and which constraints mattered.', "Then check today's implementation and tests."),
]
parts=['<svg xmlns="http://www.w3.org/2000/svg" width="1000" height="790" viewBox="0 0 1000 790" role="img" aria-labelledby="title desc">', '<title id="title">New session. New agent. Same project memory.</title>', '<desc id="desc">Illustrative three-step workflow: Dana captures a retry decision, Sam syncs and searches with Codex, then reads its original source. Synthetic content, abbreviated results.</desc>', '<rect width="1000" height="790" rx="20" fill="#08080a"/>']
def text(x,y,s,size=19,color='#c7c7d1',weight='400'):
 parts.append(f'<text x="{x}" y="{y}" fill="{color}" font-size="{size}" font-weight="{weight}" font-family="ui-monospace,Menlo,Consolas,monospace">{escape(s)}</text>')
text(35,48,'New session. New agent. Same project memory.',28,'#e8e8ec','700')
text(35,81,'Git-backed memory for the decisions behind your code.',17,'#8a8a94')
for i,(label,actor,command,line1,line2) in enumerate(steps):
 y=110+i*207
 parts.append(f'<rect x="25" y="{y}" width="950" height="189" rx="12" fill="#101015" stroke="#2d2d38"/>')
 text(47,y+31,label,17,'#a79bff','700');text(47,y+59,actor,14,'#91919e')
 text(47,y+96,'$ '+command,19,'#22d3ee')
 text(47,y+130,line1,17);text(47,y+157,line2,17)
text(35,755,'Illustrative workflow / synthetic content / abbreviated results',15,'#91919e')
parts.append('</svg>')
Path('docs/assets/demo.svg').write_text('\n'.join(parts)+'\n')
