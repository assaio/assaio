# 8. The signal catalog is the answer to "what can this tell me about my data"

## Status
Accepted (2026-08-02)

## Context
`assaio` publishes a per-source **depth** matrix (`B83`): Gemini CLI carries tokens but no
edits, Copilot totals a session rather than a turn. That answers "what can this *source*
tell me". It does not answer the question a person actually has, which is "can it tell me
**this**" — can it tell me AI lines per active day, or rework, or which sub-agent the tokens
went to. Today the only way to find out is to run `analyze` and read whichever verdict came
back thin.

`B90` proposed two things at once: a catalog of every analyzer-visible signal, and an
`AnalyzerContext` that analyzers read instead of querying SQLite, "so the store stops being
the public API". Reading the code, the second half rests on a premise that is already false.
Validators do not query SQLite: they are pure functions of `analyze.Input`, which the CLI
builds. What leaks is not queries but **types** — `Input` carries `[]store.UsageRow`,
`[]store.SessionRow`, `[]store.AttributionRow` — and those same types are the published
metric-plugin wire (ADR 0004), shipped in v0.6.0.

## Decision
Ship the catalog. Do not ship a second way to read data.

- **A signal is a declared, stable id for one thing assaio can report**, carrying what a
  reader needs in order to judge it: a unit, whether it is observed / estimated / derived,
  the grains it is meaningful at, and — the field that earns the catalog its keep — **what a
  zero means**. "Zero rework" and "this source never recorded rework" are different facts, and
  every metric built on the second one is a lie told confidently.
- **A signal declares its meaning; a source declares what it can answer.** Capability lives on
  `parser.Depth.Answers` as a list of signal ids, not on the signal as a required capability,
  so neither file can become a second opinion about the other. This was not the first design:
  the catalog originally declared "needs tokens / activity / attribution", reusing the depth
  matrix's three axes — and running `signals coverage` against real data immediately showed
  that overstating support. `Activity` is one bit over a source that records changed lines but
  no edit count, no tool calls and no turns, which is exactly what Copilot CLI is, so sixteen
  of eighteen signals read as fully supported when ten are. The three axes stay as the tier
  table's summary; `Answers` is the per-signal truth, and a test asserts the two never
  contradict each other. Adding a parser now means declaring what it answers, which is a
  better question than "does it have activity, yes or no".
- **`signals list`, `signals describe <id>` and `signals coverage`** are the surface.
  `coverage` is the one with teeth: it reads the window actually in the store and reports,
  per signal, the share of tokens coming from sources that can answer it — *full*, *partial*
  with the share named, or *none*. It is computed from real data and `parser.Depth`, never
  from a claim in a table.
- **`signals coverage` and the `coverage` validator answer different questions** and must not
  drift into each other. The validator answers "how much of this window is high-confidence
  data" on three axes for the verdicts already printed. The command answers "which of the
  things assaio knows how to say can be said at all about my setup". They share the same
  token-by-tool computation so the two can never disagree on the underlying arithmetic.
- **No `AnalyzerContext` yet, deliberately.** Introducing it now means two read surfaces
  living side by side — the classic maintenance smell, and an open-source contributor's first
  question would be which one to use. Migrating the nineteen validators onto it instead means
  breaking the metric-plugin wire published in v0.6.0 a day earlier, for zero user-visible
  gain: no plugin author gets anything from an in-process Go API, because exec plugins speak
  JSON. So the context is deferred until the git evidence collector (`B91`) is a real second
  consumer with real requirements, and it is designed against those rather than guessed at.
  What "the store stops being the public API" needs is tracked separately (`B102`).
- **The catalog is not a second source of truth about coverage.** It declares what a signal
  *is*; whether your data supports it is computed from the window in the store. A capability
  claim that outruns what a parser really extracts is a bug the coverage command surfaces
  rather than hides — it surfaced one on its first real run, which is the argument for the
  command more than any of the prose here.
- **Ids are a public surface** from the moment they are printed, and they follow the event
  contract's naming (ADR 0007): `<domain>.<thing>.<measure>`, so `ai.lines.added` and
  `vcs.commits.count` read as one vocabulary once the collectors land. Renaming one after
  `v1.0` is a semver event (`B23`).
- **`attributed` is named but not registered.** The catalog's status vocabulary is observed /
  estimated / derived today; `attributed` lands with attribution edges (`B85`), which is what
  will produce the first signal that deserves it. Same posture as ADR 0007: commit the
  vocabulary in the decision, register what can actually be produced.

## Consequences
- A person can ask, before trusting any report, what this tool can and cannot say about their
  machine — and get an answer computed from their own data rather than from a feature list.
  That is the depth matrix's promise at the resolution people actually care about.
- Adding a signal is one catalog entry. Adding one whose capability is wrong is caught by the
  coverage command reporting support the metric does not really have, which is a better
  failure than silence.
- The nineteen existing validators are untouched, the metric-plugin wire is untouched, and no
  golden moves. The catalog describes what those validators consume without yet being how they
  consume it — an honest half-step, named as one.
- The drill-down's safety — that a project-scoped validator never reads a window-only field
  the caller did not populate — was until now upheld only by every such validator happening to
  be `WindowScoped`. It is now asserted by a test, because it is exactly the kind of invariant
  a twentieth validator would break silently.
