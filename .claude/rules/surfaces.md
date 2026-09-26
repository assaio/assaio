---
paths:
  - "README.md"
  - "FEATURES.md"
  - "CHANGELOG.md"
  - "BACKLOG.md"
  - "ROADMAP.md"
  - "PRIVACY.md"
  - "CITATION.cff"
  - "site/**"
  - "docs/**"
---

# Published surfaces

- **The site describes the latest release, not `main`** — and every push to `main` publishes it,
  ungated (`docs/site.md`). Nothing on it names a version (`site.yml` fails on a bare `X.Y.Z`),
  loads a remote asset, or links a `.html` path. `site/index.html` is the only hand-written page;
  `site/llms.txt` and `site/robots.txt` are hand-written too.
- **Every `.md` under `docs/` is published or excused**: linked from a path in `docs/index.md`
  (which sets reading order, sidebar and previous/next) or listed in `unpublished` in
  `internal/docs/guides.go`; a new file in neither fails `make test`. `docs/work/` and `docs/adr/`
  are skipped as directories. `docs/reference.json`, `site/reference.html`, `site/docs.html`,
  `site/docs/*` and `site/sitemap.xml` are generated: `make docs`, never edited.
- **Lifecycle**: a shipped item is *deleted* from `BACKLOG.md` (ids never reused, never `[x]`),
  gets one entry under `CHANGELOG.md` `[Unreleased]` (seven headings only), and a row in
  `FEATURES.md` with its release — all in the shipping PR. `consistency.yml` checks the mechanical half.
- **A corrected figure updates two files**: the changelog line and its post-mortem in
  `docs/corrections.md`, linked by anchor.
- **One fact, one place.** Counts and lists that `internal/docs` can check live in
  `data-claim` spans; prose repeats nothing a linked page already states. Before adding a
  sentence, ask which existing sentence it duplicates or contradicts.
- **A new parser changes `PRIVACY.md`, `README.md`'s source table, `site/llms.txt`, `AGENTS.md`'s
  layout, `CITATION.cff`'s abstract and the tool line in `docs/assets/make-og.py` (then redraw
  `site/og.png`)** in the same commit.
- **`ROADMAP.md` carries direction with exit criteria and dated, footnoted sources**; a claim
  about the market names its source and date. An ADR whenever a change makes a commitment a
  future contributor could unknowingly undo; ADRs are amended with a status note, never rewritten.
- `docs/README.md` links every ADR (`TestADRIndexNamesEveryRecord`); every internal link inside
  `site/` must resolve (`TestEveryLinkInsideTheSiteResolves`).
