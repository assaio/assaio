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
sessions from other stored projects, sessions with unavailable projects, and sessions under this
repository's name whose repository never resolved (`identityUnresolvedSessions`) separately,
excluding all three from that denominator.

On forge-merged history, coverage measures what commit timing can link, not what shipped: a session
whose work reached `main` in a squash can stay ambiguous or unmatched.

The document identifies algorithm `session-commit/v2`. Each result includes method, confidence,
provenance, ambiguity, reason, candidates, and alternatives. Each candidate includes its commit hash
and time, the time used to judge its relation (`evidenceAt`), plus the git observation's source,
time source, provenance, privacy class, and content-free file-category counts. The hash is git's
stable identifier and lets readers inspect the candidate in their local repository.

## Methods and confidence

The engine uses only evidence in the current contracts:

| Method | Result | Confidence | Rule |
|---|---|---|---|
| `manual-confirmation` | matched | high | A confirmed commit wins, even one the rules below exclude; other plausible candidates remain alternatives. The public command has no correction UI or ledger yet. |
| `project-time-overlap` | matched | medium | Every same-repository commit with an author or committer time inside the session remains linked, except a forge merge commit; only author time counts for a commit the forge rebased. A commit the forge landed never counts as an overlap. Several commits are a valid many-to-many answer, not ambiguity by count alone. |
| `project-time-following` | matched | low | When no commit overlaps the session, exactly one same-repository commit follows it, measured from the earliest of its times within 48 hours after the session. |
| `project-time-forge-landing` | matched | low | Exactly one candidate is a commit the forge landed while the session ran or up to 48 hours after it. Its time is the merge, an upper bound on when the work happened. |
| `project-time-following` | ambiguous | insufficient | When no commit overlaps the session, several following or landed commits fit the same bound; no available signal separates them. |
| `none` | unmatched | insufficient | No candidate or no usable evidence. The reason says which; `later-forge-landing` means a commit the forge landed after the 48-hour bound may hold the session's work. |

Git records two times for a commit: when it was written and when it reached its branch. A forge
rewrites them. A squash merge stamps both with the merge, a rebase merge keeps the written time,
and a merge commit is stamped at the merge and carries no lines. session-commit/v2 judges a forge's
rebase at the written time, treats a squash only as a low-confidence landing, never links a forge's
merge commit, and judges every other commit at both times. An amend can thus link the commit to
both the session that wrote it and the one that amended it. A merge made locally stays a candidate
because it can carry a conflict resolution. Only GitHub.com's own committer identity,
`noreply@github.com`, is recognised. Git applies that filter itself, so assaio reads no e-mail
address. GitLab, Bitbucket, Gitea and GitHub Enterprise Server commit as the person who merged, so
their merges are judged as local ones. A commit whose committer was set to that address by hand is
misread as the forge's; a personal `…@users.noreply.github.com` address is not. Each document's
`forge` block names its detection scope and counts landed and rebased commits, excluded merge
commits, and sessions left unmatched before a landing after their following window ([ADR 0021](adr/0021-commits-a-forge-writes.md)).

The 48-hour following boundary is inclusive; a commit one second later is excluded. The local git
collector sees only commits reachable from `HEAD`. Input order does not affect results: sessions and
commits are sorted by stable identifiers and times before matching.

## Privacy and limits

The command reads each stored session's id, tool, project name, the repository its rows resolved
to, and first and last timestamps.
Git observations contain only commit hashes, timestamps, parent and line counts, revert indication,
and counts across six file categories. The collector briefly reads paths and commit subjects to
derive categories and revert indications. Prompts, model responses, code, diffs, commit messages,
and branch names never appear in observations, edges, or output.

A session is a candidate only when its rows resolved to the repository `--repo` names ([ADR
0019](adr/0019-repository-identity.md)): another checkout with the same name is another project, and
a session whose repository never resolved — stored before assaio recorded repositories, or read
after its directory was gone — is counted in `identityUnresolvedSessions`, never matched. When no
stored usage resolved to that repository, the document's `project` is empty (the text shows `—`)
rather than another repository's name. Commit observations lack authors, so identity cannot
distinguish overlapping users; the conformance corpus requires that case to stay ambiguous. Output
has no member, person, score, or rank field and is not intended for performance evaluation.

A `matched` label is an attribution observation. It does not prove an AI session caused a commit or
show AI impact. v0.27 has no PR, review, CI, merge, attributable-survival, or other outcome
correlation. Those need new observations, connector privacy policy, and conformance cases.
