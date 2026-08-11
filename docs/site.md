# The website

<https://assaio.dev/> is `site/index.html`, written by hand. No build step, no framework, no
CDN — the favicon is an inline `data:` SVG and every link is absolute, so the page renders
correctly opened straight from disk. Keep it that way; it is the same posture as the offline
dashboard the tool itself writes. `wrangler.toml` says where the directory goes and under which
domains — deployment configuration, not a build step.

Three other files sit beside it in `site/`. One is generated, and two are there because they are
read by something that is **not** the browser rendering the page:

- **`reference.html`** — the second page, and the only generated one:
  `assaio-agent docs export --format html`, committed, with `make test` failing when it differs
  from what the binary would write. It publishes what can be enumerated — every signal, source,
  validator, command, flag, configuration key and metric-contract field — so the hand-written
  page never has to list any of it. Regenerate with `make docs`; do not edit it.
  **It is served at `/reference`, not `/reference.html`**: Workers Assets drops the extension and
  `307`s the file name away, the same handling that serves `index.html` at `/`. A link or a
  canonical naming the file therefore points at a redirect — which looks correct in review and in
  a browser opened on the file, and is only wrong against the deployed site. The `servedpaths`
  job holds every internal link and canonical to a path the host actually serves.

- **`og.png`** — the card a shared link renders as. Every Open Graph tag was present except
  this one, so LinkedIn and the rest showed a bare grey line; a `data:` URI cannot stand in,
  because no major crawler accepts one for `og:image`. It does not weaken the "loads nothing"
  promise: the page never requests it, a crawler fetches it out of band when someone pastes the
  link. Redraw it with `python3 docs/assets/make-og.py site/og.png` rather than editing the PNG,
  and upload the same image by hand to the repository's Settings → Social preview, which has no
  API. The `sharecard` job fails if the tags go missing or point at a file that is not served.
