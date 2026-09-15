# 18. Local session-to-commit evidence is recomputed, content-free and allowed to abstain

## Status
Accepted (2026-09-15)

## Context
The git collector already emits content-free commit observations (ADR 0009), and the
attribution corpus already states the answers an honest engine must preserve (ADR 0010).
Neither gave a user a join they could inspect. Building the whole session→PR→review→CI→merge
funnel would require a connector and the correlation threat model it implies; persisting a
graph first would require a migration and retention contract before the useful edge was known.

The smallest public question is narrower: for sessions associated with one local repository,
which locally reachable commits are candidates, where is the evidence ambiguous, and how much
of the session population has no candidate?

## Decision
`assaio-agent evidence` computes algorithm `session-commit/v1` in one local pass.

- The command reads aggregated sessions from the default local store and fresh commit
  observations from one `--repo`. It has no `--db`, rejects member-bearing rows, makes no
  network request and persists neither observations nor results.
- Project basename and time are the only matching signals available. Every commit inside a
  session is retained as a medium-confidence many-to-many candidate. If none overlaps, exactly
  one same-project commit in an inclusive 48-hour following window is a low-confidence match;
  several remain ambiguous with insufficient confidence. No candidate remains unmatched.
- A manual confirmation in the engine contract is high-confidence manual provenance and wins
  over the heuristic while keeping plausible alternatives. v0.27 exposes no correction UI or
  durable correction ledger.
- Every result carries status, method, confidence, provenance, ambiguity, reason, candidates
  and alternatives. Population, candidate coverage and resolved coverage are explicit.
- The collector's content-free boundary is inherited unchanged. Output can carry a session id,
  project basename and commit hash because it is local inspection, but no member/person, path,
  branch, message, diff, code, prompt or response field exists. No ranking surface exists.
- A match is described as an observation from project and time proximity, never a causal claim
  or evidence of AI impact.

The production engine runs against all ten existing conformance scenarios. The
`overlapping-users` case remains ambiguous because a commit observation carries no identity;
one-session-many-commits and many-sessions-one-commit remain many-to-many; manual confirmation
survives replay; and input order cannot change the answer.

## Consequences
- The first Evidence Graph slice is useful without a connector, schema migration or public
  protocol change, and a rerun is idempotent because it writes nothing.
- A store created by v0.26.1 needs no new migration for this feature. The B101 project-conflict
  correction lands first, so contradictory project claims become an explicit unavailable
  population rather than a false edge.
- Repository basenames can collide. The command states this limit; stronger project identity
  belongs with durable commit identity and the correlation privacy decision.
- Durable observations and corrections, explicit markers, branch/identity/category methods,
  PR/review/CI/merge observations, outcome signals and the full funnel remain open work.
