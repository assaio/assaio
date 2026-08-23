# Privacy

`assaio-agent` is designed to be safe to run on a work machine. This document states
exactly what it reads, what it stores, and what it never touches. If anything here is
inaccurate, that is a bug — please report it.

## What it reads

The agent reads local session logs written by AI coding tools:

- **Claude Code** — `~/.claude/projects/**/*.jsonl`
- **OpenAI Codex CLI** — `~/.codex/sessions/**` and `~/.codex/archived_sessions/**`
- **Gemini CLI** — `~/.gemini/tmp/<hash>/chats/session-*.jsonl`
- **GitHub Copilot CLI** — `~/.copilot/session-state/*/events.jsonl` (honors `COPILOT_HOME`)
- **Cline** — the extension's global storage (`saoudrizwan.claude-dev`) under VS Code, VS Code
  Insiders, VSCodium and Cursor, and `~/.cline/data/tasks`

It reads these files; it never modifies or deletes them.

Two files that are **not** session logs are read as well, both for one number each and neither
stored:

- `/Library/Application Support/ClaudeCode/managed-settings.json` on macOS, or
  `/etc/claude-code/managed-settings.json` on Linux — the machine-wide managed policy.
- `~/.claude/settings.json` — your own Claude Code settings.

`assaio-agent doctor` decodes exactly one key from the first of those that sets it,
`cleanupPeriodDays`, to state how long Claude Code keeps its own transcripts — which is the
real ceiling on how far back any report can ever reach. Nothing else in either file is decoded,
and the value is printed, never stored. A project's own `.claude/settings.json` is deliberately
not read: `doctor` answers for the machine, and one repository's setting is not the machine's.

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

Rule plugins (`rules:` in `config.yaml`) are opt-in the same way and receive strictly
less: only the validator verdicts — titles, verdict labels, figures, caveats, and
takeaways — never usage rows, sessions, or prices. The ranked bar lists are **stripped
before the plugin is called**, because those are where project, skill, and sub-agent names
appear and a rule gates on a verdict, not on which repository produced it.
They run in `assaio-agent check` and can only emit alerts back.

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
- tool-call purpose counts — the same turn's tool calls split into how many read, searched,
  ran a command, wrote, or did something else. The tool's **name** is matched against a
  fixed allowlist while parsing and then dropped: only the five counts are stored, never the
  name, its arguments, or its output.
- tool error counts — how many of a turn's tool calls came back an error
- a sub-agent flag — whether the turn ran inside a sub-agent (`1` or `0`), read from the
  log's own marker

Two fields on this list are **short text labels rather than counts**, and they are the only
ones:

- skill — the skill the tool itself attributed the turn to (e.g. `code-review`)
- sub-agent type — the kind of sub-agent the turn ran as (e.g. `general-purpose`)

Both are category labels the tool assigns, not anything you typed, and they exist so a
report can say where the spend went. They are names people choose, though, so a shared
report treats them exactly like project names: `--anonymize` (the default for the team
server and the published dashboard) replaces them with stable pseudonyms.

That is the complete list of what a usage record holds; the step timeline below adds its own,
and nothing else is stored. Apart from those two labels, the fields are **numeric counts
only** — how much AI produced, how efficiently, and with how much friction. No field
carries prompt text, model output, or code content: a diff line contributes a `+1` or a
`-1` and nothing else.

### The step timeline

Beside those per-turn records, `assaio` stores a session's **sequence**: one row per step,
holding only what kind of step it was (from a closed list: assistant turn, read, search,
command, edit, other, compaction), its position in the sequence, the model, its token cost,
and how it ended (from a closed list: ok, error, denied, truncated, or nothing when the log did
not say).

One field there deserves its own paragraph, because it is the only one that stands for a file:

- **target** — an **integer** assigned in first-seen order within a single sequence, and
  nothing else. Not comparable across sequences: `3` in a session's main transcript and `3` in
  one of its sub-agents are unrelated. It exists so a sequence can show that the same thing was touched nine times.
  It is deliberately **not** a hash of the path: a hash is reversible by anyone holding the
  repository, because file paths carry almost no entropy. The path is read in memory while
  parsing one transcript, used to decide which integer to reuse, and discarded — the same
  discipline the rework counts already follow. `3` is not recoverable to a file name by
  anyone, including you.

  Since v0.21.0 the integer is read from the **call's own arguments** rather than from its
  result, which widens *which* steps carry one -- a read, and an edit that failed, now do -- and
  changes nothing about what is stored: still an integer, still per sequence, still no path. A
  path named relatively is resolved against the session's working directory before the integer is
  chosen, so one file cannot hold two of them; a relative path with no working directory to
  resolve it against is left unnumbered rather than guessed at.

