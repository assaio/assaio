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

*Amended in v0.21.0:* the integer is read from the **call's own arguments**, not from the result
the call produced. The first reading took it from the edit result, which measured out as a
capability gap rather than a design: a result exists only for a call that completed, so all 344
failed edits and every one of 36,846 reads on the audited store carried no target at all, and two
of the detectors this substrate was built for could not be written. Nothing about what is stored
changes -- still an integer, still per timeline, still no path -- and a path named relatively is
resolved against the session's working directory before the integer is chosen, so one file cannot
hold two of them. What does change is that a re-parse may renumber, which is why `target_ref` is
now *assigned* on restatement rather than kept at the higher of two values, exactly as `ordinal`
already was and for the same reason.

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

*Amended in v0.21.0:* the table has readers. Two detectors read it (`edit-loops`, `recovery`), and
three consequences of that were not obvious until they were measured:

- **A detector's scope is its denominator, and it belongs in the interface.** 89% of the main-loop
  sequences on the audited store are SDK calls holding 5.7% of its steps, so a figure over "every sequence"
  describes one-shot API invocations and prints them as sessions. A validator reading the timeline
  implements `analyze.TraceReader` and names one scope; one place computes what asking for it left
  out, and every detector renders the same sentence for it.
- **Reading the sequences costs about 2.5s on a 339,000-step store**, most of it row scanning
  rather than SQL. It is skipped when no registered validator wants it, and `trace.horizon_days`
  bounds it. The metric-plugin wire carries the whole set unconditionally -- about 44MB at that
  size -- because the alternative is an extension surface that cannot write what the core just
  gained; letting a plugin declare its needs is `B168`.
- **Per-step and per-turn answers differ, and only one of them is a cost.** A tool call carries no
  tokens, and the steps after a failure are more heavily assistant turns than the window is, so the
  aftermath of a failure reads 1.06x the window's average *step* and 1.02x its average *turn* --
  1.35x against 1.03x over a three-step window. The first number measures the sample's composition.
  Any future detector dividing tokens by steps inherits that trap.
