# Add a data source

*Part of [Extending assaio](../extending.md). No Go or PR needed: [write a parser
plugin](parser-plugin.md).*

A data source is one Go package under `internal/parser/<tool>/`. It converts a tool's on-disk
session logs into normalized `usage.Record` values. The core handles pricing, aggregation, storage,
and rendering.

A parser exposes exactly two functions:

```go
// Discover returns the log files (or task directories) under one root this tool has
// written. Sources with more than one root (Codex, Cline) are looped by the caller —
// one Discover call per root, never a []string of roots.
func Discover(root string) ([]string, error)

// Parse reads one log and returns its normalized usage records, plus the count of
// lines that failed to unmarshal as JSON.
func Parse(r io.Reader) ([]usage.Record, int, error)
```

`Discover` runs a filesystem glob under a root the core resolves (see
[`internal/paths`](../../internal/paths/paths.go)). Keep the glob narrow: `~/.gemini` is shared with
Antigravity CLI, so Gemini matches only `tmp/*/chats/session-*.jsonl` and Antigravity matches only
`antigravity-cli/brain/*/.system_generated/logs/transcript.jsonl`. `Parse` accepts an `io.Reader`,
not a path, so you can test it with a fixture. A source that processes directories may expose
`ParseDir(dir string) ([]usage.Record, int, error)` instead: Cline reads `ui_messages.json` with
`task_metadata.json`, while Antigravity CLI reads one file but gets the conversation id only from
its directory name. Keep a reader-based core (`cline.ParseTask`, `agy.ParseTranscript`) that accepts
the id as an argument, so fixture tests stay hermetic. Prefer the file-based `Parse(io.Reader)` form
when possible.

**While developing a parser, run `assaio-agent backfill --full`.** Ingest skips unchanged inputs it
has already parsed. Stored state uses the build's identity, which stays constant for local builds so
rebuilding does not reparse every file. Released binaries invalidate that state automatically;
development builds do not. Use `--full` to see parser changes.

