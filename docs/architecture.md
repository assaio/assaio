# Architecture

One binary, one path. A session log a vendor wrote becomes a figure a person reads through
ten hand-offs, each owned by exactly one package. Every one of those surfaces is documented
somewhere — [ADR 0001](adr/0001-language-and-architecture.md) says why it is a single Go
binary, [AGENTS.md](../AGENTS.md) lists what lives where, [`extending.md`](extending.md)
walks each extension point — and nothing walks the route end to end. This does.

Each step below names the package that owns it and the type or function the hand-off
crosses on. Where an out-of-tree plugin attaches, and where the team server splits off, are
marked in place rather than described afterwards: they are steps in this path, not a second
one.

The whole path is local. The only two places it touches a network are the opt-in server
steps, both marked below; what crosses a machine boundary, and under whose control, is
[`threat-model.md`](threat-model.md) and [`PRIVACY.md`](../PRIVACY.md).

## 1. Discovery — `internal/paths`, `internal/ingest`

`internal/paths` answers where each tool writes: `ClaudeRoot`, `CodexRoots`, `GeminiRoot`,
`CopilotRoot`, `ClineRoots`, plus `DataDir`/`DBPath`/`ConfigPath` for assaio's own files
(all XDG-aware). `paths.Resolve(configured, defaults...)` is the override rule: a non-empty
`sources.<tool>` list **replaces** the built-in roots entirely and is never merged with
them, so a configured path can never silently combine with a stale default.

`internal/ingest` then asks each parser for its own files — `discoverSources`,
`discoverClaude`, `discoverClineDirs` calling the per-package
`Discover(root string) ([]string, error)`. Claude Code is discovered twice, main
transcripts and `claude.DiscoverSubagents`, because a completed sub-agent appears in both
its parent's transcript (as a last-turn summary) and its own file (in full), and only the
second is authoritative.

An input this build already parsed unchanged is skipped before it is opened: `ingest`'s
`skipper` reads the `ingest_file` table (path, size, mtime, build) and counts the skip as
`Result.Unchanged`. A new build never trusts the previous one's state and re-reads
everything once, which is how stored history gains a signal an older parser could not
extract.

## 2. Parse — `internal/parser/<tool>`

One data source, one package. The contract is `Parse(io.Reader) ([]usage.Record, int, error)`
— records, skipped-line count, error — with two documented shapes of the same thing:
`cline.ParseDir` reads a task directory rather than a stream, and `claude.ParseAll` and
`codex.ParseAll` return the records *and* the step sequence from a single scan, with
`Parse` (and, for Claude, `ParseSteps`) as wrappers over it.

The shared half lives in `internal/parser` itself and is what makes the parsers agree:
`NewScanner` (bounded by `MaxLineBytes`, 16 MiB), `NonNeg` and `SumNonNeg` for counts a log
can state as garbage, `Subset` for a portion clamped to its whole, `VocabularyToken` for a
vendor's closed-vocabulary label, `ToolCounts`/`StepKind` for the tool-name allowlist (the
name is classified and then dropped), and `Rework` for the per-file add-then-remove budget.

The policy is skip-and-count, never discard-on-error: a line that fails to decode increments
the skipped counter and the scan continues, and a file that fails partway through still
returns everything it recovered before the failure. `ingest.ingestParsed` folds that outcome
in — `parseErr` marks the file `Failed` and never throws away its records.

Adding one: [`extending/data-source.md`](extending/data-source.md). Every parser ships a
native `FuzzParse` with a seed corpus.

## 3. Normalize — `internal/usage`, `internal/projectid`

`usage.Record` is the normalized event — one turn, or one whole-session aggregate, with
`Granularity` saying which so the two are never summed into a total nobody can interpret.
`usage.Step` is the ordered reading of the same transcript
([ADR 0012](adr/0012-session-step-timeline.md)).

Two fields are decided here rather than by the parser. `ingest.resolveProjects` passes each
record's `Cwd` to `projectid.Resolve`, which walks up to the nearest `.git` and returns the
repository root's basename and the working directory's path relative to it — so a monorepo's
subdirectories roll up into one project. `Record.Cwd` is tagged `json:"-"` and is never a
stored column; the full path exists only in memory for the length of that walk.

`ingest.dated` drops a record whose timestamp is the zero value and counts it as skipped.
The reason is that every report, validator and dashboard window is bounded by `ts >= ?`: a
zero-stamped row would be stored, invisible to all of them, and still counted in the store's
size and totals.

What a source can and cannot answer is declared beside the parsers, in `parser.Depth` and
queried through `parser.Answers` / `SourcesAnswering` — one place, so no metric keeps a
second opinion about a source's capability ([ADR 0008](adr/0008-signal-catalog.md),
[ADR 0011](adr/0011-capability-gated-metrics.md)).

