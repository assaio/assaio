# Local session-to-commit evidence

`assaio-agent evidence` is the first small Evidence Graph interface. It compares local-store
sessions with commits reachable from `HEAD` in one local repository:

```console
$ assaio-agent evidence --repo . --since 30d
$ assaio-agent evidence --repo ../service --format json
```

It makes no network requests, accepts no `--db` override, rejects stores with team member rows, and
writes nothing. Repeating it reads the same inputs and yields the same attribution answer. It
persists neither commit observations nor derived edges.

## What one result means

Each project-scoped session has exactly one outcome:

- `matched` — available evidence supports one or more candidates without choosing one from a
  many-to-many relationship;
- `ambiguous` — at least two candidates have equivalent available evidence, so all remain;
- `unmatched` — no candidate is supported, the following window remains open, or the session lacks a
  usable time window.

The summary uses the same project-session population for both fractions: candidate coverage is
`(matched + ambiguous) / population`, and resolved coverage is `matched / population`. It reports
sessions from other stored projects and sessions with unavailable projects separately, excluding
both from that denominator.

The document identifies algorithm `session-commit/v1`. Each result includes method, confidence,
provenance, ambiguity, reason, candidates, and alternatives. Each candidate includes its commit hash
and time, plus the git observation's source, time source, provenance, privacy class, and
content-free file-category counts. The hash is git's stable identifier and lets readers inspect the
candidate in their local repository.

## Methods and confidence

The engine uses only evidence in the current contracts:

| Method | Result | Confidence | Rule |
|---|---|---|---|
| `manual-confirmation` | matched | high | A confirmed commit wins; other plausible candidates remain alternatives. The public command has no correction UI or ledger yet. |
| `project-time-overlap` | matched | medium | Every commit in the same stored project whose time falls inside the session remains linked. Several commits are a valid many-to-many answer, not ambiguity by count alone. |
| `project-time-following` | matched | low | When no commit overlaps the session, exactly one same-project commit occurs after it and no later than 48 hours after it. |
| `project-time-following` | ambiguous | insufficient | When no commit overlaps the session, several following commits fit the same bound; no available signal separates them. |
| `none` | unmatched | insufficient | No candidate or no usable evidence. The reason says which. |

The 48-hour following boundary is inclusive; a commit one second later is excluded. The local git
collector sees only commits reachable from `HEAD`. Input order does not affect results: sessions and
commits are sorted by stable identifiers and times before matching.

## Privacy and limits

The command reads each stored session's id, tool, project basename, and first and last timestamps.
Git observations contain only commit hashes, timestamps, parent and line counts, revert indication,
and counts across six file categories. The collector briefly reads paths and commit subjects to
derive categories and revert indications. Prompts, model responses, code, diffs, commit messages,
and branch names never appear in observations, edges, or output.

Project identity uses only the basename assaio stores, so this method cannot distinguish
repositories with the same basename. Commit observations lack authors, so identity cannot
distinguish overlapping users; the conformance corpus requires that case to stay ambiguous. Output
has no member, person, score, or rank field and is not intended for performance evaluation.

A `matched` label is an attribution observation. It does not prove an AI session caused a commit or
show AI impact. v0.27 has no PR, review, CI, merge, attributable-survival, or other outcome
correlation. Those need new observations, connector privacy policy, and conformance cases.
