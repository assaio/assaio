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
metric plugin is any executable that answers two verbs with `ASSAIO_METRIC_PROTOCOL=1` in
its environment, each writing a one-line handshake
(`{"assaio_metric":<version>,"name":"<name>"}` — `1` when this ADR was written, `4` since
v0.25) followed by exactly one JSON document:

- `<command> describe` — nothing on stdin; **one Declaration** naming what this metric
  reads: `needs` (capabilities, from `internal/analyze`'s closed set), and optionally
  `fields` (which columns of a section) and `where` (which rows, by column value).
- `<command> analyze` — **one Input envelope** on stdin, versioned as
  `assaio_metric_input: 4`, carrying exactly what the declaration asked for and nothing
  else; **one `Result`** on stdout, in the shape `analyze --format json` emits.

- **The contract is the data format, not a Go API** — both sides of it are shapes
  assaio already documents as public surfaces (the store aggregation and the analyze
  JSON output). Field names are camelCase to mirror the Result JSON; only the version
  keys stay snake_case, matching `assaio_plugin`.
- **Declared under `metrics:` in config**, same `{name, command, timeout}` entry shape
  as `plugins:`, same opt-in-only rules (no PATH scanning, no downloading). One binary
  may appear in both lists and serve both protocols.
- **The envelope is a projection, not a dump** (v0.25). A plugin declares the sections it
  reads, the columns inside them, and the rows a predicate admits; the core sends that.
  Grain is not a separate vocabulary: in this store the grain *is* a column —
  `usage.granularity` picks turn or session rows, `trace.scope` picks a population — so a
  predicate over a string column expresses both. Predicates address a top-level row only:
  dropping steps from inside a sequence would leave that sequence's ordinals and step count
  describing a set nobody declared.
- **What the plugin did not ask for is absent, and what it asked for and did not get is
  named.** Three states a plugin must be able to tell apart, and one empty array is none of
  them: a section outside `projection.needs` was never requested; a capability in
  `withheld` was requested and the local config denied it; a section present but empty is a
  window that holds nothing. Every predicate reports `projection.rows[section] = {sent,
  available}` — without the second number, pushdown would be a new way to publish "all of
  them are X" over a set the plugin itself chose.
- **The declaration lives in the handshake; config constrains it** (v0.25). It was in
  `config.yaml` from v0.24 (`needs:`, for the step timeline), which put the requirement on
  the person pasting a config entry rather than on the author who knows it: omit the line
  and the plugin got an empty timeline with a withheld annotation, diagnosable only from the
  plugin's own documentation. `needs:` stays as the reader's **veto** — this is a
  data-disclosure boundary and the disclosure decision is theirs — with explicit
  intersection semantics: the run carries `declared ∩ allowed`; an empty `needs:` is *no
  constraint* rather than an empty grant, because almost no entry carries the key; allowing
  more than the plugin declared grants nothing extra; and allowing less names the difference
  in `withheld` and adds a caveat to the rendered verdict.
- **The envelope carries capability, not only counts** (`answers`, added in v0.11). Every
  activity column on a row is zero for a source that does not record it, so a plugin without
  the depth matrix cannot tell a measurement from a silence — which made every out-of-tree
  metric structurally exposed to the failure [ADR 0011](0011-capability-gated-metrics.md)
  fixed in-tree, with no way to ask. Only the tools present in the window are sent: a plugin
  needs the capability of the data it was handed, and shipping the whole matrix would make
  the envelope a second publication of `internal/parser`. Removing this field is removing a
  plugin author's only way to be honest, not trimming a redundant one.
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
- **The team server never executes metric plugins.** `GET /` rebuilds the dashboard per
  request; spawning config-declared subprocesses per request would be a denial-of-service
  vector. (The route was also unauthenticated when this was written; it is not since v0.24, and
  the reasoning above never depended on that.) Exec metrics run in `analyze`, `dashboard`, and
  `metrics verify`; compiled-in validators still cover the served dashboard. The
  drill-down section likewise re-runs built-ins only, and `demo` stays deterministic.
- **Pre-1.0 instability is explicit.** The envelope and result are versioned; a release
  that changes either must say so (RELEASING.md), the same stance as the SQLite schema.
- **The contract is published as vectors, not only as prose** (v0.25). `docs/conformance/`
  holds every document this boundary accepts and refuses — declaration, result, alerts, and
  the ADR 0003 parser record — each with its verdict and the reason for it, so a plugin can
  have its own CI without the assaio binary. The same files drive assaio's own tests and
  seed its fuzzers: a vector nobody runs is a vector that lies.

Rejected alternatives:
- **Paging the envelope.** A metric emits one verdict over one window, so paged input would
  require either merging verdicts across pages — which no metric can do honestly, since a
  rate over page 3 is not a rate — or holding every page anyway, which is the memory the
  paging was for. Projection and pushdown buy the same bound with a contract a plugin can
  state and a reader can check, and `projection.rows` keeps the denominator honest. If a
  section ever outgrows that, the answer is a streaming verb of its own, not a page number
  bolted onto a document protocol.
- **Negotiating on one pipe** — request, answer, envelope, result, all through one process.
  It halves the process count and costs every plugin author a flushing bug; the shell and
  Python skeletons this ships with would each get it wrong differently. Two one-shot runs
  cost one extra spawn per metric per `analyze`; the envelope they size is measured in
  megabytes.
- **Registering plugins as `analyze.Validator` adapters** — uniform flow (drill,
  server) but breaks the documented purity of `Analyze(Input)`, smuggles a context into
  a struct field, and would put subprocess execution behind the server's
  served GET.
- **Embedded scripting (Starlark/WASM)** — sandboxing for free, but a heavy new
  dependency, a new language forced on metric authors, and a second extension contract
  alien to the ADR 0003 model already shipped.

## Consequences
- A company writes one Python file, points `metrics:` at it, and its metric renders in
  `analyze`, `analyze --format json`, and the Assay dashboard (pseudonymization
  included, via `barsPseudonym`) — no fork, no rebuild, no release wait.
- The prepared-`Input` JSON becomes a versioned public surface; widening it is cheap,
  reshaping it is a breaking protocol change to be called out in release notes.
- The in-tree validator path (docs/extending.md) remains the route for upstreamable
  metrics; the dynamically loaded in-process Go API stays roadmap.
- **The envelope went from a dump to a projection, and the saving is measured, not asserted**
  (v0.25). On the maintainer's real store over a 30-day window — 522 usage rows, 3,376
  sessions, 4,279 step sequences, 424,310 steps — serializing the same window under four
  declarations:

  | What the plugin gets | Bytes |
  |---|---|
  | protocol 3, no `needs:` line (everything except `trace`) | 1,237,872 (1.18 MB) |
  | protocol 3, `needs: [trace]` (everything) | 55,864,147 (53.28 MB) |
  | protocol 4, a token-share metric: `needs: [usage]`, four columns | 43,779 (0.04 MB) |
  | protocol 4, a step detector: `needs: [trace]`, three sequence and two step columns, `trace.scope = interactive` | 7,531,004 (7.18 MB) |

  28× for the ordinary metric, 7.4× for the one that reads the timeline, and the second is
  the case that mattered: 53 MB per plugin per `analyze` is a wall a team-year walks into,
  not an optimization. Widening the window to 365 days moves the projected figures to
  72,368 and 7,547,221 bytes, because the step retention horizon — not the window — is what
  bounds the timeline. Reproduce with `ASSAIO_MEASURE_DB=<store> go test ./internal/plugin/
  -run TestEnvelopeSizeOnARealStore -v`, which copies the named store before opening it and
  sizes the same envelope the runtime builds.
- **A protocol-3 plugin fails loudly and migrates in two edits.** It has no `describe` verb,
  so the run fails on the handshake naming the version rather than on a figure computed over
  sections that were not there. The migration is: bump the handshake integer to `4`, and add
  a `describe` branch printing the same handshake plus `{"needs":[…]}` naming every
  capability it reads — `{"needs":["usage","sessions","trace","attribution","turn-sizing",
  "cache-misses","prices"]}` reproduces protocol 3's payload exactly. Narrowing from there is
  optional and is where the bytes are. A `needs:` line already in `config.yaml` keeps
  working and now reads as a veto: it constrains that declaration instead of extending it.
- **The envelope carries every `analyze.Input` field, and that is now a test rather than an
  intention** (v0.17). It was not true when this ADR was written: six fields — `windowStart`,
  `planMonthlyCost`, `skills`, `agents`, `turnSizing`, `cacheMisses` — never crossed the
  boundary, while five shipped validators read them, so "the same prepared `Input` bundle
  every built-in validator reads" was a claim the wire did not honour and an out-of-tree
  author could not reproduce a third of what the core published. A reflective canary now fails
  the build when a new `Input` field reaches neither the envelope nor a listed exception with
  its reason; the only standing exceptions are `recent` (sent as `recentDays`) and
  `ingested`/`parsedBy` (which the core stamps onto the plugin's own `Result`). An extension
  surface weaker than the core it extends is a demonstration, not a contract.
