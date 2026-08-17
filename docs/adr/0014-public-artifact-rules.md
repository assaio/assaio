# 0014 — What a publicly postable artifact may say

Status: accepted · 2026-08-17 · shipped in v0.23.0 (`B149`)

## Context

`assaio-agent share` renders a window as an image, a video and a block of text a person
posts on a public timeline. That makes it unlike every other surface this binary has.

A wrong number in `report` is read by the person who ran it, in a terminal, beside the
caveats that qualify it, and it is corrected by running the command again. A wrong number
on a shared card is read by strangers, without the terminal, without the caveats, and it
cannot be recalled — the image outlives the store it came from, and a re-shared image
arrives with the caption stripped off.

The commitments below are the ones a future contributor could unknowingly undo while
making the card look better, which is exactly the pressure a marketing surface applies to
a measurement tool. Each is stated with the failure it prevents, because each was found by
committing it first.

## Decision

### 1. The package originates no figure

Every number on a card is read from what `internal/analyze` published for the same window,
including its published *string*. `internal/share` may select, omit and lay out; it may not
compute, re-round or re-unit.

The rule is not tidiness. Re-formatting a published `92.6%` with `%.0f` printed `93%`, and
a card that disagrees with `assaio analyze` is worse than a card that says less: it makes
the tool's own output unciteable. The same rule caught a derived share rendering `99.6%` as
`100%`, a bound the data never reached.

One figure is derived rather than quoted — the share of classified tool calls that ran a
command, because `explore-produce` publishes it as a Bar and the index reads only Figures.
It uses that validator's own denominator and the same percentage formatter, and it is the
only one; a second exception needs a reason recorded here.

### 2. Redaction is structural, never a flag

No field on `share.Assay` may hold a repository, member, path, branch, skill or sub-agent
name. There is deliberately no `--no-anonymize`: a flag is a thing to get wrong, and the
one artifact built for publication is the wrong place to offer the choice.

Tools and models are named. Which coding agent and which model ran is a public fact about a
vendor, not about the person who ran it. That argument does **not** reach an out-of-tree
parser, whose tool is stored as `plugin:<name>` where the name comes from the user's own
config (ADR 0003) — `plugin:acme-internal-billing` is exactly what this rule exists to keep
off a card, so any tool outside the built-in set renders as `a plugin source`.

Repositories appear as a count and never as a sorted spend distribution. The distribution
was in the original item and was dropped rather than deferred: ordering and proportions are
what somebody who knows the setup reads a pseudonym back out of, which is the same reason
this project refuses pseudonymous project labels on a public card.

### 3. Every rendered frame carries its own limits

The measurement layer, whether a cost figure is complete or a floor, and a sample-data
marker under `--demo` are drawn **on the artifact**, not listed beside it.

This is the commitment most easily lost, because it was lost once already: the caveats were
computed, rendered into the preview page, and absent from the PNG and the video — which are
the only parts that leave the machine. A limit that stays on the page qualifies nothing.

### 4. No rank, no percentile, no population

The archetype describes a shape of working and carries no percentile, no rarity and no
position in any ordering. A percentile needs a population, and this project refuses cohort
comparison without a minimum cohort size and explicit consent (`BACKLOG.md`).

The profile marks exactly one reserve per card, by construction rather than by judgement,
and it names the pole the data actually leans toward. A card that marked none would read as
"nothing to improve"; a card that marked several would read as a list of faults; and a card
that named the opposite pole — which shipped in review — publishes a claim about a person
that their own data contradicts.

### 5. The preview requests nothing, and the command imports only its own store

The page is self-contained: inline CSS, one inline script, no external font, image or
request. Image and video are produced in the viewer's browser from a canvas.

`share` has no `--db`. It imports through `backfill`, which writes `paths.DBPath()`
unconditionally and prunes trace steps past the horizon before it parses anything — so
accepting the flag turned "read from somewhere else" into an unrequested and irreversible
delete here. `init` refuses the flag for the same reason; a test now fails on the pairing
rather than on either command by name.

## Consequences

A figure that cannot be rendered under these rules is not rendered. That is the intended
cost, and it has already been paid: the spend distribution, the percentile, and every
archetype description that asserted something no condition tested.

The rules are enforced where they can be. Redaction is covered by a test that plants a
user-chosen string into every field a row carries and fails if it reaches any surface,
including the marshalled payload — a field nothing draws today is still a field that left
the machine. The quoting rule and the reserve's pole are each covered by a test with a
verified negative control. The rest is review.
