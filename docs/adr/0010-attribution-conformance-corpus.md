# 10. The attribution conformance corpus defines honesty before an engine exists

## Status
Accepted (2026-08-04)

## Context
`B85` will link a session to the change it produced. Every earlier decision in this project
made that link easier to get wrong: the store holds counts rather than code, a commit
observation is content-free by construction ([ADR 0009](0009-local-git-evidence-collector.md)),
and the evidence is timing and category overlap. That is exactly the shape of data where a
plausible heuristic produces a confident answer nobody can check.

The failure mode is specific and it is not a crash. An engine that always names its
best-scoring candidate will look correct on almost every real repository, because most
sessions really do have one obvious commit. It will be wrong precisely where being wrong
matters — two people in one window, two commits a minute apart, a session that produced
nothing — and it will be wrong silently, because a link with no alternatives attached reads
like a fact.

`B93` asks for the corpus that catches this, and asks for it **before** the GitHub connector
and before the engine. Writing it after would mean writing the expectations against whatever
the engine happened to do.

## Decision
`internal/attribution` is a conformance corpus and nothing else: ten scenarios, each of which
builds a real git repository and the sessions that ran against it, and states what any engine
must conclude — most importantly, where it must refuse to conclude anything.

- **The corpus speaks the smallest vocabulary honesty can be judged in.** A scenario's
  expectation is: which commits a session may be linked to, whether the link must stay
  ambiguous, and which commit a human confirmed. It deliberately does **not** define an edge
  type carrying method, confidence, evidence and alternatives — that is `B85`'s design, and a
  corpus that fixed it now would be testing one engine's shape rather than the property.
  Reducing an engine's answer to `Links` is the engine's job.
- **Fixtures are real repositories read through the real collector.** A scenario is data —
  commits with tags, branches, file lists and offsets from a fixed epoch — and `Build` lands
  them with pinned author and commit dates, then reads them back through `vcs.Collect`. The
  corpus therefore judges an engine against the observations it will actually receive, not
  against a shape invented for the test.
- **Ambiguity is verified structurally, not asserted.** A scenario declaring itself ambiguous
  proves nothing; the test reads every signal an engine could separate the candidates by —
  project, file categories, position against the session window — and fails if any of them
  differs. A fixture that quietly became separable would otherwise keep passing while
  teaching the opposite lesson.
- **The corpus must have teeth and must be satisfiable.** Two stand-in engines run against it
  in its own tests: the nearest-commit heuristic everyone writes first, which must fail the
  ambiguous scenarios, and an engine answering what the scenarios describe, which must pass
  all ten. Without the first the corpus proves nothing; without the second it tells an
  implementer nothing.
- **`overlapping-users` is ambiguous, and that is a finding rather than a compromise.** `B85`
  lists "identity compatibility" among the methods an edge may carry. A commit observation
  carries no author — the payload is counts and categories — so as the evidence stands today,
  two people committing in one window cannot be separated by identity at all. The corpus
  records that as required ambiguity. Making that scenario separable is a change to what an
  observation carries, with the privacy decision that implies (`B100`), not a smarter ranking.

Rejected alternatives:
- **Assert against a reference engine.** Circular: the engine would define correct, which is
  the thing the corpus exists to define independently.
- **Synthesize commit observations directly.** Faster and no git dependency, but it would
  stop testing the collector's own reading — the rename, merge and category behaviour that
  decides what an engine even sees.
- **Wait for `B85` and write the corpus alongside it.** The expectations would be written
  with the implementation in view, which is how a test ends up asserting what the code does.

## Consequences
- `B85` starts against a specification rather than a blank page: ten named properties, each
  with a one-line statement of what it defends, printed with any violation.
- The corpus is not reachable from the binary and is not meant to be. It is a contract other
  packages' tests consume, the same way a golden file is.
- Adding a scenario is one entry in `scenarios.go`. Adding a *kind* of expectation is not,
  and should be resisted: every field added to `Expectation` is a way for the corpus to start
  describing an implementation.
- The corpus does not yet judge confidence, method or evidence, because none of them exists.
  When `B85` defines them, the question is whether the corpus should assert on them at all —
  a scenario that pins a confidence number would break on every re-weighting, so the likely
  answer is bands and ordering rather than values.
