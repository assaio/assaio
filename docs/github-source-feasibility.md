# GitHub source feasibility and bounded review dismissals

`assaio-agent evidence --github` adds raw review-dismissal observations to local delivery evidence.
They describe a source-stated PR outcome, never an AI session result. Exact review rounds,
comparable workflow lifetimes and CI rates, authoritative one-parent merge methods and explicit
PR revert relations remain deferred under B232.

This guide records a finite source study captured on 2026-10-10. Field availability, observed
cases and comparable measurement populations are separate questions. A schema capture answers
only the first.

## Currently shipped reads

Two fixed GraphQL repository walks run through the user's own `gh`, ordered by PR `updatedAt`,
newest first. `--since` bounds that update-time walk, not every event's occurrence time. The base
walk reads 50 PRs per page, at most 20 pages, plus a top reread of at most 20 pages. The independent
history walk reads 5 PRs per page with the same page and reread caps. Both share one invocation
`observedAt`, while retaining separate PR counts and read-back windows. Repository totals cover
all repository PRs, rather than the selected window. Each API call has a 60-second timeout and an
8 MiB response cap. Source errors fail the document.

| Read | Exact selected fields and bounds | Meaning and population |
| --- | --- | --- |
| Base PR and commits | PR `id`, `number`, `state`, `updatedAt`, `mergedAt`; `mergeCommit { oid parents(first: 2) { totalCount } }`; `commits(first: 100) { totalCount nodes { commit { oid } } }` | Current PR snapshot and first 100 currently listed commits; parent count is topology. |
| Current reviews | `reviews(last: 100) { totalCount nodes { id state submittedAt updatedAt commit { oid } } }` | Current review states, with actual submission time kept separate from occurrence-time fallbacks. |
| Raw dismissals | `timelineItems(last: 100, itemTypes: [REVIEW_DISMISSED_EVENT])`, aliased as `reviewDismissals`; `totalCount`, `filteredCount`, `pageCount`, `pageInfo { hasPreviousPage }`; nodes contain only `__typename`, `id`, `createdAt`, `previousReviewState`, nullable `review { id submittedAt commit { oid } }` | Last 100 selected dismissal events per PR. Actual event `createdAt` is required; there is no fallback. |
| Latest-head checks | `statusCheckRollup.state`; `contexts(last: 100) { totalCount nodes { __typename } }`, with CheckRun `id`, `status`, `conclusion`, `startedAt`, `completedAt`, or StatusContext `id`, `state`, `createdAt`, `updatedAt` | Current-head contexts; separate from historical suites and runs. |
| Independent history | PR `id`, `number`, `updatedAt`; first 100 listed commits with `oid`; last 10 `checkSuites` per commit with `totalCount`, `id`, `status`, `conclusion`, `createdAt`, `updatedAt`; last 5 `checkRuns` per suite with `checkType: ALL`, `totalCount`, `id`, `status`, `conclusion`, `startedAt`, `completedAt` | Bounded history on currently listed commits, with total/listed coverage at each layer. It does not recover lost commits or complete workflow lifetimes. |

Only PRs named by candidates or alternatives receive details. Historical data joins by PR node id
and number only when commit totals and returned first-100 SHA sets agree, without duplicate SHAs
in either prefix. Matching truncated prefixes do not establish complete history; a mismatch
withholds the join.

Dismissal coverage preserves five distinct facts:

| Field | Meaning |
| --- | --- |
| `totalCount` | All PR timeline events, including kinds outside the dismissal filter. |
| `filteredCount` | All selected review-dismissal events in that timeline. |
| `pageCount` | The source-reported count in the returned selected slice. |
| `listed` | Actual retained dismissal nodes. |
| `hasPreviousPage` | Source-stated truncation of the selected dismissal population. |

For the production `last: 100` read, usable coverage requires consistent nonnegative counts,
`listed = pageCount = min(filteredCount, 100)` and
`hasPreviousPage = (filteredCount > pageCount)`. Missing coverage remains null/unavailable;
independently validated raw events may still appear. An explicit usable connection with
`filteredCount = 0`, an empty slice and no previous page establishes selected absence for that
PR at capture time, even when `totalCount` is positive.

A dismissal carries its opaque event id, PR number, source-stated previous review state and
actual dismissal time. A null `review` leaves review id, reviewed commit hash and submission time
absent. The envelope retains `gh` source, parsed provenance, local-only privacy and the shared
invocation observation time. The event belongs to the PR outcome layer. Its previous state does
not restore a current review snapshot or enter a new count-derived metric or rate.

