# Query your own data

*Part of [Extending assaio](../extending.md). This page has its own column notes; the [generated
reference](https://assaio.dev/docs/reference) covers commands, flags, config keys, signals, and the
metric contract, but not the storage schema.*

Everything `assaio` collects lives in one SQLite file:

```
~/.local/share/assaio/assaio.db
```

The location honors `XDG_DATA_HOME`. Query this ordinary SQLite database directly with `sqlite3`, DB
Browser, or any client. `assaio` never phones home; this file holds all your data.

## Schema

Two tables hold your data. `usage_record` is one row per API response
([`internal/store/migrations/0001_init.sql`](../../internal/store/migrations/0001_init.sql));
`session_step` is one row per step of a session's sequence and is described after it — measured
on the maintainer's store it is the **larger of the two**, 136.3 MB of table and indexes against
`usage_record`'s 69.7 MB (re-measured for v0.26), which is why it is the one table with a
retention horizon.

`usage_record`:

| Column | Type | Notes |
|--------|------|-------|
| `id` | `INTEGER PRIMARY KEY` | Row id. |
| `tool` | `TEXT` | Source: `claude-code`, `codex`, `gemini-cli`, `copilot-cli`, `cline`, `agy` (Antigravity CLI), or `plugin:<name>` for an out-of-tree parser. |
| `session_id` | `TEXT` | The tool's session/conversation ID. |
| `ts` | `TEXT` | UTC RFC3339 timestamp. Day is `substr(ts,1,10)`. |
| `model` | `TEXT` | Model name as recorded by the tool, or `''` when the source records none — Antigravity CLI writes one nowhere in its format. For an in-tree source, a later `backfill` fills a blank (Cline reads the name from a sidecar) and replaces a name when the re-read of the same file states a different one; exec-plugin rows keep their first answer. |
| `input_tokens` | `INTEGER` | Non-cached input tokens. |
| `output_tokens` | `INTEGER` | Output tokens. |
| `cache_read_tokens` | `INTEGER` | Tokens served from cache. |
| `cache_write_tokens` | `INTEGER` | Tokens written to cache. |
| `reasoning_tokens` | `INTEGER` | Reasoning tokens, when reported. |
| `dedupe_key` | `TEXT` | Unique with `tool` (`UNIQUE(tool, dedupe_key)`). |
| `project` | `TEXT` | Basename of the resolved git repository root, or `''`. Monorepo subdirectories share one value here. |
| `subpath` | `TEXT` | Working directory relative to that repository root (e.g. `apps/mobile`), or `''` at the root. |
| `git_branch` | `TEXT` | Branch name, or `''`. |
| `entrypoint` | `TEXT` | Invocation label, or `''`. |
| `granularity` | `TEXT` | `turn` or `session`. |
| `lines_added` | `INTEGER` | AI-added lines (from diff `+` markers), or `0`. |
| `lines_removed` | `INTEGER` | AI-removed lines (from diff `-` markers), or `0`. |
| `edits` | `INTEGER` | File-editing tool calls, or `0`. |
| `tool_calls` | `INTEGER` | All tool-use calls, or `0`. |
| `rejected` | `INTEGER` | Tool proposals the human declined, or `0`. |
| `compactions` | `INTEGER` | Context-compaction events attributed to the record, or `0`. |
| `rework_lines` | `INTEGER` | AI-added lines later undone within the same transcript+file, or `0`. |
| `member` | `TEXT` | `''` for purely local usage; non-empty only on a central store synced from a team member (see [The team server](team-server.md)). |
| `tool_reads` | `INTEGER` | Tool calls that read a file, `0` for sources that do not name their calls. |
| `tool_searches` | `INTEGER` | Tool calls that searched. |
| `tool_commands` | `INTEGER` | Tool calls that ran a command. |
| `tool_writes` | `INTEGER` | Tool calls that wrote a file. |
| `tool_other` | `INTEGER` | Tool calls in none of the above. The five sum to `tool_calls`. |
| `tool_errors` | `INTEGER` | Tool calls that failed outright. |
| `sidechain` | `INTEGER` | `1` when the turn belongs to a sub-agent rather than the main transcript. |
| `skill` | `TEXT` | The skill this turn was attributed to, `''` when none. |
| `agent` | `TEXT` | The sub-agent this turn was attributed to, `''` when none. |
| `cache_write_1h` | `INTEGER` | The portion of `cache_write_tokens` that bought a 1-hour lifetime. A subset, never added to it. |
| `cache_miss_reason` | `TEXT` | The vendor's own stated reason a cache read missed, `''` when unstated. |
| `project_conflict` | `INTEGER` | `1` when the same completed Claude sub-agent aggregate carried competing non-empty projects. Its `project` and `subpath` stay empty; usage remains counted once. |

Claude Code and Codex parsers populate the activity columns (`lines_added` … `rework_lines`). Since
v0.6, GitHub Copilot CLI also populates `lines_added`/`lines_removed` once per session, not per
turn. Only Claude Code populates `rejected`. Gemini CLI and Cline record no line or edit signals, so
they store `0` throughout: **absent, not zero**. Figures using these columns must filter by source
capability (ADR 0011). These columns hold **counts only**, never the counted code.

**`agy` has the reverse hazard.** Antigravity CLI records `edits`, `tool_calls`, and five purpose
counts but **no token counter anywhere in its format**. Every token column on an `agy` row is
therefore a structural zero. A `SELECT tool, SUM(input_tokens) … GROUP BY tool` based on this page
would show a real source at zero tokens and, if priced, a fabricated `$0`. The binary withholds that
figure, but direct SQL bypasses the check. For token and cost queries, use `WHERE tool <> 'agy'`, or
check the depth matrix (`assaio-agent doctor`, `signals coverage`) before summing across sources.

`report --format csv|json` carries `in`, `out`, `cache_read`, `cache_write`, `cache_write_1h`,
`cache_write_tiered` and `cost`, so a row that holds one model (the default rows and `--by model`)
reproduces its cost from its token columns and that model's prices; a row grouped `--by tool`,
`project` or a label mixes models billed at different rates. `cache_write_1h` is the part of
`cache_write` bought at the 1-hour cache lifetime and billed at its own rate; never add it to
`cache_write`. Only Claude Code states the tier: `cache_write_1h` is `null` (an empty CSV cell) for
a row with no such source, and `cache_write_tiered` is the part of `cache_write` from sources that
do, so a 1-hour share is `cache_write_1h / cache_write_tiered`. Writes with no stated tier are
billed at the standard write rate. Claude Code rows imported before assaio read the tier, whose
transcripts are gone, carry `0` there. In CSV the two are the last columns. `effectiveness --format
csv` adds activity and `$`/100-lines columns.

**Cost is not stored.** The database holds tokens, not dollars. Reports compute cost from the
embedded price table because prices change and unpriced models must remain blank. For cost figures,
use `assaio-agent report --format csv`, which includes a `cost` column, instead of SQL.

**`session_step` holds each sequence:** one row per step, with its kind, position, model, token
total, ending, and an integer representing the touched file, never a path (see
[PRIVACY.md](../../PRIVACY.md)). `trace.horizon_days` bounds retention (default 30). This is the
store's only retention rule; `0` disables it, so the table grows without bound.

**Three bookkeeping tables hold no usage:** `ingest_file` records each parsed input's path, size,
mtime, and parsing build, making repeat `backfill` nearly free. `ingest_source` records files found,
files read, records, skipped lines, and zero-token records for each source and run; [format-drift
canaries](../format-resilience.md) compare against it. `digest_snapshot` stores each `digest` run's
verdicts and totals for the next comparison. All three are caches. Dropping them causes one slow
re-parse, resets the drift baseline, and leaves one digest with nothing to compare against, but has
no other cost. Each pass prunes `ingest_file` to files on disk; `ingest_source` keeps only the
newest runs per tool. None grows with installation age. Run `assaio-agent compact` to return freed
pages to the filesystem; SQLite does not do this automatically.

**Stability.** The schema is not a public API, before or after v1.0
([compatibility](../compatibility.md)): a release may add, reshape or rewrite tables when a
correction needs it. Pin direct queries to the version you wrote them against. For anything another
program depends on, read `report`, `analyze` or `evidence` with `--format json`; those outputs are
what v1.0 freezes.

## Ready-made queries

```sh
DB=~/.local/share/assaio/assaio.db
```

**Token spend per project, last 30 days** (the dimension behind `report --by project`; join your
price sheet for dollars or use the CSV report):

```sh
sqlite3 -header -column "$DB" "
  SELECT project,
         SUM(input_tokens)      AS in_tok,
         SUM(output_tokens)     AS out_tok,
         SUM(cache_read_tokens) AS cache_read
  FROM usage_record
  WHERE ts >= date('now','-30 days')
  GROUP BY project
  ORDER BY out_tok DESC;"
```

**Total tokens per model:**

```sh
sqlite3 -header -column "$DB" "
  SELECT model, SUM(input_tokens + output_tokens) AS total_tok
  FROM usage_record
  GROUP BY model
  ORDER BY total_tok DESC;"
```

**Cache efficiency per project** — cache reads divided by input plus cache reads, as shown in
`Cache%`:

```sh
sqlite3 -header -column "$DB" "
  SELECT project,
         ROUND(100.0 * SUM(cache_read_tokens)
               / NULLIF(SUM(input_tokens + cache_read_tokens), 0), 1) AS cache_pct
  FROM usage_record
  GROUP BY project
  ORDER BY cache_pct DESC;"
```

**Busiest days:**

```sh
sqlite3 -header -column "$DB" "
  SELECT substr(ts,1,10) AS day,
         SUM(input_tokens + output_tokens) AS total_tok,
         COUNT(*)                          AS records
  FROM usage_record
  GROUP BY day
  ORDER BY total_tok DESC
  LIMIT 10;"
```

---
