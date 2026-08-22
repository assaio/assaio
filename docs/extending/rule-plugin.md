# Write a rule plugin (any language)

*Part of [Extending assaio](../extending.md). Rules read what [metrics](metric-validator.md) produce.*

**When to reach for this instead of a metric.** A rule plugin is your own **gate**: an
executable that reads the verdicts assaio just computed and answers one question — is
this window acceptable? It runs inside `assaio-agent check`, so an `error` alert exits
non-zero and reddens CI or blocks a push. Reach for a [metric
plugin](metric-plugin.md) when you want to *measure* something new;
reach for a rule when the measurement exists and what you need is *your* threshold on it.
Thresholds are exactly what assaio refuses to ship built-in — the right number is
organizational, and a number we picked would be a claim about your team we cannot honestly
make.

Rule plugins are **opt-in only**, declared under `rules:` in
`~/.config/assaio/config.yaml` — never discovered from `PATH`, never downloaded. The
entry shape is the same as `plugins:` and `metrics:`, and one binary may appear in all
three lists, serving all three protocols (`scan`, `analyze`, and `evaluate` argv):

```yaml
rules:
  - name: budget-drift       # required, [a-z0-9-]+; stamped on every alert it raises
    command: /path/to/assaio-rule-budget      # required; PATH lookup if not absolute
    timeout: 15s             # optional, default 60s
```

A rule sees **less than a metric does**: no usage rows, no sessions, no prices — only the
verdict array `analyze --format json` already prints. Nothing crosses the process boundary
that you could not print yourself with one command.

## The protocol

`assaio` invokes `<command> evaluate` with `ASSAIO_RULE_PROTOCOL=1` in the environment,
writes one JSON envelope to the plugin's **stdin**, closes it, and reads stdout.

**stdin** — this window's verdicts, versioned:

```json
{
  "assaio_rule_input": 1,
  "verdicts": [
    {"name":"adoption","title":"Adoption","describe":"...","read":{"key":"watch","label":"WATCH"},
     "purity":0.42,"howToRead":"...","figures":[{"label":"AI lines","value":"1,204"}],
     "bars":[{"label":"web","value":"800","frac":1}],"takeaway":"...","caveats":["..."]}
  ]
}
```

Each entry is one [`Result`](metric-validator.md#what-a-validator-returns-result) — every registered
validator, plus every configured [metric plugin](metric-plugin.md)
(named `plugin:<name>`), in the order `analyze` renders them. Like the metric envelope, it
is **versioned but pre-1.0 unstable** — a release that reshapes it says so explicitly (see
`RELEASING.md`).

**stdout** — a one-line handshake, then exactly **one** JSON alerts document
(pretty-printed is fine; anything after it is a violation):

1. `{"assaio_rule": 1, "name": "<name>"}` — version must be `1`, `name` must equal the
   configured name.
2. `{"alerts": [...]}`, each alert:

| Field | Required | Meaning |
|-------|----------|---------|
| `rule` | yes | Stable id of the check that fired, e.g. `premium-share`. |
| `severity` | yes | `info`, `warn`, or `error`. **Only `error` fails the gate.** |
| `message` | yes | One line a human reads on a red build. |
| `validator` | no | The verdict this alert is about, echoed back for the reader. |

An empty `alerts` array is a normal, passing answer. Anything written to stderr passes
through prefixed `[rule/<name>] `.

## What the boundary enforces

A document that fails **any** check is rejected whole — assaio never applies a
partially-sanitized alert set, because a silently dropped alert would weaken a gate you
believe is running.

- `severity` must be exactly `info`, `warn`, or `error` (lower-case).
- `rule` (≤ 64 chars) and `message` (≤ 400) are required; `validator` (≤ 64) is optional.
- Max 50 alerts per plugin; no control characters anywhere (terminal-escape guard).
- Unknown JSON fields are rejected: a misspelled `alerts` or `severity` key must fail
  loudly rather than quietly disarm the gate.
- Stdout is capped at 1 MiB; the run is killed after `timeout` (default 60s).
- The emitting plugin's name is stamped onto every alert at the boundary, so an alert is
  always attributable and a plugin cannot claim to be another.

**Failure is fail-closed.** A rule that could not be evaluated — bad handshake, timeout,
non-zero exit, contract violation — is reported on stderr *and* fails `check`, while the
remaining rules still run. A gate that did not run is not a gate that passed.

## A complete example (Python)

```python
#!/usr/bin/env python3
"""assaio-rule-premium: gate on how much of the window runs on premium models."""
import json, sys

verdicts = {v["name"]: v for v in json.load(sys.stdin)["verdicts"]}
print(json.dumps({"assaio_rule": 1, "name": "premium"}))

fit = verdicts.get("model-fit")
if fit is None:
    print(json.dumps({"alerts": [{"rule": "model-fit-missing", "severity": "warn",
                                  "message": "model-fit did not report this window."}]}))
    sys.exit(0)

alerts = []
if fit["read"]["key"] == "watch":
    alerts.append({"rule": "premium-share", "severity": "error", "validator": "model-fit",
                   "message": fit["takeaway"][:400]})
print(json.dumps({"alerts": alerts}))
```

Make it executable, declare it under `rules:` as above, and run the gate:

```console
$ assaio-agent check --since 30d
budget check · last 30d
  total tokens: 4812004
  total cost:   $61.20 (API-equivalent estimate)

  no budget set -- pass --max-tokens or --max-cost to gate.

rules
  [error] premium/premium-share: Nearly all tokens run on premium models -- consider
  delegating routine work to cheaper models or sub-agents. (model-fit)

Cost is an estimate at public pay-as-you-go API prices -- not your actual spend; ...
error: rule gate failed: premium/premium-share
$ echo $?
1
```

**Where it deliberately does not run:** `analyze` (a per-validator report, and its
`--format json` array is the metric-plugin contract — alerts would either reshape that
public surface or print in text what JSON omits), the dashboard, and the team server
(`GET /` rebuilds per request, same reasoning as metric plugins). `check` is the gate, and the
gate is where rules live. See [ADR 0005](../adr/0005-exec-rule-plugin-protocol.md) for the
full rationale.

---