No author, actor, reviewer, dismissal message, title, body, content or person field is selected.
Requests send repository identity and cursors, never session or store data. Observations remain
in memory for one command, outside the store, sync, dashboards, share and plugins. See
[PRIVACY.md](../PRIVACY.md), [the evidence guide](evidence.md) and
[ADR 0025](adr/0025-bounded-review-dismissals.md).

## Four-track feasibility matrix

The endpoints and projections below are investigative candidates unless explicitly identified
as shipped above. They do not add requests to `evidence --github`.

| Track | Investigated source and exact projection | Meaning, privacy and population limits | Missing evidence and decision |
| --- | --- | --- | --- |
| Review transitions / rounds | GraphQL `ReviewDismissedEvent`: production fields listed above. `pr-timeline.graphql` also selected `review.state` and `pullRequestCommit { commit { oid } }`; review/request/removal events selected their ids and times. | A dismissal proves one source-stated transition. Request/removal events describe requests, without collecting the requested person. Production uses only the narrower dismissal projection and its own coverage. | Three real approved→dismissed cases exist. No real changes_requested dismissal, nullable-review case or population over 100 dismissals was captured. Complete state transitions and authoritative round boundaries remain missing; exact rounds stay withheld. |
| Comparable CI/workflow lifetime | GraphQL `HeadRefForcePushedEvent { id createdAt beforeCommit { oid } afterCommit { oid } }`. Schema inspection established `CheckSuite.workflowRun` and WorkflowRun field availability, including `id`, `createdAt`, `updatedAt`, `event`, `runAttempt`, `runNumber`, `checkSuite`. REST candidates are repository workflow-run listing, individual-run and run-attempt reads. | Force-push endpoints identify two head commits, not every removed commit or its checks. A suite/run association does not establish a complete comparable PR lifecycle. Schema queries read field names, without workflow names, branch values, titles, actors or output. No real REST workflow-lifetime projection was captured. | Lost/force-pushed commits, attempts, deletion, synthetic merge commits and reliable PR association need a complete source/population contract and real cases. No vanished-commit workflow-lifetime proof exists; comparable CI rates remain withheld. |
| Authoritative merge method | `pr-timeline.graphql` selected `mergeCommit { oid parents(first: 2) { totalCount } }`, `MergedEvent { id createdAt commit { oid } }`, `autoMergeRequest { mergeMethod }`, and auto-merge/rebase/squash enabled or disabled event ids and times. | More than one merge parent proves a merge. One parent cannot identify squash versus rebase. `MergedEvent.commit` identifies the landing commit; its parents are topology. `autoMergeRequest.mergeMethod` and enabled event kinds express intent, not authoritative execution. No actor or branch field is selected. | An authoritative read-side applied-method source and real distinguishable one-parent counterexamples remain missing. Do not infer execution from topology, configured intent or commit subjects. |
| Content-free revert relation | `pr-timeline.graphql` selected `CrossReferencedEvent { id createdAt isCrossRepository source { __typename } }`, with source PR `id`, `number`, `state`, `mergedAt`, or Issue `id`, `number`, `state`. `revertPullRequest` was examined as a mutation candidate, never executed. | A cross-reference is a general reference, not proof of reversion. The mutation expresses write intent; its result cannot supply an existing reliable read-side PR revert population. No title, body or subject heuristic is admissible. | An explicit content-free original/revert relation, a defined read population and real positive, negative and unknown cases remain missing. There is no actual revert receipt in this study. |

