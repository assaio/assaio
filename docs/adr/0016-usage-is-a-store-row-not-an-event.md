# ADR 0016 — AI usage is a store row, not an event

Status: accepted (2026-09-02)

Amends [ADR 0007](0007-canonical-event-contract.md); settles the open posture in
[ADR 0009](0009-local-git-evidence-collector.md).

## Context

[ADR 0007](0007-canonical-event-contract.md) committed one envelope for every observation the
evidence graph would hold, AI and non-AI alike, and shipped `event.FromRecord` to adapt a
parsed `usage.Record` into `ai.usage.observed` and `ai.edit.observed`.
[ADR 0009](0009-local-git-evidence-collector.md) gave the contract its first real caller, but
only for `vcs.commit.observed`.

The AI half has had none since it shipped in v0.7.0, eighteen releases ago. `deadcode
./cmd/...` reports `FromRecord`, `Event.with` and both AI payloads as unreachable from the
binary. So `assaio-agent` carries two canonical models of the same fact: `store.UsageRow`,
which every report, metric, dashboard, rule and plugin actually reads, and `event.Event`,
which the contract says they read. That is not a transitional state. It is the state in which
a correction has two places to land and one of them is never visited — and the one never
visited is the one the architecture documentation points at.

`B104` framed this as a missing caller. The roadmap framed it as a choice: make the contract
the persisted, correction-aware ingestion spine, or delete the unused half. Three things
already in the tree decide it, and none of them is tidiness.

**The correction-aware spine exists, and it is not this package.** `usage_record` holds the
dedupe contract, the restatement path that lets a re-read raise a figure it had understated,
and, since v0.24, the count of the rows a re-read moved *down*. Moving that onto an event
spine buys a migration on a store that is 240 MB on the maintainer's machine, and SQLite does
not return those pages on `DELETE` — `compact` exists because of that. ADR 0007 rejected
persisting events on precisely this ground; making them the ingestion spine is the same bill
with a proven mechanism discarded to pay it.

**The consumer `B104` predicted does not want the adapter.** `B85` links a session to a
commit, and the conformance corpus that specifies it ([ADR 0010](0010-attribution-conformance-corpus.md))
is already shipped: `Fixture.Commits` is `[]event.Event` and `Fixture.Sessions` is
`[]usage.Record`. The specification an engine will be judged against reads the two shapes side
by side, and what its scenarios defend — surviving ambiguity, resisting proximity, honouring a
human correction — is untouched by whether both arrive in one envelope.

**What the envelope buys differs by domain.** For a commit, a pull request, a review round or
a check run it removes a record type and an analyzer coupling that would otherwise be invented
per source. For AI usage — canonical since the first release, with a schema, a dedupe key and
a restatement path — it buys a second encoding of numbers that already have one.

## Decision

`internal/event` is the observation contract for the domains that have no store row of their
own. AI usage is not one of them.

- **The AI half is deleted, not deprecated.** `FromRecord`, `Event.with`, the `Usage` and
  `Edit` payloads, their tests and the golden file are gone: 136 lines of contract, 187 of test
  and a 54-line golden file. A deprecation window protects callers, and there are none:
  `internal/` is unimportable outside the module, and the published metric-plugin wire (ADR
  0004) carries store row shapes, so no out-of-tree plugin can be holding an event.
- **The committed vocabulary loses its AI names.** ADR 0007 committed `ai.session.observed`,
  `ai.usage.observed` and `ai.edit.observed`; all three are withdrawn. `scm.pull_request`,
  `scm.review`, `ci.check` and `delivery.merge|revert|survival` stand, each still landing with
  the collector that fills it. Withdrawing a name is cheap while it has no producer and
  expensive afterwards, which is the same reason ADR 0007 gave for renaming `ai.session`
  before anything emitted it.
- **Grain shrinks with the payloads that counted in it; the classifications do not.** A grain
  is the unit a payload counts in, so `turn` and `session` leave with the AI payloads and a
  pull request will add its own. `privacy`, `provenance` and `time_source` keep their full
  sets whatever this build emits: they classify any observation, and the meaning of
  `local-only` *is* the contrast with `pseudonymous`. A one-value classification classifies
  nothing.
- **One error posture, and it is skip-and-count.** ADR 0009 recorded that the collector skips
  and counts while `FromRecord` failed a whole batch on the first rejected record, and that
  having both was not defensible. The batch-failing posture had exactly one implementation and
  it is deleted with it. `Validate` judges one observation and never a batch; a collector skips
  what it cannot build and reports the count, the way every parser treats a corrupt line. The
  split is settled by removing the half that disagreed, not by choosing between two live ones.
- **AI usage as an observation is a future decision, not a resumption.** If an OTel export
  (`B98`) or a connector that must carry sessions and commits over one wire ever needs it, it
  arrives with that consumer and costs one constant, one payload struct and one grain — the
  same one-line-per-collector change every other domain makes. What it must not do is exist
  first and wait to be needed.

Rejected alternatives:

- **Give the adapter a caller.** The only caller available today is ingest, adapting every
  parsed record into events nothing reads, validating a second time what the store already
  validates. That is a call site written to satisfy `deadcode`, and it puts work on the
  ingestion path whose entire output is discarded. An unproven contract half is at least
  honest about being unproven; a fabricated caller makes the same code look load-bearing.
- **Make events the persisted ingestion spine.** The honest maximal version, and the one the
  roadmap named. It is a migration and a size bound in front of the stage that pays for the
  product, to replace a mechanism that works with one that has no consumer. See the context
  above.
- **Keep the payloads as documentation of the intended shape.** A struct is worse documentation
  than a paragraph and better bait, because it compiles and therefore reads as available. This
  record and ADR 0007 are where the shape is described.
- **Rename the package to say it is git-only.** It is not: the connector domains (ADR 0007's
  `scm.*`, `ci.*`, `delivery.*`) land here next, and naming it after its single current
  producer would have to be undone by the change this contract exists to make cheap.

## Consequences

- `deadcode ./cmd/...` reports nothing in `internal/event`. On the tree this landed against it
  went from 70 unreachable functions to 63, and the seven that went are exactly the AI half —
  no other entry moved. What remains reachable is the envelope, its validation and the commit
  payload: the code `survival` runs on every pass.
- Nothing user-facing moves. No command, no column, no figure, no stored byte; the deleted
  code never ran outside its own tests.
- ADR 0007 is narrowed rather than superseded. Its envelope, its two clocks, its privacy
  classes, its "id is the source's own key" rule and its refusal to persist all stand; what it
  promised and this record withdraws is the symmetry between AI usage and everything else, and
  the three AI type names. It is amended in place where it made those claims.
- The evidence graph's next collector is unaffected. Adding `scm.pull_request.observed` is
  still one constant, one payload struct and one package — the shape ADR 0009 demonstrated,
  now the only shape this contract claims.
- The accepted risk is re-writing about seventy lines if AI usage ever does need an observation
  form. That is the price of not carrying a second model of every token count — in the store,
  in the reader's head, and in the place a future correction has to be applied twice.
