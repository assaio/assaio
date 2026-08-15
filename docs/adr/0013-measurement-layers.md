# 0013 — Every figure states its measurement layer

**Status:** accepted (v0.22.0)

## Context

`ROADMAP.md` has carried this since the project started, under "Four layers, never relabeled":

> Everything `assaio` reports sits on exactly one of four layers, and the product says which.
>
> - **Activity** — tokens, turns, tool calls, edits. What happened.
> - **Output** — changed lines, commits, PRs, tests. What was produced.
> - **Outcome** — merged, survived, passed CI, review burden, reverted. Whether it held.
> - **Impact** — value to a user or a business. Needs context `assaio` does not have.

and, among the promises that hold regardless of demand:

> **Never relabel a layer.** Activity is not output, output is not outcome, outcome is not
> impact. A metric states which one it is.

Nothing said which one. There was no `Layer` field on `analyze.Result` or on `signal.Signal`,
and the only place a layer was ever named to a reader was one sentence of `throughput`'s prose.
The promise existed as prose about the code rather than as anything in it.

That is not a cosmetic gap, because the whole reason the line is in the roadmap is that
promoting an output figure to an outcome claim is the most likely way this project starts
lying. Three shipped defects were that gap costing something concrete:

- `throughput` reads growth in AI lines as `RAMPING`, a green verdict with a gauge rising in
  the line count, while its own `HowToRead` says "not a quality score". Colour, label and gauge
  all say more lines is better.
- `$/100 lines` divides a whole window's cost by a denominator only some sources contribute
  to. `effectiveness` disclosed that and the dashboard colophon disclosed it; `status` printed
  the same ratio with nothing.
- The digest mails an output trend to an inbox, the one surface designed to be read out of
  context, with comparability and cost caveats and nothing about lines being an output measure.

Each was fixable on its own. None of them would stay fixed, because nothing recorded that the
distinction existed at the point a contributor adds the next metric.

## Decision

**A closed vocabulary, in its own package.** `internal/layer` holds the four constants and
`Valid`. It imports nothing, so both the validator framework and the signal catalog can depend
on it without depending on each other.

**A metric declares its layer through the interface, not through a field.** `Validator` gained
`Layer() layer.Layer`. A field is a field one metric will forget; a method is a compile error.
`Register` additionally panics on a value outside the vocabulary, so a fifth layer cannot enter
the registry at start-up. `Evaluate` stamps the declaration onto the `Result`, so a `Result`
cannot claim a layer its metric did not.

**The declaration is what the verdict rests on.** A `Result` carries figures that may sit on a
*different* layer as context — `model-fit` prints a lines-per-token note, which is output, beside
a token-share verdict, which is activity; `skill-economics` puts a line count in every bar under
the same activity read — and one label per metric is what the roadmap promises. The label answers
"what is this verdict a claim about", because that is what a reader acts on, and the dashboard
tooltip says so in those words. A metric whose verdict would move to another layer is a new
metric, not a relabeled one.

**Signals declare it too.** `signal.Signal` gained the same field, for the reason the catalog
exists at all: an ID is not a layer. `ai.step.outcome` is an **activity** signal — how one call
ended inside a session — and reading it as an outcome, which is whether the code held, is the
exact relabeling this ADR forbids. A test pins that one by name.

**The extension surface is not weaker than the core it extends.** The metric-plugin protocol
(ADR 0004) requires `layer` on a result document and rejects the metric whole when it is
missing or outside the vocabulary — the same boundary treatment `read.key` gets. Built-ins are
forced by a compile error; a plugin gets the equivalent at the wire. This is a **breaking
protocol change**, called out in `CHANGELOG.md` under Breaking.

**It renders, and it is published.** The CLI text report puts the layer on the header line beside
the verdict, the dashboard ledger prints it under the read with the four-layer explanation as its
tooltip, `analyze --format json` carries it as `layer`, and the generated reference — the surface
`llms.txt` calls the one list that cannot fall behind the build — carries a Layer column for both
validators and signals. A label nobody sees would have been the same gap in a new place.

## Consequences

What the layers make visible, immediately: of twenty-one built-in validators, **eighteen are
activity and three are output** — `throughput`, `rework` and `concentration`. **Not one claims
an outcome.** That is the honest state of a local-first tool reading session logs: it can say
what happened and what was produced, and it cannot say whether any of it held. `survival` is
the one local signal reaching toward outcome, and it is a command rather than a validator,
directional by its own admission.

`TestNoValidatorClaimsAnOutcome` fails if a built-in validator ever declares `outcome` or
`impact`. That is not a permanent ban — the server stage is where outcome evidence comes
from — but it makes the promotion a deliberate act with an ADR behind it rather than a line
changed while adding a feature.

The cost: twenty-one one-line methods, twenty-five catalog entries and a protocol field, all of
which have to be right rather than merely present. A layer stated wrongly is worse than none,
because it launders a claim. That is why the assignment rule is written down above and why the
one genuinely tempting mis-assignment has its own test.

What this does **not** do: it does not make `throughput` a better metric, and it does not
resolve whether a green verdict on a rising line count should exist at all. It makes the claim
legible so that question can be asked. `B180` remains open on those terms.
