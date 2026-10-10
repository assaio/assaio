# 24. Independent bounded GitHub history with explicit populations

## Status
Accepted (2026-10-09)

## Context
The bounded PR, review and current-head read cannot describe historical suites/runs or exact
review rounds. A deep all-at-once real query exceeded source resources. An unlimited walk would
hide both operational bounds and unequal histories. Current review states are snapshots, not
state transitions; currently listed commits omit force-pushed-away work.

## Decision
Keep two fixed repository queries through the user's own `gh`. The base walk reads 50 PRs per
page for at most 20 pages; the independent historical walk reads 5 per page for at most 20 pages.
Each can additionally re-read at most 20 pages from the top. Both follow PR update times under
`--since` and share the same invocation `Read.ObservedAt`, while retaining independent PR counts
and read-back windows. Repository source
totals cover the entire repository, not the selected window. Each call has a 60-second timeout
and 8 MiB cap. Source errors fail the document; a resource-limit partial result is not accepted.

The base read retains the first 100 listed commits, last 100 reviews and last 100 latest-head
contexts. Reviews add actual reviewed commit hashes and submitted times; merge commits add parent
counts. The historical query reads the first 100 currently listed commits, last 10 suites per
commit and last 5 runs per suite with check type ALL. Historical suite and run observations carry
content-free identifiers, hashes, closed status/conclusion fields and source times. They retain
`gh` source, parsed provenance and local-only privacy. Source timestamps remain separate from
occurrence-time fallbacks; a fallback cannot establish an actual review submission time.

Join snapshots only by PR node id and number, matching commit totals and identical returned
first-100 commit SHA sets with no duplicate SHAs in either prefix. Both lists may be truncated;
matching prefixes retain their coverage gaps. A mismatched prefix or total withholds history. Only PRs named by candidates or alternatives receive
details. Each layer carries total and listed counts: PRs, commits, reviews, head contexts, suites
and runs. Null connections, counts or nodes mean unavailable; explicit empty connections with
total zero mean known empty. Latest-head contexts and historical runs are separate populations
and never summed together.

Nullable payload fields are limited to `PR.MergeParents` (`*int64`),
`PullRequestCommit.Suites` (`*Population`) and `CheckSuite.Runs` (`*Population`).
`Population` contains only numeric `Total` and `Listed` counts. These fields do not permit
free text, content or person fields or bypass boundary validation.
Raw observations remain separate from derived counts and rates, with closed reasons for every
withheld result.

Distinct `changes_requested` reviewed commit hashes are counted only from a complete usable
review population with matching observation counts, unique present ids, actual submission times
and hashes, and no dismissed, pending or unusable states. Exact rounds are always withheld.
The request-change snapshot share uses the entire named merged PR population, including
alternatives, and requires a nonempty usable review population for every PR. Ineligible PRs do
not shrink its denominator. After-merge reviews contribute to a current snapshot; the share
is not a pre-merge fact.

A comparable PR pipeline CI rate is always withheld. Bounded suites/runs on currently listed
commits do not establish a whole workflow lifetime or a comparable pipeline population. More
than one merge-commit parent proves a merge; one parent cannot distinguish squash from rebase.
Absent or unusable merge evidence withholds the method. No `mergeMethod` source field is assumed.

## Consequences
ADR 0022's candidate rules and `session-commit/v3` remain unchanged; offline evidence remains
`session-commit/v2`. ADR 0023's suite/history deferral is superseded only by this bounded slice.
Observations and links stay in memory for one local command: no persisted graph, sync, dashboard,
share or plugin surface, no source content or person fields, and no credentials handled by assaio.
Requests send repository identity and cursor, never session or store data.

Exact request rounds require authoritative review-state transitions and round boundaries.
Comparable PR workflow lifetimes need force-pushed-away commits and more source evidence.
Authoritative squash-versus-rebase methods and trustworthy content-free revert relations need
source support and conformance cases. These remain B232 work; no subject or topology heuristic
establishes PR reversion, and no PR observation establishes an AI session outcome.
