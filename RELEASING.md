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
| **Major** `v1.0.0` | The stability promise: public plugin/SDK API and the SQLite schema become stable per [ROADMAP.md](ROADMAP.md). | — |

Cadence: patch releases ship as soon as a meaningful fix lands (days, not weeks);
minor releases when a coherent feature set is ready. No date-driven schedule.

## Schema changes (hard rule)

The store applies migrations by **filename**:
[`internal/store/schema.go`](internal/store/schema.go) records each applied file in a
`schema_migration` table and skips any name it has already seen.

- **Before the first public release** — zero users, no shipped database anywhere — edit
  [`internal/store/migrations/0001_init.sql`](internal/store/migrations/0001_init.sql) in
  place. There is no upgraded DB to migrate, so a clean rebuild is the whole story.
- **After the first public release, editing a shipped migration is forbidden.** Every
  schema change **MUST** be a **new** file (`0002_*.sql`, `0003_*.sql`, …). Never touch a
  migration that has already gone out in a release.

Why this is a hard rule and not a style preference: an upgraded user's database already
has `0001_init.sql` recorded as applied, so the runner **skips it** — your edited SQL
never executes, and the new column never lands on their DB. Their `assaio` then queries a
column that does not exist and breaks, while a fresh install works, making the bug
invisible in your own testing. A new `0002_*.sql` has a name the runner has never seen,
so it runs exactly once on every database, new and old alike. A shipped migration is
immutable — the same discipline as an immutable release tag.

## The changelog flow (exact, tag-coupled)

`CHANGELOG.md` and the version tag move in lockstep; `make release` enforces it, so
none of this relies on memory:

1. **During development**, every user-facing change lands under `## [Unreleased]` in
   the same PR that ships it (the PR template's checklist item). `[Unreleased]` always
   means: merged to `main`, **in no tagged release yet**.
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

## Rules

- Tags are immutable: never delete or re-point a published tag. A bad release is
  fixed by the next patch release.
- Release only from `main`. No release branches while pre-1.0; introduce
  `release-vX.Y` branches only if/when backports become necessary.
- A release that changes the record schema or the plugin protocol must say so
  explicitly in the notes (**Breaking** or **Compatibility** section).
