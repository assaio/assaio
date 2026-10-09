# 22. `evidence --github` joins sessions to pull requests through the user's own `gh`

## Status
Accepted (2026-09-30)

## Context
`evidence` (ADR 0018, ADR 0021) sees the commits reachable from `HEAD`. On a repository that
squash-merges, the work a session did lives on branch commits `HEAD` never reaches, and the only
commit left on `main` is the squash, stamped at the merge: session-commit/v2 offers it as a
low-confidence landing among every other squash in the following window, so most sessions stay
ambiguous. The forge knows which commits each pull request carried. ADR 0020 set how assaio may
ask: on an explicit per-invocation flag, through the user's own client, naming a repository and a
window and nothing the store holds, with allowlisted fields, results `local-only`.

Two things the forge's answer does not settle. A commit being in this clone's object store says
nothing about who wrote it: a fetch brings every teammate's pushed branch. And several commits
belonging to one pull request is not evidence about which of them holds a session's work.

## Decision
**The connector.** `evidence --github` runs `gh` in the repository: `gh repo view --json
nameWithOwner,url,isFork` names the repository and host, with `GH_REPO` and `GH_HOST` removed from
`gh`'s environment so the clone, not a shell variable, decides; then `gh api graphql --hostname
<host>` with one compile-time query, owner, name and cursor passed as raw strings. The query asks
for each pull request's node id, number, state, `updatedAt`, `mergedAt`, merge commit and up to 100
listed commit hashes with their total — nothing else. Pages of 50, newest update first, stop at the
first pull request updated before `--since` or after 20 pages; the walk then re-reads from the top
down to the newest update its first page held — the forge's clock, not the local one — so a pull
request that moved during the read is not lost, and duplicates are dropped by node id. Each call
has a 60-second timeout and an 8 MiB output cap. An `errors` field, a missing repository, a cap hit
or a failed page fails the command: there is no partial document. `gh` holds the credentials;
assaio reads, passes and stores no token.

**The observations** (ADR 0007). `scm.pull_request.observed`, grain `change`, id the forge's node
id, `occurredAt` its `updatedAt`: a snapshot of something that keeps changing, of which the latest
`observedAt` is current. `scm.pull_request.commit.observed`, grain `commit`, id `<node id>:<hash>`,
`occurredAt` the reading time with time source `ingest-time`, because the forge states no time for
a commit joining a pull request. Both `local-only`, both `parsed`.

**A listed commit is a candidate only when it was made here**: when a `HEAD` reflog of any of this
clone's worktrees records creating it — a commit, an amend, a rebase pick, a cherry-pick, a revert,
an applied patch or a merge git made — and not when HEAD merely moved onto it by a checkout, a
pull, a reset or a fast-forward. Git filters the entries and prints the hashes alone. Such a
commit is read by the same collector as the commits on `HEAD`, forge detection included. A listed
commit in the object store that no reflog records as made here, one absent from it, one git
cannot read, and one made here before the window are counted apart and never become candidates.
On a partial clone, a git older than 2.44 would fetch a missing object from the remote, so the
command refuses rather than read.

**session-commit/v3 is session-commit/v2 plus four rules**, and applies only to a document built
with pull-request data; without `--github`, `evidence` keeps session-commit/v2 byte for byte.
- R1: a listed commit made here is a candidate, judged by v2's rules.
- R2: each candidate and alternative is linked to every observed pull request that lists it
  (`listed`) or whose merge wrote it (`merged-as`): the merge commit, and for a rebase merge each
  commit on its first-parent line that the forge replayed, up to the pull request's commit count.
- R3: a result names one pull request (`change`) only when every candidate is linked and their
  pull-request sets share exactly one. Status, method, confidence and ambiguity follow v2's rules:
  grouping commits into a pull request is not new evidence about which commit holds a session's
  work. The summary fractions are computed over the candidate set R1 enlarged, so a v3 coverage
  figure is comparable only with another v3 document, never with a v2 one.
- R4: an unmatched session gets `listed-commit-not-readable-here`, not `no-commit-candidate`, when
  a pull request updated after it started lists a commit this clone cannot read.

**What the document shows.** A `changes` block, absent without `--github`: when the pull requests
were read and how far back, how many listed commits R1 added as candidates, the counts behind every
absence (lists cut at 100 — the newest commits are the ones cut —, listed commits no reflog records
as made here, not local, unreadable, made before the window), the sessions R3 placed in one pull
request and those whose candidates span several, and the pull requests a candidate or alternative
names, by number, with state, merge time and which of the two names it. ADR 0023 supersedes this
counting rule: its document counts pull requests read, named and unmatched within the read, while
listing only named pull requests. A candidate line names its pull requests and how it is
linked, never their state; state sits in the pull-request block, labelled as the forge reported it
when read and as the pull request's outcome, not a session's. Each link carries its own provenance
— stated by the forge — apart from the time relation assaio derived.

Rejected alternatives:
- **Object presence as the test.** On a clone with the default refspec it admits every teammate's
  branch, and a teammate's commit written during a session would become a medium overlap.
- **Any `HEAD` reflog entry as the test.** A checkout to review a teammate's pull request, or a pull
  of a shared branch, puts their commit there too.
- **The configured git identity as the test.** A stronger signal, but the first use of identity in
  attribution; it needs its own record, and a commit made under another address would drop out.
- **Commit dates from the API for every listed commit.** Reaches commits this clone never had,
  mixes provenance, and asks the forge for more.
- **Grouping one pull request's candidates into a `matched` result.** It raises resolved coverage
  with no new evidence and hides two people's commits in one pull request.
- **Asking the forge which pull request holds each candidate commit.** It tells the forge which
  commits sat near AI sessions (ADR 0020).
- **An HTTPS client holding a token.** Credential handling becomes assaio's to get wrong.

## Consequences
- A document built with `--github` is a snapshot: pull-request state moves, force-pushes change
  commit lists, and git expires reflog entries a ref no longer reaches after 30 days by default,
  so an older session's candidates can disappear on a later run. Determinism (ADR 0018) holds for
  the offline form.
- A worktree that has since been removed takes its reflog with it, so commits made only there are
  not candidates. In the maintainer's repository, where work runs in short-lived worktrees, most of
  the listed commits counted as not made here were made in such worktrees.
- Two of the user's own sessions running at once in sibling worktrees are as indistinguishable as
  two people: a commit made during both is a candidate for both.
- A fork's pull requests live on its parent; `gh`'s default repository decides which is read, and
  the command fails when none of the pull requests read touches this clone's history.
- The query runs as the user and can appear in an organization's API audit log.
- GitHub Enterprise Server is reached through `gh`'s host for the repository, untested; other
  forges are not reached. ADR 0023 supersedes the reviews-and-checks deferral: B229 adds bounded
  review states and current-head check contexts, with total and listed counts; B231 retains
  rounds, suites and historical runs, merge method, revert relations and derived rates.
- `internal/github` is a connector, not a log parser: the `internal/parser` contract does not apply
  to it, and its response parser still has a native fuzz target.
- The conformance corpus (ADR 0010) gains pull requests in its fixtures and one change-level
  expectation.
