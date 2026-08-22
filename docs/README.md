# Documentation map

Where everything lives, in one place.

## Using assaio

- [`README.md`](../README.md) — install, quick start, the command table, supported
  tools and accuracy caveats.
- [`FEATURES.md`](../FEATURES.md) — the maintained inventory of what exists today,
  with the release each capability arrived in.
- [`extending.md`](extending.md) — the map of every extension surface and the honesty rules
  that bind all of them, with one page per surface under [`extending/`](extending): the
  [in-tree metric validator](extending/metric-validator.md) and its
  [worked example](extending/metric-validator-example.md),
  [exec parser plugins and custom log paths](extending/parser-plugin.md),
  [exec metric plugins](extending/metric-plugin.md), [exec rule plugins](extending/rule-plugin.md),
  [the team server](extending/team-server.md), [direct SQL on the store](extending/query-your-data.md),
  [adding an in-tree parser](extending/data-source.md), and
  [what each source's log carries](extending/source-fields.md).
- [`reference.json`](reference.json) — generated: every signal, source, validator, command,
  flag, config key and protocol field this build has, from its own registries. `make docs`
  rewrites it and `make test` fails when it and the binary disagree, which is also what keeps
  [assaio.dev/docs/reference](https://assaio.dev/docs/reference) from falling behind.
- [`reconcile.md`](reconcile.md) — checking the `$` estimate against a vendor's own
  billing or usage export, offline: how columns bind, how to read the unexplained delta,
  and what no export can answer.
- [`config.example.yaml`](../config.example.yaml) — every config key, documented, with
  defaults.
- [`PRIVACY.md`](../PRIVACY.md) — exactly what is read, extracted, stored, and never
  touched.

## Project direction

- [`ROADMAP.md`](../ROADMAP.md) — the ordered direction: each milestone's exit criteria, the
  gates that have not been met, and what `v1.0` has to mean. Direction, not commitment.
- [`compatibility.md`](compatibility.md) — what `v1.0` freezes, what it deliberately does not,
  and what is deferred. The single answer the roadmap and the release guide link to.
- [`BACKLOG.md`](../BACKLOG.md) — the ranked pool of concrete candidate items
  (`B01`–…): proposals and effort estimates, not commitments.
- [`CHANGELOG.md`](../CHANGELOG.md) — what actually shipped, per release
  (Keep a Changelog format).

Lifecycle: an item graduates BACKLOG → CHANGELOG `[Unreleased]` → a row in FEATURES,
all in the shipping PR.

## Contributing and policies

- [`CONTRIBUTING.md`](../CONTRIBUTING.md) — the authoritative rules: code style, the
  local gate, git workflow, honesty rules.
- [`GOVERNANCE.md`](../GOVERNANCE.md) — how decisions are made.
- [`CODE_OF_CONDUCT.md`](../CODE_OF_CONDUCT.md) — Contributor Covenant 2.1.
- [`SECURITY.md`](../SECURITY.md) — private vulnerability disclosure.
- [`RELEASING.md`](../RELEASING.md) — maintainers only: versioning, the immutable-
  migrations rule, cutting a release.
- [`format-resilience.md`](format-resilience.md) — how vendor log-format drift is
  detected and fixed: current defenses, known gaps, and the report → fixture → patch
  release loop (`format-drift` label).

## Architecture Decision Records

- [ADR 0001](adr/0001-language-and-architecture.md) — Go, one static binary, the
  `internal/` core and dependency direction.
- [ADR 0002](adr/0002-code-standards-and-enforcement.md) — code standards: what
  tooling gates vs what review holds.
- [ADR 0003](adr/0003-exec-plugin-protocol.md) — exec plugin protocol for out-of-tree
  **parsers** (subprocess, handshake + JSONL).
- [ADR 0004](adr/0004-exec-metric-plugin-protocol.md) — exec plugin protocol for
  out-of-tree **metrics** (Input on stdin → one Result), and why the team server never
  executes them.
- [ADR 0005](adr/0005-exec-rule-plugin-protocol.md) — exec plugin protocol for
  out-of-tree **rules** (verdicts on stdin → severity alerts), why `check` is their only
  host, and why the gate fails closed.
- [ADR 0006](adr/0006-session-annotations.md) — session annotations are **closed-vocabulary
  labels**, never free text, and they never leave the machine.
- [ADR 0007](adr/0007-canonical-event-contract.md) — the canonical **event contract** the
  evidence graph is built on.
- [ADR 0008](adr/0008-signal-catalog.md) — the **signal catalog**: what assaio can report,
  what a zero means, and which sources can answer each one.
- [ADR 0009](adr/0009-local-git-evidence-collector.md) — the local **git evidence
  collector** observes commits and stores none of their content.
- [ADR 0010](adr/0010-attribution-conformance-corpus.md) — the **attribution conformance
  corpus** defines an honest session→commit link before an engine exists.
- [ADR 0011](adr/0011-capability-gated-metrics.md) — a metric reads only the sources that
  record its field, so a structural silence never averages in as a zero.
- [ADR 0012](adr/0012-session-step-timeline.md) — the **session step timeline**: what a
  session did in what order, why it carries an integer rather than a path, and why its
  retention horizon is load-bearing.
- [ADR 0013](adr/0013-measurement-layers.md) — every figure states its **measurement
  layer**, so an output figure can never be read as an outcome claim.
- [ADR 0014](adr/0014-public-artifact-rules.md) — what a **publicly postable** artifact may
  say: no figure originates in the renderer, redaction is structural rather than a flag, and
  every frame carries its own limits because a shared image outlives its caption.
- [ADR 0015](adr/0015-structured-recommendations.md) — advice is a **typed record**, not a
  sentence: evidence, risk, rollback, review window and the follow-up figure are required, the
  rendered text projects the record, and a thin window abstains rather than suggesting.
