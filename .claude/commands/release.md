---
description: Prepare the next release per RELEASING.md — gate, price table, surface audit, changelog — and stop at the tag command
argument-hint: "[patch|minor|vX.Y.Z]"
allowed-tools: Bash, Read, Edit, Write, Grep, Glob, WebFetch, Agent
---

Prepare a release. Requested bump: $ARGUMENTS (empty means: decide it from what is under
`## [Unreleased]` in `CHANGELOG.md` and say why).

Run the **release-captain** agent to drive `RELEASING.md` end to end, and have it delegate the
prose half of the public-surface check to the **surface-auditor** agent.

Two things this command must never do:

- tag or push anything — print the exact `make release-*` and `git push origin <tag>` commands
  for the maintainer instead;
- proceed past a red gate, a stale `internal/pricing/litellm.json`, or a `[Unreleased]`
  section that still holds entries after the retitle.

Finish with: the version chosen and why, what changed on each published surface, the two
commands to run, and the post-release checks (attestation, and the live-page diff against
`site/index.html`).