The root for `Discover` can be the built-in default or a team override. This needs no code change;
see [Custom log-source paths](parser-plugin.md#custom-log-source-paths).

### Declare what your source can answer

Add a row for the parser in [`internal/parser/depth.go`](../../internal/parser/depth.go). Set
`Answers` to the ids of signals the source can produce (`assaio-agent signals list` shows them). A
tier summarizes three axes; `Activity: true` does not identify which activity signals exist. Copilot
CLI records changed lines but no edit count, tool calls, turns, or rework. It lists two activity
signals, not the other four. Declaring only the axis made `signals coverage` claim full support for
sixteen of eighteen signals when only ten were supported.

List fields your source writes but your parser does not read in [What each source's log
carries](source-fields.md), with reasons for each omission.

For each signal, ask: **would a figure computed from my records be right, or merely non-empty?** If
the log lacks the data, omit the signal. An absent signal says the source cannot answer; a declared
signal produces a number people may trust. This applies to tokens: `ai.tokens.reasoning` is declared
per source because Claude Code and Cline never expose a thinking count. Claiming it for them
reported full support for a figure their records leave at zero. A test checks that listed ids exist
and agree with the tier axes ([ADR 0008](../adr/0008-signal-catalog.md)).

**Metrics check this row before reading your records.** For per-session figures such as session mix,
context health, session length, and rejection rate, a validator calls `parser.Answers(tool, id)` and
includes only sessions the source can answer for. A field your parser never writes is missing, not
zero; including it would distort the average. Omitting a signal excludes your sessions from that
figure. Use the `parser.Signal*` constant for each id, never a literal: a typo returns false for
every tool and empties a metric without failing the build.

The row also tells downstream code which tools exist and what they can answer through
`parser.Tools()` and `parser.Answers()`: sync validation, `clear --tool`, every verdict's confidence
envelope, and tool-specific caveats. Wire the parser into `internal/ingest` and the `doctor` scan
separately. Two tests require the sets read by ingest, scanned by doctor, and published by the
matrix to match. Without a row, the team server rejects the parser's records and users cannot delete
its data by source. This happened to Copilot CLI between v0.6 and v0.8.

### Corrupt-line policy: skip and count

Session logs are live, append-only files that a tool may write while `assaio` reads them. Truncated
final lines and bad bytes are expected. `Parse` counts each line that fails `json.Unmarshal` in the
returned `skipped` int and continues, preserving records around it. Lines that parse but contain no
usage are filtered, not counted as skipped; only unmarshal failures count. Scanner failures such as
`bufio.ErrTooLong` beyond `parser.MaxLineBytes` affect the file as a whole and return an error with
context. At the file level, `internal/ingest.Run` counts files it cannot open or parse as `Failed`
and continues, so one corrupt log does not block the rest of a `backfill`.

## The `usage.Record` contract

Each emitted record is one normalized usage event. Fill in fields the log provides in
[`internal/usage/record.go`](../../internal/usage/record.go), and leave the rest at their zero
values.

| Field | Type | Meaning | Rules |
|-------|------|---------|-------|
| `Tool` | `string` | Stable identifier for the source, e.g. `"claude-code"`. | Constant per package. Becomes the `tool` column and pairs with `DedupeKey` for uniqueness. |
| `SessionID` | `string` | The tool's own session/conversation ID. | Pass through verbatim; do not synthesize. |
| `Timestamp` | `time.Time` | When the usage occurred. | Stored as UTC RFC3339. Parse the log's timestamp; do not use "now". |
| `Model` | `string` | Model name as the tool records it. | Pass through verbatim — normalization to the price table happens in the core. |
| `InputTokens` | `int64` | Non-cached input tokens. | If the log's input count **includes** cached tokens (Codex, Gemini do this), subtract them so input and cache-read never double-count. |
| `OutputTokens` | `int64` | Generated output tokens. | Fold tool-use tokens here only if the vendor bills them as output (Gemini); document the choice in a one-line comment and a `doctor` caveat. |
| `CacheReadTokens` | `int64` | Tokens served from prompt cache. | Feeds the `Cache%` column and cache-read pricing. |
| `CacheWriteTokens` | `int64` | Tokens written to prompt cache. | |
| `ReasoningTokens` | `int64` | Reasoning/thinking tokens, when reported separately. | A **subset** of `OutputTokens`, never added to it — clamp with `parser.Subset`, and sum fields with `parser.SumNonNeg`, because plain `+` on int64 overflows into a negative that `NonNeg` then reads as zero. Recorded for transparency; whether they are billed separately is model-dependent. |
| `DedupeKey` | `string` | Stable per-record identity within a `Tool`. | **Must be deterministic** — see below. |
| `Cwd` | `string` | The session's full working-directory path, exactly as the log reports it. | **Never persisted.** `internal/ingest` reads it only to resolve `Project`/`Subpath` (`internal/projectid`) and then discards it. Leave `""` if the log has no cwd — never fabricate one. |
| `Project` | `string` | The **basename of the git repository root** containing the session's working directory. | Set it as a **fallback only** — `filepath.Base(cwd)` — for when ingest cannot resolve a repository root (e.g. `Cwd` left `""`). Whenever `Cwd` is set, ingest overwrites this with the resolved repo-root basename, so a monorepo's subdirectories roll up to one project. |
| `Subpath` | `string` | `Cwd` relative to the resolved repository root (e.g. `apps/mobile`), or `""` at the root. | Set by ingest, not by parsers — leave it at its zero value. Always relative; never an absolute path. |
| `GitBranch` | `string` | Branch name, when the log carries it. | Else `""`. |
| `Entrypoint` | `string` | How the tool was invoked, e.g. `"cli"`, `"sdk-py"`. | Else `""`. |
| `Granularity` | `string` | `"turn"` for per-request records, `"session"` for session aggregates. | **Honesty rule** — see below. |
| `LinesAdded` | `int64` | AI-added code lines for this record — the primary effect proxy. | Count only the `+`-prefixed lines of the edit's diff hunks (or a sub-agent's reported added lines). **The code on the line is never stored — only the count.** `0` if the source exposes no diff. |
| `LinesRemoved` | `int64` | AI-removed code lines. | The `-`-prefixed diff lines, same rule. `0` if unknown. |
| `Edits` | `int64` | File-editing tool calls (`Edit`/`Write`/`NotebookEdit`/`MultiEdit`). | A subset of `ToolCalls`. `0` if the log does not expose tool calls. |
| `ToolCalls` | `int64` | All tool-use calls in the turn, edits included. | `0` if unknown. |
| `Rejected` | `int64` | Tool proposals the human declined — a friction signal. | `0` if unknown. |
| `Compactions` | `int64` | Context-compaction events attributed to this record — a context-strain signal. | `0` if the source exposes no compaction/summarization marker. |
| `ReworkLines` | `int64` | AI-added lines later removed by a subsequent edit to the same file, within one transcript — a rework/thrash proxy. | Computed via the shared [`internal/parser.Rework`](../../internal/parser/rework.go) helper. The file path used to detect it is read transiently and **never** copied onto the record. `0` if unknown. |

