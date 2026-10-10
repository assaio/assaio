# 25. Bounded raw GitHub review-dismissal observations

## Status
Accepted (2026-10-10)

## Context
Current review states cannot recover a dismissed review's earlier state. A finite real source
capture found approved→dismissed events in github/docs #46200 and microsoft/vscode #339957 and
#340803, with actual event times and content-free review joins. Separate CLI samples had no
selected dismissals while other timeline events were present. That observed absence does not
establish API incapability or repository-wide absence.

The source distinguishes all timeline events, selected dismissals and the returned slice.
A nullable review join and missing coverage must remain unknown rather than being reconstructed.
The [source feasibility guide](../github-source-feasibility.md) records the captures, exact
projections, count demonstration and remaining evidence gaps.

## Decision
Extend ADR 0024's existing base query with
`timelineItems(last: 100, itemTypes: [REVIEW_DISMISSED_EVENT])`, aliased as `reviewDismissals`.
Keep its 50-PR pages, at most 20 pages plus at most 20 pages of top reread. The independent
5-PR historical walk, its bounds and joins remain unchanged. Both walks share the invocation
observation time; API calls retain the 60-second timeout and 8 MiB response cap.

Select only `__typename`, `id`, `createdAt`, `previousReviewState` and nullable
`review { id submittedAt commit { oid } }`, alongside connection `totalCount`, `filteredCount`,
`pageCount` and `pageInfo { hasPreviousPage }`. Do not select author, actor, reviewer,
`dismissalMessage`, title, body, content or person fields.

Emit `scm.review.dismissed` as a raw source-local PR outcome observation. Its payload carries
PR `number`, source-stated `previousState`, required actual `dismissedAt`, and optional `reviewId`,
`commit` and `submittedAt`. The envelope carries the event id, `gh` source, parsed provenance,
local-only privacy, the invocation `observedAt` and an occurrence time stated by the source.
The actual event `createdAt` is required; neither PR update time nor observation time substitutes
for it. A null source review leaves its id, hash and submission time absent. Optional review
metadata is never manufactured.

Retain nullable `TimelinePopulation` coverage independently of raw observations:

- `totalCount` counts all PR timeline events;
- `filteredCount` counts selected review dismissals;
- `pageCount` is the source-reported returned slice count;
- `listed` counts actual retained nodes;
- `hasPreviousPage` states selected-population truncation.

Validate counts and nodes against the fixed last-100 bound. Missing coverage remains
null/unavailable, while independently validated raw events may still appear. An explicit usable
selected-empty connection is known empty at read time; a positive whole-timeline total does not
contradict it. Invalid source data fails the document.

Keep raw transitions separate from current reviews. `previousState` never restores a snapshot,
defines a review round or feeds a new rate or count-derived metric. Existing reviewed-revision
eligibility and request-change snapshot-share calculations remain unchanged. Offline
`session-commit/v2`, GitHub `session-commit/v3` and all matching rules remain unchanged.

## Consequences
The machine-readable addition consists of one event type, its allowlisted payload and nullable
PR dismissal coverage. It adds no schema migration, stored graph, sync, dashboard, share or plugin
surface. Requests send repository identity and cursors, never session or store data. No PR
observation establishes what an AI session caused.

The production-shaped query and response are retained as
`internal/github/testdata/b232/production-dismissals.graphql` and `production-dismissals.json`.
Their capture ran from 2026-10-10T21:30:48Z to 2026-10-10T21:30:50Z. The separate investigative
last-zero count demonstration is not a production query or proof of a real population exceeding
100 dismissals. Real changes_requested dismissals, nullable review joins and more than 100
selected dismissals remain corpus gaps; synthetic cases cannot close them.

B232 remains open for authoritative review transitions and round boundaries, complete comparable
workflow lifetimes including lost/force-pushed commits, an authoritative one-parent applied
merge-method source and an explicit content-free PR revert relation. Each needs source coverage,
a defined comparable population and real positive, negative and unknown cases. ADR 0024's
existing bounded-history decision remains in force.

Reverting the one additive feature PR reverses this addition without a migration. Published tags
and shipped historical records remain immutable. See [ADR 0024](0024-bounded-github-history.md),
[the evidence guide](../evidence.md) and [compatibility](../compatibility.md).