- **`llms.txt`** — the [llmstxt.org](https://llmstxt.org/) summary an assistant reads instead of
  scraping the page. It states what assaio measures, what it refuses to measure, and links the
  documents that answer the rest.

Anything added to `site/` is served, so it inherits the page's own rules: the `noversion` job
reads every `.html`, `.txt` and `.md` under that directory, not just the page.

## What it is allowed to say

**The site describes the latest released version, not `main`.** It is read by people deciding
whether to install something, so a feature that exists only on `main` must not appear on it —
`[Unreleased]` in the changelog means "installing the latest release does not give you this."

**The page names no version at all, and a CI job holds it to that.** It used to carry an
`Assay report · vX.Y.Z` stamp in the footer, guarded against the highest `v*` tag, because the
page had once run three releases behind the binary. Both the stamp and that guard are gone, and
the reasoning is worth keeping: a version copied onto the page is a fact that has to be updated
at every tag, and the release that forgets to update it is, by definition, the one nobody
notices. A fact worth stating once belongs in the one place it cannot rot. For "which version is
current" that place is the releases page, which the site now links instead of copying.

What replaced the guard is its inverse: `site.yml` fails if a bare `X.Y.Z` appears anywhere under
`site/`. Reintroducing a stamp is therefore a deliberate act with a red check attached, not an
easy convenience that quietly creates a chore. Two shapes are masked before the search, because
neither is a stamp: SVG geometry attributes, whose optimized path data writes implicit separators
(`d="M12.5.5…"`), and dotted quads, because `reference.html` publishes `serve --addr`'s default of
`127.0.0.1:8787`. Masking is the same move both times — say what is *not* a version, then look.

### The claims the page is held to

The rest of what a release used to re-read by hand is now a test. `site/index.html` declares which
of its claims are checkable — `data-claim="command.digest"` on the entry that describes it,
`data-covers="commands"` on the list that promises to enumerate them, `data-claim="sources.count"`
on the word "five" — and `internal/docs` checks exactly those against the binary's own registries.
It runs in both directions, and the second one is the one that matters: a claim with nothing
behind it fails, **and so does a shipped capability the page never names**. `digest` and
`mark --suggest` shipped in v0.17.0 and this page said neither for a whole release.

What it deliberately does not check is prose. "Nineteen reads, one faceplate" has its number
verified and its sentence trusted; the questions-it-answers section, the honest-scope section and
the colophon are checked by a reader, not a regex. A guard that implied otherwise would be the
false green this project exists to refuse. The set of covered claims is `commands`, `sources`,
`signals`, `validators` and `config`, and only top-level commands are enumerable — a page has to
name every capability, not every subcommand of one.

`site/llms.txt` cannot carry attributes, so it gets the weaker form of the same rule: every
source has to be named somewhere in it, matched the way prose writes names ("Claude Code"
satisfies `claude-code`). It states no counts at all any more — it points at `reference.html`
and tells the assistant reading it to answer capability questions from there.

### Saying what does not exist yet

The page no longer describes unreleased work at all. It used to carry a roadmap section — eight
cards, 10.5% of the file — with an explicit *not released yet* marker on the next-release panel.
That section was deleted rather than guarded, for the reason the version stamp was: it duplicated
[ROADMAP.md](../ROADMAP.md), which is linked, and it had to be re-read at every tag. Describing a
capability anywhere on the page is now, without exception, claiming it ships today.

The one place a version-shaped string survives is `v1.0`, used as the **name of a milestone**
rather than as a stamp. The guard draws the same line by requiring three components: `v1.0`
passes, `v1.0.0` does not.

### The page fetches nothing

The colophon ends by promising the page loads nothing: no fonts, no analytics, no third-party
requests. That is a property of the artifact, so `site.yml` checks it. Every fetching attribute —
`src`, `srcset`, `poster`, `data` — must hold a `data:` URI, in any quoting form including none,
because `src="…"` alone would miss the three other ways a pasted embed writes it. A `<link>` may
only be `canonical` or `alternate`, or carry a `data:` URI: nearly every other `rel` fetches, so
the check allows the two that do not rather than trying to list the ones that do. And no
`@import` or `url(//…)`, protocol-relative included. The check exists because the sentence is one
pasted embed away from being false, and a marketing badge is precisely the shape that arrives as
one. The Product Hunt badge in the hero is the worked
example: the official SVG, both themes, inlined. Inlining it also froze its upvote counter, so
the counter was removed rather than shipped stale — a page arguing that a number should never
look more certain than it is cannot display a stale one.

## How it deploys

**Cloudflare Workers Builds** is connected to this repository and runs `npx wrangler deploy`
on every push to `main`. It publishes `site/` as a Worker whose only content is that directory
— `[assets]` with no script is a complete Worker when there is nothing to compute. There is no
build command; the deploy is an upload.

**`package.json` pins the deploy tool, and that is all it does.** Nothing here builds the page;
the pin exists because `npx wrangler deploy` resolves `latest` at build time, which made the
deploy a hostage of whatever npm published in the preceding minutes. It failed exactly that way
once: `wrangler@4.121.0` went up at 19:25:24Z, the build ran 88 seconds later, and its brand-new
`miniflare` dependency was not resolvable yet — `ETARGET`, three deploys skipped, the site
serving a previous commit while every other check was green. A version resolved at build time is
an unpinned input like any other, and this one belongs in a diff for the same reason the domains
in `wrangler.toml` do.

Both custom domains are declared in `wrangler.toml`, so the deploy creates the DNS records and
issues the certificates itself. That is what the file is for: which domains serve this page is
a fact about the project, and keeping it in the repository is what stops it from becoming a
setting in a dashboard nobody remembers — which is how the site deploy stayed broken for five
releases.

`site.yml` no longer deploys. It runs the three artifact guards — no version named, nothing
fetched at render time, a share card that exists — over every served page on each push, tag and
pull request, and nothing else. The content guards are Go tests, so they run in `make test`
locally and in `ci.yml`: keeping them in the language that owns the registries is what lets them
compare against the registries rather than against another copy of the answer.

### What that costs, stated plainly

Publishing is no longer downstream of the guards. Cloudflare builds from the commit, not from a
green check, so a `main` carrying a stale page **will go live** and the guards will report it
afterwards rather than hold it back. They still fail the branch, which is what a reviewer and the
release checklist see — but "the site cannot be published stale" is not true, and the
release-time step in [RELEASING.md](../RELEASING.md#after-the-workflow-finishes) is what catches
it. Removing the version stamp shrank what can go stale in the first place, which is a better
answer than a guard: what is not written down cannot fall behind.

The alternative is to deploy from GitHub Actions with two repository secrets, which keeps the
guard upstream of publication. Both work; this one trades that guarantee for having no
credential to manage.

## One-time setup

**1. Domain onto Cloudflare.** Dashboard → *Add a site* → `assaio.dev` → Free plan. Cloudflare
issues two nameservers; set them at the registrar. The zone flips to *Active* once they
propagate (usually minutes, up to 24 h). Expect the domain to stop resolving entirely between
the nameserver switch and the first deploy: the registrar's parking records are gone and the
zone is empty until the first `wrangler deploy` creates the real ones.

**2. Connect the repository.** *Workers & Pages* → *Create* → *Workers* → *Import a repository*
→ `assaio/assaio`. Worker name **`assaio`**, no build command, deploy command
`npx wrangler deploy`.

**Leave *Builds for non-production branches* unchecked.** Cloudflare does not document whether
a non-production build touches the production Worker or its custom domains, and a branch that
is not `main` has no business publishing this page either way.

No repository secrets are needed on this path: Cloudflare authenticates through its own Git
connection.

**3. Verify.** Push to `main` and load <https://assaio.dev/>. The footer of a release is the
last check in [RELEASING.md](../RELEASING.md#after-the-workflow-finishes): the live page must
name the version just published.

**Optional.** `www.assaio.dev` serves the same assets, so nothing is broken without this — but
the page's `<link rel="canonical">` points at the apex, so a zone-level *Redirect Rule* sending
`www` there is the tidier end state. It is a rule, not a route: it runs before the Worker and
needs no change to `wrangler.toml`.

## Why not GitHub Pages

It would work, but it means configuring two systems instead of one — GitHub for hosting,
Cloudflare for DNS — plus a slower certificate path and an SSL mode (`Full`) that is easy to get
wrong. The domain is on Cloudflare either way; there is nothing to gain by splitting it.

## Why a Worker rather than Cloudflare Pages

Cloudflare does not call Pages obsolete, and for one static file either would serve the page
identically. What decided it is where the configuration lives. Pages wants the project created
by hand and each custom domain attached in the dashboard; a Worker takes both from
`wrangler.toml`, so what serves this page and under which domains is reviewable in a diff. The
failure this repository actually had was not a hosting limitation — it was five releases of a
red job whose fix was four dashboard steps nobody had written down anywhere the build could
see. Connecting the repository to Workers Builds keeps one of those steps in the dashboard;
the domains, which are the part that rots quietly, stay in the file.
