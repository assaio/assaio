# 21. A commit a forge wrote is judged by the time that means work

## Status
Accepted (2026-09-30)

## Context
`evidence` (ADR 0018) matched a session to the commits reachable from `HEAD` by the time git
reports for each commit — its committer time. That is when a commit reached its branch, which is
when the work happened only if the author committed straight onto the branch. The three ways a
forge brings a pull request onto the main line all break that:

- a **squash merge** writes one new commit whose author and committer time are both the merge
  (checked on this repository: GitHub.com stamps both, and commits as `GitHub
  <noreply@github.com>`); the branch commits are not reachable from `main`;
- a **rebase merge** replays each branch commit, keeping its author time and stamping the merge
  as its committer time;
- a **merge commit** joins the branch with a two-parent commit that carries no lines, stamped at
  the merge.

Local rewrites do the same to one commit: `--amend`, a local rebase and a cherry-pick keep the
author time and move the committer time. The conformance corpus (ADR 0010) held none of these, so
session-commit/v1 was never judged on them. Run against seven new scenarios it links a forge-merge
to whichever session was running at the click, gives a rebase-merged commit to that session
instead of the one that wrote it, and drops the session that wrote a commit later amended. On
this repository, which squash-merges every pull request, it rated all eleven sessions of a month
medium-confidence overlaps; 35 of their 42 candidate commits were squash commits GitHub wrote at
the merge.

## Decision
**The collector reads two more facts, and keeps the committer time as the observation's time.**
The commit payload gains `authoredAt` and `committedByForge`. The envelope's `occurredAt` stays
the committer time, so `survival`'s ages and the `--since` window do not move. The forge flag is
read by git itself (`git log --fixed-strings --regexp-ignore-case
--committer=<noreply@github.com> --format=%H`): assaio receives hashes, never a name or an
e-mail. The angle bracket keeps a user's own `…@users.noreply.github.com` address out.

**session-commit/v2 judges each commit by the time that means work:**
- a forge commit with two or more parents is not a candidate — the branch commits it joins carry
  the work, and they are observed too;
- a forge commit written and committed at one moment (a squash; also a web edit, the Revert
  button, an applied review suggestion) was stamped when it landed. It stays a candidate for a
  session running then or ending up to 48 hours before, as `landed-by-forge`, and never above low
  confidence: the time is an upper bound on when the work happened, not evidence of who did it;
- a forge commit written earlier (a rebase merge) is judged at its author time;
- any other commit is judged at both of its times, so an amend reaches the session that wrote the
  commit and the one that amended it;
- a local merge commit stays a candidate: git reports no lines for any merge, and one made by
  hand can carry a conflict resolution;
- a session with no candidate while a forge-landed commit lies after its window gets the reason
  `later-forge-landing`, not `no-commit-candidate`: its work may have shipped in that commit.

**Detection is stated, never implied.** Only GitHub.com's own committer identity is recognised.
GitLab, Bitbucket, Gitea and GitHub Enterprise Server commit as the person who merged, so their
merges are judged as local ones; a user who sets that address as their own committer is misread
as the forge. Every `evidence` document names its detection scope and counts the landed and
rebased commits, the excluded merge commits and the sessions left before a later landing, and its
text says that on forge-merged history coverage measures what commit timing can link, not what
shipped.

Rejected alternatives:
- **Use the author time as the observation's time.** It moves `survival`'s commit ages and puts
  commits older than `--since` into the window, and it drops the session that amended a commit.
- **Exclude squash commits.** On a repository that squash-merges minutes after the session that
  wrote the work, that session's only evidence is the squash; excluding it would leave it
  unmatched where v1 was right.
- **Ask the forge which pull request holds each commit.** It needs the network and tells the
  forge which commits sat near AI sessions (ADR 0020); it is `B92`'s job, with its own policy.

## Consequences
- The corpus holds seventeen scenarios. Its stand-in engines now include one that trusts only the
  committer time and one that trusts only the author time; each must fail a landing scenario.
- `evidence` output moves on any repository a forge merges into: fewer medium links, more
  ambiguous and low ones, and a new relation, method, reason and document block. A
  `session-commit/v1` document from such a repository should be discarded, not compared.
- A confirmation keyed by a branch commit's hash does not follow the commit through a forge
  rebase or squash; it stays `confirmed-commit-unobserved`.
- Stacked pull requests, and any link from a squash to the branch commits it contains, need pull
  request data (`B92`).
