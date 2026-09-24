# Format resilience — detecting and reacting to vendor log-format drift

Every format `assaio` parses is **vendor-internal**. The tools do not document session logs as
stable interfaces, and any release may change their shape without notice; `doctor` discloses this on
every run. This guide covers safeguards for the numbers, detection of silent drops, and the detect →
triage → fix → release process.

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

The checks above catch parse failures. They miss these cases, which need separate defenses:

1. **Semantic drift.** A renamed or moved token field may still produce valid JSON but stop matching
   the usage shape or map to zero tokens. `skipped` counts *unparseable* lines, not renamed keys, so
   totals can silently shrink.
2. **Discovery drift.** If a tool moves its log directory or renames files, `Discover` finds fewer
   or none. `backfill` and `doctor` *show* file counts, but did not flag a drop from 300 files to 0.
3. **Additive drift.** If a vendor starts *recording something new*, every figure assaio already
   publishes stays exactly as correct as it was. Nothing shrinks or errors, so no canary fires, but
   assaio misses data the source now provides.

The first two can cause **plausible-looking underreporting** that an honesty-first tool must flag.
Additive drift needs a separate defense because the checks above target drops and failures.

### Additive drift has no canary, and cannot have one

The canaries below compare a source with **its own history**: file counts, records per file, skips,
and zero-token records. Adding a field or event changes none of these. Detect it with a **periodic
field audit**: list every key path in a current corpus and compare it with what the parser reads.
[source-fields.md](extending/source-fields.md) records each section's corpus and capture date; a
stale audit can miss a field present on disk for months.

The v0.26 re-audit found Codex's `event_msg/item_completed`:

| | |
|---|---|
| First seen | 2026-08-20 (Codex CLI ~0.148) |
| Coverage when found | **1,614 of 1,614** September rollouts, 833 of 1,001 August ones, 0 of 10 July ones |
| Volume | 14,268 events across 2,625 rollouts |
| What it carries | per-step wall-clock duration; a `CommandExecution` `exit_code` — the only place Codex says a command *failed* rather than *returned* |
| Canaries that fired | **none, correctly**: records per file, files found, skips and zero-token share were all unchanged |
| Audit that would have found it | the field audit — whose Codex section was taken on 21 rollouts, all of them older than the event |

**The field audit is a detector, not documentation.** Repeat it regularly on a current corpus or it
describes an outdated tool. The audit's Codex section records the measured finding: the code
deliberately does not read that event because its ids join to nothing.

## Detection — three channels, and a fourth that is not automatic

0. **Audit fields manually and regularly.** Only this detects additive drift, and nothing runs it
   for you. Recheck [source-fields.md](extending/source-fields.md) against a current corpus,
   recording the corpus and date for each section. The three checks below compare a source with its
   history and cannot find a field the parser never read.

1. **Local canaries run automatically.** After each `backfill`, four of five compare a source with
   its recent history; one checks a condition:

   | Canary | Fires when | Abstains below |
   |---|---|---|
   | `discovery` | no files found where there used to be some, or fewer than half the recent median | a median of 20 files, for the partial-drop half |
   | `yield` | records per file read collapse to under a quarter of the historical median | 20 files read this pass |
   | `skipped` | skips average one or more per file read | 50 skipped inputs |
   | `zero-token` | at least a quarter of parsed records carry no tokens at all | 50 records, **and** a source whose depth row declares a token counter |
   | `barren` | files are found and no run on record has read a usage record out of them | nothing — the condition is absolute |

   The baseline is a **median** of recent runs, so one odd run cannot move it. Every comparison uses
   a **ratio**, making an incremental run of four files comparable with a full run of six thousand.
   Canaries that calculate shares have a **sample floor**; below it, too few records support a
   judgment. A source without files, such as an exec plugin, gets only the `zero-token` check, which
   needs no files.

   `barren` checks a condition instead: a source that never worked has a zero baseline, so
   comparisons cannot detect a drop. This was measured by setting all four sample floors to `1` and
   rerunning the real corpus; neither build triggered them. It reads all history because one
   incremental run whose single changed input yields nothing does not mean the source is barren.

   `zero-token` also checks capability, not just sample size. A source whose depth row has no token
   signal is exempt. Antigravity CLI has no counter in its format, so a healthy run has 100%
   zero-token records. Checking that share would warn on every backfill and fail `doctor --strict`
   on correct data. An unknown source is still checked: lack of a matrix entry does not show that it
   has no tokens.

   A breach prints `warning: possible format drift in <tool>`; `barren` prints
   `warning: nothing read from a detected source`, since the condition alone is not a diagnosis.
   Both appear in `doctor`'s drift section, and `doctor --strict` exits non-zero for cron or CI.
   While a discovery canary is active, per-input ingest state is frozen rather than pruned: missing
   files and failed discovery look alike, and the state is evidence.
