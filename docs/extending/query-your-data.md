# Query your own data

*Part of [Extending assaio](../extending.md). The column notes below are this page's own; the
[generated reference](https://assaio.dev/docs/reference) covers commands, flags, config keys,
signals and the metric contract, not the storage schema.*

Everything `assaio` collects lives in one SQLite file:

```
~/.local/share/assaio/assaio.db
```

The location honors `XDG_DATA_HOME`. It is an ordinary SQLite database — point `sqlite3`,
DB Browser, or any client at it and query directly. `assaio` never phones home, so this
file is the whole of your data.

## Schema

Two tables hold your data. `usage_record` is one row per API response
([`internal/store/migrations/0001_init.sql`](../../internal/store/migrations/0001_init.sql));
`session_step` is one row per step of a session's sequence and is described after it — measured
on the maintainer's store it is the **larger of the two**, 102.0 MB of table and indexes against
`usage_record`'s 58.3 MB, which is why it is the one table with a retention horizon.

`usage_record`:

| Column | Type | Notes |
|--------|------|-------|
| `id` | `INTEGER PRIMARY KEY` | Row id. |
| `tool` | `TEXT` | Source: `claude-code`, `codex`, `gemini-cli`, `copilot-cli`, `cline`, `agy` (Antigravity CLI), or `plugin:<name>` for an out-of-tree parser. |
| `session_id` | `TEXT` | The tool's session/conversation ID. |
| `ts` | `TEXT` | UTC RFC3339 timestamp. Day is `substr(ts,1,10)`. |
| `model` | `TEXT` | Model name as recorded by the tool, or `''` when the source records none — Antigravity CLI writes one nowhere in its format, and a source that learns the name later (Cline reads it from a sidecar) fills the blank on the next `backfill`. |
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

The activity columns (`lines_added` … `rework_lines`) are populated by the Claude Code and
Codex parsers, and `lines_added`/`lines_removed` by GitHub Copilot CLI since v0.6 (once per
session, not per turn); `rejected` is Claude-Code-only. Gemini CLI and Cline record no line or
edit signal at all, so they store `0` throughout — **absent, not zero**, which is why every
figure over these columns filters by what a source can answer (ADR 0011). They hold **counts only** — never the code content of the lines
they count.

**`agy` is that hazard inverted, and it is the one this page can hand you.** Antigravity CLI
records `edits`, `tool_calls` and the five purpose counts and publishes **no token counter
anywhere in its format**, so every token column on an `agy` row is a structural zero. A
`SELECT tool, SUM(input_tokens) … GROUP BY tool` written off this page therefore reports a real
source at zero tokens, and priced, at a fabricated `$0` — the figure the binary refuses to
print, on the one surface that bypasses its withholding. Filter token and cost queries with
`WHERE tool <> 'agy'`, or read the depth matrix (`assaio-agent doctor`, `signals coverage`)
before summing a column across sources.

`report --format csv` covers tokens and cost, with one gap tracked as `B197`: it carries `in`,
`out`, `cache_read`, `cache_write` and `cost`, but **not `cache_write_1h`**, and the 1-hour
cache-write tier bills at its own rate. Recomputing cost from the four published token columns
therefore misses `cache_write_1h × (1h rate − standard write rate)` — measured against the
maintainer's store that remainder is **$2,582.38** for `claude-opus-5` alone. Read
`cache_write_1h` from the table above when a check has to reconcile. `effectiveness --format
csv` adds the activity and `$`/100-lines columns.

**Cost is not stored.** The database holds tokens only; dollar cost is computed at report
time against the embedded price table, because prices change and unpriced models must stay
honestly blank. For cost figures, use `assaio-agent report --format csv` (which carries a
`cost` column) rather than SQL.

**Beside those two, `session_step` holds the sequence**: one row per step, carrying the kind of
step, its position, the model, its token total, how it ended, and an integer standing for the
file it touched — never a path (see [PRIVACY.md](../../PRIVACY.md)). It is bounded by
`trace.horizon_days` (30 by default), which is the only retention rule in the store; `0` turns
it off and the table then grows without bound.

**Three bookkeeping tables sit beside them**, none holding usage: `ingest_file` (one row per
input already parsed — path, size, mtime, parsing build) makes a repeat `backfill` nearly
free, and `ingest_source` (one row per source per run — files found, files read, records,
skipped lines, zero-token records) is the baseline the [format-drift
canaries](../format-resilience.md) compare against, and `digest_snapshot` (the verdicts and
totals each `digest` run reported, so the next one can say what moved). All three are caches:
dropping them costs one slow re-parse, a reset drift baseline and a digest with nothing to
compare against, nothing more. `ingest_file` is pruned to what is
actually on disk after each pass, and `ingest_source` keeps only the newest runs per tool,
so none of them grows with how long assaio has been installed. Use `assaio-agent compact` to
return freed pages to the filesystem — SQLite does not do that on its own.

**Stability.** The schema may still evolve before v1.0. Changes will be additive where
possible — new nullable columns rather than renames — but treat direct queries as coupled
to a version you have pinned, not a frozen contract. The report/JSON/CSV output is the
more stable surface.

## Ready-made queries

```sh
DB=~/.local/share/assaio/assaio.db
```

**Token spend per project, last 30 days** (the dimension behind `report --by project`;
join to your own price sheet for dollars, or use the CSV report):

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

**Cache efficiency per project** — cache reads as a share of input + cache reads, the same
ratio the `Cache%` column shows:

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
