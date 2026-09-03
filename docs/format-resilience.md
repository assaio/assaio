# Format resilience — detecting and reacting to vendor log-format drift

Every format `assaio` parses is **vendor-internal**: none of the tools document their
session logs as a stable interface, and any release of theirs can change shape without
notice (`doctor` discloses this on every run). This document is the operating flow for
that reality: what protects the numbers, how a change that shrinks them without erroring
gets caught, and the detect → triage → fix → release loop.

## What protects us today

| Defense | Where | What it catches |
|---------|-------|-----------------|
| Narrow `Discover` globs | each `internal/parser/<tool>/discover.go` | Foreign files in shared directories never reach a parser. |
| Skip-and-count | every `Parse` + exec-plugin boundary | A line that stops unmarshaling is counted, never fatal; `backfill` prints `skipped=` / `failed=` per source. |
| Scanner caps | `internal/parser` (`MaxLineBytes`), plugin stdout caps | A pathological file cannot wedge or OOM an import. |
| `NonNeg` clamps + boundary validation | shared parser helpers, `internal/plugin` | Corrupt counts cannot go negative or smuggle in impossible values. |
| Golden files | each parser's `testdata/` | Any change in **our** parsing of a captured shape shows up as a reviewable diff. |
| Fuzzing (`make fuzz`) | every parser + the metric-result decoder | No panic on arbitrary bytes; invariants hold on whatever is accepted. |
| Granularity rule | `usage.Record` contract | A format change that degrades detail must be re-labeled `session`, never silently kept as `turn` — and a report marks a mixed total rather than reading session data as per-turn. Re-reading a local file restates the label on rows already stored, so a correction reaches history instead of only new records. |
| Capability gate on every metric | `parser.Depth.Answers` + `analyze` + the metric-plugin wire's `answers` | A figure is computed only over the sources that record its field — per-session figures and rates over stored columns alike — so a source that never writes one is absent from it rather than averaged in at zero. A generic test varies both row shapes, so a new validator reading an ungated column fails rather than shipping a quiet dilution. |
| Deterministic dedupe keys | parser contract + golden tests | Re-importing after a fix never double-counts old records. |
| Drift canaries | `internal/drift`, after every `backfill` | The first two failure modes below: numbers that shrink without anything erroring. The third, additive drift, no canary can see. |

## The three silent failure modes

Everything above catches a parse that *fails*. None of these does — which is why they
needed their own defense:

1. **Semantic drift.** If a vendor renames or moves a token field, the line often still
   parses as valid JSON — it just stops matching the usage shape, or maps to zero tokens.
   `skipped` counts *unparseable* lines, not renamed keys, so totals quietly shrink
   instead of failing loudly.
2. **Discovery drift.** If a tool moves its log directory or changes file naming,
   `Discover` finds fewer (or zero) files. `backfill` and `doctor` *show* the counts, but
   nothing flagged "this used to be 300 files and is now 0" as an anomaly.
3. **Additive drift.** If a vendor starts *recording something new*, every figure assaio
   already publishes stays exactly as correct as it was. Nothing shrinks, nothing errors,
   no canary can fire — and the tool silently stops being as deep as its source allows.

The first two share one property: the failure mode is **plausible-looking underreporting**,
which is exactly what an honesty-first tool must not do silently. The third is different in
kind, and worth stating separately because the defenses above are all aimed at the first two.

### Additive drift has no canary, and cannot have one

Every canary below judges a source against **its own history**: fewer files, fewer records per
file, more skips, more zero-token records. A vendor adding a field or an event moves none of
those. The detection is a **periodic field audit** — enumerate every key path a current corpus
holds, diff it against what the parser reads — and the audit's own currency is the risk:
[source-fields.md](extending/source-fields.md) states each section's corpus and when it was
taken, because a stale audit reports the absence of a field that has been on disk for months.

The worked example is Codex's `event_msg/item_completed`, found in the v0.26 re-audit:

