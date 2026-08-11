# Gating CI on what a window cost

*Part of [Extending assaio](../extending.md). The rules that decide a verdict live in [rule plugins](rule-plugins.md).*

`check` is the only command that exits non-zero on purpose. It fails on a token or
API-equivalent `$` budget, and it fails when a configured [rule plugin](rule-plugins.md) raises
an `error` alert — or cannot be evaluated at all. Everything below is a complete, working
invocation; every command and flag on this page is checked against the binary's own command
tree, so a renamed flag breaks the build rather than a reader's pipeline.

## The one thing to decide first

**Tokens or dollars.** `--max-tokens` is plan-independent: it counts what was spent regardless of
what anybody pays per token. `--max-cost` is an API-equivalent estimate, and on a subscription it
is not your bill — it is what the same usage would have cost at list price.

A cost gate refuses to pass on a partial figure: a window carrying usage the price table cannot
price **fails** rather than reporting the priced part as if it were the whole. That is deliberate,
and it is why a token gate is the better default for a team that has not configured its pricing
basis yet.

```sh #budget-tokens
# Fails when the last 7 days exceeded 50M tokens.
assaio-agent check --since 7d --max-tokens 50000000
```

```sh #budget-cost
# API-equivalent dollars. Fails on an unpriced model rather than under-reporting.
assaio-agent check --since 30d --max-cost 1500
```

## As a pre-push hook

The cheapest place to notice a runaway week, because it costs nothing until you push.

```sh #pre-push-hook
#!/usr/bin/env sh
# .git/hooks/pre-push — warn, never block a push on somebody else's budget.
if ! assaio-agent check --since 7d --max-tokens 50000000; then
  echo "assaio: this week is over budget. Pushing anyway; see: assaio-agent analyze --since 7d" >&2
fi
exit 0
```

Blocking a push on a *team* budget punishes whoever pushes last, which is why the recipe warns.
Block on a rule plugin instead when the thing you want stopped is a property of the change.

## As a GitHub Actions job

**`check` reads a store, and a CI runner has none.** This is the part every "put it in CI" recipe
gets wrong, including an earlier draft of this one: a runner's own filesystem holds no agent
sessions, so a gate that just installs the binary and runs `check` reads an empty window and
passes at any spend, forever. There is no command that pulls a window either — `sync` is
push-only.

So the store has to arrive from somewhere, and the job has to say where. Two honest shapes: run
the gate **on the machine that has the store** (the [team server](../extending/team-server.md)
host, via `--db`), or carry the store into the job as an artifact something else published. The
recipe below does the second, because it is the one that fits an Actions workflow.

```yaml #actions-job
name: ai-budget
on:
  schedule: [{cron: '0 7 * * 1'}]     # Monday morning, not per-PR: a budget is a window, not a diff
  workflow_dispatch:

permissions:
  contents: read

jobs:
  budget:
    runs-on: ubuntu-latest
    steps:
      - name: Install assaio
        env:
          # Pin it. `latest/download` cannot be used here: the asset name carries the version,
          # so the URL has to name it too. Set the repository variable to a real release.
          VERSION: ${{ vars.ASSAIO_VERSION }}
        run: |
          curl -fsSL "https://github.com/assaio/assaio/releases/download/$VERSION/assaio_${VERSION#v}_linux_amd64.tar.gz" \
            | tar -xz -C /usr/local/bin assaio-agent
      - name: Restore the store this gate reads
        uses: actions/download-artifact@v4
        with: {name: assaio-store, path: store}
      - name: Gate
        run: assaio-agent check --db store/assaio.db --since 7d --max-tokens 50000000
```

Note the schedule rather than a `pull_request` trigger. A per-PR budget gate reads as a
per-person one within a week, and this project refuses to build those.

## What a non-zero exit does and does not mean

`check` exits non-zero for three different reasons and it is worth knowing which you got:

| Exit | Meaning |
|------|---------|
| over budget | the window exceeded the number you set |
| an `error` alert | a rule plugin judged something and said so |
| a rule that could not be evaluated | the gate fails closed rather than passing on an unanswered question |

The third is the one people patch out and should not: a rule that failed to run has told you
nothing, and treating nothing as a pass is how a gate quietly stops gating.

## Before you gate on cost, check the cost

A budget on an estimate is only as good as the price table under it. `doctor --strict` fails when
too much of *your* store carries no price at all, which is the condition that makes a `--max-cost`
gate misleading rather than wrong.

```sh #doctor-before-cost
assaio-agent doctor --strict
```

Run it in the same job, before the gate. `pricing.max_unpriced_share` sets the ceiling; the
default is 5% and `0` disables the check.
