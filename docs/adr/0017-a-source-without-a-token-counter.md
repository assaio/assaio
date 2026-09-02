# ADR 0017 — A source with no token counter is a tier, not a row of zeros

Status: accepted (2026-09-02)

Applies [ADR 0011](0011-capability-gated-metrics.md) to the axis it had never been tested on;
extends the vocabulary of [ADR 0008](0008-signal-catalog.md)'s depth matrix.

## Context

Every source assaio read before this one published token counts. `deep`, `standard` and
`import-only` all assume it: `standard` is defined by which *activity* gaps a source
documents, and `import-only` by whether its figures can be attributed to a session at all.
Neither axis is "does it count tokens", because until now the answer was always yes.

[ADR 0011](0011-capability-gated-metrics.md) made capability a precondition of a figure and
listed the fields a source might not record — turns, edits, compactions, rework. It did not
list tokens, and the gate was never exercised on them. That left a class of fabricated zero
nobody could see: a token figure computed over a source that keeps no token counter.

Antigravity CLI is that source. Its CLI (`agy`, binary `~/.local/bin/agy`, verified at
1.1.23) keeps a rich local corpus — 500 conversations, 1,537 transcript entries and 1,975
steps on the machine this was measured on — and **no input, output, cache or reasoning
counter anywhere in it**. The search was exhaustive rather than cursory: every varint path in
every `gen_metadata` and `steps.metadata` protobuf blob across 500 conversations was
correlated against the printable length of the same step's payload. A token counter of that
text sits near 3–5 characters per unit with a spread under 3x; nothing does. The tightest
paths sit at 46 chars per unit with a spread of 1.0x, which is the signature of a constant.

One real figure does exist, and it is not a billing figure. `gen_metadata` field path
`1.9.10.1` is prompt/context tokens occupied — monotonic across steps in 198/198
conversations with more than one row, and `used ≤ window` in 638/638 observations against
`1.9.10.4`, the model's context window, which takes exactly {80000, 128000, 160000, 256000}
and pairs with the model name in the same blob 218/218 times. The account behind it runs on a
plan quota, not API billing, so even a complete token record would answer subscription fit and
never a dollar.

Running the source through the existing pipeline before any of this was declared produced
exactly the sentence ADR 0011 exists against, on six surfaces at once: `report` totalling
`$0.00`, `effectiveness` totalling `0` AI lines beside `$0.00` under the line "Every source in
this table records changed lines", `status` reporting `peak context ~0 tokens` and `0 output
tokens/session`, `burn-anomaly` reading a typical day of 0 tokens, and `turn-efficiency`
reading 0 output per turn. None of them errored. Every one of them was a confident statement
about someone's work drawn from a field their tool never kept.

## Decision

**A source declares whether it counts tokens, and every token- and dollar-denominated figure
reads only the sources that do.**

- **`activity-only` joins the tier vocabulary.** It carries sessions and what was done in
  them, and no token or cost accounting at all. It is a tier rather than a `standard` row with
  `Tokens: false` because the tier is the summary a reader scans: "standard" beside a source
  whose every `$` figure withholds reads as a source that can be added to a cost report. It is
  deliberately *not ranked* against `import-only` — one knows what happened and not what it
  cost, the other the reverse — and the ordering comment says so rather than implying a
  ladder.
- **The zero is declared, not merely produced.** The parser leaves every token field at zero
  *and* the depth row answers no token signal. Those are two different statements and both are
  required: an undeclared zero is indistinguishable from a parser that forgot to read a field,
  which is the whole failure mode.
- **`ai.tokens.total` is the capability question every token figure asks.** Six surfaces now
  ask it. `report.SessionStats` gained two bases beside `Turned` — a turn count and a token
  count are different capabilities and the first never implied the second. `Row.Tokened` and
  `EffRow.Tokened` are the token column's `LineCapable`. A total withholds on the same
  condition its rows do, because a total is the cell a reader trusts most and checks least.
- **The zero-token drift canary is capability-gated.** A source declaring no counter produces
  100% zero-token records on a perfectly healthy run; judged by the share it would fire on
  every backfill forever and fail `doctor --strict` on correct data. A source the matrix has
  never heard of stays under the canary: not knowing a tool is not evidence that it keeps no
  tokens.
- **Nothing observed is stored in a column every other source derives.** `PeakContextTokens`
  is `cache_read + input` on every source that has one; `agy`'s figure is read from the vendor
  instead. Two different measurements in one column, with no field to say which is which, is
  the blend AGENTS.md forbids. The evidence is recorded in `B194` rather than wired.
- **What cannot be told apart is claimed neither way.** `steps.has_subtrajectory` is false on
  all 1,975 captured steps, which cannot separate "no sub-agent ran" from "this build never
  sets it". Delegation is therefore *unverified on this source*, a different entry in the
  matrix from answered-and-zero.

Rejected alternatives:

- **Store context occupancy as input or cache-read tokens.** It would give every `agy` session
  a price. The number is real; the price would be invented, and a plan quota is not a rate.
- **Call it `standard` and let `Tokens: false` carry the difference.** The axis is one bit in
  a struct; the tier is the word `doctor` prints. A reader who reads one of them reads the
  second.
- **Refuse the source.** It is a real tool with a real local corpus, and "assaio cannot see
  Antigravity" is a worse answer than "assaio sees its sessions and states that it cannot see
  its cost".
- **Read the conversation databases for the model name and step vocabulary.** It means opening
  another tool's live WAL-mode SQLite file and walking unnamed protobuf fields, for a model
  name covering 218 of 500 conversations, frequently several per conversation with no request
  count to choose between them. The brittleness is bought for an attribution that would be
  wrong more often than absent.

## Consequences

- Antigravity CLI is the most brittle of the six sources and `docs/format-resilience.md` says
  so with the reasons — a self-updating binary (two builds from consecutive days sat on the
  audited machine), an unpublished schema, and a data root shared with Gemini CLI — along with
  which canary catches which drift and the one that nothing guards: 26 tool calls across 500
  conversations is too sparse a baseline for any share to judge.
- The depth matrix now names the CLI version a source was verified against, because a
  self-updating binary makes "supported" a claim with a date on it.
- A `report` or `effectiveness` table in which nothing priced totals `—`. That is a visible
  change for any window whose only rows are unpriced, not just for `agy`.
- Adding a source is now also a question about which *columns* it removes itself from, not
  only which signals — which is the same lesson ADR 0011 drew, one axis further out.
