# Governance

## Roles

**Maintainer** — has merge rights, sets technical direction, and is a code owner
(see `.github/CODEOWNERS`). Currently: [@karauda](https://github.com/karauda).

**Contributor** — anyone who has a PR merged. No special rights beyond authorship
credit; contributors are encouraged to review open PRs.

## How decisions are made

Day-to-day technical decisions (bug fixes, small features, connector additions) are
made by whichever maintainer reviews the PR, using the code philosophy in `AGENTS.md`
and the rules in `CONTRIBUTING.md` as the standard.

Decisions with lasting architectural impact (new core interfaces, storage or data model
changes, licensing) are recorded as an ADR under `docs/adr/` before implementation.
Disagreements are resolved by maintainer consensus; if maintainers cannot agree, the
most senior maintainer (by tenure) makes the final call and documents the reasoning in
the ADR.

## Becoming a maintainer

Maintainer status is offered, not requested. A contributor who has consistently
submitted high-quality PRs and thoughtful reviews over time may be invited to become a
maintainer by existing maintainers. There is no fixed quota or term; the bar is
sustained, trustworthy judgment on this codebase.

## Contribution rules

All contributions go through the process in `CONTRIBUTING.md`:

- One commit per pull request (squashed before review).
- Every commit signed off per the [Developer Certificate of Origin](https://developercertificate.org/) (DCO).
- `main` is protected: a pull request and green CI are required for every change.
  External contributions additionally require an approving review from a maintainer.
  While the project has a single maintainer, that maintainer self-merges their own
  changes after green CI (GitHub does not permit self-approval); the review requirement
  takes effect for everyone as soon as there is a second maintainer.

## Code of conduct

Participation in this project is governed by `CODE_OF_CONDUCT.md`.