How much of this history is kept is capped by `trace.horizon_days` (30 by default; `0` keeps
everything). `assaio-agent clear` erases the timeline under exactly the same scope as the
records it erases.

## What you add yourself

One thing in the store is not read from a log at all: the labels you attach with
`assaio-agent mark`. Session logs record what a tool did, never what the work was *for*, and
that intent cannot be recovered by reading prompts — which are never collected anyway.

A label is three **closed vocabularies**, and nothing else:

- task class — `bugfix`, `feature`, `test`, `refactor`, `docs`, `research`, `review`, `other`
- outcome — `done`, `partial`, `abandoned`
- difficulty — `low`, `medium`, `high`

There is **no free-text field**. A value outside its vocabulary is rejected, so the table is
free of content by construction: there is nowhere to put a branch name, a ticket title, a
commit message, or a prompt. Deliberately absent, too, is any issue or branch reference —
see [ADR 0006](docs/adr/0006-session-annotations.md) for why that belongs to attribution
work rather than here.

Labels are **local**. `sync` sends usage records; it does not send labels, and no other path
does either. They are also the only data in the store that no re-import can rebuild, so
`clear --all`, `clear --older-than` and `clear --tool` leave them alone and report how many
they kept — only `clear --labels` deletes them.

Labeling is optional and stays optional: an unlabeled session is counted in full by every
metric, a report grouped by a label always shows the `unlabeled` group rather than hiding
it, and nothing in `assaio` scores how much you label.

## A vendor export you point it at

`assaio-agent reconcile <file>` reads one more thing, and only when you name it on the
command line: a billing or usage export you downloaded yourself. It is read, never written
and never stored — the comparison happens in memory and only its result is printed. No
credential is asked for and no network call is made to fetch it; getting the file out of a
vendor console is your step, deliberately, so the tool never holds an API key that could
pull your account data.

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

Beside the usage records, the same file holds three small bookkeeping tables. None holds
usage, and each exists only so a repeat command stays cheap and honest:

- `ingest_file` — one row per input already parsed: its **path on your disk**, size,
  modification time, and which build read it. This is the only place `assaio` stores a
  full local path, and it is never synced, exported, or shown in a dashboard. Rows for
  files that are no longer on disk are dropped on the next pass.
- `ingest_source` — one row per source per `backfill` run, holding only counts (files
  found, files read, records, skipped lines, records carrying no tokens) and a timestamp.
  It is what the format-drift canaries compare against, and only the newest runs per tool
  are kept.
- `digest_snapshot` — one row per `digest` run, holding the verdicts and totals that run
  reported plus the build that parsed them, so the next digest can say what *moved*. A few KB
  each, never usage rows, and all but the newest few are deleted whenever one is written.

All three can be deleted at any time; the only cost is one slow re-parse, a reset drift
baseline, and a digest with nothing to compare against.

A fourth table, `session_label`, holds the labels described under "What you add yourself" —
one row per session you marked, carrying only the three vocabulary values and a timestamp.
Unlike the three above it is **not** a cache: nothing can rebuild it, so it is the one table
`clear` never removes without being asked. Its size is bounded by how many sessions you
marked by hand (~80 bytes each), not by how much you have ingested.

## How to delete it

```sh
assaio-agent clear --all --yes
assaio-agent compact          # actually return the freed space to the filesystem
```

`clear` refuses to run without an explicit scope (`--all`, `--older-than`, `--tool`, or
`--labels`) and the `--yes` confirmation flag. The first three delete usage records and keep
your session labels, reporting how many they kept; `--labels` is the deliberate way to
delete those too, and deleting the database file removes everything at once. Note that deleting rows frees pages *inside* the
database file without shrinking it — `compact` is what returns that space to your
filesystem. You can also simply delete the database file. Because all data is local and
self-contained, this makes `assaio` straightforward to operate under GDPR-style deletion
requirements.

## Network