| | |
|---|---|
| First seen | 2026-08-20 (Codex CLI ~0.148) |
| Coverage when found | **1,614 of 1,614** September rollouts, 833 of 1,001 August ones, 0 of 10 July ones |
| Volume | 14,268 events across 2,625 rollouts |
| What it carries | per-step wall-clock duration; a `CommandExecution` `exit_code` — the only place Codex says a command *failed* rather than *returned* |
| Canaries that fired | **none, correctly**: records per file, files found, skips and zero-token share were all unchanged |
| Audit that would have found it | the field audit — whose Codex section was taken on 21 rollouts, all of them older than the event |

The lesson is not that a canary was missing. It is that **the field audit is a detector, not
documentation**, and it decays: re-take it against a current corpus on a cadence, or it reports
last year's tool. What that particular finding changed in the code is recorded in the audit's
own Codex section — it is measured, and deliberately *not* read, because its ids join to
nothing.

## Detection — three channels, and a fourth that is not automatic

0. **The field audit, by hand and on a cadence.** Named first because it is the only detector
   additive drift has, and the only one nothing runs for you:
   [source-fields.md](extending/source-fields.md), re-taken against a current corpus, each
   section stating the corpus it was taken from and when. The three below all judge a source
   against its own history and none of them can see a field that was never read.

1. **Local canaries, automatic.** After every `backfill`, four of the five judge a source
   against its own recent history, and one judges a condition:

   | Canary | Fires when | Abstains below |
   |---|---|---|
   | `discovery` | no files found where there used to be some, or fewer than half the recent median | a median of 20 files, for the partial-drop half |
   | `yield` | records per file read collapse to under a quarter of the historical median | 20 files read this pass |
   | `skipped` | skips average one or more per file read | 50 skipped inputs |
   | `zero-token` | at least a quarter of parsed records carry no tokens at all | 50 records, **and** a source whose depth row declares a token counter |
   | `barren` | files are found and no run on record has read a usage record out of them | nothing — the condition is absolute |

   The design rules behind those numbers matter more than the numbers: the baseline is a
   **median** of recent runs, so one odd pass cannot move it; every comparison is a
   **ratio**, so an incremental pass that read four files stays comparable to a full one
   that read six thousand; and every canary that computes a share has a **sample floor**
   below which it says nothing, because a share computed from a handful of records is not
   evidence. A source with no files — an exec plugin — is judged only by `zero-token`, the
   one canary that needs none.

   `barren` is the exception to all of it, and deliberately so: a comparison cannot see a
   source that never worked, because its baseline is zero and there is no drop to detect.
   That was measured, not assumed — setting all four sample floors to `1` and re-running
   the real corpus fired nothing on either build. It reads the whole history rather than
   one run, because an ordinary incremental pass whose single changed input yields nothing
   is not a barren source.

   `zero-token` carries a second floor the others do not, and it is a capability rather than a
   sample size: a source whose depth row answers no token signal is exempt. Antigravity CLI
   publishes no counter anywhere in its format, so 100% of its records are zero-token on a
   perfectly healthy run — judged by the share it would fire on every backfill forever and fail
   `doctor --strict` on data that is exactly right, which is how a canary stops being evidence.
   A source the matrix has never heard of is still judged: not knowing a tool is not evidence
   that it keeps no tokens.

   A breach prints `warning: possible format drift in <tool>` — or, for `barren`, `warning:
   nothing read from a detected source`, because a condition is not a diagnosis — and appears
   in `doctor`'s drift section; `doctor --strict` turns it into an exit code for cron or CI. While a
   source's discovery canary is lit, its per-input ingest state is frozen rather than
   pruned: "the files are gone" and "we stopped finding them" are indistinguishable, and
   the state is the evidence.
