# 4. Exec metric plugin protocol for out-of-tree analyzers

## Status
Accepted (2026-07-17)

## Context
ADR 0003 gave out-of-tree **parsers** a language-agnostic path into assaio; metrics had
no equivalent. A custom analyzer required forking the repo and rebuilding the binary,
because `internal/analyze` is compiler-enforced unimportable — the right call for the Go
API (ADR 0001), but a real adoption cost for a company that wants one private metric on
its own dashboard. The product promise is a framework whose analyzers are extensible
the same way its data sources are: out-of-tree, any language, selected in config.

## Decision
Out-of-tree metrics are subprocesses speaking a stdio protocol, mirroring ADR 0003. A
metric plugin is any executable that, invoked as `<command> analyze` with
`ASSAIO_METRIC_PROTOCOL=1`, reads one JSON envelope on stdin — the same prepared
`Input` bundle every built-in validator reads (usage, sessions, delegation, byModel,
byProject, totals, prices), versioned as `assaio_metric_input: 1` — and writes to
stdout a one-line handshake (`{"assaio_metric":1,"name":"<name>"}`) followed by exactly
one JSON `Result` document, in the same shape `analyze --format json` emits.

- **The contract is the data format, not a Go API** — both sides of it are shapes
  assaio already documents as public surfaces (the store aggregation and the analyze
  JSON output). Field names are camelCase to mirror the Result JSON; only the version
  keys stay snake_case, matching `assaio_plugin`.
- **Declared under `metrics:` in config**, same `{name, command, timeout}` entry shape
  as `plugins:`, same opt-in-only rules (no PATH scanning, no downloading). One binary
  may appear in both lists and serve both protocols.
- **Reject, never fabricate.** The boundary whitelists `read.key`, requires the prose
  fields, caps counts and lengths, refuses control characters, and clamps
  `purity`/`frac`. A violating result is dropped whole with a warning — assaio never
  renders a fabricated or partially-sanitized verdict. `metrics verify` reports every
  violation with its reason.
- **Namespaced verdicts.** `Result.Name` is always stamped `plugin:<name>` at the
  boundary, so a plugin cannot shadow or impersonate a built-in validator — the same
  rule that namespaces parser plugins' stored tool labels.
- **Orchestrated beside the registry, not inside it.** Plugin results are computed by
  the CLI next to the in-process validators (the `ingestPlugins` precedent) and
  appended to the same rendering path. `Validator.Analyze` stays a pure function of
  `Input`; subprocess lifecycle, context, and timeouts stay at the driver layer.
- **The team server never executes metric plugins.** `GET /` is unauthenticated and
  rebuilds the dashboard per request; spawning config-declared subprocesses per request
  would be a denial-of-service vector. Exec metrics run in `analyze`, `dashboard`, and
  `metrics verify`; compiled-in validators still cover the served dashboard. The
  drill-down section likewise re-runs built-ins only, and `demo` stays deterministic.
- **Pre-1.0 instability is explicit.** The envelope and result are versioned; a release
  that changes either must say so (RELEASING.md), the same stance as the SQLite schema.

Rejected alternatives:
- **Registering plugins as `analyze.Validator` adapters** — uniform flow (drill,
  server) but breaks the documented purity of `Analyze(Input)`, smuggles a context into
  a struct field, and would put subprocess execution behind the server's
  unauthenticated GET.
- **Embedded scripting (Starlark/WASM)** — sandboxing for free, but a heavy new
  dependency, a new language forced on metric authors, and a second extension contract
  alien to the ADR 0003 model already shipped.

## Consequences
- A company writes one Python file, points `metrics:` at it, and its metric renders in
  `analyze`, `analyze --format json`, and the Assay dashboard (pseudonymization
  included, via `barsAreProjects`) — no fork, no rebuild, no release wait.
- The prepared-`Input` JSON becomes a versioned public surface; widening it is cheap,
  reshaping it is a breaking protocol change to be called out in release notes.
- The in-tree validator path (docs/extending.md) remains the route for upstreamable
  metrics and for signals the wire envelope does not carry; the dynamically loaded
  in-process Go API stays roadmap.