The core analysis commands — `backfill`, `report`, `effectiveness`, `analyze`, `status`,
`dashboard`, `share`, `reconcile` — make **no network calls**. The model price table is embedded into the binary
at build time, so every report works fully offline; nothing is fetched, uploaded, or
phoned home.

Three **optional commands are the exception**, and only when you invoke them.
`assaio-agent sync` uploads your usage records to a team server, and `assaio-agent serve` runs
that server; both talk only to infrastructure **you** stand up and point them at (see below).
`assaio-agent runtime inspect --vllm-url/--dcgm-url` reads a metrics endpoint **you name** —
a plain GET with no header, body or credential, bounded by `--timeout`, `--max-bytes` and
`--max-redirects`, storing nothing; `--vllm-file`/`--dcgm-file` read a saved snapshot and need
no network at all. If you never run those three, `assaio` never touches the network.

`assaio-agent share` is not an exception to this — it makes no request, and neither does the
page it writes. It is the one command that **starts another program**: after writing the file
it asks your desktop to open it (`open`, `xdg-open`, `rundll32`), which launches your browser.
That is the only program `assaio` ever launches, and `--no-open` writes the file and stops.

## Sharing a dashboard, and sharing an assay

`assaio` writes two shareable artifacts, and their guarantees differ. Read this section before
generalizing from one to the other.

`assaio-agent dashboard` writes a single self-contained HTML file — all styling is inline,
with no external fonts, scripts, or requests, so it renders offline and phones nowhere.
Because that file is meant to be shared, it **pseudonymizes project names by default**
(a stable `project-xxxx` label); pass `--no-anonymize` to keep real names. The interactive
CLI tables always show real *project* names — the setting governs only the shareable export.
Member names are stricter and not governed by it: `report` renders a pseudonym in every format,
and `--identify` is the only door to raw ones (see below).

`assaio-agent share` is the artifact built *for* publication, and its redaction is
**structural rather than a flag** — there is no `--no-anonymize` equivalent, because no field
it renders can hold a repository, member, path, branch, skill or sub-agent name. Repositories
appear only as a count, labelled *never named*; an out-of-tree parser's tool label collapses to
`a plugin source`, since that name comes from your own config rather than from a vendor. Tools
and models **are** named, deliberately: which coding agent and which model ran is a fact about a
vendor, not about you. The preview page is self-contained — inline CSS, one inline script, no
external font, image or request — and the PNG, MP4 or WebM it offers are produced in your own
browser from a canvas and saved by your own download. Nothing is transmitted at any point.

## Telemetry

None. No usage pings, no analytics, no crash reporting. The agent does not know we
exist and does not tell us it ran.

One distinction worth stating, because a card makes it look otherwise: a card you choose to
post carries the project's hashtag and its install line **on the image**, deliberately, so a
re-shared image keeps them. That is a caption `assaio` wrote, not a signal it sent, and it
travels only if you post it.

## The optional team server

v0.1 ships an early, self-hostable **team server** so a team can pool its usage in one
place: `assaio-agent serve` runs a central collector and `assaio-agent sync` pushes each
member's records to it, with a per-member team dashboard. It is the **largest of the networked exceptions**
named above — the offline guarantee is about the local analysis, not this. You run
the server on **your own infrastructure** and control what reaches it. It is an honest MVP:
no TLS of its own (put a reverse proxy in front), meant for a trusted network — not yet
production-hardened. Every route requires a bearer token, the dashboard included, and
configuring one secret per member makes the server decide who a request is rather than
believing the name in its body.

Each synced member is **pseudonymized by default** (a stable `member-xxxx` label); team
views are aggregated by default. A per-member, real-name view is never silent — it is a
deliberate, governed opt-in an admin enables, not what the default configuration produces,
and never a performance-evaluation leaderboard.

The pseudonym holds on the way **out** as well as on the way in. Reading a central store with
`report` labels members in every format — table, JSON and CSV alike — and a metric plugin, which
is an out-of-tree subprocess, receives labels rather than names. Raw identity has exactly one
door: `report --identify`, which names individuals and says so in its own output. Nothing else
in this repository turns a synced name back into a printed one. The **deeper** stage — correlating with
your git remotes and issue tracker for survival / bug / quality signals — is still ahead;
see [ROADMAP.md](ROADMAP.md).
