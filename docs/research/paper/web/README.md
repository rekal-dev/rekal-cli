# `/paper` — drop-in site assets

Serves the flagship paper at **`https://rekal.dev/paper`**: an SEO-rich HTML
landing page that inlines the PDF (arXiv-abstract style), with the raw PDF at
`/paper/rekal-paper.pdf`. Built here because the site lives in a separate
repo — copy these two files across and wire one route.

## Files

| File | Serves at | Purpose |
|---|---|---|
| `index.html` | `/paper` | SEO landing: Scholar `citation_*` tags (incl. `citation_arxiv_id` 2607.14390), OpenGraph, Twitter card, JSON-LD `ScholarlyArticle` with `sameAs` arXiv + `identifier`, crawlable abstract + key stats, inlined PDF viewer |
| `rekal-paper.pdf` | `/paper/rekal-paper.pdf` | local PDF mirror (Scholar/`citation_pdf_url` and primary download point at arXiv: `https://arxiv.org/pdf/2607.14390`) |

Regenerate the PDF from `../rekal-paper.typ` (or the LaTeX in `../arxiv/`) and
re-copy it here whenever the paper changes.

## Wire the route (pick your host)

**Static / directory-based (GitHub Pages, most SSGs):** put the folder at
`paper/` in the site root — `paper/index.html` resolves at `/paper`
automatically. Done.

**Netlify** — `_redirects`:
```
/paper    /paper/index.html   200
```

**Vercel** — `vercel.json`:
```json
{ "rewrites": [{ "source": "/paper", "destination": "/paper/index.html" }] }
```

**Next.js** (app router) — drop `index.html` + PDF in `public/paper/`, then in
`next.config.js`:
```js
async rewrites() { return [{ source: '/paper', destination: '/paper/index.html' }]; }
```

**nginx**:
```nginx
location = /paper { try_files /paper/index.html =404; }
```

## SEO checklist (already in `index.html`, plus two site-level steps)

- [x] `<title>`, meta description, canonical `https://rekal.dev/paper`
- [x] Google Scholar tags (`citation_title/author/pdf_url/...`) — how academic search indexes it
- [x] OpenGraph + Twitter `summary_large_image` — link previews on HN/X/Slack/LinkedIn
- [x] JSON-LD `ScholarlyArticle` — Google rich results
- [x] Crawlable abstract text on the page (PDF-only pages index poorly)
- [ ] **Add an OG image** at `/paper/og.png` (1200×630) — referenced by the
      OG/Twitter tags; without it, social cards fall back to text-only. A
      screenshot of the title + Figure 1, or a simple title card, works.
- [ ] **Add to `sitemap.xml`**:
      ```xml
      <url><loc>https://rekal.dev/paper</loc><changefreq>monthly</changefreq><priority>0.9</priority></url>
      ```
      and confirm `robots.txt` doesn't disallow `/paper`.

## Note on "the PDF directly at /paper"

`/paper` serves an HTML page that **inlines the full PDF** (via `<object>`),
so a visitor lands on the paper immediately — while search engines and social
crawlers get real metadata and text they can index, which a bare PDF cannot
provide. The raw file is one click (or `/paper/rekal-paper.pdf`) away. If you
truly want `/paper` to return the PDF bytes with no landing page, point the
route at `rekal-paper.pdf` instead — but you lose Scholar indexing, link
previews, and the stat cards, so the landing page is the recommended default.
