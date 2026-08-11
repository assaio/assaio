# Running it without being asked

*Part of [Extending assaio](../extending.md). The full scheduling reference: [Automation](../automation.md).*

[Automation](../automation.md) covers the schedulers themselves — cron, launchd, editor status
lines. This page is the half it deliberately leaves out: what to do with the output once
something is running unattended, and which failures to make loud.

Every command here is checked against the binary's own command tree, so a renamed flag fails
the build rather than a Monday morning.

## The weekly loop, in the order it has to happen

Import, then read, then report what moved. Running `digest` against a store nothing refreshed
produces a confident summary of a stale window, which is the one outcome worth designing away.

```sh #weekly-loop
#!/usr/bin/env sh
set -eu

assaio-agent backfill                 # import whatever the tools wrote since last time
assaio-agent doctor --strict          # stop if a source drifted or the price table is behind
assaio-agent digest --weekly          # markdown: what moved, and where the comparison is weak
```

`doctor --strict` before `digest` is the point of the recipe. It exits non-zero on suspected
format drift, on a configured source with no inputs, and above `pricing.max_unpriced_share` of
stored tokens carrying no price — each of which makes the digest below it a description of
something other than your week.

## Delivering the digest

`digest` writes markdown to stdout and stops there, on purpose: delivery is a decision about
your infrastructure, not about measurement. Two shapes cover most of it.

```sh #digest-to-file
# Keep a dated archive; the digest compares against its own last run, not against these.
out="$HOME/assaio-digests/$(date +%Y-%m-%d).md"
mkdir -p "$(dirname "$out")"
assaio-agent digest --weekly > "$out"
```

```sh #digest-to-webhook
# Post it wherever your team reads things. --dry-run first if you are still tuning the window:
# it prints the digest without recording this run as the basis the next one compares against.
body=$(assaio-agent digest --weekly)
curl -fsS -X POST -H 'Content-Type: application/json' \
  --data "$(jq -Rn --arg t "$body" '{text:$t}')" \
  "$WEBHOOK_URL"
```

**`--dry-run` is the flag that matters while you are still iterating.** Every real run records a
comparison basis, so three test runs in an afternoon leave the next genuine digest comparing
against an afternoon rather than against last week.

## When the schedule itself is the thing that broke

A scheduled job that stops running produces no output, and no output is indistinguishable from a
quiet week. The cheap guard is to let the *store's* age answer instead of the job's:

```sh #freshness
# Non-zero when nothing has been imported recently enough to trust a report.
assaio-agent statusline
```

`statusline` prints today's tokens, AI lines, cost basis and how fresh the data is, and it never
fails loudly — which makes it right for a prompt and wrong as a monitor. For a monitor, read the
data age it prints and decide in your own script; assaio will not invent a staleness threshold,
because how stale is too stale depends on how often your team actually runs an agent.

## What not to automate

Nothing on this page writes a label. `mark --accept-suggested` is safe to schedule in a
repository whose conventions you wrote and trust — but a scheduled labeller in a repository
without a convention writes nothing at all, and a scheduled labeller in a repository whose
convention you guessed writes the wrong thing everywhere at once. Run
[`--suggest`](label-rules.md) by hand until its output is boring.
