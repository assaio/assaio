# 6. Session annotations are category labels, and they stay local

## Status
Accepted (2026-07-31)

## Context

Session logs record what a tool did. They never record what the work was *for*. That gap
makes the questions this project exists to answer unanswerable: "did AI-written code hold
up" averaged over research, a refactor and a greenfield feature is a number nobody can act
on, and the same is true of cost per line, model fit, and friction.

Intent cannot be recovered from the logs. Reading prompts to infer it is not an option —
prompts are never collected, and inference would manufacture a fact the data does not
contain. So intent has to be stated explicitly, by a person, after the fact.

That makes annotations unlike everything else in the store. Every other row is derived from
a file on disk and can be rebuilt by re-running `backfill`. An annotation cannot: delete it
and the only copy is gone.

## Decision

**1. Category labels only.** An annotation is three closed vocabularies — task class,
outcome, difficulty — defined in `internal/label` and validated in Go. There is no free-text
field, so the table is free of content by construction: there is nowhere to put a branch
name, a ticket title, a commit message, or a prompt. A value outside its vocabulary is an
error, never a silently stored string.

**2. No issue or branch reference here.** An earlier sketch of this feature carried an
optional issue id or branch name. It is deliberately excluded. A pointer from a session to a
commit or a PR is not a label, it is a *claim*, and ADR-level honesty requires a claim to
carry its method, its confidence, and the alternatives it beat. That is the attribution-edge
work (`B85`), and storing a bare string here would have to be migrated into that shape
later. Keeping it out also keeps decision 1 absolute.

**3. Local only.** `sync` sends `[]usage.Record`; annotations are per-session and are not
part of that payload, and no other path sends them. Whether a team ever pools intent is a
separate decision with its own consent question, not a side effect of shipping this.

**4. Never deleted as a side effect.** `clear --all`, `clear --older-than` and `clear --tool`
leave `session_label` untouched and report how many labels they kept. Only `clear --labels`
deletes them. Nothing prunes them automatically: a session outside the retention window can
return, and an annotation once deleted cannot.

**5. Unlabeled is a first-class state, never a failure.** Unlabeled sessions stay fully
counted in every unfiltered metric. A report grouped by an annotation always renders the
`unlabeled` group rather than hiding it. The `intent` validator has no unfavorable verdict:
scoring how diligently someone labels, or judging a work mix that is genuinely all one kind,
would turn optional bookkeeping into a judgement about a person.

**6. Opt-in at the query layer.** `store.Usage` and every other existing query are unchanged
and carry no annotation join. Reading usage grouped by annotation is a distinct call
(`UsageByLabel`), and filtering is a distinct call (`UsageFiltered`, and the `*Filtered`
siblings of the four other window queries). The metric-plugin wire (ADR 0004) is therefore
unchanged: `usageWire`/`sessionWire` map fields explicitly and do not carry annotations.

**7. A filter narrows every input or none.** A label filter is applied to all five window
queries at once. Narrowing only the usage rows would state one subset's figures beside the
whole window's delegation, attribution and turn mix. Validators marked `WindowScoped` —
whose answer belongs to the whole window, like a flat plan price — are skipped under a
filter and named as skipped, rather than being restated over a slice they do not describe.

## Consequences

- Adding a category is a one-line change in `internal/label`, not a migration. There is
  deliberately no SQL `CHECK` constraint, which would be a second source of truth.
- The store grows by one row per session a person marked (~80 bytes), bounded by hand rather
  than by history: it does not grow with the volume of logs ingested.
- A `--task` filter necessarily reports a smaller sample than the unfiltered run, and the
  confidence envelope (`B81`) says so on its own. That is the intended behaviour: a verdict
  about twelve sessions must not read like a verdict about three hundred.
- A Claude session id is reused across `--resume`, so one label can span work that changed
  character later in the same session. This cannot be fixed at this layer; it ships as a
  stated caveat on the `intent` validator and its `explain` page.
