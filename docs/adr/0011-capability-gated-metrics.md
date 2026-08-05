# 11. A metric reads only the sources that record its field

## Status
Accepted (2026-08-05)

## Context
[ADR 0008](0008-signal-catalog.md) gave every signal a `ZeroMeans` line, because "zero rework"
and "this source never recorded rework" are different facts. It declared what a zero means. It
did not make anything act on it.

Reading the code found what that gap costs. `store.SessionRow` carries `Turns`, `ActiveMinutes`,
`Edits` and `Compactions` for every session, and three of the five in-tree sources leave some or
all of them at zero forever — Gemini CLI and Cline record no edits, Copilot CLI totals a whole
session and so records no turn at all. Four validators and the `status` block averaged those
zeros in. The output was not a thin number with a caveat; it was a confident sentence about
someone's work, drawn from a field their tool does not keep:

```
conversational: 100% (no file edits)
Takeaway: Most sessions are conversational -- design and debugging work that rarely edits files.
```

Nothing in the window was wrong. Every session really did carry zero edits. The parser simply
never had any to write, and no layer between it and the verdict asked.

This is the failure mode the project is least able to see. It produces no error, no thin
sample, no low-coverage label — the confidence envelope read `activity coverage 0%` beside the
sentence above, which describes the *window* and cannot describe the *field*. It is invisible
on the maintainer's own machine, whose store is Claude Code and Codex, and it gets worse with
every source added, which is the direction the roadmap goes.

## Decision
Capability is a precondition of a figure, not a caveat on it. A metric over a field that only
some sources record keeps the rows whose source answers the corresponding signal, computes over
those, and declares that reach.

- **The gate is `parser.Answers(tool, id)`, and the id is a `parser.Signal*` constant.** The ids
  move into the package that publishes capability, so asking is a compile-time-checked
  expression rather than a string literal a typo turns into "no source answers this" — which
  empties a metric silently instead of failing a build. `internal/signal` keeps declaring what
  each id *means*; neither file becomes a second opinion about the other.
- **An empty subset withholds the verdict.** `Read` goes neutral and the takeaway names the
  missing capture, because a favourable verdict earned from a silence is worse than no verdict:
  it is the exact shape of the sentence this decision exists to prevent.
- **A figure with no subset prints `—`, never `0`.** The two are different claims and the
  reader cannot tell them apart from a number alone.
- **The reach is declared as signal coverage** (`covering()`, ADR 0007's envelope as extended in
  v0.9). "This mix describes 6 of 12 sessions" is then in the data, not only in prose.
- **A field with no signal id gets one before a metric reads it.** `ai.compactions.count` and
  `ai.rejected.count` were both stored columns and both the subject of a shipped verdict while
  being undeclarable, which is precisely why the metrics over them could not tell absence from
  zero. Adding one is a catalog entry plus the matrix rows that answer it.
- **The shared session summary gates too.** `report.SessionStats` is read by `analyze` and by
  `status`, so gating it in one place fixes both; it carries a basis count per group rather than
  a single sample size, since its groups do not rest on the same sessions.
- **Not every field is gated, and the test is what the field means at another grain.** Output
  tokens and a session's start time are honest for a whole-session record; turn depth, peak
  context, focused minutes and an edit count are not. The question is never "is this field
  populated" but "would the figure be right, or merely non-empty" — the same question ADR 0008
  asks a parser about `Answers`.

Rejected alternatives:
- **State it as a caveat and keep averaging.** What shipped for three releases. A caveat under a
  verdict does not stop the verdict from being read, and the verdict was the wrong sentence.
- **Weight by capability instead of filtering.** Produces a number between the true one and the
  fabricated one, which is worse than either: it is defensible nowhere and looks precise.
- **Gate in the store's queries.** `Sessions` serves every consumer, and each wants a different
  subset — `rhythm` needs every timed session for its bands and only the paced ones for length.
  A query that pre-filtered would push the choice out of the only place that knows the question.
- **A `WindowScoped`-style marker interface per capability.** Declarative, but it says which
  metric is affected rather than which *rows* are, and the same metric usually needs several
  different subsets.

## Consequences
- A window from a source that records nothing per-session now reads as no verdict plus a stated
  reach, instead of a confident mix. That is a visible change to what `analyze` and `status`
  print, and a changelog entry rather than an implementation detail.
- A twentieth validator can still break this by reading `SessionRow.Edits` directly. A test
  asserts the property generically: every registered validator must return an identical `Result`
  for a window whose gated fields are zero and one where they are not, when the source answers
  none of them.
- Adding a source is now also a question about what it *removes* itself from. Leaving a signal
  out of a depth row takes that source's rows out of the matching figure — which is the honest
  outcome, and the opposite of the old incentive to claim an axis.
- The reach being data (`signalCoverage`) rather than prose is what lets `check` and the future
  recommendation engine (`B84`) refuse to fire on a figure that describes a sliver.
