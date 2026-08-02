# 7. Canonical event contract for the evidence graph

## Status
Accepted (2026-08-02)

## Context
Everything assaio has stored for five releases is one shape: `usage.Record`, a per-turn or
per-session row read from an AI tool's log. The evidence graph (`B89`–`B94`, ROADMAP
"Evidence graph") needs shapes that row cannot hold — a commit, a pull request, a review
round, a check run, a merge, a revert — produced by collectors that are not parsers (a local
git reader, a GitHub connector) and read by analyzers that must not know which of them
produced what.

With no common envelope, every collector invents its own record and every analyzer learns
every one. Worse, the fields that make this project defensible — where a fact came from, when
it actually happened, how coarse it is, whether it is safe to send anywhere — get re-invented
per source and drift apart. That is already visible: granularity lives on `usage.Record`,
freshness and parser build live on `store.Provenance`, per-source capability lives on
`parser.Depth`, and an analyzer reaches through `store` to get any of it.

`usage_record` is deliberately **not** what this migrates. It has a dedupe contract, a
migration history, and 119,896 rows on the maintainer's machine alone; reshaping it into
"events" would be a large, risky change that buys this milestone nothing.

## Decision
A canonical event is a small versioned envelope wrapping one closed payload. It is the shape
collectors emit and the shape `AnalyzerContext` (`B90`) will serve. Today's `usage.Record`
adapts into it; nothing else is rewritten.

- **An event is an interface contract, not a storage format.** There is no event table and no
  migration in this decision. Storage stays per domain — the git collector (`B91`) gets its
  own observation table, the connector (`B92`) its own — and events are what crosses the
  boundary between producing and consuming code. An event log would have meant at least one
  row per usage record, doubling a 38 MB store to carry a second copy of what it already
  holds, and SQLite never returns those pages to the filesystem on `DELETE` (`compact`
  exists precisely because of that). A contract that costs a `VACUUM` to adopt is the wrong
  contract.
- **Every event type is a past-tense observation**, `<domain>.<thing>.observed`. The
  committed vocabulary is `ai.session.observed`, `ai.usage.observed`, `ai.edit.observed`,
  `vcs.commit.observed`, `scm.pull_request.observed`, `scm.review.observed`,
  `ci.check.observed`, and `delivery.merge|revert|survival.observed`. `B89` sketched the
  first as a bare `ai.session`; it is renamed here, because a versioned contract is cheap to
  correct now and expensive later. An *observation* is all any of these ever are — a claim
  about a link between two of them is an attribution edge (`B85`), a different thing with
  different fields.
- **This document commits the vocabulary; the code registers what it can actually produce.**
  `internal/event` today knows `ai.usage.observed` and `ai.edit.observed`, because those are
  what the `usage.Record` adapter emits. Each remaining type lands with the collector that
  produces it — one registry line and one payload struct — rather than shipping now as an
  empty shape nobody fills. Declaring a name is a commitment; declaring a struct with no
  producer is speculative abstraction.
- **The envelope, v1**: `spec_version`; `type`; `id`; `source` as `{name, version, build}` —
  who produced it, the source's own format or API version, and the assaio build that read it;
  `occurred_at` and `observed_at`; `time_source`; `grain`; `privacy`; `provenance`; and a
  `subject` of `{project, session, member}`. Each payload is one closed struct per type.
- **`id` is the source's own key, never a generated one.** Re-reading an artifact must
  produce the same event, so the adapter derives it from the parser's existing dedupe key.
  Idempotency is a property of the identifier, not of a de-duplicating consumer.
- **Two clocks, and a field saying how much to trust them.** `occurred_at` is when the thing
  happened per the source; `observed_at` is when assaio read it. `time_source` states which
  of `source-stated`, `file-mtime` or `ingest-time` produced `occurred_at`, because those are
  three very different qualities of evidence and only the first is the source's own claim.
  This is also where `B79` stops being invisible: tool logs carry `Z` timestamps with no
  offset, so an event knows an instant and never a local calendar day.