## 4. Validate at the boundary — `internal/usage`

`usage.CheckTimestamp` (range: `TimestampFloor` .. now + `FutureSkew`) and
`usage.CheckCounts` (non-negative, under `MaxCount`, subsets not larger than their wholes)
are the bounds every record arriving from **outside this process** must satisfy.

There are exactly two such boundaries, and they share these functions on purpose — they had
drifted apart before, and the same year-9999 record was rejected over the wire and accepted
from a subprocess:

- an exec parser plugin's stdout — `plugin.parseRecordLine` → `wireRecord.toRecordAt`, which
  decodes strictly (`DisallowUnknownFields`), bounds every string at `maxWireStringLen`, and
  namespaces the result as `plugin:<name>`;
- a team member's HTTP push — `server.validateRecord`, which adds what only a server needs:
  a `Tool` from the known set or a well-formed `plugin:` namespace, a known `Granularity`,
  `maxStringField`, `cacheMissReasonPattern`, and `validateToolPurposeSplit`.

An in-tree parser is not re-validated here. Its output is clamped where it is read, in step
2, by the shared helpers — which is why those helpers exist rather than a validation pass.

## 5. Upsert — `internal/store`

One embedded SQLite file, opened by `store.Open` (WAL, migrations applied on open). Every
insert is idempotent on `(tool, dedupe_key)`, and the three entry points differ in what a
*duplicate* is allowed to do:

- `Store.InsertLocal` — for files this store read itself. A duplicate is handed to
  `restateActivitySQL`: token counts take `MAX(stored, offered)` because they are the
  vendor's own figure read off an append-only log, while every count assaio *derived* is
  assigned, so a corrected attribution rule can reach history. A restatement that moves a
  figure **down** is counted separately (`Result.Lowered`, printed as `restated-down`) —
  from the store's side a fix landing and a parser regression look identical, so the number
  is reported rather than inferred.
- `Store.Insert` — first-write-wins, for rows whose caller does not own the input: exec
  parser plugins, `demo`, `share`.
- `Store.InsertSynced` — `InsertLocal`'s restate on a central store, safe only because the
  sync endpoint prefixes every dedupe key with `<member>:`, giving each row exactly one
  possible writer.

Steps go in through `ingest.ingestSteps` and are bounded by `trace.horizon_days`;
`ingest.pruneTrace` deletes what is past the horizon and reports how many, because a
deletion nobody counts is the silent loss the whole skip-and-count policy exists against.

No usage row holds a price, a path, a prompt, or a line of code. The one full local path in
the store is the `ingest_file` bookkeeping table's — the signature (path, size, mtime, build)
that decides whether an input still needs parsing — and it is never synced, exported, or
rendered.

## 6. Price — `internal/pricing`

`pricing.Load()` parses the LiteLLM snapshot embedded at build time (`//go:embed
litellm.json`) once per process. `Table.Cost(*usage.Record)` and
`Table.CostTokens(model, Tokens)` try the exact model name, then `pricing.NormalizeModel`,
and return `ok=false` when neither matches — which becomes `Priced: false` and a nil `Cost`,
never a zero. A cache write is split at its 1-hour portion and each part billed at its own
rate.

Pricing is a read-time operation. The store has no cost column, so refreshing the table
re-prices all of history and no stored row can disagree with the table the binary carries.

## 7. Aggregate — `internal/store`, `internal/report`

`Store.Usage(ctx, since)` returns `[]store.UsageRow` already grouped in SQL by
day/tool/model/project/entrypoint/member/granularity, with the sums spelled out once in
`usageAggregates` so the plain, filtered and label-grouped variants cannot drift apart on
what a total contains.

`report.Build(rows, table)` prices each row into a `report.Row` and renders member names as
pseudonyms; `report.BuildIdentified` is the deliberate, greppable opt-in for raw ones.
`report.Aggregate(rows, by)` folds to a single dimension, and `report.CollapseForTable`
folds further — but only for the display that has no column to show the difference in.

## 8. Validator — `internal/analyze`

`analyze.BuildInput` turns the aggregates into the read-only `analyze.Input` bundle,
precomputing `ByModel`, `ByProject` and `Totals` once so no metric re-groups the same rows.

One metric is one file implementing `analyze.Validator` (`Name`, `Title`, `Describe`,
`Layer`, `Analyze`) and self-registering from that file's `init()` through
`analyze.Register`; `analyze.Validators()` is the registry every surface walks. `Layer` is a
method rather than a field because a field is one a metric will forget, and a missing
measurement layer is a claim about the world ([ADR 0013](adr/0013-measurement-layers.md)).

