# Write a rule plugin (any language)

*Part of [Extending assaio](../extending.md). Rules read [metrics](metric-validator.md).*

**When to reach for this instead of a metric.** A rule plugin is an executable **gate** that reads
computed verdicts and decides whether a window is acceptable. It runs in `assaio-agent check`; an
`error` alert exits non-zero, failing CI or blocking a push. Use a [metric plugin](metric-plugin.md)
to measure something new. Use a rule when the metric exists and you need your own threshold. Assaio
ships no built-in thresholds because the right number depends on your organization; choosing one
would imply a claim about your team.

Rule plugins are **opt-in only**. Declare them under `rules:` in `~/.config/assaio/config.yaml`;
assaio never finds them through `PATH` or downloads them. Entries have the same shape as `plugins:`
and `metrics:`. One binary can appear in all three lists and serve the `scan`, `analyze`, and
`evaluate` argv protocols:

```yaml
rules:
  - name: budget-drift       # required, [a-z0-9-]+; stamped on every alert it raises
    command: /path/to/assaio-rule-budget      # required; PATH lookup if not absolute
    timeout: 15s             # optional, default 60s
```

A rule sees **less than a metric does**: only the verdict array from `analyze --format json`, with
no usage rows, sessions, or prices. You can print everything that crosses the process boundary with
one command.

## The protocol

`assaio` runs `<command> evaluate` with `ASSAIO_RULE_PROTOCOL=1`, writes one JSON envelope to the
plugin's **stdin**, closes it, and reads stdout.

**stdin** — versioned verdicts for this window:

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

Each entry is one [`Result`](metric-validator.md#what-a-validator-returns-result), in `analyze`
render order, from every registered validator and configured [metric plugin](metric-plugin.md)
(named `plugin:<name>`). Like the metric envelope, this is **versioned but pre-1.0 unstable**.
Releases that change its shape say so explicitly (see `RELEASING.md`).

**stdout** — one handshake line, then exactly **one** JSON alerts document. Pretty-printing is
allowed; any content after the document violates the protocol:

1. `{"assaio_rule": 1, "name": "<name>"}` — version must be `1`; `name` must match the configured
   name.
2. `{"alerts": [...]}` — each alert:

| Field | Required | Meaning |
|-------|----------|---------|
| `rule` | yes | Stable id of the check that fired, e.g. `premium-share`. |
| `severity` | yes | `info`, `warn`, or `error`. **Only `error` fails the gate.** |
| `message` | yes | One line a human reads on a red build. |
| `validator` | no | The verdict this alert is about, echoed back for the reader. |

An empty `alerts` array passes. Stderr passes through with the prefix `[rule/<name>] `.

## What the boundary enforces

Assaio rejects a document that fails **any** check. It never applies some alerts and drops others
silently, which could weaken a gate you expect to run.

- `severity` must be exactly `info`, `warn`, or `error` (lowercase).
- `rule` (≤ 64 characters) and `message` (≤ 400) are required; `validator` (≤ 64) is optional.
- Each plugin may emit at most 50 alerts, with no control characters anywhere (to guard terminal
  escapes).
- Unknown JSON fields are rejected so misspelled `alerts` or `severity` keys fail instead of
  silently disabling a gate.
- Stdout is limited to 1 MiB; the run is killed after `timeout` (60s by default).
- Assaio stamps the emitting plugin's name on each alert, so alerts remain attributable and plugins
  cannot impersonate each other.

**Failure is fail-closed.** A bad handshake, timeout, non-zero exit, or contract violation is
reported on stderr and fails `check`. Other rules still run. A gate that did not run has not passed.

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
