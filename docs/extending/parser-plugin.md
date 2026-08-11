# Reading a source assaio does not ship

*Part of [Extending assaio](../extending.md). To add a source in-tree instead: [Add a data source](data-source.md).*

Two escalating answers to "assaio does not read my logs". The first is a config change and no
code: the tool is supported but writes somewhere else. The second is a subprocess in any
language: the tool is not supported at all.

## Custom log-source paths

For a team whose logs don't live at the built-in default path — a custom install
location, an OS-variant path the defaults don't cover, a synced or mounted home
directory, an external volume, or a CI runner with a non-standard `HOME` — the fix is a
config change, not code. `internal/paths.Resolve` (see
[`internal/paths/resolve.go`](../../internal/paths/resolve.go)) backs every tool's root
resolution: a non-empty `sources.<tool>` list in `config.yaml` **replaces** the built-in
default roots entirely for that tool (never merged with them), so the result is always
exactly what you configured; an empty or omitted list keeps the default.

```yaml
# ~/.config/assaio/config.yaml (honors XDG_CONFIG_HOME)
sources:
  claude:
    - /Volumes/work/.claude/projects   # e.g. Claude Code logging to an external volume
  codex: []                            # default: ~/.codex/sessions, ~/.codex/archived_sessions
  gemini: []                           # default: ~/.gemini
  cline: []                            # default: VS Code global storage, and ~/.cline/data
```

Each tool accepts a **list** of roots — set more than one when a team has usage spread
across two locations (e.g. a laptop's default path plus an old profile directory that
hasn't been cleaned up yet):

```yaml
sources:
  claude:
    - ~/.claude/projects
    - /Volumes/archive/old-laptop/.claude/projects
```

Override per-tool from the environment instead of a file with `ASSAIO_SOURCES_<TOOL>`
(one root per variable — use the YAML list form above for more than one root), e.g.
`ASSAIO_SOURCES_CLAUDE=/Volumes/work/.claude/projects`. Environment variables win over
the config file, which wins over the built-in default (see `internal/config`'s
precedence: defaults < file < `ASSAIO_*` env < flags).

Verify what's actually in effect with `assaio-agent doctor`: it reports every tool's
resolved roots, whether each is the built-in default or config-overridden, and flags a
configured root that doesn't exist on disk — so a typo'd path fails loudly instead of
silently importing nothing.

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

An exec plugin is an executable that discovers and parses one tool's usage data itself
and emits normalized records to stdout. `assaio` runs it as a subprocess during
`backfill`, validates every line, and stores what passes. The contract is the data
format below — there is no Go to link against and no library version to track. The core
lives under Go's `internal/`, which the compiler forbids any external module from
importing, because freezing a public Go API before v1.0 would bind us to its shape under
semver while the data model is still moving (see [ADR 0003](../adr/0003-exec-plugin-protocol.md)).
An exec plugin's contract is the **data format** instead — a handshake line and JSONL
records over stdout — so you can write one in Python, Rust, or a shell script, and
nothing you depend on breaks when the core refactors.

Plugins are **opt-in only**: they run exclusively when declared in
`~/.config/assaio/config.yaml`. `assaio` never scans `PATH`, never auto-discovers, and
never downloads plugins.

```yaml
plugins:
  - name: mytool            # required, [a-z0-9-]+; records are stored as tool "plugin:mytool"
    command: /path/to/assaio-parser-mytool   # required; resolved via PATH lookup if not absolute
    timeout: 60s            # optional, default 60s
```

## The protocol

`assaio` invokes `<command> scan` with `ASSAIO_PLUGIN_PROTOCOL=1` in the environment.
The plugin writes to stdout:

1. **Handshake** (line 1): `{"assaio_plugin": 1, "tool": "<name>"}`. The protocol
   version must be `1` and `tool` must equal the configured `name`; any mismatch fails
   the run.
2. **Records** (every following line): one JSON object per line, snake_case:

```json
{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"some-model","input_tokens":100,"output_tokens":200,"cache_read_tokens":0,"cache_write_tokens":0,"reasoning_tokens":0,"dedupe_key":"s1:0","project":"myrepo","git_branch":"main","entrypoint":"cli","granularity":"turn"}
```

Required: `session_id`, `timestamp` (RFC3339), `model`, `dedupe_key`, and `granularity`
(`turn` or `session` — the [granularity honesty rule](data-source.md#granularity-honesty-hard-rule)
applies to plugins exactly as it does to in-tree parsers). Token fields default to 0;
`project`, `git_branch`, and `entrypoint` are optional. **A field the protocol does not
define is rejected**, as it already was for the metric and rule protocols: a plugin writing
`outputTokens` where the protocol says `output_tokens` would otherwise store a zero and be
counted as a valid record, which is a wrong number arriving quietly instead of a protocol
error arriving loudly. Emit exactly the fields above. The same
[`usage.Record` contract](data-source.md#the-usagerecord-contract) rules apply: `project` is a
directory **basename**, never a full path, and `dedupe_key` must be
[deterministic](data-source.md#dedupekey-determinism-hard-rule) so re-runs never double-count.

Anything the plugin writes to stderr passes through to `assaio`'s stderr prefixed with
`[plugin/<name>] `, so diagnostics stay attributable.

## What the boundary enforces

`assaio` validates every record line and **skips** (and counts) any line that breaks one of
the boundary invariants — the same skip-and-count policy in-tree parsers apply to corrupt
log lines:

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

Unknown fields are currently ignored rather than rejected, unlike the metric and rule
protocols — so a misspelled key stores a zero instead of raising a violation. That
inconsistency is tracked as `B143` and will change behind a handshake version bump, not
silently.

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

Make it executable, add it to `config.yaml` as shown above, and check conformance —
`plugins verify` runs the plugin and validates the full stream **without storing
anything**:

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

Once `verify` is clean, `assaio-agent backfill` ingests the plugin after the built-in
sources and reports a `plugin:mytool` line alongside them.

---
