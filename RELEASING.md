# Releasing

How assaio versions and ships. Maintainers only; contributors never need this.

## Versioning scheme

Semantic versioning, driven entirely by **git tags** (`vMAJOR.MINOR.PATCH`). There is
no version file to bump: the binary's version is injected from the tag at build time
(`internal/version` via ldflags), and Go modules resolve releases from tags.

While pre-1.0:

| Bump | When | Examples |
|---|---|---|
| **Patch** `v0.X.Y+1` | Bug fixes, docs, dependency bumps — no behavior change beyond the fixed bug. | Parser handles a format quirk; price-table refresh. |
| **Minor** `v0.X+1.0` | New features or tools; any breaking change (allowed pre-1.0, always called out in release notes under a **Breaking** heading). | New parser; new command; record-schema change. |
| **Major** `v1.0.0` | The stability promise: the contracts listed in [docs/compatibility.md](docs/compatibility.md) freeze. The SQLite schema is **not** among them — it stays an implementation detail with migration and export guarantees. | — |

Cadence: patch releases ship as soon as a meaningful fix lands (days, not weeks);
minor releases when a coherent feature set is ready. No date-driven schedule.

## Schema changes (hard rule)

The store applies migrations by **filename**:
[`internal/store/schema.go`](internal/store/schema.go) records each applied file in a
`schema_migration` table and skips any name it has already seen.

- **Before the first public release** — zero users, no shipped database anywhere — edit
  [`internal/store/migrations/0001_init.sql`](internal/store/migrations/0001_init.sql) in
  place. There is no upgraded DB to migrate, so a clean rebuild is the whole story.
- **After the first public release, a shipped migration is immutable in name and content.**
  Every schema change **MUST** be a **new** file (`0002_*.sql`, `0003_*.sql`, …). Never edit
  and never rename a migration that has already gone out in a release.

Why this is a hard rule and not a style preference: an upgraded user's database already
has `0001_init.sql` recorded as applied, so the runner **skips it** — your edited SQL
never executes, and the new column never lands on their DB. Their `assaio` then queries a
column that does not exist and breaks, while a fresh install works, making the bug
invisible in your own testing. A new `0002_*.sql` has a name the runner has never seen,
so it runs exactly once on every database, new and old alike. A shipped migration is
immutable — the same discipline as an immutable release tag.

A **rename** is the worse half of the same rule and is easy to reach for while tidying: the
runner has never seen the new name, so it re-executes a body it has already applied. `IF NOT
EXISTS` guards the DDL and nothing guards the DML — re-running
`0008_response_grain_claude.sql` moves every `claude-code` row into the archive table `doctor`
describes as unreadable by any report, and then deletes the originals.

`TestShippedMigrationsAreImmutableInNameAndContent` (`internal/store`) holds both halves: it
carries a digest per shipped file and fails on an edit, a rename or a removal. Adding a
migration means adding its digest in the same commit; changing a digest already listed is the
thing this rule forbids.

## The changelog flow (exact, tag-coupled)

`CHANGELOG.md` and the version tag move in lockstep; `make release` enforces it, so
none of this relies on memory:

