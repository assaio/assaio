# Compatibility

What `assaio` promises not to break, and what it deliberately does not promise. This file is
the single answer: [ROADMAP.md](../ROADMAP.md), [RELEASING.md](../RELEASING.md) and
[extending.md](extending.md) link here rather than restating it, because for several releases
they each carried a different version of it and nothing noticed.

`assaio` is pre-1.0. Until `v1.0`, a minor release may break any of it, and the changelog says
so under a **Breaking** heading. What follows is what `v1.0` means.

## Frozen at v1.0

These are the contracts a third party builds against. At `v1.0` each gets a version, a
compatibility test, and a written deprecation path; after it, a breaking change needs a major.

| Contract | Where it lives |
|---|---|
| Exec plugin protocols — parser, metric, rule | [extending.md](extending.md), ADR [0003](adr/0003-exec-plugin-protocol.md) / [0004](adr/0004-exec-metric-plugin-protocol.md) / [0005](adr/0005-exec-rule-plugin-protocol.md) |
| The observation envelope and its payload types | ADR [0007](adr/0007-canonical-event-contract.md) |
| Signal ids and what a zero means for each | ADR [0008](adr/0008-signal-catalog.md), `assaio-agent signals` |
| The team sync protocol | [extending/team-server.md](extending/team-server.md) |
| Machine-readable output — `analyze --format json`, `report --format json\|csv`, `docs export` | `docs/reference.json` |

## Not frozen, and guaranteed instead

**The SQLite schema is an implementation detail.** It is not a public API and will not be
frozen at `v1.0`. Reading the database directly is supported for exploration
([extending/query-your-data.md](extending/query-your-data.md)) and unsupported as an integration.

What is guaranteed in its place:

- **Forward migration** from any released version to any later one, applied automatically and
  tested from the oldest supported version. A shipped migration is immutable in name and
  content (see [RELEASING.md](../RELEASING.md#schema-changes-hard-rule)).
- **Export.** Everything the store holds is reachable through the machine-readable outputs
  above, which *are* frozen. No analysis or export requires a network, a server, or a licence.
- **Backup.** The store is one file; copying it while `assaio` is idle is a complete backup.

The reason is v0.12: correcting a semantic error needed a migration that rewrote stored rows,
and it will not be the last. A frozen store schema would have made that correction the breaking
change instead of the bug. The promise worth making is that history stays *correctable*.

## Deferred, and not a v1 contract

**An in-process Go plugin API.** Exec protocols are the extension boundary: language-neutral,
opt-in, validated at the boundary, and bounded in time and size. A dynamically loaded in-process
API needs a measured performance or deployment need that nobody has demonstrated, and Go's own
`plugin` package cannot deliver it portably. If it ever arrives it will be additive — a second
boundary beside the exec one, never a replacement for it.

**Assaio Cloud.** The managed service is not part of this repository's compatibility surface.
The contracts above are the same ones it will speak; nothing in the open-source binary may come
to require it.
