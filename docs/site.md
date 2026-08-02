# The website

<https://assaio.dev/> is one file: `site/index.html`. No build step, no framework, no CDN —
the favicon is an inline `data:` SVG and every link is absolute, so the page renders correctly
opened straight from disk. Keep it that way; it is the same posture as the offline dashboard
the tool itself writes.

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

`.github/workflows/site.yml` publishes to Cloudflare Pages on every push to `main`. There is
no build command — the deploy is an upload of `site/`. Publishing on every push rather than
only when `site/` changed is deliberate: the live page is then always exactly what `main` says,
with no state to reconcile.

Pull requests run the version guard but never deploy.

## One-time setup

**1. Domain onto Cloudflare.** Dashboard → *Add a site* → `assaio.dev` → Free plan. Cloudflare
issues two nameservers; set them at the registrar. The zone flips to *Active* once they
propagate (usually minutes, up to 24 h).

**2. Create the Pages project.** *Workers & Pages* → *Create* → *Pages* → *Use direct upload*,
project name **`assaio`** — it must match `--project-name=assaio` in the workflow. Do not
connect it to Git: the workflow is what deploys, and having both would mean two pipelines
racing to publish the same page.

**3. API credentials as repository secrets.**

- *My Profile* → *API Tokens* → *Create Token* → template **Edit Cloudflare Workers**, or a
  custom token with the **Cloudflare Pages: Edit** permission on your account. Scope it to
  this account only.
- Account ID is on the right-hand sidebar of any zone's overview page.

Add both under repo *Settings* → *Secrets and variables* → *Actions*:

| Secret | Value |
|---|---|
| `CLOUDFLARE_API_TOKEN` | the token from above |
| `CLOUDFLARE_ACCOUNT_ID` | your Cloudflare account id |

**4. Custom domain.** In the Pages project → *Custom domains* → *Set up a domain* → `assaio.dev`,
then again for `www.assaio.dev`. Because the zone is already on Cloudflare, the DNS records and
the TLS certificate are created for you. Send `www` → apex with a *Redirect Rule*: the page's
`<link rel="canonical">` points at the apex, so that is the one to keep.

**5. Verify.** Push to `main` (or run the workflow manually from the Actions tab) and load
<https://assaio.dev/>. The footer of a release is the last check in
[RELEASING.md](../RELEASING.md#after-the-workflow-finishes): the live page must name the
version just published.

## Why not GitHub Pages

It would work, but it means configuring two systems instead of one — Pages for hosting, Cloudflare
for DNS — plus a slower certificate path and an SSL mode (`Full`) that is easy to get wrong. The
domain is on Cloudflare either way; there is nothing to gain by splitting it.