1. **During development**, every user-facing change lands under `## [Unreleased]` in
   the same PR that ships it (the PR template's checklist item). `[Unreleased]` always
   means: merged to `main`, **in no tagged release yet**. Entries go under one of seven
   headings — `Breaking`, `Added`, `Changed`, `Fixed`, `Removed`, `Deprecated`,
   `Security` — and run one to three lines each.
   **A change that corrects a figure an earlier release published updates two files**:
   the entry here, and the post-mortem in [docs/corrections.md](docs/corrections.md)
   that entry links to by anchor. The changelog says what moved; the register says what
   the wrong number showed a reader, and what the fix overruled. Neither ships without
   the other — an unlinked correction is the long story with nothing pointing at it, and
   a dangling link is a claim the evidence is filed when it is not.
2. **Before tagging `vX.Y.Z`**, in one `chore(release): prepare vX.Y.Z changelog`
   commit: retitle the `[Unreleased]` section to `## [X.Y.Z] - YYYY-MM-DD`, recreate
   an empty `## [Unreleased]` above it, and update the link references at the bottom
   of the file (`[Unreleased]: …/compare/vX.Y.Z...HEAD`, plus the new
   `[X.Y.Z]: …/releases/tag/vX.Y.Z`).
3. **`make release` refuses to tag** unless both hold: `CHANGELOG.md` has a
   `## [X.Y.Z]` section, and `[Unreleased]` carries no leftover entries (they would
   be silently missing from the release's story). A version heading therefore always
   describes exactly what its tag contains.
4. The GitHub Release's generated notes (Conventional Commit subjects) link back to
   the tag's `CHANGELOG.md` — the changelog is the curated story, the notes are the
   raw commit list.

## The public surface (check before every tag)

The binary is not the only thing a release ships. Before preparing the changelog, confirm
each of these describes the version being cut — a page or a table that lags the binary is
how an honesty-first product starts making false claims about itself:

- `site/index.html` — the page served at <https://assaio.dev/>. **Most of what used to be on
  this line is now a test.** The supported-source list, the command list and the validator and
  signal counts are annotated claims checked against the binary's own registries by
  `make test`, in both directions: a claim with nothing behind it fails, and so does a shipped
  capability the page never names — which is how `digest` sat unpublished for a whole release.
  `site/reference.html` and `docs/reference.json` are generated; regenerate with `make docs`
  and commit the result. The roadmap section is gone for the same reason the version stamp was:
  it was a duplicate of `ROADMAP.md` that had to be re-read at every tag.
  What is left here is the judgement no test can make — whether the prose is still *true*: the
  narrative sections, the caveat lists, and any "on the roadmap" wording about something that
  has since shipped. It deploys itself from `main` (see [docs/site.md](docs/site.md)), so a
  merge publishes it immediately — there is no separate step that would prompt a review.
- `site/llms.txt` — the same page's machine-readable companion, deployed from the same commit
  and covered by none of those annotations. Its source list, its refusals and its index of
  documents are judgement every time.
- `README.md` — the "Every command" table, the caveat list under what it cannot measure, and the
  source list. All three have gone stale before, and once together: a shipped command missing
  from the table, a caveat that had been false for twenty minors, and a source whose second
  structural absence it never named.
  ([correction](docs/corrections.md#readme-commands-and-counts))
- `PRIVACY.md` — the directories the binary opens and the fields each parser extracts. **A new
  parser changes this file in the same commit**, and a parser that reads *less* than the others
  changes it too — that reduction is the part a reader deciding whether this is safe on a work
  machine is owed. Shipping a source without it is already on the register.
  ([correction](docs/corrections.md#privacy-md-named-three-of-five-sources))
- `AGENTS.md` — the "What this is" paragraph and the layout block. It is the first thing every
  contributor and every assistant reads, so a source or an `internal/` package missing from it
  is wrong guidance rather than a stale document.
  ([correction](docs/corrections.md#surfaces-counted-four-sources))
- `FEATURES.md` — a row per user-facing capability, with the release it arrived in.
- `ROADMAP.md` and `BACKLOG.md` — levels and items marked shipped, shipped entries deleted
  from the backlog, and any scope deliberately moved recorded on the item that inherited it.
- `docs/` — including an ADR whenever the release makes a commitment a future contributor
  could unknowingly undo.
- `CITATION.cff` — `version` and `date-released` for the tag being cut, and the tool list in
  the abstract. Mechanical now: `consistency.yml` fails when the version is neither the newest
  tag nor the one `CHANGELOG.md` is preparing, when `date-released` is not that tag's date, or
  when the abstract names a different set of tools than the parsers in `docs/reference.json`.
  It is on this list at all because it is the one published surface nothing in review opens —
  it sat at 0.1.1 for twenty-three releases, still naming four parsers a release after the
  fifth shipped.
- `internal/pricing/litellm.json` — re-download LiteLLM's
  `model_prices_and_context_window.json` and bump `SnapshotDate` in
  `internal/pricing/snapshot.go`. **This is the release's job, not a background chore**: every
  `$` assaio prints is a token count times this table, and a table that has fallen behind is
  indistinguishable from a complete one from the inside — five weeks of drift once left 45.5%
  of the maintainer's tokens unpriced and a window's estimate $15,452.42 short. Two guards
  catch the cases they can — `TestEveryCalibratedModelHasAPrice` fails when a trace in this
  repo names a model the table cannot cost, and `doctor --strict` fails on a reader's own
  store above `pricing.max_unpriced_share` — but neither sees a model the vendor shipped that
  nobody here has run yet. That gap is why this line is on the list.

This list exists because it was skipped: `site/index.html` still advertised v0.2 while
v0.5.0 was being tagged, listing an already-shipped connector as a roadmap item. The
mechanical half of it became a test after it was skipped a second time, in a quieter way —
`digest` and `mark --suggest` shipped in v0.17.0 and the page said neither.

## Cutting a release

Everything happens from a clean, up-to-date `main` with a green CI run.

```sh
# 1. Verify locally (same gate as CI):
make test lint && CGO_ENABLED=0 go build ./...

# 2. Prepare the changelog (step 2 of the flow above) and push it.

# 3. Tag (annotated; refuses to run without the changelog prepared). Pick ONE:
make release-patch CONFIRM=yes    # v0.1.2 -> v0.1.3
make release-minor CONFIRM=yes    # v0.1.3 -> v0.2.0
make release VERSION=v0.2.0 CONFIRM=yes   # explicit version

# 4. Push the tag — this triggers the release workflow:
git push origin <tag>
```

The tag push runs `.github/workflows/release.yml`: goreleaser builds
macOS/Linux/Windows (amd64/arm64) archives with checksums, publishes the GitHub
Release with a changelog generated from Conventional Commit subjects, and attaches
build-provenance attestations.

## After the workflow finishes

1. Check the [Releases](https://github.com/assaio/assaio/releases) page: artifacts,
   checksums, changelog.
2. Edit the release notes if the generated changelog needs a human touch — lead with
   user-facing highlights; put any breaking change under a **Breaking** heading first.
3. Verify provenance of one artifact:
   `gh attestation verify <artifact> -o assaio`.
4. Confirm the deploy actually ran. The page names no version, so there is no stamp to eyeball —
   and a release that changed nothing on the page would leave nothing to eyeball anyway. Compare
   the bytes instead, which works either way:

   ```sh
   diff <(curl -fsSL https://assaio.dev/) site/index.html && echo "live page matches this commit"
   ```

   The site redeploys on every push to `main`, and publishing is not gated on any check (see
   [docs/site.md](docs/site.md#what-that-costs-stated-plainly)), so this is the only thing
   standing between a failed upload and a page nobody looks at again for weeks.

## Rules

- Tags are immutable: never delete or re-point a published tag. A bad release is
  fixed by the next patch release. The `immutable-release-tags` repository ruleset
  enforces this server-side, so a force-push to a tag is rejected even for an admin.
- **If `main` is ever rewritten, published tags stay where they are** and therefore stop
  being ancestors of `main`. That is the correct outcome, not a problem to fix by moving
  them: the tag must keep pointing at the commit the release was actually built from.
  The one thing it breaks is version derivation, because `git describe` only sees tags
  reachable from `HEAD` and would propose a version that has already shipped. `LATEST_TAG`
  in the `Makefile` therefore reads the highest tag that *exists* rather than the nearest
  reachable one. The divergence is self-healing: the next release tags the current history,
  and `git describe` behaves normally again from then on.
- Release only from `main`. No release branches while pre-1.0; introduce
  `release-vX.Y` branches only if/when backports become necessary.
- A release that changes the record schema or the plugin protocol must say so
  explicitly in the notes (**Breaking** or **Compatibility** section).
