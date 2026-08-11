# Query your own data

*Part of [Extending assaio](../extending.md). Every column below is also described in the [generated reference](https://assaio.dev/docs/reference).*

Everything `assaio` collects lives in one SQLite file:

```
~/.local/share/assaio/assaio.db
```

The location honors `XDG_DATA_HOME`. It is an ordinary SQLite database — point `sqlite3`,
DB Browser, or any client at it and query directly. `assaio` never phones home, so this
file is the whole of your data.

## Schema

One table holds your data, `usage_record`
([`internal/store/migrations/0001_init.sql`](../../internal/store/migrations/0001_init.sql)):

| Column | Type | Notes |
|--------|------|-------|
| `id` | `INTEGER PRIMARY KEY` | Row id. |
| `tool` | `TEXT` | Source, e.g. `claude-code`, `codex`, `gemini-cli`, `copilot-cli`, `cline`. |
| `session_id` | `TEXT` | The tool's session/conversation ID. |
| `ts` | `TEXT` | UTC RFC3339 timestamp. Day is `substr(ts,1,10)`. |
| `model` | `TEXT` | Model name as recorded by the tool. |
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

The activity columns (`lines_added` … `rework_lines`) are populated by the Claude Code
and Codex parsers today, except `rejected`, which is Claude-Code-only; Gemini and Cline
store `0` throughout. They hold **counts only** — never the code content of the lines
they count. `report --format csv` covers tokens and cost; `effectiveness --format csv`
adds the activity and `$`/100-lines columns.

**Cost is not stored.** The database holds tokens only; dollar cost is computed at report
time against the embedded price table, because prices change and unpriced models must stay
honestly blank. For cost figures, use `assaio-agent report --format csv` (which carries a
`cost` column) rather than SQL.

**Two bookkeeping tables sit beside it**, neither holding usage: `ingest_file` (one row per
input already parsed — path, size, mtime, parsing build) makes a repeat `backfill` nearly
free, and `ingest_source` (one row per source per run — files found, files read, records,
skipped lines, zero-token records) is the baseline the [format-drift
canaries](../format-resilience.md) compare against. Both are caches: dropping them costs one
slow re-parse and a reset drift baseline, nothing more. `ingest_file` is pruned to what is
actually on disk after each pass, and `ingest_source` keeps only the newest runs per tool,
so neither grows with how long assaio has been installed. Use `assaio-agent compact` to
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
