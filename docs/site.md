# The website

<https://assaio.dev/> is one file: `site/index.html`. No build step, no framework, no CDN —
the favicon is an inline `data:` SVG and every link is absolute, so the page renders correctly
opened straight from disk. Keep it that way; it is the same posture as the offline dashboard
the tool itself writes. The only other file involved is `wrangler.toml`, which says where the
directory goes and under which domains — deployment configuration, not a build step.

## What it is allowed to say

**The site describes the latest released version, not `main`.** It is read by people deciding
whether to install something, so a feature that exists only on `main` must not appear on it —
`[Unreleased]` in the changelog means "installing the latest release does not give you this."

A CI job enforces the part that can be checked mechanically: the `Assay report · vX.Y.Z` stamp
in the hero must equal the highest `v*` tag in the repo. It runs on every push, PR and tag —
deliberately not path-filtered, because a release cut *without* touching the page is exactly
the drift worth catching.

One version ahead is also allowed, and only one: the release being prepared, recognised as the
newest `## [X.Y.Z]` section in `CHANGELOG.md` that has no tag yet. Without that exception the
rule would be unsatisfiable — [RELEASING.md](../RELEASING.md#the-changelog-flow-exact-tag-coupled)
requires the page and the changelog section to be updated *before* the tag is cut, and `main` is
protected, so a prep commit that can never be green could never be merged and the tag could never
exist. The `consistency` workflow is what keeps the pending version to exactly one.

Everything else is a review norm, listed in [RELEASING.md](../RELEASING.md#the-public-surface-check-before-every-tag):
the supported-tool list, the command list, validator counts, and any "on the roadmap" wording
about something that has since shipped.

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

`site.yml` no longer deploys. It runs the version guard on every push, tag and pull request,
and nothing else.

### What that costs, stated plainly

Publishing is no longer downstream of the guard. Cloudflare builds from the commit, not from a
green check, so a `main` whose page names the wrong version **will go live** and the guard will
report it afterwards rather than hold it back. The check still fails the branch, which is what
a reviewer and the release checklist see — but "the site cannot be published stale" is no
longer true, and the release-time step in
[RELEASING.md](../RELEASING.md#after-the-workflow-finishes) is now the thing that catches it.

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
