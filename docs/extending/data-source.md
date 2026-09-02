# Add a data source

*Part of [Extending assaio](../extending.md). No Go, no PR: [write a parser plugin](parser-plugin.md) instead.*

A data source is one Go package under `internal/parser/<tool>/`. It turns a tool's
on-disk session logs into a slice of normalized `usage.Record` values. That is the
entire job — pricing, aggregation, storage, and rendering are the core's responsibility,
not the parser's.

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

`Discover` is a filesystem glob rooted at a path the core resolves for you (see
[`internal/paths`](../../internal/paths/paths.go)). Keep the glob narrow — `~/.gemini`, for
example, is shared with Antigravity CLI, so the Gemini discoverer only matches
`tmp/*/chats/session-*.jsonl` and the Antigravity one only
`antigravity-cli/brain/*/.system_generated/logs/transcript.jsonl`. `Parse` takes an
`io.Reader` (not a path) so it is trivial to test against a fixture. A source whose unit of
work is a directory rather than a single file may expose a
`ParseDir(dir string) ([]usage.Record, int, error)` helper instead — Cline reads
`ui_messages.json` alongside `task_metadata.json`; Antigravity CLI reads one file but keeps
the conversation id in the directory name and nowhere inside it. Such a parser still keeps a
reader-shaped core (`cline.ParseTask`, `agy.ParseTranscript`) that takes the id as an argument
rather than deriving it, so the parsing stays testable against a fixture and hermetic; the
file-oriented `Parse(io.Reader)` shape is the default and the one to reach for first.

**While you work on a parser, run `assaio-agent backfill --full`.** Ingest skips inputs it
has already parsed unchanged, and the stored state is keyed on the build's identity — which
stays constant for a local build, precisely so a rebuild does not force a re-parse of every
file. A released binary invalidates the state automatically; your development build does
not, so `--full` is how you see a parser change take effect.

