# Privacy

`assaio-agent` is designed to be safe to run on a work machine. This document states
exactly what it reads, what it stores, and what it never touches. If anything here is
inaccurate, that is a bug — please report it.

## What it reads

The agent reads local session logs written by AI coding tools:

- **Claude Code** — `~/.claude/projects/**/*.jsonl`
- **OpenAI Codex CLI** — `~/.codex/sessions/**` and `~/.codex/archived_sessions/**`
- **Gemini CLI** — `~/.gemini/tmp/<hash>/chats/session-*.jsonl`
- **Cline** — the VS Code extension's global storage (`saoudrizwan.claude-dev`) and
  `~/.cline/data/tasks`

It reads these files; it never modifies or deletes them.

If you configure exec plugins (`plugins:` in `config.yaml`), each one is a program you
chose that runs as a subprocess with your user privileges and reads whatever its own
code reads. Plugins are explicit opt-in from your config file only — `assaio` never
downloads plugins, never scans `PATH` for them, and never auto-discovers them. A
plugin's output is validated at the boundary before storage, is stored under a
`plugin:<name>` label, and is limited to the same usage-accounting fields listed below.

Metric plugins (`metrics:` in `config.yaml`) follow the same opt-in rules with one
difference in direction: `assaio` **sends** each one your stored usage aggregates on
stdin — project names, model names, member pseudonyms, and token/line counts, exactly
the fields listed below, never prompts or code (which are never collected at all) — so
it can compute its metric. That data goes only to the local program you configured;
know what a metric plugin does with it before declaring one.

## What it extracts

From each session log, the parsers extract only usage accounting fields:

- token counts (input, output, cache-read, cache-write, reasoning)
- model name
- timestamp
- session ID
- project — the basename of the session's **git repository root** (e.g. `webapp`),
  never the full path. At ingest, `assaio` walks up from the session's working
  directory to the nearest `.git` and keeps only that root directory's last path
  segment, so a monorepo's subdirectories (`apps/mobile`, `apps/web`, a worktree
  checkout, …) roll up into one project instead of fragmenting by leaf directory name.
  The full working-directory path is read only transiently to do that walk — it is
  never written to the store.
- subpath — the working directory's path **relative to the project's repository root**
  (e.g. `apps/mobile`), or empty when the session ran at the root. Always relative:
  never an absolute path, never the home directory.
- git branch name, when the log records it
- entrypoint label — how the tool was invoked (e.g. `cli`)
- granularity — whether a record is a single turn or a session-level aggregate
- AI line counts — lines added and removed, **derived only from the `+`/`-` markers of
  diff hunks.** The prefix is counted; the code on the line is never stored.
- edit and tool-call counts — how many file-editing tool calls (Edit/Write/…) and how
  many tool calls in total a turn made
- rejection counts — how many tool proposals the human declined
- sub-agent usage — a completed sub-agent's own token counts and its added/removed line
  counts, recorded as its own usage record
- compaction counts — how many times a session's context overflowed and got
  auto-summarized: a context-strain signal, not a content record
- rework line counts — a proxy for "AI wrote code that didn't stick," computed (for both
  Claude Code and Codex now) by matching, within one transcript, a later edit's removed
  lines against lines the AI itself added earlier **to that same file**. The file path is
  read only transiently, in memory, to group edits by file while parsing that one
  transcript, and is discarded the moment parsing finishes — **never stored.** Only the
  resulting numeric count is kept, same as every other field on this list.

That is the complete list. These fields are **numeric counts only** — how much AI
produced, how efficiently, and with how much friction. No field carries prompt text,
model output, or code content: a diff line contributes a `+1` or a `-1` and nothing
else. Everything `assaio` needs for its reports is a count, so a count is all it keeps.

## What it never reads

- Prompt text
- Model responses / generated content
- The **content** of any file, diff, or code from your project — a diff hunk is scanned
  only to count its `+`/`-` line prefixes; the text after the prefix is never decoded or
  stored
- Anything in a session log beyond the usage-accounting fields above

The parsers walk each log line and pull out token counts, identifiers, and activity
counts. Message bodies and code are never decoded or stored.

## Where your data lives

Normalized usage is stored in a single embedded SQLite database:

```
~/.local/share/assaio/assaio.db
```

The location honors `XDG_DATA_HOME`. This file never leaves your machine.

## How to delete it

```sh
assaio-agent clear --all --yes
```

`clear` refuses to run without an explicit scope (`--all`, `--older-than`, or `--tool`)
and the `--yes` confirmation flag. You can also simply delete the database file. Because
all data is local and self-contained, this makes `assaio` straightforward to operate
under GDPR-style deletion requirements.

## Network

The core analysis commands — `backfill`, `report`, `effectiveness`, `analyze`, `status`,
`dashboard` — make **no network calls**. The model price table is embedded into the binary
at build time, so every report works fully offline; nothing is fetched, uploaded, or
phoned home.

The two **optional team commands are the exception**, and only run when you invoke them:
`assaio-agent sync` uploads your usage records to a team server, and `assaio-agent serve`
runs that server. Both talk only to infrastructure **you** stand up and point them at (see
below). If you never run them, `assaio` never touches the network.

## Sharing a dashboard

`assaio-agent dashboard` writes a single self-contained HTML file — all styling is inline,
with no external fonts, scripts, or requests, so it renders offline and phones nowhere.
Because that file is meant to be shared, it **pseudonymizes project names by default**
(a stable `project-xxxx` label); pass `--no-anonymize` to keep real names. The interactive
CLI tables always show real names — the setting governs only the shareable export.

## Telemetry

None. No usage pings, no analytics, no crash reporting. The agent does not know we
exist and does not tell us it ran.

## The optional team server

v0.1 ships an early, self-hostable **team server** so a team can pool its usage in one
place: `assaio-agent serve` runs a central collector and `assaio-agent sync` pushes each
member's records to it, with a per-member team dashboard. It is the **networked exception**
to everything above — the offline guarantee is about the local analysis, not this. You run
the server on **your own infrastructure** and control what reaches it. It is an honest MVP:
a shared bearer token, no TLS of its own (put a reverse proxy in front), meant for a
trusted network — not yet production-hardened.

Each synced member is **pseudonymized by default** (a stable `member-xxxx` label); team
views are aggregated by default. A per-member, real-name view is never silent — it is a
deliberate, governed opt-in an admin enables, not what the default configuration produces,
and never a performance-evaluation leaderboard. The **deeper** stage — correlating with
your git remotes and issue tracker for survival / bug / quality signals — is still ahead;
see [ROADMAP.md](ROADMAP.md).