2. **User reports.** A report that numbers dropped after a <tool> update suggests drift. Label the
   issue **`format-drift`**. Ask for the tool version, `assaio-agent doctor` output, and a few
   **redacted** sample lines. Follow the [connector intake
   flow](extending/data-source.md#the-intake-path-open-a-connector-issue-first): never request a
   real transcript, prompts, or code.
3. **Maintainer canary (manual).** When a covered tool ships a major release, make a fresh throwaway
   session, run `backfill` and `doctor` against a scratch store (`--db`), and inspect the counts.
   This can catch drift before users report it.

## The most brittle source, named

The six sources have different risks. **Antigravity CLI (`agy`) is the most brittle of the six** for
three reasons:

- **The binary self-updates.** Antigravity CLI 1.1.23 was verified on 2026-09-02; a `.old` copy from
  the previous day sat beside it, showing two versions on one machine in consecutive days. Users are
  not pinned to the parser's tested version, so the depth matrix names Antigravity CLI 1.1.23.
- **The schema is unpublished** and its directory is shared with another tool. `~/.gemini` also
  holds Gemini CLI, so both discoverers use narrow globs and neither scans the shared root.
- **Accounting uses unnamed protobuf fields.** The parser deliberately reads none of them (see [what
  each source's log carries](extending/source-fields.md)). It reads named JSON keys, which make this
  source readable.

Which canaries catch these failures, and which do not:

| If Antigravity CLI… | caught by |
|---|---|
| moves or renames `brain/<id>/.system_generated/logs/` | `discovery` — 500 conversations to none |
| renames `source`, `created_at` or `step_index` | `barren` on a fresh store, `yield` on an existing one: entries still parse as JSON and stop producing records |
| writes a `created_at` in another format | `skipped` — undatable turns are counted, not silently dropped |
| renames `tool_calls` | **nothing.** The corpus holds 26 tool calls across 500 conversations, which is far too sparse to form a baseline any share could be judged against. Stated here rather than guarded, because a canary computed from 26 observations is not evidence. |

The last row marks the limit. It is why `agy` is `activity-only`, not `standard`: its few figures
can be checked by eye, and none measures cost.

## Reaction — the fix loop

1. **Label and confirm.** Label the issue `format-drift`; reproduce it with the redacted sample and
   reported tool version.
2. **Capture the new shape as a fixture.** Add a synthetic fixture or field-allowlist redaction of a
   real capture beside the old one; never use a real transcript. Regenerate goldens with `-update`.
   **Keep the old fixture and keep parsing the old shape** because users still have months of older
   logs. Parser upgrades must support both shapes additively.
3. **Fix the parser.** Update mappings, add fuzz seeds for the new shape, and run `make fuzz` on
   every parser change.
4. **Guard the dedupe keys.** Do not change keys for existing records or the next `backfill` will
   double-count. If a change is unavoidable, explain it in release notes and document
   `clear --tool <name>` followed by re-backfill.
5. **Re-check the honesty surface.** If the new format changes a field's meaning, such as folded
   token classes or cache accounting, record the mapping decision in the parser package doc **and**
   a `doctor` caveat line. Keep every modeling assumption visible to users.
6. **Ship a patch release within days** (see [RELEASING.md](../RELEASING.md)). Parser fixes follow
   the "patch, days not weeks" rule. Name the tool and affected versions in the release notes.

## Out-of-tree parsers (exec plugins)

For plugins, the **plugin author** owns steps 2–5 for their tool.
`assaio-agent plugins verify <name>` checks conformance, and boundary validation makes a drifting
plugin fail loudly with skip counts and listed violations instead of storing bad data. The wire
contract is versioned (handshake + JSONL, ADR 0003; metric envelope, ADR 0004). A breaking change
requires release notes on our side.
