# 3. Exec plugin protocol for out-of-tree parsers

## Status
Accepted (2026-07-13)

## Context
The core lives under Go's `internal/` on purpose: freezing a public Go API before v1.0
would bind its shape under semver while the data model is still moving (ADR 0001). But
"send a parser PR and wait for a release" is a real adoption cost — teams with an
internal AI tool, or one we have not covered yet, need a way to feed assaio without
touching our tree, in whatever language they already use.

## Decision
Out-of-tree parsers are subprocesses speaking a stdio protocol, not linked code. A
plugin is any executable that, invoked as `<command> scan` with
`ASSAIO_PLUGIN_PROTOCOL=1`, writes a one-line JSON handshake
(`{"assaio_plugin":1,"tool":"<name>"}`) then JSONL usage records to stdout. The plugin
owns discovery and parsing; assaio owns validation, storage, and pricing.

- **The contract is the data format, not a Go API.** The record shape mirrors the
  `usage_record` schema, which we already document as a public surface. Core refactors
  cannot break a plugin; no semver freeze is spent.
- **Any language.** Python, Rust, shell — the reference example in `docs/extending.md`
  is 20 lines of Python.
- **Opt-in only, from config.** Plugins run exclusively when declared under `plugins:`
  in `config.yaml`. No PATH scanning, no auto-discovery, no downloading. The sandboxing
  story stays honest: a plugin is a program the user chose to run, with the user's own
  privileges, stated plainly in `PRIVACY.md`.
- **Boundary validation enforces the honesty rules.** Records with negative tokens, an
  empty dedupe key, a bad timestamp, or an invalid granularity are skipped and counted.
  Stored tool labels are namespaced `plugin:<name>`, so a plugin can never impersonate
  a built-in source and the `(tool, dedupe_key)` keyspace never collides.
- **Failure isolation mirrors ingest.** Timeout (per-plugin, default 60s), a 64 MiB
  stdout cap, stderr passthrough prefixed `[plugin/<name>]`, and a failed plugin counts
  as Failed while the run continues. `plugins verify` is the conformance tool.

Rejected alternatives:
- **Go's `plugin` package** — dead end: Linux/macOS only, exact-toolchain coupling,
  no unloading, effectively frozen upstream.
- **hashicorp/go-plugin or WASM** — real plugin *runtimes* with versioned RPC and (for
  WASM) sandboxing, but they force a public API definition now and add heavy machinery
  a local CLI does not need. Revisit at the team-server stage, where long-lived
  connector processes and stronger isolation earn that cost.

## Consequences
Anyone can extend assaio today without Go and without waiting on us; the exec contract
must now be versioned (the handshake carries the protocol number, currently 1) and kept
backward compatible. Spawning subprocesses per backfill is trivially cheap at this
scale. The Go library API remains unfrozen until the team server, as planned — exec
plugins take the pressure off freezing it early.
