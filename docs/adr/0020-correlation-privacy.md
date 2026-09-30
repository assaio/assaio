# 20. Correlating sessions with delivery data stays on the machine that ran them

## Status
Accepted (2026-09-30)

## Context
Until the evidence graph, everything assaio held was about AI usage: counts, models, times, a
project name. Correlation adds a second party. A session joined to a commit, and later to a pull
request, review or check run, says which piece of shipped work an AI session sat near — and the
delivery side is readable by anyone with access to the repository, including who authored each
change. ADR 0007 defined the privacy classes an observation carries (`local-only`,
`pseudonymous`, `public-metadata`) and deferred the policy to `B100`. ADR 0009 and ADR 0018 made
commit observations and session→commit results `local-only` and never stored, by choosing the
strictest answer rather than deciding one.

Designing the GitHub connector (`B92`) showed why a policy has to come first:

- A pull-request number or a commit hash is one lookup from its author on a forge the reader can
  open. "No person is fetched" does not mean "no person is identifiable".
- `sync` already sends each row's branch name to the team server. Anyone holding the server's
  data and read access to the repository can join a branch and a time to the pull request built
  from that branch, and to its author — re-identifying a member pseudonym without assaio ever
  shipping an edge. The pseudonym itself is an unkeyed hash of hostname and OS user name, which
  anyone holding the server's data can confirm by guessing, and a synced session id joins to an
  account wherever the coding tool's own telemetry records both.
- A query to a forge is itself a disclosure: asking which pull requests contain *these* commits
  tells the forge which commits sat near AI sessions.
- A per-session list of "an AI session touched this change" is one sort away from a ranking of
  people, which the product refuses in every form.

## Decision

**Correlation data is `local-only`.** Commit observations, pull-request, review and check
observations, and every edge between a session and any of them are computed on the user's
machine, printed to its own stdout, and never stored, synced, exported by `report`, rendered on
a dashboard or a `share` artifact, or sent to a metric or rule plugin. `evidence` keeps no
`--db` and refuses a store holding member rows.

**A connector runs only when asked, as the user.** A command that reaches a forge must do so on
an explicit per-invocation flag; without it the command is byte-identical to its offline form, so
local analysis stays offline-capable. It goes through the user's own client (`git`, `gh`), which
holds the credentials; assaio reads, stores and passes no token. The request carries nothing
derived from the store: it names a repository and a window, never the commits or sessions the
store associates with AI. The query names only allowlisted fields — no title, body, branch name,
label, comment text, check name, author or reviewer — and the connector's own test holds its
query to that list.
The threat model's network inventory lists every client assaio executes beside the `net/http`
users, because an exec'd client opens sockets that a grep for `net/http` cannot see; `git` runs
with lazy fetching disabled, so a read of a partial clone never reaches the remote.

**Field-level sync policy.** A field may leave the machine only as ADR 0007's `pseudonymous`
class allows, and a new field needs PRIVACY.md and the threat model changed in the same commit;
a test pins the set of fields a synced record carries, so none is added unnoticed. A field that
joins to an identity-bearing record outside assaio — a branch name, a commit hash, a
pull-request number, an issue key — and a member label must leave only as a digest keyed so the
server cannot reverse it, or not at all. Today `sync` breaks this rule twice: it sends branch
names as they are, and its member label is an unkeyed hash. `B227` brings both under the rule;
until it ships, PRIVACY.md and the threat model name those re-identification paths, and the
session-id join as well.

**Retention.** Nothing correlated is stored today, so there is nothing to retain. The change that
first stores a correlation observation or an edge (`B103`) ships its horizon, its `clear` scope
and its size bound in the same commit, the way `trace.horizon_days` bounds the step timeline.

**Server views.** No server view of correlation data exists. One that is added must aggregate over
at least five members, show no member's own changes, sessions or edges at any cohort size, and
rank nobody. The team panel that ships today has no such guard — it shows each member's session
bar at any team size — and `B228` adds it.

**No ranking surface.** No correlation output carries a person's name, login, e-mail, role or a
rank, score or percentile; a structural test walks the `evidence` document's serialized field
names, nested types included, and fails on one — it checks names, not values, which is why the
document carries no free string from a forge. Pull requests, once observed, must be counted and
listed by number for local inspection and never ordered by who made them.

## Consequences
- `B92` builds against this: a repository-and-window query through `gh`, allowlisted fields,
  results `local-only`, no pull-request data in any stored or synced shape.
- What stays true after this record, and is stated rather than solved: a pull-request number or
  commit hash printed locally identifies work to anyone who can read the repository; on a
  repository owned by a user, the repository path itself contains that user's login; and a
  small team's timing is identifying on its own. The defaults keep all of it on the machine.
- Sending a branch name, a hash or a number anywhere, or storing an edge, is now a decision that
  names this record, not a side effect of a feature.