Where `Discover`'s root itself comes from — the built-in default, or a team's own
override — is a separate, non-code concern; see [Custom log-source
paths](parser-plugin.md#custom-log-source-paths).

### Declare what your source can answer

A parser also adds one row to the depth matrix in
[`internal/parser/depth.go`](../../internal/parser/depth.go), and the part that matters most is
`Answers`: the ids of the signals your source can actually produce (`assaio-agent signals
list` prints them all). Do not reach for the nearest tier and move on — the tier's three
axes are a summary, and `Activity: true` says nothing about *which* activity. Copilot CLI
records changed lines and no edit count, no tool calls, no turns and no rework, so it lists
two activity signals and not the other four; declaring the axis alone made `signals coverage`
report sixteen of eighteen signals as fully supported when the truth was ten.

What your source writes but you choose *not* to read belongs in the audit below — [What each
source's log carries](source-fields.md) — with the reason,
so the next reader can tell a deliberate omission from an oversight.

The rule is one question per signal: **would a figure computed from my records be right, or
merely non-empty?** If the log does not carry it, leave it out — an absent signal is reported
as "this source cannot answer it", which is a useful fact, while a claimed one becomes a
number someone trusts. That applies inside the token group too: `ai.tokens.reasoning` is
declared per source rather than inherited, because Claude Code and Cline never surface a
thinking count and claiming it for them reported full support for a figure their records can
only leave at zero. A test asserts the ids you list are real and that they do not contradict
the tier axes ([ADR 0008](../adr/0008-signal-catalog.md)).

**A metric reads this row before it reads your records.** A validator that reports a
per-session figure — the session mix, context health, how long sessions run, the rejection
rate — first asks `parser.Answers(tool, id)` and keeps only the sessions your source can
answer for, because a field you never write is not a zero: it is a silence, and averaging it
in reports a fact about someone's work that came from your parser's gap. Leaving a signal out
therefore *removes* your sessions from that figure rather than dragging it down. Spell the id
with the `parser.Signal*` constant, never as a literal — a typo answers false for every tool
and empties a metric instead of failing a build.

The row does more than describe your source. `parser.Tools()` and `parser.Answers()` are how
everything downstream asks what exists and what it can do — sync validation, `clear --tool`,
the confidence envelope on every verdict, and every caveat that used to spell out tool names.
Wiring your parser into `internal/ingest` and `doctor`'s scan is still a separate step, and
two tests bind the three together: the set ingest reads, the set doctor scans, and the set the
matrix publishes must be identical. A parser that ships without its row is not merely
undocumented — its records get rejected by the team server and its data cannot be deleted per
source, which is exactly what happened to Copilot CLI between v0.6 and v0.8.

### Corrupt-line policy: skip and count

Session logs are live, append-only files a tool can be writing to while `assaio` reads
them — a truncated final line or one bad byte is expected, not exceptional. `Parse`
therefore never aborts a file over one malformed line: a line that fails `json.Unmarshal`
is counted in the returned `skipped` int and parsing continues, so the records on either
side of it are never lost to one corrupt entry. A log line that unmarshals fine but
carries no usage is simply *filtered*, not counted as skipped — only unmarshal failures
count. The scanner itself can still fail (e.g. `bufio.ErrTooLong` past `parser.MaxLineBytes`);
that is a structural problem with the whole file, not one line, so it is returned as an
error, wrapped with context. `internal/ingest.Run` mirrors this at the file level: a file
that cannot be opened or parsed at all is counted as `Failed` and the run continues with
the remaining files, so one corrupt log never blocks a `backfill` of the rest.

## The `usage.Record` contract

Every record you emit is one normalized usage event. The struct lives in
[`internal/usage/record.go`](../../internal/usage/record.go); fill in what the log gives you
and leave the rest at its zero value.

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

Records with no token usage should be skipped, not emitted with zeros.

### Parsers stay hermetic — project resolution is ingest's job

A parser's only filesystem access is the `io.Reader` `Parse` was handed. It must never
open, stat, or walk anything else — in particular, it must not import
`internal/projectid` or otherwise try to resolve `Cwd` to a repository root itself. Emit
`Cwd` verbatim from the log and, as a fallback `Project`, your own best guess (typically
`filepath.Base(cwd)`); `internal/ingest` re-resolves `Project` (and fills `Subpath`) for
every record after `Parse` returns, by walking the real filesystem via
`internal/projectid`. This split keeps parsers trivially testable against a fixture
reader — no temp directories, no `.git` scaffolding — and keeps the one place that
touches the filesystem for identity resolution auditable in one file
([`internal/ingest/project.go`](../../internal/ingest/project.go)). It is also why a
per-file metric can't be built from stored data: the file path itself never survives
past this step (see [What a validator reads: Input](metric-validator.md#what-a-validator-reads-input)).

### Activity fields are optional (honesty note)

`LinesAdded`, `LinesRemoved`, `Edits`, `ToolCalls`, `Rejected`, `Compactions`, and
`ReworkLines` are session-level activity signals that power the `effectiveness` report
(AI output vs. cost) and the `analyze` validators. A new parser **MAY** populate them
where its source genuinely exposes edit/diff data, and **MUST** leave them at `0` where
it does not — an honest zero, never a guess. When you do count lines, count only the
`+`/`-` diff markers; the content of the line is never stored.

**Today the Claude Code and Codex parsers are the two that populate the full set —
`LinesAdded`, `LinesRemoved`, `Edits`, `ToolCalls`, `Compactions`, and `ReworkLines`**
(Copilot CLI carries the two line counts per session and none of the rest) (Claude Code from structured edit
results, sub-agent tool stats, and compaction-boundary lines; Codex from
`patch_apply_end` diffs, function/custom tool-call events, and `compacted` events — both
share the [`internal/parser.Rework`](../../internal/parser/rework.go) helper for rework
detection). `Rejected` is Claude-Code-only: Codex's rollout logs don't surface tool-use
denials the way Claude Code's do. Gemini and Cline report token usage but leave every
activity field at `0`, so they contribute cost but not line counts — which is exactly what
the `effectiveness` view discloses.

### DedupeKey determinism (hard rule)

Inserts are idempotent: the store's uniqueness constraint is `(tool, dedupe_key)`, so
`backfill` is safe to run repeatedly. That guarantee only holds if **re-parsing the same
file always produces the same keys**. A `DedupeKey` must therefore be a pure function of
the log's content, never of wall-clock time, iteration randomness, or map ordering.

- When the log gives you a stable per-record UUID, use it directly — Claude Code keys on
  the message `uuid`.
- Otherwise, derive a positional key like `fmt.Sprintf("%s:%d", sessionID, index)` where
  `index` counts emitted records in file order — Codex, Gemini, and Cline do this.

If two parses of one unchanged file disagree on keys, you will silently double-count on
the next `backfill`. The golden test below is your guard against exactly that.

### Granularity honesty (hard rule)

`assaio` will not let session-level data masquerade as per-turn data. If your source only
reports totals for a whole session (a daily vendor aggregate, a single end-of-session
summary), you **must** set `Granularity: "session"`. Emit `"turn"` only when each record
genuinely corresponds to one request/response. When in doubt, choose `"session"` — an
honest coarse label beats a precise-looking lie.

The rule bites inside a source, not only between them: a Claude Code transcript is per-turn
throughout except for the one record summarizing a completed sub-agent, which totals a whole
run and is therefore `"session"`. It was labelled `"turn"` until v0.10, which let every
per-turn figure count it as a single very large turn. If one shape in your log aggregates,
label that shape — the field is per record, not per parser.

## Golden-file testing

Parsers are tested against captured fixtures under the package's `testdata/` directory,
compared to a checked-in `.golden` snapshot of the parsed records. The convention (see
[`internal/parser/claude/claude_test.go`](../../internal/parser/claude/claude_test.go)):

- A fixture (`testdata/session.jsonl`) and its golden output (`testdata/session.golden`,
  the records marshaled as indented JSON).
- An `-update` flag that regenerates the golden file:

  ```sh
  go test ./internal/parser/<tool>/ -run TestParseGolden -update
  ```

  Run it once you have eyeballed the parse, then commit the `.golden` file. Review it in
  the diff on every future change — a golden mismatch is how you catch a vendor changing
  their format out from under you.

- **A fixture is synthetic, or a field-allowlist redaction of a real capture. Never a real
  transcript.** Fabricating a minimal log that exercises the fields and edge cases you care
  about (dedupe, model switches mid-session, cache tokens, missing cwd) is the default and
  keeps the test's intent legible. A redaction is the stronger option where you have a real
  corpus, and it is only a redaction if it is by allowlist: every field the parser reads stays
  verbatim, every field it does not is replaced — a body by the same number of placeholder
  lines, an identifier or path by a stand-in — and nothing the parser never reads is copied at
  all. Either way the rule the fixture exists to keep is the same one, and it is absolute:
  no prompt, no code, no path, no name reaches the repository. `internal/calibration` records
  which of the two a trace is, in its `capture` field, because a constructed sample proves the
  reading and only a real one also proves the shape is still what the vendor writes.

Add a second, assertion-style test for behavior the golden file cannot make obvious —
that duplicates collapse, that non-usage lines are filtered, that dimensions land on every
record.

## Fuzzing

Every parser must ship a native Go fuzz test — `FuzzParse`, or a name saying which entry point
it drives when the parser has more than one (`FuzzParseTask` for Cline, `FuzzParseTranscript`
for Antigravity CLI). Add it to `make fuzz`, which names each fuzzer explicitly. It seeds `f.Add` with the package's `testdata/` fixture
plus a few hand-written edge seeds (empty input, `{}`, a truncated JSON line, int64-max
token values, invalid UTF-8), and asserts the parser's invariants on every returned
record: `Parse` never panics (a non-nil error returns early, which is fine), `skipped >= 0`,
no token field is negative, a field the log states as a portion of another stays inside it
(`ReasoningTokens <= OutputTokens`, `CacheWrite1hTokens <= CacheWriteTokens`), `Tool` equals
the package constant, and `DedupeKey` is
non-empty. `make fuzz` runs each fuzzer for `FUZZTIME` (default `20s`); a discovered
crasher is committed as a corpus file under `testdata/fuzz/` so it becomes a permanent
regression seed.

## Wire it in

Three touch points connect a finished parser to the CLI.

1. **Ingest** — [`internal/ingest/ingest.go`](../../internal/ingest/ingest.go). Add your
   discovery call and append a `source` (tool name + discovered files + `Parse` function)
   to the `sources` slice. Directory-oriented sources append a `dirSource` instead.
   Add the root resolver to [`internal/paths`](../../internal/paths/paths.go). If your
   source populates `Cwd`, project/subpath resolution happens automatically — every
   `source` and the Cline branch already run through
   [`internal/ingest/project.go`](../../internal/ingest/project.go) before `Insert`;
   nothing more to wire up.

2. **Doctor** — [`internal/cli/doctor.go`](../../internal/cli/doctor.go). Print a discovery
   line so `assaio-agent doctor` reports how many files were found, and add a one-line
   caveat for any modeling assumption your parser makes (folded token classes, recomputed
   cost, shared directories). Every honesty compromise the parser makes belongs in
   `doctor` output.

3. **Its size** — measure what the source costs the store, on a real corpus, before you call
   it done. `SELECT name, pgsize FROM dbstat` before and after an ingest into a throwaway
   store gives the per-record and per-day figure; state both, and state whether any of it is
   reachable by a retention rule. `trace.horizon_days` prunes the step timeline and nothing
   else, so a source that emits no steps accumulates in `usage_record` forever and only
   `clear` plus `compact` frees it. AGENTS.md requires a bound with a cleanup path for every
   growth, and a parser is a growth: SQLite never shrinks on DELETE, so an unbounded source
   nobody measured is discovered as a full disk rather than as a number.

## The intake path: open a connector issue first

Before writing code, open a **Connector request** issue
([`.github/ISSUE_TEMPLATE/connector.yml`](../../.github/ISSUE_TEMPLATE/connector.yml)). It
captures the tool, which channels its data is available through (local logs, vendor API,
OTLP, editor/CLI hooks), and — most importantly — a redacted sample of the log format.
That sample becomes the synthetic fixture, and the discussion settles the token-mapping
questions (does input include cache? how is reasoning billed?) before they turn into
wrong numbers. A connector is a well-scoped first contribution; the issue is where it
starts.

---
