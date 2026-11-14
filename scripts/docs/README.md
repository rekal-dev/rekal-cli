# Shared marketing demo

`demo-story.json` is the canonical, synthetic story for both the README SVG and
rekal.dev's interactive terminal. Do not put captured conversations in this file.

After editing the story, run from the repository root:

```sh
python3 scripts/docs/render-demo.py
```

Commit the story and `docs/assets/demo.svg` together. Demo consistency CI checks
that the generated SVG matches. The README image links to the interactive demo.

The website checks out this repository and imports the story during deployment.
It refreshes every six hours, on website pushes, or through its manual Deploy
workflow. A failed sync blocks deployment, retaining the last successful site.
Its checked-in JSON is a development snapshot; edit the canonical file here.
The two renderers share content, while the website adds interactive controls.
