# 9. The local git evidence collector observes commits, and stores none of them

## Status
Accepted (2026-08-03)

## Context
[ADR 0007](0007-canonical-event-contract.md) committed `vcs.commit.observed` as a name and
said each type lands with the collector that fills it. `B91` is that collector: the first
producer of observations assaio does not parse out of an AI tool's log, and the first test of
whether the contract earns its keep — until now `internal/event` had no caller in the binary
at all.

Two things about a repository make this different from a session log. Its content is the
thing assaio promises never to collect, and a commit is the unit an attribution edge (`B85`)
will point at, so what the collector records now decides what the evidence graph can say
later. ADR 0007 also anticipated that this collector "gets its own observation table"; that
turned out to be a decision worth making separately rather than inheriting.

## Decision
A collector, not a store. `internal/vcs` reads a repository and returns
`[]event.Event`; nothing is persisted.

- **Content-free by construction, categories rather than paths.** A commit observation carries
  parent count, changed-file count, added and removed lines, a six-way file-category split
  (test / source / docs / config / generated / other) and a revert flag. There is no field for
  a path, a branch name, a commit message or a diff, and the contract's own test enforces that
  as the payload grows. The categories are a **naming heuristic**, stated as one wherever they
  are printed: `app_test.go` is a test because of its name, not because anything read it.
- **`other` is a category, not a fallback.** An image or a binary is neither source nor
  config, and folding it into source would inflate the number a reader is most likely to act
  on. The split must sum to the file count beside it — the payload refuses to validate
  otherwise, so four categories can never be read as the whole of a commit.
- **The revert flag reads the message and keeps none of it.** Git writes `Revert "…"` for a
  revert it generates; the collector tests that prefix and discards the subject, the same way
  a parser reads a diff to count lines. An undo phrased any other way is invisible, which is
  why the field is an indicator rather than a count.
- **`local-only` is the privacy class, and needs no argument to be right.** Repository
  identity plus timing is exactly what the correlation threat model (`B100`) exists to reason
  about. The strictest class is the answer that costs nothing today and does not pre-empt that
  decision.
- **A commit is its own grain.** `grain` gains `commit` beside `turn` and `session`. Reusing
  `session` would let a consumer average two things that are not the same unit, which is the
  mistake the field exists to prevent.
- **The id is the commit hash.** Idempotency is a property of the identifier (ADR 0007), and a
  hash is the source's own key, so re-reading a repository produces the same observations
  rather than a second set. A keyed pseudonymous digest is what a *syncable* variant would
  need; it lands with the connector and the threat model, not before.
- **Skip and count, like every parser.** A commit whose header this build cannot read is
  counted and skipped rather than aborting the pass, and the count is printed — a short
  history is never quietly short.
- **No observation table yet.** ADR 0007 expected one; nothing needs it. `survival` collects
  and consumes in one pass, and persisting commits would mean a migration, a size bound and a
  cleanup path (the store-size discipline `compact` exists for) in exchange for nothing a user
  can see. The requirement that decides the shape is `B85`'s: attribution edges need commits
  queryable across passes, and that is when the table gets designed — against a consumer,
  not ahead of one.
- **`survival` becomes the first consumer.** It no longer shells out to git for its commit
  set: it reads observations, and reports what the window changed by category beside the
  survival rate. Paths still exist for the blame pass — `vcs.TouchedFiles` is documented as
  local-only and its output never reaches a `Result` — because blame has to name a file.

Rejected alternatives:
- **Store paths behind an opt-in flag.** `B91` allows it; nothing needs it. A flag that
  turns a content-free dataset into a path-level one is a privacy decision, and adding it
  before a consumer exists means deciding it in the abstract.
- **Emit one observation per changed file.** Truer to the data and far more of it: a 32-commit
  window here is 793 file changes. The category split answers the questions `B94` asks at a
  fraction of the size, and per-file evidence can be added later without moving the commit.
- **Reuse `session` grain.** See above; a wrong unit is worse than a new vocabulary value.

## Consequences
- `internal/event` has a caller. The contract's validation now runs on real data on every
  `survival` run instead of only in its own tests.
- Adding the next collector (`B92`, the GitHub connector) is one registry line, one payload
  struct and one package — the shape ADR 0007 promised, now demonstrated once.
- Three requirements for `AnalyzerContext` (`B102`) came out of building this rather than
  being guessed at, and are recorded on that item: analyzers receive `[]store.UsageRow`, not
  observations, so no validator can read a commit today; `signals coverage` weights support by
  tokens, so a source that has none cannot appear in it at all, which is why no `vcs.*` signal
  is registered yet; and capability lives in `internal/parser`, which a collector is not.
- One posture is still split and should be settled when the adapter gains a caller: this
  collector skips and counts an observation it cannot build, while `event.FromRecord` returns
  an error for the whole batch on the first record the contract rejects. Both are defensible;
  having both is not, and the log parsers' skip-and-count is the precedent.
- The category heuristic is a published behaviour the moment it is printed. Changing how a
  path is classified changes a reported number, so it is a change with a changelog entry, not
  an implementation detail.
