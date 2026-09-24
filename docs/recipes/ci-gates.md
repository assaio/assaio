# Gating CI on what a window cost

*Part of [Extending assaio](../extending.md). Verdict rules are in [rule plugins](rule-plugins.md).*

`check` is the only command designed to exit non-zero. It fails on a token or API-equivalent `$`
budget, or when a configured [rule plugin](rule-plugins.md) raises an `error` alert or cannot run.
Every invocation below works, and every command and flag is checked against the binary's command
tree so a renamed flag fails the build before it breaks a pipeline.

## The one thing to decide first

**Tokens or dollars.** `--max-tokens` counts usage regardless of the price paid per token.
`--max-cost` estimates API-equivalent list-price cost; on a subscription, it is not your bill.

A cost gate **fails** if the window includes usage missing from the price table; it will not treat a
partial cost as the whole. Use a token gate by default until your team has configured its pricing
basis.

```sh #budget-tokens
# Fails when the last 7 days exceeded 50M tokens.
assaio-agent check --since 7d --max-tokens 50000000
```

```sh #budget-cost
# API-equivalent dollars. Fails on an unpriced model rather than under-reporting.
assaio-agent check --since 30d --max-cost 1500
```

## As a pre-push hook

Catch a runaway week here at no cost until you push.

```sh #pre-push-hook
#!/usr/bin/env sh
# .git/hooks/pre-push — warn, never block a push on somebody else's budget.
if ! assaio-agent check --since 7d --max-tokens 50000000; then
  echo "assaio: this week is over budget. Pushing anyway; see: assaio-agent analyze --since 7d" >&2
fi
exit 0
```

A push gate on a *team* budget penalizes whoever pushes last. To stop a property of a change, gate
on a rule plugin instead.

## As a GitHub Actions job

**`check` reads a store, and a CI runner has none.** A runner has no agent sessions, so installing
the binary and running `check` against its empty window passes regardless of spend. `sync` only
pushes; no command pulls a window.

Provide the store to the job. Run the gate **on the machine that has the store**, such as the [team
server](../extending/team-server.md) host with `--db`, or publish the store as an artifact and bring
it into the job. The recipe below uses an artifact for an Actions workflow.

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

Use the schedule shown, not a `pull_request` trigger. A per-PR budget gate acts like a per-person
weekly budget, which this project does not build.

## What a non-zero exit does and does not mean

`check` exits non-zero for three reasons; identify which one occurred:

| Exit | Meaning |
|------|---------|
| over budget | the window exceeded the number you set |
| an `error` alert | a rule plugin judged something and said so |
| a rule that could not be evaluated | the gate fails closed rather than passing on an unanswered question |

Do not ignore the third reason. A rule that failed to run gave no verdict; treating that as a pass
silently disables the gate.

## Before you gate on cost, check the cost

An estimated-cost budget depends on the price table. `doctor --strict` fails when too much of *your*
store has no price, which makes a `--max-cost` gate misleading rather than wrong.

```sh #doctor-before-cost
assaio-agent doctor --strict
```

Run it before the gate in the same job. `pricing.max_unpriced_share` sets the limit; it defaults to
5%, and `0` disables the check.
