# Shared marketing demo

`demo-story.json` is the canonical, synthetic story for both the README SVG and
rekal.dev's interactive terminal. Do not put captured conversations in this file.

After editing the story, run from the repository root:

```sh
python3 scripts/docs/render-demo.py
# GIF rendering needs CairoSVG and Pillow.
python3 scripts/docs/render-demo-gif.py
```

Commit the story, renderers and generated files in `docs/assets/` together.
The README uses a looping GIF rendered from the SVG, showing one conversation
at a time for mobile readability. Install CairoSVG and Pillow to regenerate it. Demo consistency CI checks
that the generated SVG matches. The README image links to the interactive demo.

The website checks out this repository and imports the story during deployment.
It refreshes every six hours, on website pushes, or through its manual Deploy
workflow. A failed sync blocks deployment, retaining the last successful site.
Its checked-in JSON is a development snapshot; edit the canonical file here.
The two renderers share content, while the website adds interactive controls.
