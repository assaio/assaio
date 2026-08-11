# The website

<https://assaio.dev/> is one page: `site/index.html`. No build step, no framework, no CDN —
the favicon is an inline `data:` SVG and every link is absolute, so the page renders correctly
opened straight from disk. Keep it that way; it is the same posture as the offline dashboard
the tool itself writes. `wrangler.toml` says where the directory goes and under which domains —
deployment configuration, not a build step.

Two other files sit beside it in `site/`, and both are there because they are read by something
that is **not** the browser rendering the page:

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

What replaced the guard is its inverse: `site.yml` fails if a bare `X.Y.Z` appears anywhere in
`site/index.html`. Reintroducing a stamp is therefore a deliberate act with a red check attached,
not an easy convenience that quietly creates a chore.

This removes a mechanical check without removing the obligation behind it. Everything a release
still has to confirm by hand is listed in
[RELEASING.md](../RELEASING.md#the-public-surface-check-before-every-tag) — the supported-tool
list, the command list, validator counts, and the roadmap section. None of that was ever
mechanical; the version stamp was the only part that was, and it was also the only part that had
to change on a release that changed nothing else on the page. "On the roadmap" wording about
something that has since shipped belongs to the same read-through.

### Saying what does not exist yet

That rule governs what the page presents as **available**. The roadmap section is the one place
it may name work that does not exist, and it earns that only by leaving no room to misread: the
section states up front that nothing in it is built, the next release carries an explicit *not
released yet* marker, and the installable build is a link to the releases page rather than a
claim. Anywhere else on the page, describing a capability is claiming it ships today.

That section names no release either, which is the same rule applied twice. It says *the next
release*, not `v0.16.0` — the ordering is the information, and a number would only add something
to keep in sync. This also keeps the page consistent with
[ROADMAP.md](../ROADMAP.md#the-next-milestones), which deliberately assigns no version to a
promise for its own stated reason.

The one exception is `v1.0`, which the section uses as the **name of a milestone** rather than as
a stamp — it is what that promise has been called since the roadmap's first draft, and it names
no release date and no release contents. The guard draws the same line by requiring three
components: `v1.0` passes, `v1.0.0` does not.

What the section does need at release time is the obvious thing: an item that shipped leaves it.
That is a review norm in [RELEASING.md](../RELEASING.md#the-public-surface-check-before-every-tag),
alongside the tool and command lists, and it is the same judgement they need.

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

Both custom domains are declared in `wrangler.toml`, so the deploy creates the DNS records and
issues the certificates itself. That is what the file is for: which domains serve this page is
a fact about the project, and keeping it in the repository is what stops it from becoming a
setting in a dashboard nobody remembers — which is how the site deploy stayed broken for five
releases.

`site.yml` no longer deploys. It runs the two page guards — no version named, nothing fetched
at render time — on every push, tag and pull request, and nothing else.

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
