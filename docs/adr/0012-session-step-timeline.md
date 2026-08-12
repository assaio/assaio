# 12. The session step timeline

Date: 2026-08-12

## Status

Accepted.

## Context

Every figure assaio publishes is an aggregate, and an aggregate cannot say *why* a session was
expensive. The logs already carry the order — plan, search, read, edit, failure, retry,
compaction, abandon — and nothing reads it. `B147` proposed storing that order; this record
fixes the shape it takes, because three of the decisions below were made against measurements
that contradicted the proposal, and one of them contradicted an earlier draft of this record.

## Decision

**A step is content-free, and the thing it acted on is an integer.** A step carries its kind
and outcome from closed vocabularies, its position, the model, and a token cost. The one field
that stands for a file is a small integer assigned in first-seen order within one *timeline* --
never across a session, since a session's main transcript and each of its sub-agents are
separate files and `3` in one is unrelated to `3` in another. It is deliberately not a digest of
the path: a digest is reversible by anyone holding the
repository, since paths carry almost no entropy. Repetition stays visible; identity does not
survive parsing.

**The timeline is part of a step's identity, not only its ordering.** A sub-agent transcript
records its *parent's* session id, so numbering per file and keying per session put 222 steps at
position 1 on the worst session in the maintainer's store and collided 127,139 of 334,288 rows.
Scoping the ordinal fixed the ordering and left the identity wrong: a *forked* sub-agent replays
its origin's whole prefix under a new agent id, so one `message.id` can legitimately belong to
three sequences (141 such ids in a 400-file sample). With the uniqueness key still
`(tool, dedupe_key)`, 13,897 steps vanished across 43 sequences, one of them stored from ordinal
3,637 with no record that its opening was missing. The key is `(tool, timeline, dedupe_key)`.

A stored sequence may still legitimately begin above ordinal 1, because the horizon cuts the
opening off a timeline that straddles it. That is a property a reader has to allow for, not a
loss to detect.

**One pass produces both readings.** `ParseAll` returns the records and the steps from a single
scan; `Parse` and `ParseSteps` are wrappers. An earlier implementation used a second scan to
avoid touching ~80 call sites in tests, and the two orders of checks drifted apart exactly where
a shared line struct could not help: a denial attributed by a backward walk instead of by the
id the line names, and a token total folded by different arithmetic. An unexported single pass
plus thin wrappers costs nothing at any call site and removes the class.

**The outcome vocabulary holds only values something can produce.** `max_tokens` is read and
maps to `truncated`; it is rare (5 of 5,706 audited transcripts) rather than absent. There is
deliberately no `aborted`: `interrupted:true` occurs in none of those 5,706 transcripts and
nothing reads it, so a member for it would be a bucket no code can fill — the silent zero these
rules exist against. Codex's `turn_aborted` is the same case and arrives when Codex gains a step
reading.

**The timeline is bounded by a horizon, and the bound is load-bearing.** Measured on the
maintainer's store after a full rebuild: 335,527 steps against 178,016 records (1.88 stored
steps per record), and 101.9 MB of table and indexes against `usage_record`'s 58.3 MB — roughly
1.7x the table it describes. The row multiplier alone said the growth was modest and that
reading was wrong: bytes are dominated by per-row overhead and indexes, not column count. `trace.horizon_days` defaults to
30, which is also all that can ever be rebuilt: Claude Code deletes transcripts at 30 by
default. An explicit `0` means keep everything and is honoured rather than coerced.

**A detector declares its scope, and its denominator is that scope.** 86% of sessions on the
audited machine are SDK calls contributing 5% of rows. "89% of sessions end without an edit"
would be true, precise and worthless. Scope comes from what the tool itself recorded —
`entrypoint` and `sidechain` — and the excluded share renders beside the figure.

## Consequences

The store grows by roughly 1.7x the usage table for anyone who ingests Claude Code transcripts,
bounded by the horizon and erasable by `clear` under the same scope as the records. Widening the
horizon later does not bring pruned steps back on its own: the ingest watermarks skip transcripts
already read, so it takes a `backfill --full`, and only while the source still has the files.

`export` and the team-server sync carry usage records only; the timeline stays local. That is a
deliberate limit rather than an oversight — the sync contract is aggregate usage, and a
sequence is the most re-identifying thing assaio holds even in content-free form.

Nothing reads the table yet. The detectors that do are the work this substrate exists for, and
until they ship the timeline costs storage and returns nothing.