Each `analyze.Result` carries its own honesty apparatus: the `Read` verdict, the `Layer`,
`Figures`, `Caveats`, `Withheld` (declared inputs this window could not supply, which is not
the same as a metric that looked and found nothing), and `Confidence`. A metric reads only
the sources that record its field, so a structural silence never averages in as a zero.

Adding one: [`extending/metric-validator.md`](extending/metric-validator.md).

## 9. Render — `internal/report`, `internal/analyze`, `internal/dashboard`, `internal/share`

Four renderers over the same results, none of which originates a figure:

- `report.RenderTable` / `RenderJSON` / `RenderCSV`, and `analyze.RenderResultText` /
  `RenderRankingText` for the CLI;
- `dashboard.Build(in, window, anonymize, subpaths, extra) dashboard.Data` then
  `dashboard.RenderHTML` — one self-contained HTML file, all styling inline, no external
  font, script or request. Project names are pseudonymized by default through
  `internal/pseudonym` (a per-install HMAC key, not a bare hash), because the file is meant
  to be shared;
- `share.Build` for the artifact built *for* publication, whose redaction is structural
  rather than a flag: no field it renders can hold a repository, member, path, branch, skill
  or sub-agent name ([ADR 0014](adr/0014-public-artifact-rules.md)).

The dashboard walks `Data.Verdicts` generically, so a newly registered validator appears on
the page with no template change.

## 10. Export — `internal/store`, `internal/docs`

Four things leave the process, and only when asked:

- `Store.Export(ctx, since)` returns raw `usage.Record` rows — the source data
  `assaio-agent sync` pushes, and the one place a record travels un-aggregated. It never
  scans `cwd`, because that is not a column;
- `dashboard` and `share` write an HTML file to a path you name (`share` then asks the
  desktop to open it, the only program assaio launches);
- `report --format json|csv` writes to stdout;
- `docs.Export(root)` projects the binary's own live registries into `docs/reference.json`
  and `site/`, regenerated by `make docs` and held to a no-diff check by `make test`, so a
  published page cannot describe a capability this binary lacks.

## Where an exec plugin attaches

All three protocols are opt-in from `config.yaml` only — never discovered from `PATH`,
never downloaded — resolved by `plugin.Resolve` / `plugin.ResolveMetric` (which validates
the name, the declared `needs`, and the timeout before `exec.LookPath` touches the
filesystem) and run as a subprocess under `Config.Timeout`.

- **Parser** ([ADR 0003](adr/0003-exec-plugin-protocol.md)) — enters at step 4.
  `ingest.ingestPlugins` → `plugin.Run`, which invokes `<command> scan` with
  `ASSAIO_PLUGIN_PROTOCOL=1`, requires a handshake line, and reads JSONL records past it.
  Records land through `Store.Insert` (first-write-wins) under `parser.PluginPrefix`, so a
  plugin can never impersonate a built-in source.
- **Metric** ([ADR 0004](adr/0004-exec-metric-plugin-protocol.md)) — enters at step 8.
  `plugin.RunMetric(ctx, cfg, *analyze.Input)` writes one document on stdin and reads one
  `analyze.Result` back, appended to the built-in verdicts *before* anonymization so the
  same pseudonymization applies. The team server never executes them.
- **Rule** ([ADR 0005](adr/0005-exec-rule-plugin-protocol.md)) — enters after step 8, in
  `assaio-agent check` only. `plugin.RunRule(ctx, cfg, verdicts)` sends the window's
  verdicts with their ranked `Bars` stripped by `buildRuleInput` (those are where project,
  skill and sub-agent names appear, and a rule gates on a verdict, not on which repository
  produced it) and reads severity alerts back. An `error` alert fails `check`.

Everything a plugin emits is validated at the boundary before it is stored or rendered.

## Where the team server splits off

The team server does not fork the path; it re-enters it.

`assaio-agent sync` reaches step 10, then POSTs the exported records to `/v1/usage`. The
server (`internal/server`) authenticates the bearer token **before reading a byte of the
body**, re-enters at step 4 through `validateRecord` — rejecting a bad push whole rather
than partially poisoning a shared store — tags each record with its member, prefixes the
dedupe key, and re-enters at step 5 through `InsertSynced`.

`assaio-agent serve` re-runs steps 7 through 9 over that central store:
`server.handleDashboard` builds the same `dashboard.Data` with `anonymize = true` and no
metric plugins, and serves it behind the same token.

The MVP boundary is stated in `internal/server`'s package doc and repeated in
[`threat-model.md`](threat-model.md): no TLS of its own, so it belongs behind a reverse
proxy on a network you trust.
