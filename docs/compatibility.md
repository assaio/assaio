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

## Not frozen, and guaranteed instead

**The SQLite schema is an implementation detail.** It is not a public API and will not be frozen at
`v1.0`. Direct database reads are supported for exploration
([extending/query-your-data.md](extending/query-your-data.md)), not integration.

These guarantees take its place:

- **Forward migration** from any released version to any later one, applied automatically and tested
  from the oldest supported version. A shipped migration's name and content never change (see
  [RELEASING.md](../RELEASING.md#schema-changes-hard-rule)).
- **Export.** Machine-readable outputs above expose everything in the store, and those outputs *are*
  frozen. Analysis and export need no network, server, or license.
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