Skip records with no token usage; do not emit them with zeros.

### Parsers stay hermetic — project resolution is ingest's job

A parser may read only the `io.Reader` passed to `Parse`. Do not open, stat, or walk other files, or
import `internal/projectid` to resolve `Cwd` to a repository root. Emit `Cwd` as logged and set a
fallback `Project` from your best guess, usually `filepath.Base(cwd)`. After `Parse`,
`internal/ingest` walks the filesystem through `internal/projectid` to resolve `Project` and fill
`Subpath` for every record. This keeps fixture tests free of temporary directories and `.git`
scaffolding, and confines identity resolution to
[`internal/ingest/project.go`](../../internal/ingest/project.go). The file path does not survive
this step, so stored data cannot support a per-file metric (see [What a validator reads:
Input](metric-validator.md#what-a-validator-reads-input)).

### Activity fields are optional (honesty note)

`LinesAdded`, `LinesRemoved`, `Edits`, `ToolCalls`, `Rejected`, `Compactions`, and `ReworkLines` are
session-level activity signals used by `effectiveness` (AI output versus cost) and `analyze`
validators. A new parser **MAY** fill them when its source exposes edit or diff data and **MUST**
leave them at `0` otherwise. Never guess. For line counts, count only `+`/`-` diff markers; never
store line content.

**Claude Code and Codex currently populate the full set: `LinesAdded`, `LinesRemoved`, `Edits`,
`ToolCalls`, `Compactions`, and `ReworkLines`.** Copilot CLI provides only the two per-session line
counts. Claude Code uses structured edit results, sub-agent tool stats, and compaction-boundary
lines; Codex uses `patch_apply_end` diffs, function/custom tool-call events, and `compacted` events.
Both use [`internal/parser.Rework`](../../internal/parser/rework.go) to detect rework. Only Claude
Code populates `Rejected`: Codex rollout logs do not expose tool-use denials the same way. Gemini
and Cline report token usage but leave all activity fields at `0`, so they add cost but no line
counts, as the `effectiveness` view discloses.

### DedupeKey determinism (hard rule)

Inserts are idempotent because the store enforces uniqueness on `(tool, dedupe_key)`, making
repeated `backfill` runs safe. This requires **the same file to produce the same keys every time**.
Derive each `DedupeKey` only from log content, never wall-clock time, random iteration, or map
ordering.

- Use a stable per-record UUID from the log when available; Claude Code uses the message `uuid`.
- Otherwise, derive a positional key such as `fmt.Sprintf("%s:%d", sessionID, index)`, where `index`
  counts emitted records in file order; Codex, Gemini, and Cline do this.

Different keys from two parses of the same unchanged file silently double-count on the next
`backfill`. The golden test below guards against this.

### Granularity honesty (hard rule)

`assaio` distinguishes session totals from per-turn data. If a source reports only whole-session
totals, such as a daily vendor aggregate or one end-of-session summary, **must** set
`Granularity: "session"`. Use `"turn"` only when each record represents one request and response. If
unsure, use `"session"`.

Set granularity for each record, even within one source. Claude Code transcripts are per-turn except
for the record summarizing a completed sub-agent's whole run, which is `"session"`. Before v0.10 it
was labeled `"turn"`, so per-turn figures counted it as one very large turn. Label any aggregate
record by its own shape; the field is per record, not per parser.

## Golden-file testing

Test parsers against captured fixtures in the package's `testdata/` directory and compare parsed
records with a checked-in `.golden` snapshot. Follow
[`internal/parser/claude/claude_test.go`](../../internal/parser/claude/claude_test.go):

- A fixture (`testdata/session.jsonl`) and golden output (`testdata/session.golden`, parsed records
  as indented JSON).
- An `-update` flag to regenerate the golden file:

  ```sh
  go test ./internal/parser/<tool>/ -run TestParseGolden -update
  ```

  Inspect the parse, run the update once, and commit the `.golden` file. Review future diffs: a
  mismatch can reveal a vendor format change.

- **Use a synthetic fixture or an allowlist-redacted real capture. Never commit a real transcript.**
  The default is a minimal synthetic log covering relevant fields and edge cases (dedupe,
  mid-session model switches, cache tokens, missing cwd). A real capture is stronger evidence if
  redacted by allowlist: keep every field the parser reads verbatim; replace fields it does not
  read, using the same number of placeholder lines for a body and stand-ins for identifiers and
  paths; copy nothing else. No prompt, code, path, or name may reach the repository.
  `internal/calibration` marks each trace's type in `capture`: a constructed sample proves parsing,
  while a real one also confirms the vendor's current format.

Add an assertion-style test for behavior the golden file does not show clearly, such as duplicate
collapse, filtering non-usage lines, or dimensions on every record.

## Fuzzing

Every parser needs a native Go fuzz test named `FuzzParse`, or a name that identifies the entry
point if there are several (`FuzzParseTask` for Cline, `FuzzParseTranscript` for Antigravity CLI).
List each fuzzer explicitly in `make fuzz`. Seed `f.Add` with a `testdata/` fixture and hand-written
cases: empty input, `{}`, truncated JSON, int64-max token values, and invalid UTF-8. For every
returned record, assert that `Parse` never panics (a non-nil error may return early),
`skipped >= 0`, token fields are nonnegative, portions stay within totals
(`ReasoningTokens <= OutputTokens`, `CacheWrite1hTokens <= CacheWriteTokens`), `Tool` matches the
package constant, and `DedupeKey` is nonempty. `make fuzz` runs each fuzzer for `FUZZTIME` (default
`20s`). Commit discovered crashers under `testdata/fuzz/` as permanent regression seeds.

## Wire it in

Connect a finished parser to the CLI in three places.

1. **Ingest** — In [`internal/ingest/ingest.go`](../../internal/ingest/ingest.go), add discovery and
   append a `source` containing the tool name, discovered files, and `Parse` function to `sources`.
   Use `dirSource` for directory-based sources. Add the root resolver to
   [`internal/paths`](../../internal/paths/paths.go). If the source fills `Cwd`, project and subpath
   resolution is automatic: every `source` and the Cline branch pass through
   [`internal/ingest/project.go`](../../internal/ingest/project.go) before `Insert`.

2. **Doctor** — In [`internal/cli/doctor.go`](../../internal/cli/doctor.go), add a discovery line so
   `assaio-agent doctor` reports the file count. Add a one-line caveat for each modeling assumption,
   such as folded token classes, recomputed cost, or shared directories. Report every parser
   compromise in `doctor`.

3. **Its size** — Measure storage growth on a real corpus before finishing. Compare
   `SELECT name, pgsize FROM dbstat` before and after ingest into a throwaway store. Report size per
   record and per day, and whether a retention rule covers it. `trace.horizon_days` prunes only the
   step timeline. A source with no steps grows `usage_record` indefinitely; only `clear` followed by
   `compact` frees space. AGENTS.md requires a bound and cleanup path for every source of growth.
   SQLite does not shrink on DELETE, so unmeasured growth can fill a disk.

## The intake path: open a connector issue first

Before coding, open a **Connector request** issue
([`.github/ISSUE_TEMPLATE/connector.yml`](../../.github/ISSUE_TEMPLATE/connector.yml)). Include the
tool, available data channels (local logs, vendor API, OTLP, editor/CLI hooks), and a redacted log
sample. The sample becomes the synthetic fixture. Use the discussion to settle token mapping,
including whether input includes cache and how reasoning is billed, before producing wrong numbers.
A connector is a scoped first contribution that starts with this issue.

---
