# Local session-to-commit evidence

`assaio-agent evidence` is the first small Evidence Graph surface. It compares sessions in
the local store with commits reachable from `HEAD` in one local repository:

```console
$ assaio-agent evidence --repo . --since 30d
$ assaio-agent evidence --repo ../service --format json
```

It makes no network request, accepts no `--db` override, refuses a store containing team
member rows and writes nothing. Repeating the command re-reads the same inputs and produces
the same attribution answer; neither commit observations nor derived edges are persisted.

## What one result means

Every project-scoped session has exactly one outcome:

- `matched` — the available evidence supports one or more candidates without forcing a
  many-to-many relation into one winner;
- `ambiguous` — at least two candidates carry equivalent available evidence, so all remain;
- `unmatched` — no candidate is supported, the following window is still open, or the
  session has no usable time window.

The summary reports two fractions over the same project-session population. Candidate coverage
is `(matched + ambiguous) / population`; resolved coverage is `matched / population`. Sessions
for another stored project and sessions whose project is unavailable are named separately and
never silently added to that denominator.

The document stamps algorithm `session-commit/v1`; each result states method, confidence,
provenance, ambiguity, reason, candidates and alternatives. Each candidate carries the commit hash and time plus the
git observation's source, time source, provenance, privacy class and content-free file-category
counts. A hash is shown because it is the source's own stable identifier and lets a local reader
inspect the candidate in their own repository.

## Methods and confidence

The engine uses only evidence the current contracts actually carry:

| Method | Result | Confidence | Rule |
|---|---|---|---|
| `manual-confirmation` | matched | high | A confirmed commit wins; other plausible candidates remain alternatives. The public command has no correction UI or ledger yet. |
| `project-time-overlap` | matched | medium | Every commit in the same stored project whose time falls inside the session remains linked. Several commits are a valid many-to-many answer, not ambiguity by count alone. |
| `project-time-following` | matched | low | When no commit overlaps the session, exactly one same-project commit occurs after it and no later than 48 hours after it. |
| `project-time-following` | ambiguous | insufficient | When no commit overlaps the session, several following commits fit the same bound; no available signal separates them. |
| `none` | unmatched | insufficient | No candidate or no usable evidence. The reason says which. |

The 48-hour following boundary is inclusive. A commit one second beyond it is not a candidate.
Commits unreachable from `HEAD` are absent because the local git collector does not observe
them. Input order does not change the answer: sessions and commits are ordered by their stable
identifiers and times before matching.

## Privacy and limits

The command reads the stored session id, tool, project basename and first/last timestamps. From
git it returns only commit hashes, timestamps, parent and line counts, revert indication and a
six-way file-category count. Paths and commit subjects are read transiently by the collector to
derive categories and the revert indication; prompt text, model responses, code, diffs, commit
messages and branch names are never part of an observation, edge or output.

Project identity is only the basename already stored by assaio. Two repositories with the same
basename cannot be separated by this method. Commit observations carry no author, so identity
cannot separate overlapping users; the conformance corpus requires that case to remain
ambiguous. The output has no member, person, score or rank field and is not intended for
performance evaluation.

A `matched` label is an attribution observation, not proof that an AI session caused the commit
and not evidence of AI impact. v0.27 has no PR, review, CI, merge, attributable-survival or other
outcome correlation. Those require new observations, connector privacy policy and their own
conformance cases.
