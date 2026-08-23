# ADR 0015 — Structured recommendations

Status: accepted (v0.24)

## Context

Every verdict `analyze` produces ends in a `Takeaway string`. A sentence can explain a result.
It cannot say what an action requires before it makes sense, what it risks, how to undo it,
when to look again, or which figure would show whether it worked — so nothing assaio has ever
suggested could be checked afterwards. A product whose stated chain ends in *verified
recommendation* cannot have its final link be prose.

The risk is asymmetric. A recommendation magnifies whatever error sits under it: a precise
action derived from a biased baseline is worse than a missing metric, because it moves someone.
That is why this ADR lands in the same release as the trust reset that withdrew fourteen
unsourced verdicts (`B177`) and fixed the recovery baseline (`B175`) — advice built on those
would have inherited them.

## Decision

`internal/recommend` holds a typed `Record`. Its required fields are exactly the ones prose
could not carry: evidence, confidence, scope, one action, prerequisites, expected effect,
risks, rollback, review window, follow-up metric, and a status from a closed vocabulary
(`proposed`, `accepted`, `rejected`, `running`, `verified`, `inconclusive`).

Four rules hold the contract:

1. **Rendered text projects the record.** Every line of the CLI block comes from a field. A
   renderer cannot grow a claim the record does not hold.
2. **A record missing a required field is dropped, not printed with a warning.** Advice that
   cannot be checked is what this package exists to prevent; shipping it with a caveat is
   shipping it.
3. **Abstention is the default.** A family builds on a verdict only where its confidence is
   `high` or `medium` and none of its declared inputs were withheld. A thin window produces
   nothing, and the empty output says it is an abstention rather than a clean bill.
4. **No predicted number.** `Effect` is a direction in words. assaio has no counterfactual;
   a predicted percentage would be the fabricated figure this project refuses. What it can do
   is name the follow-up figure *before* the result is known.

One family, one file, mirroring the metric validators — because precision is measured per
family, and a family that keeps being rejected has to be disableable as a unit.

## Consequences

**The lifecycle is declared and not yet reachable.** Every record ships `proposed`. Moving one
to `accepted`, `running`, `verified` or `inconclusive` needs stored state, a migration, and a
comparison of a baseline window against an intervention window with drift checks — a milestone,
not a field. The vocabulary is complete from the start anyway: a status added later would leave
every earlier record unable to say which of these it had been.

**Recommendations quote verdicts rather than recomputing them.** A family reads the `Result`
`analyze` published for the same window, so a recommendation and the report can never disagree
about the same figure.

**The first families are about assaio's own evidence.** Pricing coverage comes before advice
about how someone works — an engine that recommends changing a workflow while its own cost basis
is missing a fifth of the window has the order backwards. That ordering is explicit in
`familyOrder`, not alphabetical: a reader acts on what is at the top.

**A family that cannot state its condition honestly does not ship.** A capture-gap family — "part
of this window was read by a build that captured less, run `backfill`" — was written twice and
withdrawn twice before release, because the coverage ratio it needed does not exist on a
`Result`: the only available denominator counts every tool call, including those from sources
that structurally record no failure, so on a two-tool store it fired on a gap no backfill can
close. Shipping a third guess would have made the one recommendation carrying a checkable
follow-up the one that never checks out.
