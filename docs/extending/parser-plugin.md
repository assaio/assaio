# Reading a source assaio does not ship

*Part of [Extending assaio](../extending.md). To add a source in-tree, see [Add a data
source](data-source.md).*

If assaio does not read your logs, check two cases. A supported tool writing elsewhere needs a
config change, with no code. An unsupported tool needs a subprocess plugin, which can use any
language.

## Custom log-source paths

If a supported tool writes logs outside its default path—such as a custom install, an unsupported OS
path variant, a synced or mounted home directory, an external volume, or a CI runner with a
non-standard `HOME`—change config. `internal/paths.Resolve` (see
[`internal/paths/resolve.go`](../../internal/paths/resolve.go)) resolves every tool's roots. A
non-empty `sources.<tool>` list in `config.yaml` **replaces** that tool's default roots; it is never
merged with them. An empty or omitted list keeps the defaults.

```yaml
# ~/.config/assaio/config.yaml (honors XDG_CONFIG_HOME)
sources:
  claude:
    - /Volumes/work/.claude/projects   # e.g. Claude Code logging to an external volume
  codex: []                            # default: ~/.codex/sessions, ~/.codex/archived_sessions
  gemini: []                           # default: ~/.gemini
  cline: []                            # default: VS Code global storage, and ~/.cline/data
```

Each tool accepts a **list** of roots. Use multiple roots when a team's logs span locations, such as
a laptop's default path and an old profile directory:

```yaml
sources:
  claude:
    - ~/.claude/projects
    - /Volumes/archive/old-laptop/.claude/projects
```

Set one root per tool through `ASSAIO_SOURCES_<TOOL>`, or use the YAML list above for multiple
roots. For example: `ASSAIO_SOURCES_CLAUDE=/Volumes/work/.claude/projects`. Environment variables
override the config file, which overrides built-in defaults (see `internal/config`: defaults < file
< `ASSAIO_*` env < flags).

Run `assaio-agent doctor` to see each tool's resolved roots and whether they come from defaults or
config. It flags configured roots missing from disk, so a typo does not silently import nothing.

