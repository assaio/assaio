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
| Drift canaries | `internal/drift`, after every `backfill` | The two failure modes below: numbers that shrink without anything erroring. |

## The two silent failure modes

Everything above catches a parse that *fails*. Neither of these does — which is why they
needed their own defense:

1. **Semantic drift.** If a vendor renames or moves a token field, the line often still
   parses as valid JSON — it just stops matching the usage shape, or maps to zero tokens.
   `skipped` counts *unparseable* lines, not renamed keys, so totals quietly shrink
   instead of failing loudly.
2. **Discovery drift.** If a tool moves its log directory or changes file naming,
   `Discover` finds fewer (or zero) files. `backfill` and `doctor` *show* the counts, but
   nothing flagged "this used to be 300 files and is now 0" as an anomaly.

Both share one property: the failure mode is **plausible-looking underreporting**, which is
exactly what an honesty-first tool must not do silently.

## Detection — three channels

1. **Local canaries, automatic.** After every `backfill`, each source is judged against
   its own recent history:

   | Canary | Fires when | Abstains below |
   |---|---|---|
   | `discovery` | no files found where there used to be some, or fewer than half the recent median | a median of 20 files, for the partial-drop half |
   | `yield` | records per file read collapse to under a quarter of the historical median | 20 files read this pass |
   | `skipped` | at least a tenth of the lines read fail to unmarshal | 50 skipped lines |
   | `zero-token` | at least a quarter of parsed records carry no tokens at all | 50 records |

   The design rules behind those numbers matter more than the numbers: the baseline is a
   **median** of recent runs, so one odd pass cannot move it; every comparison is a
   **ratio**, so an incremental pass that read four files stays comparable to a full one
   that read six thousand; and every canary has a **sample floor** below which it says
   nothing, because a share computed from a handful of records is not evidence. A source
   with no files — an exec plugin — is judged only by the two canaries that need none.

   A breach prints `warning: possible format drift in <tool>` and appears in `doctor`'s
   drift section; `doctor --strict` turns it into an exit code for cron or CI. While a
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

## Reaction — the fix loop

1. **Label and confirm.** Tag the issue `format-drift`; reproduce from the redacted
   sample and the reported tool version.
2. **Capture the new shape as a fixture.** Add a **new synthetic** fixture beside the
   old one (never a real transcript) and regenerate goldens with `-update`. **Keep the
   old fixture and keep parsing the old shape** — users' disks still hold months of
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
