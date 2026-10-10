# Compatibility

This page defines what `assaio` promises to keep compatible and what it does not.
[ROADMAP.md](../ROADMAP.md), [RELEASING.md](../RELEASING.md), and [extending.md](extending.md) link
here because their separate versions disagreed for several releases without detection.

`assaio` is pre-1.0. Before `v1.0`, a minor release may break any contract; the changelog marks such
changes **Breaking**. The sections below define the `v1.0` commitments.

## Frozen at v1.0

These are the contracts a third party builds against. At `v1.0` each gets a version, a
compatibility test, and a written deprecation path; after it, a breaking change needs a major.

| Contract | Where it lives |
|---|---|
| Exec plugin protocols — parser, metric, rule | [extending.md](extending.md), ADR [0003](adr/0003-exec-plugin-protocol.md) / [0004](adr/0004-exec-metric-plugin-protocol.md) / [0005](adr/0005-exec-rule-plugin-protocol.md) |
| The observation envelope and its payload types | ADR [0007](adr/0007-canonical-event-contract.md) |
| Signal ids and what a zero means for each | ADR [0008](adr/0008-signal-catalog.md), `assaio-agent signals` |
| The recommendation record and its lifecycle | ADR [0015](adr/0015-structured-recommendations.md) |
| The team sync protocol | [extending/team-server.md](extending/team-server.md) |
| Machine-readable output — `analyze --format json`, `report --format json\|csv`, `evidence --format json`, `docs export` | `docs/reference.json`, [evidence.md](evidence.md) |

## Additive GitHub evidence fields

`evidence --github --format json` adds observation type `scm.review.dismissed` and nullable PR
`reviewDismissals` coverage. The event uses the existing envelope for its opaque id, source,
provenance, privacy, occurrence time and invocation `observedAt`.

| Addition | Machine-readable fields and meaning |
| --- | --- |
| `ReviewDismissal` payload | Required `number` (PR number), `previousState` (source-stated closed review state), `dismissedAt` (actual event time); optional `reviewId`, `commit` (reviewed hash), `submittedAt` (actual review submission time). A null source review leaves all three optional fields absent. |
| Nullable `reviewDismissals` / `TimelinePopulation` | `totalCount` (all timeline events), `filteredCount` (selected dismissals), `pageCount` (source-reported returned slice count), `listed` (actual retained nodes), `hasPreviousPage` (source-stated selected truncation). Missing coverage is null/unavailable; independently validated raw events may still appear. |

The source selects only `__typename`, `id`, `createdAt`, `previousReviewState` and nullable
`review { id submittedAt commit { oid } }` for each dismissal. Actual event time is required,
with no fallback. The last-100 slice remains bounded. Consumers must accept the additional event
type and optional coverage while preserving absent values as unknown.

These additions do not change offline `session-commit/v2`, GitHub `session-commit/v3`, matching,
current review snapshot metrics or existing field meanings; the GitHub algorithm version is not
bumped. No store/schema migration, sync, dashboard, share or plugin contract is added. This is an
additive output extension, without an actual breaking change requiring a Breaking entry. See
[ADR 0025](adr/0025-bounded-review-dismissals.md) and
[the source/count guide](github-source-feasibility.md).

## Not frozen, and guaranteed instead

**The SQLite schema is an implementation detail.** It is not a public API and will not be frozen at
`v1.0`. Direct database reads are supported for exploration
([extending/query-your-data.md](extending/query-your-data.md)), not integration.

These guarantees take its place:

- **Forward migration** from any released version to any later one, applied automatically and tested
  from the oldest supported version. A shipped migration's name and content never change (see
  [RELEASING.md](../RELEASING.md#schema-changes-hard-rule)).
- **Export.** At v1.0 the machine-readable outputs above expose everything in the store, and those
  outputs *are* frozen. Today they are aggregates and do not yet expose every stored record a
  consumer needs to reproduce its totals; the versioned snapshot that closes that gap is `B211` in
  the backlog. Analysis and export need no network, server, or license.
- **Backup.** The store is one file. Copy it while `assaio` is idle for a complete backup.

The v0.12 migration corrected a semantic error by rewriting stored rows, and more corrections may be
needed. Freezing the schema would have made that correction a breaking change instead of a bug fix.
The guarantee is that stored history stays *correctable*.

## Deferred, and not a v1 contract

**An in-process Go plugin API.** Exec protocols are the extension boundary: language-neutral,
opt-in, validated at the boundary, and limited in time and size. No measured performance or
deployment need yet justifies a dynamically loaded in-process API, and Go's `plugin` package is not
portable. If added, it will be a second boundary alongside exec, not a replacement.

**Assaio Cloud.** The managed service is outside this repository's compatibility surface. It will
use the contracts above. The open-source binary will not require it.
