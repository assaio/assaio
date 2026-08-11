# Rule plugins you can run today

*Part of [Extending assaio](../extending.md). The contract these implement: [Write a rule plugin](../extending/rule-plugin.md).*

A rule plugin reads the window's verdicts and decides whether any of them should stop a build.
It is the only extension point that can fail `check`, and the only one where "what my company
considers unacceptable" belongs — assaio ships thresholds for nothing, because a threshold is a
policy and policies are local.

Each plugin below is complete. Every one is executed by `TestRecipeRulePlugins` against a
fixture window and asserted on: they are run, not merely printed.

## Wiring one up

```yaml #rule-config
rules:
  - name: house-rules
    command: ./scripts/house-rules.py
    timeout: 30s
```

Then `assaio-agent check --since 7d` runs it after the budget. Only an `error` fails the gate;
`warn` and `info` are printed and pass. A rule that cannot be evaluated fails the gate too —
the check refuses to read silence as approval.

## Fail when a verdict is withheld for lack of data

The most useful first rule has nothing to do with thresholds: it catches the case where assaio
could not answer at all. A verdict withheld for want of data is not a green light, and a
pipeline that treats it as one has stopped measuring without noticing.

```python #withheld
#!/usr/bin/env python3
"""Fail when a metric could not be computed: no data is not the same as no problem."""
import json, sys

doc = json.load(sys.stdin)
alerts = []
for v in doc.get("verdicts", []):
    label = (v.get("confidence") or {}).get("label")
    if label == "insufficient":
        alerts.append({
            "rule": "withheld-verdict",
            "severity": "warn",
            "validator": v["name"],
            "message": f'{v["title"]} could not be computed from this window -- '
                       f'{v.get("takeaway", "no reason given")}',
        })

print(json.dumps({"assaio_rule": 1, "name": "house-rules"}))
print(json.dumps({"alerts": alerts}))
```

Start it at `warn`. Move it to `error` once the window it runs on is one you trust to be
complete — otherwise the first sparse week fails every build and the rule gets deleted.

## Fail when a named read turns bad

The general shape: pick the verdicts you care about by name, and act on the `read.key` assaio
already computed. Names come from
[the reference](https://assaio.dev/docs/reference#validators); `watch` is the key every
validator uses for "worth a closer look".

```python #named-read
#!/usr/bin/env python3
"""Fail the build when a chosen read goes to `watch`; ignore the rest."""
import json, sys

WATCHED = {"rework": "error", "context": "warn"}

doc = json.load(sys.stdin)
alerts = [
    {
        "rule": f'{v["name"]}-watch',
        "severity": WATCHED[v["name"]],
        "validator": v["name"],
        "message": f'{v["title"]}: {v.get("takeaway", "flagged")}',
    }
    for v in doc.get("verdicts", [])
    if v["name"] in WATCHED and (v.get("read") or {}).get("key") == "watch"
]

print(json.dumps({"assaio_rule": 1, "name": "house-rules"}))
print(json.dumps({"alerts": alerts}))
```

**Read a verdict, not a figure.** Parsing `figures[].value` out of a rendered string — `"1,204"`,
`"12.4%"`, `"—"` — is how a rule starts failing on a thousands separator. The `read.key` is the
part of the contract that stays stable, and an em dash is a real answer meaning "not computable",
which is what the first recipe is for.

## A rule that says why it could not decide

The failure mode worth designing for: your rule depends on a figure this window does not carry.
Saying so out loud is better than silently passing, and it is what separates a gate from
decoration.

```python #cannot-decide
#!/usr/bin/env python3
"""Judge premium-model share, and say plainly when the window cannot answer."""
import json, sys

doc = json.load(sys.stdin)
fit = next((v for v in doc.get("verdicts", []) if v["name"] == "model-fit"), None)

if fit is None:
    alerts = [{
        "rule": "premium-share",
        "severity": "warn",
        "message": "model-fit did not run in this window, so premium share was not judged",
    }]
else:
    alerts = [] if (fit.get("read") or {}).get("key") != "watch" else [{
        "rule": "premium-share",
        "severity": "error",
        "validator": "model-fit",
        "message": f'model fit is flagged: {fit.get("takeaway", "")}',
    }]

print(json.dumps({"assaio_rule": 1, "name": "house-rules"}))
print(json.dumps({"alerts": alerts}))
```

## Checking one before you trust it

There is no `rules verify` — a rule plugin is exercised by the gate that hosts it. Run `check`
against a real window and read what it prints:

```sh #verify-rule
assaio-agent check --since 7d
```

Anything the plugin writes to stderr is passed through prefixed `[rule/<name>] `, which is the
channel to use while writing one.
