# Running it without being asked

*Part of [Extending assaio](../extending.md). For scheduling, see [Automation](../automation.md).*

[Automation](../automation.md) covers cron, launchd, and editor status lines. This page covers
unattended output and failures that should be visible.

Every command here is checked against the binary's command tree, so a renamed flag fails the build.

## The weekly loop, in the order it has to happen

Import, read, then report changes. If nothing refreshes the store before `digest`, it confidently
summarizes a stale window.

```sh #weekly-loop
#!/usr/bin/env sh
set -eu

assaio-agent backfill                 # import whatever the tools wrote since last time
assaio-agent doctor --strict          # stop if a source drifted or the price table is behind
assaio-agent digest --weekly          # markdown: what moved, and where the comparison is weak
```

Run `doctor --strict` before `digest`. It exits non-zero on suspected format drift, a configured
source with no inputs, or stored tokens without prices exceeding `pricing.max_unpriced_share`. Any
of these can make the digest misrepresent the week.

## Delivering the digest

`digest` writes markdown to stdout but does not deliver it; delivery depends on your infrastructure.
Two approaches cover most cases.

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

**Use `--dry-run` while iterating.** Each real run records the basis for the next comparison. Three
test runs in one afternoon would make the next weekly digest compare against that afternoon instead
of last week.

## When the schedule itself is the thing that broke

A stopped scheduled job produces no output, which looks like a quiet week. Check the *store's* age
to catch it:

`statusline` shows today's tokens, AI lines, cost basis, and data age. It **never fails loudly**:
every path exits 0. That suits a prompt, not an alarm.

`doctor` exits non-zero when a configured source has no inputs, which a stopped schedule eventually
causes.

```sh #freshness
# Exits non-zero on suspected drift, on a configured source with no inputs, on a store it
# cannot read, and above the unpriced-token ceiling.
assaio-agent doctor --strict
```

For staleness, read the data age and choose your own limit. assaio sets none because the right limit
depends on how often your team runs an agent.

## What not to automate

Nothing here writes a label. You can schedule `mark --accept-suggested` in a repo with conventions
you defined and trust. Without conventions, it writes nothing; with guessed conventions, it can
write wrong labels everywhere. Run [`--suggest`](label-rules.md) manually until its output is
predictable.