- **`provenance` says how the event came to exist**: `parsed` from a source artifact,
  `derived` by assaio from other events, or `manual` from a person. Estimated and attributed
  are deliberately absent — they describe signals (`B90`) and edges (`B85`), not observations.
- **`privacy` is a field, and content is impossible by construction.** Three classes:
  `local-only` (must never leave the machine), `pseudonymous` (syncable once identities are
  pseudonymized), and `public-metadata` (says nothing about a person or a repository's
  contents). Payloads carry only numbers, closed-vocabulary values, and identifiers the
  source itself assigned — there is no free-text field to put a prompt, a diff, a branch name
  or a commit message in, and a test asserts that as the contract grows. The classes are the
  hook the correlation threat model (`B100`) turns into a field-level sync policy; this
  decision defines the vocabulary and no policy.
- **Confidence is not an envelope field.** An observation either happened or was not
  observed. The confidence envelope (`B81`) belongs to a verdict, and attribution confidence
  to an edge; putting a number here would invite an analyzer to average two things that mean
  different things. What an event does carry is the raw material confidence is computed
  *from* — grain, time source, provenance, source version.
- **Reuse, never a second source of truth.** `grain` extends `usage.Record.Granularity`
  rather than restating it; `source.build` is what `store.Provenance` already reports as
  `ParsedBy`; per-source capability stays in `parser.Depth` and is referenced, not copied.
- **Versioned, and validated at the boundary.** `spec_version` is an integer; fields may be
  added within a version, and removing or retyping one is a new version. Every event is
  validated where it is constructed — unknown type, unknown vocabulary value, empty
  identifier, missing clock or negative count is rejected rather than carried. This is the
  same posture as the exec plugin protocols (ADR 0003–0005): reject, never fabricate.
- **The two clocks are not ordered against each other.** A draft of this decision rejected
  `occurred_at` after `observed_at` as a reversed clock. Run against 324,289 real records it
  dropped 51, every one of them from a session still being written while the pass ran:
  `observed_at` is a *reading* time stamped once for a batch, so a file read late legitimately
  holds records newer than the stamp. The rule would also have made the evidence path stricter
  than the store, dropping records the store keeps — two disagreeing views of the same data is
  worse than either policy alone. A vendor's clock is not ours to police; what a consumer gets
  instead is `time_source`, which says how much the timestamp is worth.

Rejected alternatives:
- **Migrate `usage_record` into an event table.** The honest version of this milestone's data
  model, and far too large a change to make before a single consumer exists. It also puts a
  schema migration in front of every future collector, which is exactly the coupling the
  contract is meant to remove.
- **Persist events beside the existing tables.** Doubles the store to hold a second copy of
  what it already has, and makes retention a two-place problem. If a collector later needs
  durable events, that is its own decision with its own size bound and cleanup path.
- **Adopt OpenTelemetry GenAI conventions as the internal model now.** The mapping is
  planned (`B98`), but OTel's model has no room for attribution ambiguity or an evidence
  grain, and adopting an external schema before knowing what this milestone needs would hand
  the shape of the product to a spec written for tracing.
- **One event type per source** (`claude.turn`, `github.pr`). Every analyzer would then learn
  every source, which is the coupling this exists to prevent.

## Consequences
- A collector is now a function producing events, and can be written and tested with no
  store, no SQL and no migration. That is what makes `B91` and `B92` small.
- Adding an event type is one registry line plus one payload struct. Adding a *field* to the
  envelope is a contract change, and adding one after `v1.0` is a semver event (`B23`).
- The adapter means one `usage.Record` becomes one or two events: usage always, and an edit
  observation only when the record carries line, edit or tool-call activity. A source with no
  activity extraction therefore emits *no* edit observation rather than one full of zeros —
  which is the same honesty `parser.Depth` states, now structural.
- Nothing user-facing changes in this step. There is no new command, no new column, and no
  figure moves; the contract earns its keep only once `B90` serves it and `B91` fills it.
- Events are in-process only today. Publishing them — over sync, to a plugin, or as OTel —
  is a separate decision, and the privacy classes are what that decision will be argued in.
