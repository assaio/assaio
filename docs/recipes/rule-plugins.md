# Rule plugins you can run today

*Part of [Extending assaio](../extending.md). See the contract in [Write a rule
plugin](../extending/rule-plugin.md).*

A rule plugin reads a window's verdicts and decides whether to stop a build. It is the only
extension that can fail `check` and the place for company-specific limits. assaio sets no thresholds
because policy depends on the team.

Each complete plugin below is **executed** by `TestRecipeRulePlugins` on a fixture window, with its
alerts checked. assaio's config loader reads the config block. The one `check` invocation is checked
against the command tree; the surrounding shell is not run.

## Wiring one up

```yaml #rule-config
rules:
  - name: house-rules
    command: ./scripts/house-rules.py
    timeout: 30s
```

Then `assaio-agent check --since 7d` runs the rule after the budget. An `error` fails the gate;
`warn` and `info` print and pass. A rule that cannot be evaluated also fails, so missing evidence
cannot count as approval.

## Fail when a verdict is withheld for lack of data

Start with a rule for missing answers, without thresholds. A verdict withheld for lack of data is
not a pass; treating it as one lets a pipeline stop measuring unnoticed.

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

Start at `warn`. Switch to `error` once you trust the window's completeness; otherwise, a sparse
first week can fail every build and get the rule removed.

## Fail when a named read turns bad

Select verdicts by name, then act on their `read.key`. Names are in [the
reference](https://assaio.dev/docs/reference#validators); every validator uses `watch` for "worth a
closer look".

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

**Read a verdict, not a figure.** Do not parse `figures[].value` from rendered strings such as
`"1,204"`, `"12.4%"`, or `"—"`; a thousands separator can break the rule. `read.key` is the stable
contract. An em dash means "not computable", which the first recipe handles.

## A rule that says why it could not decide

Plan for a missing figure in the window. Report that the rule cannot evaluate it instead of silently
passing; a gate must say when it has no answer.

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

There is no `rules verify`. The gate runs rule plugins. Run `check` on a real window and inspect its
output:

```sh #verify-rule
assaio-agent check --since 7d
```

The host passes through plugin stderr with a `[rule/<name>] ` prefix. Use that channel while writing
a plugin.
