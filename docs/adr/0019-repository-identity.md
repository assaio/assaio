# 19. A repository's identity is a local key beside its name, and a name two repositories share is split

## Status
Accepted (2026-09-29)

## Context
Every usage row stored the basename of its repository root as `project`
(`internal/ingest/project.go`), and nothing else about the repository. Two unrelated checkouts
named `api` therefore shared every aggregate, and every join that starts from a directory joined
both: `survival` summed both repositories' AI lines, `evidence` offered both repositories' sessions
as candidates for one repository's commits, and `mark` could label the other checkout's session. A
pseudonym is derived from the name, so no later export could separate them either.

The evidence graph (ROADMAP milestone 2) needs to know which repository a session ran in before it
joins anything across repositories; temporal proximity does not answer that. The store may not hold
a working-directory path (PRIVACY.md), and nothing that leaves the machine may carry a path or a
hash of one.

## Decision

**A local key names a repository root.** Ingest resolves a session's working directory to its
repository root as before (a worktree to its main checkout) and, only when the directory exists and
a `.git` entry was found, derives `projectid.Key`: `v1:` and 128 bits of an HMAC-SHA256 over the
root's canonical spelling — symlinks resolved, each component spelled as its parent directory lists
it, so a case or Unicode variant of one path is one key. The HMAC is keyed by a 32-byte salt stored
in the store itself (`repository_salt`), so a restored backup keeps its keys and rotating
`pseudonym.key` does not split history. The version prefix lets a later derivation coexist with this
one.

The key names a root *on this machine*, not an upstream project. Two clones are two keys, and a
checkout moved to a new path is a new key. The alternatives each merge what should stay apart or
cost more than they return: an HMAC of the root commits merges two products started from one
starter repository, needs a full history walk per root at ingest, has no answer before the first
commit and gives a shallow clone another key; a remote URL is absent in most local repositories and
still needs the path as a fallback. Splitting is the direction that never adds a stranger's usage to
a total.

**Rows carry a registry id, not the key.** `repository(id, label, repo_key, ordinal)` holds each
(label, key) once; `usage_record.repo_id` is `0` for an unresolved row (no working directory, no
repository above it, a directory gone when read, a synced or plugin record, a row stored before
v0.35.0), `-1` for an ambiguous one (a completed sub-agent aggregate two reads assigned to two
repositories; sticky), and otherwise the registry id. The text key lives once per repository: on the
maintainer's 220,856-row corpus a text column measured +8.4 MB against +0.4 MB for the integer.
With the partial index the liveness check needs, the store grew 2.9 MB on 221,168 rows, 2.3 MB of
it the index: about 13 bytes per row whose repository resolved.

**One rule names what a reader sees**, in one SQL CTE (`store.shownNames`) every read joins. Only
repositories some row still carries count. A label with one such repository is shown as the label on
every row, resolved or not — the store holds no evidence of a second one, which is the presumption
every earlier release made. A label two or more share is split: the first by registration keeps the
bare label, so an established name never renames; the rest are `label (n)`; rows that cannot be
placed under one of them are `label (?)`. A generated name some row already carries as a label
becomes `label (#n)`, so one name never stands for two repositories. Ordinals are assigned once and
never reused; the bare name moves to the next repository only when every row of the first is
deleted, and `digest` records which repositories each split name stood for, so that shows as a
caveat. A report row takes the name of its own row's repository; a session-grain surface names a
session from the one repository its rows resolved to, so an unresolved row inside a resolved
session counts under that repository there and under `(?)` in a report total.

**Joins from a directory use the key.** The directory resolves the way ingest resolves a session
(a worktree to its main checkout), and the store looks the key up first, so another spelling of the
path finds the rows its canonical spelling stored. `evidence` makes a session a candidate only when
its rows resolved to that repository and counts sessions under the name without one as
`identityUnresolvedSessions`; `survival` counts only AI lines resolved to it and prints the
unresolved lines under the name as not counted; `mark` targets the newest session with a row
resolved to it. The presumption that folds unresolved rows into a name's only repository serves
totals and never a join. When nothing resolved to the repository, `evidence` names no project,
`survival` prints `—`, `mark` refuses, and each says so instead of borrowing another repository's
usage or name.

**Disclosure is deliberate and local.** No surface prints the key. `assaio-agent repos` lists the
names and, given directories, prints the name each one's usage is stored under: the user supplies
the path, and nothing is stored or sent.

**The team server learns nothing new.** The key and the id never leave the machine; `usage.Record`
tags the key `json:"-"`, so `sync` cannot send it. The server keeps pooling by name, as before, and
`sync` warns when two local repositories share a name it is about to send. An explicit mapping of
local repositories to team names is deferred (`B226`): a rolling re-push relabels only its window,
and keyless rows would keep the raw name, so one repository would appear under two names.

## Consequences
- `clear --all` empties the registry and draws a new salt, so a store with no usage keeps no
  repository name or keyed hash of a path; a scoped `clear` keeps both, so numbering holds.
- Migration 0014 adds the column, the registry and the salt, and clears the ingest watermarks of the
  sources that record a working directory, so the next `backfill` restates every row whose
  transcript is still on disk. History whose transcript is gone stays unresolved forever; under a
  split name it is shown as `(?)`.
- A store restored on a machine where the same repositories live at other paths gives them new keys,
  and every name that also appears under the old keys splits.
- The store file itself does not protect the key: its salt sits beside it, and `ingest_file` already
  holds session-log paths. Anyone holding the file can test a guessed path. The key protects what
  leaves the store.
- A new join across repositories inherits the rule: it may not match through an unresolved or
  ambiguous identity, and the `B211` snapshot may report identity as unresolved.
