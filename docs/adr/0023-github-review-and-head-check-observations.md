# 23. `evidence --github` observes reviews and latest head checks

## Status
Accepted (2026-10-08)

## Context
ADR 0022 joined sessions to pull requests and deferred reviews and checks to B229. Its bounded read could show a pull request’s state and listed commits, but not what the forge reported about review or the checks on its head. A bounded snapshot must also show what it did not see: a missing connection is not a count of zero, and a connection limited to 100 entries is not complete when its total is larger.

These forge observations describe a pull request. Neither its state nor its reviews or checks establish an AI session’s outcome, even when a session has a candidate commit linked to that pull request.

## Decision
**The shipped first slice of B229 extends the explicit `--github` read.** The user’s own `gh` runs the fixed repository-level GraphQL query. For each pull request it additionally asks for `reviews(last: 100)` with `totalCount` and each review’s id, state, `submittedAt` and `updatedAt`. It asks for the head’s `statusCheckRollup` state and `contexts(last: 100)` with `totalCount`. A CheckRun supplies id, status, conclusion, `startedAt` and `completedAt`; a StatusContext supplies id, state, `createdAt` and `updatedAt`. The query fetches no reviewer, author, review body, check name or check output.

**Each returned node becomes a validated observation.** A review is `scm.review.observed`, grain `review`, identified by its forge review node id. Its closed-vocabulary state is observed at `submittedAt`, falling back to `updatedAt`. A head context is `ci.check.observed`, grain `check`, identified by the pull-request node id joined to the context node id with `:`. Its kind distinguishes CheckRun from StatusContext; run status and conclusion, and status-context state, use closed vocabularies. A run occurs at completion, falling back to its start; a StatusContext uses `updatedAt`, then `createdAt`, and finally ingest-time `observedAt` if neither source time is present. The source is `gh`, provenance is `parsed`, and privacy is `local-only`. Every event is validated; a malformed or unknown observation fails the whole read.

**Coverage accompanies the snapshot.** Pull-request observations retain both each connection’s `totalCount` and its returned-node count. The query requests the last 100 entries in connection order; `listed < total` means incomplete, never zero. A null reviews connection or null head rollup means unavailable. The rollup describes the pull request’s current head, not every CI result in its history.

The walk still re-reads from the top until it reaches the first pass’s newest `updatedAt`. Reaching the re-read’s 20-page bound before that point now fails the read instead of returning an apparently complete document. The first pass continues to report its own 20-page bound through `readBackTo`. Pull-request snapshots are deduplicated by node id before review and check events are made.

**The document keeps pull-request facts apart from session results.** It counts pull requests read, named by at least one candidate or alternative, and unmatched within this read (`read − named`). Only named pull requests are listed by number, with review and head-check state counts and connection coverage. Candidate lines name their pull-request links without putting PR state beside a session. Session status, confidence and candidate selection do not change in this slice; no review or CI rate or session outcome is inferred.

## Consequences
ADR 0022’s statements that reviews and checks remain B229 work and that unmatched pull requests are not counted are superseded to the extent above. Its session-commit/v3 rules remain in force.

ADR 0007 and ADR 0020 still govern these observations: correlation runs locally only on explicit `--github`, with no store, sync, dashboard, share or plugin surface and no free forge text or person field in an event or document. Check suites, merge method, revert relation and derived rates remain outside this slice.

## Continuation (2026-10-09)
[ADR 0024](0024-bounded-github-history.md) supersedes those deferrals for bounded historical
suite/run observations, eligible distinct reviewed-revision counts, a request-change snapshot
share with a full named merged PR denominator, and multi-parent merge identification. The history
read is independent of the base read, with separate bounds and layer coverage. Exact rounds,
comparable CI histories and rates, authoritative one-parent merge methods and PR revert links
remain deferred. These are local PR observations; attribution selection and the content/person
exclusions are unchanged.