2. **User reports.** A "my numbers dropped after updating <tool>" bug is the classic
   drift signature. Such issues get the **`format-drift`** label. Ask for: the tool's
   version, `assaio-agent doctor` output, and a few **redacted** sample lines — the
   same rules as the [connector intake
   flow](extending/data-source.md#the-intake-path-open-a-connector-issue-first): never a real
   transcript, prompts, or code.
3. **Maintainer canary (manual).** When a covered tool ships a major release,
   generate one fresh throwaway session with it and run `backfill` + `doctor` against
   a scratch store (`--db`); eyeball the counts. Cheap, and catches drift before users
   do.

## The most brittle source, named

Not all six are equally exposed, and pretending otherwise is the same mistake as a bare
checkmark on a depth matrix. **Antigravity CLI (`agy`) is the most brittle of the six**, on
three counts at once:

- **The binary self-updates.** Antigravity CLI 1.1.23 was verified on 2026-09-02, with a `.old` copy of
  the previous build sitting beside it from the day before — two versions from two consecutive
  days on one machine. Nothing pins a user to the version this parser was read against, and the
  depth matrix names Antigravity CLI 1.1.23 for exactly that reason.
- **The schema is unpublished** and the format is under a directory shared with another tool.
  `~/.gemini` holds Gemini CLI as well, which is why both discoverers glob narrowly and neither
  scans the shared root.
- **What accounting exists is in unnamed protobuf fields.** The parser deliberately reads none
  of them (see [what each source's log carries](extending/source-fields.md)); the fields it does
  read are named JSON keys, which is the single reason this source is readable at all.

Which canary catches which failure here, and which does not:

| If Antigravity CLI… | caught by |
|---|---|
| moves or renames `brain/<id>/.system_generated/logs/` | `discovery` — 500 conversations to none |
| renames `source`, `created_at` or `step_index` | `barren` on a fresh store, `yield` on an existing one: entries still parse as JSON and stop producing records |
| writes a `created_at` in another format | `skipped` — undatable turns are counted, not silently dropped |
| renames `tool_calls` | **nothing.** The corpus holds 26 tool calls across 500 conversations, which is far too sparse to form a baseline any share could be judged against. Stated here rather than guarded, because a canary computed from 26 observations is not evidence. |

The last row is the honest limit. It is also the reason `agy` is `activity-only` and not
`standard`: the figures it feeds are few enough to check by eye, and none of them is a cost.

## Reaction — the fix loop

1. **Label and confirm.** Tag the issue `format-drift`; reproduce from the redacted
   sample and the reported tool version.
2. **Capture the new shape as a fixture.** Add a new fixture beside the old one — synthetic,
   or a field-allowlist redaction of a real capture, never a real transcript — and regenerate
   goldens with `-update`. **Keep the old fixture and keep parsing the old shape** — users' disks still hold months of
   history in the previous format; a parser upgrade must handle both, additively.
3. **Fix the parser.** Update mappings; add fuzz seeds for the new shape; `make fuzz`
   is mandatory on any parser change.
4. **Guard the dedupe keys.** A fix must not change how existing records' dedupe keys
   are derived — that would double-count on the next `backfill`. If a key change is
   truly unavoidable, the release notes must say so and document the
   `clear --tool <name>` + re-backfill path.
5. **Re-check the honesty surface.** If the new format changes what a field *means*
   (folded token classes, different cache accounting), the mapping decision goes into
   the parser's package doc **and** a `doctor` caveat line — every modeling assumption
   stays user-visible.
6. **Ship a patch release within days** (see [RELEASING.md](../RELEASING.md)): parser
   fixes are exactly the "patch, days not weeks" case. Name the tool and its affected
   versions in the release notes.

## Out-of-tree parsers (exec plugins)

Plugins get the same posture with inverted ownership: the **plugin author** owns steps
2–5 for their tool, `assaio-agent plugins verify <name>` is their conformance check,
and the boundary validation means a drifting plugin fails *loud* (skipped counts,
violations listed) rather than storing garbage. The wire contract itself
(handshake + JSONL, ADR 0003; metric envelope, ADR 0004) is versioned — a breaking
change to it is a release-notes event on our side, never a silent one.
