# 5. Exec rule plugin protocol for out-of-tree gates

## Status
Accepted (2026-07-26)

## Context
ADR 0003 opened **parsers** and ADR 0004 opened **metrics**; the third unit the product
has always described — a **rule** that turns verdicts into a gate — had no protocol
(`B13`, ROADMAP "Ecosystem & extensibility"). `check` could gate on a token or
API-equivalent cost budget and nothing else, so "fail CI when adoption stalls" or "warn
when premium-model share climbs" meant re-deriving a verdict assaio had already computed,
in a shell script parsing `analyze --format json`. The thresholds that matter are
organizational, not universal: they belong to the team, not in our tree.

## Decision
Out-of-tree rules are subprocesses speaking a stdio protocol, the third instance of the
ADR 0003/0004 shape. A rule plugin is any executable that, invoked as `<command>
evaluate` with `ASSAIO_RULE_PROTOCOL=1`, reads one JSON envelope on stdin — the window's
validator verdicts, versioned as `assaio_rule_input: 1` — and writes to stdout a one-line
handshake (`{"assaio_rule":1,"name":"<name>"}`) followed by exactly one JSON document,
`{"alerts":[{"rule","severity","message","validator"?}]}`.

- **A rule reads verdicts, never usage.** The envelope carries exactly the `Result` array
  `analyze --format json` already prints — no rows, no sessions, no prices. A rule plugin
  therefore sees strictly less than a metric plugin does, which is why it needs no privacy
  note of its own: the surface is a report the user can already print.
- **Declared under `rules:` in config**, same `{name, command, timeout}` entry shape as
  `plugins:` and `metrics:`, same opt-in-only rules (no PATH scanning, no downloading).
  One binary may appear in all three lists and serve all three protocols.
- **Reject, never fabricate.** The boundary whitelists `severity` (`info|warn|error`),
  requires `rule` and `message`, caps the alert count and every string length, refuses
  control characters, and rejects unknown JSON keys — a misspelled `alerts` key must fail
  loudly rather than silently disarm a gate. A violating document is dropped whole, with
  the first reason folded into the error.
- **Stamped provenance.** Each accepted alert carries the configured plugin name, set at
  the boundary and never read off the wire, so an alert is always attributable — the same
  rule that namespaces parser tools and metric verdicts as `plugin:<name>`.
- **`check` is the only host, and it fails closed.** An `error` alert exits non-zero;
  `warn` and `info` are printed and pass. A rule that could not be evaluated — bad
  handshake, timeout, non-zero exit, contract violation — is warned about on stderr and
  *also* fails the run: a gate that did not run is not a gate that passed. Rules run over
  the built-in verdicts plus configured metric plugins, so a company can gate on its own
  exec metric.
- **Not in `analyze`, not on the server.** `analyze` stays a per-validator report and its
  `--format json` array stays exactly the metric-plugin `Result` contract; folding alerts
  in would either reshape that public surface or print in text what JSON omits. The team
  server executes no config-declared subprocess per served request, as in ADR
  0004.
- **Pre-1.0 instability is explicit.** Envelope and alerts are versioned; a release that
  changes either must say so (RELEASING.md), the same stance as the SQLite schema.

Rejected alternatives:
- **Thresholds as config (`rules: - metric: adoption, max: 0.4`)** — no subprocess, but
  it invents a rule DSL that must then grow comparators, windows, and boolean logic, and
  it can only ever express what the schema anticipated. The exec contract expresses any
  condition in any language today.
- **In-tree rule validators** — every threshold in this repo would be a claim that our
  number is the right one for every team, which the honesty rules forbid.
- **Reusing the metric protocol** (a metric plugin whose `read.key` gates) — conflates a
  measurement with a policy, gives every gate a dashboard cell it does not want, and
  would hand raw usage to something that only needs verdicts.

## Consequences
- A team writes one script, points `rules:` at it, and `assaio-agent check` becomes their
  own gate in CI or a pre-push hook — no fork, no rebuild, no threshold of ours baked in.
- `check` now depends on the verdict pipeline (and, when configured, on metric plugins)
  instead of only on the usage rollup — but only when `rules:` is non-empty; with no rules
  configured its cost and output are unchanged.
- Fail-closed means a broken rule reddens CI for a non-usage reason. That is the intended
  trade: the alternative is a green build that proves nothing.
- The verdict array becomes a versioned public surface in a second place (after `analyze
  --format json`), which is an argument for freezing it earlier rather than later (`B23`).
