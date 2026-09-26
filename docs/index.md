# assaio documentation

These guides are grouped by what you want to do. Each page is rendered from the repository's
Markdown or generated from the binary; a build check fails if a published page disagrees with it.

## Understand the figures

Learn how assaio links sessions to commits, compares vendor numbers, handles log changes, and
reads each source.

- [Local session-to-commit evidence](evidence.md)
- [Reconciling against the vendor's own numbers](reconcile.md)
- [Format resilience — detecting and reacting to vendor log-format drift](format-resilience.md)
- [What each source's log carries, and what assaio reads](extending/source-fields.md)
- [Runtime inspect (experimental)](runtime-inspect.md)

## Run it for a team

Set up the self-hosted team server, push usage with sync, and automate runs with hooks or a
schedule.

- [The team server](extending/team-server.md)
- [Automating assaio — hooks, scheduled sync, and survival](automation.md)

## Extend assaio

Find extension points for metrics, rules, parsers and data sources, query the documented schema
with SQL, and check compatibility.

- [Extending assaio](extending.md)
- [Adding a metric validator](extending/metric-validator.md)
- [Worked example: Weekend Usage](extending/metric-validator-example.md)
- [Write a metric plugin (any language)](extending/metric-plugin.md)
- [Write a rule plugin (any language)](extending/rule-plugin.md)
- [Reading a source assaio does not ship](extending/parser-plugin.md)
- [Add a data source](extending/data-source.md)
- [Query your own data](extending/query-your-data.md)
- [Compatibility](compatibility.md)

## Use an example

Copy examples for rules, CI gates and extensions; the test suite checks the recipes.

- [Label rules you can paste in](recipes/label-rules.md)
- [Gating CI on what a window cost](recipes/ci-gates.md)
- [Rule plugins you can run today](recipes/rule-plugins.md)
- [Running it without being asked](recipes/automation.md)
- [Extensions, written out in full](recipes/extensions.md)

See [the generated reference](https://assaio.dev/docs/reference) for every signal, source,
validator, command, flag and setting, generated from the binary.
