---
name: release
description: Prepares the next release per RELEASING.md — green gate, LiteLLM price-table refresh, the public-surface audit through surface-auditor, CITATION.cff, the changelog retitle commit — and stops at the exact tag and push commands for the maintainer. Use "release", "cut a version", "prepare vX.Y.Z", "tag".
argument-hint: "[patch|minor|vX.Y.Z]"
allowed-tools: Bash, Read, Edit, Write, Grep, Glob, WebFetch, Agent
---

Requested bump: $ARGUMENTS (empty means: decide it from what is under `## [Unreleased]` in
`CHANGELOG.md` and say why — a new feature, command or any record-schema / protocol change is
minor while pre-1.0).

Run the **release-captain** agent to drive `RELEASING.md` end to end, and have it delegate the
prose half of the public-surface check to the **surface-auditor** agent. `/review repo` first
when the release carries a figure change.

Two things this skill must never do:

- tag or push anything — print the exact `make release-*` and `git push origin <tag>` commands
  for the maintainer instead (the tag push publishes a GitHub Release and a Homebrew formula);
- proceed past a red gate, a stale `internal/pricing/litellm.json`, or a `[Unreleased]` section
  that still holds entries after the retitle.

Finish with: the version chosen and why, what changed on each published surface, the two
commands to run, and the post-release checks (attestation, and the live-page diff against
`site/index.html`).
