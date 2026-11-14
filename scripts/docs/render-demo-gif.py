"""Render a looping mobile-friendly view of the SVG's agent conversations."""
import copy
import io
import re
from pathlib import Path
import xml.etree.ElementTree as ET

import cairosvg
from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
source = ET.fromstring((ROOT / 'docs/assets/demo.svg').read_text())
ns = '{http://www.w3.org/2000/svg}'
for element in list(source):
    if element.tag == ns + 'style':
        source.remove(element)
frames = []
# Show one conversation at a time so the text remains readable on a phone.
for tick in range(128):
    elapsed = tick / 4
    frame = copy.deepcopy(source)
    for group in frame.findall(ns + 'g'):
        if group.get('class') != 'row':
            continue
        start = float(re.search(r'animation-delay:([0-9.]+)s', group.get('style', '')).group(1))
        group.attrib.pop('style', None)
        if elapsed < start:
            frame.remove(group)
            continue
        for line in group.findall(ns + 'text'):
            if line.get('class') == 'typed':
                line_start = float(re.search(r'animation-delay:([0-9.]+)s', line.get('style', '')).group(1))
                line.text = (line.text or '')[:max(0, int((elapsed - line_start) * 45))]
            line.attrib.pop('style', None)
    png = cairosvg.svg2png(bytestring=ET.tostring(frame))
    img = Image.open(io.BytesIO(png)).convert('RGB')
    left = 20 if elapsed < 13 else 560
    frames.append(img.crop((left, 100, left + 520, 700)))
frames[0].save(ROOT / 'docs/assets/demo.gif', save_all=True,
               append_images=frames[1:], duration=250, loop=0, optimize=False)
print('Rendered 32-second looping conversation GIF')

# CI verifies that the committed animation matches these generation inputs.
import hashlib
import json
paths = ['scripts/docs/demo-story.json', 'scripts/docs/render-demo.py',
         'scripts/docs/render-demo-gif.py', 'docs/assets/demo.svg', 'docs/assets/demo.gif']
(ROOT / 'docs/assets/demo-manifest.json').write_text(json.dumps(
    {p: hashlib.sha256((ROOT / p).read_bytes()).hexdigest() for p in paths},
    indent=2) + '\n')