GitHub documents the dismissal and connection fields in
[ReviewDismissedEvent](https://docs.github.com/en/graphql/reference/pulls#reviewdismissedevent)
and [PullRequestTimelineItemsConnection](https://docs.github.com/en/graphql/reference/pulls#pullrequesttimelineitemsconnection).
The force-push source is
[HeadRefForcePushedEvent](https://docs.github.com/en/graphql/reference/pulls#headrefforcepushedevent);
workflow-run and attempt read candidates are documented in
[REST workflow runs](https://docs.github.com/en/rest/actions/workflow-runs).
[CheckSuite](https://docs.github.com/en/graphql/reference/objects#checksuite) and
[WorkflowRun](https://docs.github.com/en/graphql/reference/objects#workflowrun) document the schema
associations, without establishing comparable PR lifetimes.
[MergedEvent](https://docs.github.com/en/graphql/reference/pulls#mergedevent) and
[AutoMergeRequest](https://docs.github.com/en/graphql/reference/pulls#automergerequest) support the
landing-commit and requested-method distinction.
[CrossReferencedEvent](https://docs.github.com/en/graphql/reference/objects#crossreferencedevent)
describes references; [revertPullRequest](https://docs.github.com/en/graphql/reference/mutations#revertpullrequest)
is a mutation, not a read-side revert relation.

## Finite real evidence

The production-shaped dismissal capture ran from `2026-10-10T21:30:48Z` to
`2026-10-10T21:30:50Z`. Its query uses `last: 100`, the discriminator and exactly the shipped
allowlist. Reproduction starts with the committed
[query](../internal/github/testdata/b232/production-dismissals.graphql) and
[response](../internal/github/testdata/b232/production-dismissals.json).

| Captured PR | Actual dismissal time | Previous state | total / filtered / page / listed / previous page |
| --- | --- | --- | --- |
| github/docs #46200 | 2026-10-06T17:06:15Z | APPROVED | 23 / 1 / 1 / 1 / false |
| microsoft/vscode #339957 | 2026-10-06T00:59:25Z | APPROVED | 24 / 1 / 1 / 1 / false |
| microsoft/vscode #340803 | 2026-10-09T21:42:28Z | APPROVED | 22 / 1 / 1 / 1 / false |
| cli/cli #14640 | No selected event | — | 25 / 0 / 0 / 0 / false |

All three positives include review id, actual submission time and reviewed commit hash. These
cases establish approved→dismissed observations, not every possible previous state or join shape.

The study's newest-by-update 30 CLI PRs of all states and separate newest-by-update 30 merged CLI
PRs each had zero selected dismissal events while other timeline events were present
(`cli-dismissal-root-page1.json`, `cli-dismissal-merged-root-page1.json`). The samples overlap and
are not 60 distinct PRs. Their source totals were 4,703 all PRs and 3,267 merged PRs. The separate
30-merged-PR count captures found one selected dismissal in github/docs and two in
microsoft/vscode (`github-docs-dismissal-counts.json`, `vscode-dismissal-counts.json`), with source
merged-PR totals of 15,616 and 55,268. These are finite observed populations; none establishes API
incapability or absence throughout a repository or GitHub.

For github/docs #46200, `dismissal-truncation.graphql` and
`github-docs-dismissal-truncation.json` compare investigative `last: 0` and `last: 1` reads.
The zero slice reports total 23, filtered 1, page 0 and previous page true; the one slice retains
the event and reports previous page false. This demonstrates count distinctions and source-stated
truncation. `last: 0` is never the production query and does not prove a real population exceeding
100 dismissals.

The initial `pr-timeline.graphql` investigation read 10 PRs per repository. In assaio/assaio,
source total 106, it returned 11 force-push events, 9 merged events, 5 cross-references, 5 review
requests and no dismissals (`assaio-timeline-page1.json`). In cli/cli, source total 4,703, it
returned 13 review nodes, 25 review requests, 11 removals, 19 cross-references and no dismissals
(`cli-timeline-page1.json`). Its first-10 commits, first-50 reviews and last-100 mixed timeline
slice are investigation bounds, not the production read contract.

`schema-pr-response.json`, `schema-transitions-response.json` and `schema-workflow-response.json`
record schema availability. They neither recover missing histories nor establish semantic
comparability. Synthetic conformance cases remain separate from real source captures and cannot
close any of the real-corpus gaps above. No feature test or release-gate execution is asserted by
this source study.

## Remaining B232 gates

B232 remains open. Review rounds require authoritative transitions and round boundaries, explicit
coverage and real cases including changes_requested dismissals, null joins and selected
truncation beyond 100. Comparable workflow lifetimes require a source that covers lost and
force-pushed commits, attempts and relevant PR associations, with comparable populations and real
positive, negative and unknown cases. One-parent merge methods require an authoritative applied
method source and distinguishable real squash/rebase cases. Revert relations require an explicit
content-free read-side relation and real original/revert, negative and unknown cases.

Current review snapshot calculations, offline `session-commit/v2` and GitHub `session-commit/v3`
matching remain unchanged. The addition is reversible by reverting one feature PR; it needs no
store migration. Published tags remain immutable. See
[ADR 0024](adr/0024-bounded-github-history.md) and
[ADR 0025](adr/0025-bounded-review-dismissals.md).