This surface changes *where* the existing parsers look; it does not change what they
parse. To make `assaio` understand a log format it doesn't already know, see [Add a data
source](data-source.md) (in-tree) or [Write a plugin](#write-a-plugin-any-language)
(out-of-tree).

---

## Write a plugin (any language)

**When to reach for this instead of a validator.** A [metric
validator](metric-validator.md) only *reads* usage that is already in the store —
it cannot manufacture tokens, lines, or sessions that were never ingested. Reach for a
plugin when the gap is upstream of that: an entirely new **tool** `assaio` has no parser
for yet (an internal AI tool, a vendor not covered by a built-in parser). A plugin's job
is narrow and specific — discover that tool's logs and emit normalized `usage.Record`
rows into the store — after which every existing surface (`report`, `effectiveness`,
`analyze`, `dashboard`, and any validator you've added) sees its data like any other
source. If the tool is one your organization alone uses, a plugin is almost always the
right call over an in-tree parser PR, since it needs no review from this project and no
release wait.

An exec plugin discovers and parses one tool's usage data, then writes normalized records to stdout.
During `backfill`, `assaio` runs it as a subprocess, validates each line, and stores valid records.
Its contract is the data format below: a handshake line and JSONL records, with no Go library or
library version to track. Go prevents external modules from importing core code under `internal/`.
Freezing a public Go API before v1.0 would bind a changing data model to semver (see [ADR
0003](../adr/0003-exec-plugin-protocol.md)). The **data format** contract lets you write a plugin in
Python, Rust, or shell without depending on core code that may be refactored.

Plugins are **opt-in only**: they run only when declared in `~/.config/assaio/config.yaml`. `assaio`
never scans `PATH`, discovers them automatically, or downloads them.

```yaml
plugins:
  - name: mytool            # required, [a-z0-9-]+; records are stored as tool "plugin:mytool"
    command: /path/to/assaio-parser-mytool   # required; resolved via PATH lookup if not absolute
    timeout: 60s            # optional, default 60s
```

## The protocol

`assaio` runs `<command> scan` with `ASSAIO_PLUGIN_PROTOCOL=1` in the environment. The plugin writes
to stdout:

1. **Handshake** (line 1): `{"assaio_plugin": 1, "tool": "<name>"}`. The protocol version must be
   `1`, and `tool` must match the configured `name`; otherwise the run fails.
2. **Records** (each later line): one snake_case JSON object per line:

```json
{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"some-model","input_tokens":100,"output_tokens":200,"cache_read_tokens":0,"cache_write_tokens":0,"reasoning_tokens":0,"dedupe_key":"s1:0","project":"myrepo","git_branch":"main","entrypoint":"cli","granularity":"turn"}
```

Required: `session_id`, `timestamp` (RFC3339), `model`, `dedupe_key`, and `granularity` (`turn` or
`session`; plugins follow the same [granularity honesty
rule](data-source.md#granularity-honesty-hard-rule) as in-tree parsers). Token fields default to 0.
`project`, `git_branch`, and `entrypoint` are optional. **A field the protocol does not define is
rejected**: writing `outputTokens` instead of `output_tokens` would otherwise store zero as valid
data. Emit only the fields above. The [`usage.Record`
contract](data-source.md#the-usagerecord-contract) also applies: `project` is a directory
**basename**, never a full path, and `dedupe_key` must be
[deterministic](data-source.md#dedupekey-determinism-hard-rule) to prevent double-counting on
reruns.

Anything the plugin writes to stderr reaches `assaio`'s stderr with the prefix `[plugin/<name>] `,
keeping diagnostics attributable.

## What the boundary enforces

`assaio` validates every record line and **skips** and counts lines that violate boundary
invariants, as in-tree parsers do with corrupt log lines:

| Rejected | Why |
|---|---|
| empty `session_id` or `dedupe_key` | `dedupe_key` is half the store's uniqueness constraint; a blank one collapses rows onto each other. |
| unparseable `timestamp` | a record that cannot be placed in time can appear in no window. |
| a field the protocol does not define | a misspelled field is a silent zero; naming it is the only way the plugin author finds out. |
| `timestamp` before 2020-01-01 or more than 48h in the future | since v0.14. Every query is `ts >= ?` with no ceiling, so a year-9999 record sits inside every `--since` window forever. Identical to what the sync endpoint enforces on the same shape — the two are one shared check (`internal/usage`). |
| invalid `granularity` | see the [granularity honesty rule](data-source.md#granularity-honesty-hard-rule). |
| a negative count, or one above 1,000,000,000 | a negative renders impossible percentages; an overflow-magnitude one distorts every `SUM()` it lands in. |
| `reasoning_tokens` above `output_tokens` | since v0.14. Reasoning is a *subset* of output, and a record claiming more renders a reasoning share above 100%. |
| a string field over 512 bytes | these are identities and labels, not free text. |

Stored records get the tool label `plugin:<name>`, so a plugin can never impersonate a
built-in source and its dedupe keyspace `(tool, dedupe_key)` never collides with anyone
else's. A plugin that exits non-zero, times out, or fails the handshake is reported as
failed for that run; the rest of the backfill continues. Stdout is capped at 64 MiB per run.

Unknown fields are currently ignored, unlike in the metric and rule protocols. A misspelled key
therefore stores zero instead of raising a violation. This inconsistency is tracked as `B143` and
will change only with a handshake version bump.

## A complete example (Python)

```python
#!/usr/bin/env python3
"""assaio-parser-mytool: emit usage records for the fictional mytool CLI."""
import json, sys
from pathlib import Path

print(json.dumps({"assaio_plugin": 1, "tool": "mytool"}))

for log in sorted(Path.home().glob(".mytool/sessions/*.jsonl")):
    for i, line in enumerate(log.read_text().splitlines()):
        entry = json.loads(line)
        print(json.dumps({
            "session_id": entry["session"],
            "timestamp": entry["ts"],            # RFC3339
            "model": entry["model"],
            "input_tokens": entry["in_tokens"],
            "output_tokens": entry["out_tokens"],
            "dedupe_key": f'{entry["session"]}:{i}',
            "granularity": "turn",
        }))
```

Make the plugin executable and add it to `config.yaml` as shown above. Then run `plugins verify` to
validate its full output **without storing anything**:

```console
$ assaio-agent plugins verify mytool
mytool: handshake OK
records ok: 42
skipped:    1
violations:
  line 17: empty dedupe_key
$ assaio-agent plugins list
mytool            /path/to/assaio-parser-mytool  (timeout 1m0s)
```

After `verify` passes, `assaio-agent backfill` ingests the plugin after built-in sources and reports
a `plugin:mytool` line beside them.

---
