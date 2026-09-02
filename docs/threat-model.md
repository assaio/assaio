# Threat model

What assaio trusts, what it checks at each boundary, and what an attacker who controls a
given surface can and cannot reach. It is the security counterpart to
[`PRIVACY.md`](../PRIVACY.md), which states what is read and stored, and to
[`architecture.md`](architecture.md), which is the path the data takes.
Vulnerability reporting is [`SECURITY.md`](../SECURITY.md); this document does not restate
it.

Scope is the `assaio-agent` binary and the team server in this repository. Out of scope:
the AI coding tools whose logs assaio reads, the operating system, and the reverse proxy an
operator puts in front of `serve`.

Every claim below names the code that enforces it. Where a boundary is weaker than it
looks, that is stated rather than omitted — an unenforced promise is worse than an absent
one.

## 1. Trust surfaces

### Local session logs — read

**Trusted:** their location, and nothing else. `internal/paths` resolves the built-in roots;
`sources.<tool>` in config replaces them entirely (`paths.Resolve`).

**Never trusted:** their content. assaio opens each file read-only and never writes,
renames, or deletes one.

**Checked at the boundary** (`internal/parser`, shared by every parser):

| Check | Enforced by |
|---|---|
| A single line is bounded at 16 MiB; a longer one aborts that file's scan | `parser.MaxLineBytes`, `parser.NewScanner` |
| A line that will not decode is skipped and counted, never fatal | each parser's scan loop; `ingest.ingestParsed` |
| Negative and overflow-magnitude counts are clamped | `parser.NonNeg`, `parser.SumNonNeg` |
| A portion larger than its whole is clamped to the whole | `parser.Subset` |
| A vendor "closed vocabulary" field that is not vocabulary-shaped is dropped | `parser.VocabularyToken` |
| A tool name is matched against a fixed allowlist and then discarded | `parser.ToolCounts`, `parser.StepKind` |
| A record with no timestamp is dropped and counted as skipped | `ingest.dated` |
| No panic, no negative count, no empty dedupe key, on any input | each parser's `FuzzParse` (`make fuzz`) |

**An attacker who controls a log file can:** distort your own figures. The in-tree parsers
clamp shape, not magnitude — `usage.CheckCounts`'s `MaxCount` guards the two *external*
boundaries (below), not a local file — so a crafted transcript can store an implausible
token count. They can also consume disk, bounded by nothing but the store's own growth,
which `doctor` reports and `compact` reclaims.

They can also set two stored text fields directly: `skill` and `agent`. These are read
verbatim from the vendor's own attribution fields, with no vocabulary or length clamp on the
local path — the clamp exists only where a record arrives from outside the process
(`server.maxStringField`, 512 bytes). Both are pseudonymized on every shareable surface, and
`share` cannot render either at all. This asymmetry is deliberate on the reading side and
worth knowing on the writing side.

**They cannot reach:** code execution — the parsers decode JSON and count, and nothing in a
log is ever evaluated, expanded, or executed. Nor can they exfiltrate content, because no
content is decoded: prompt text, model output, and the code on a diff line are never read
past the `+`/`-` prefix. A path is read into memory to group edits and to resolve a project
root, and discarded (`usage.Record.Cwd` is `json:"-"` and is not a column).

### The store — write

**Trusted:** it is a local file, owned by you, at `paths.DBPath()` (`XDG_DATA_HOME`-aware).
Anyone who can write it already has code execution as your user, so it is not a security
boundary — it is a durability and blast-radius one.

**Checked:** SQL in `internal/store` is never assembled from anything but compile-time
constants, and every value is bound as a parameter. Migrations are append-only by repository
rule ([`RELEASING.md`](../RELEASING.md)). `store.Open` applies them on open.

**The blast radius is deliberately shaped:**

- `assaio-agent clear` has no `--db` flag. It opens `paths.DBPath()` and nothing else, and
  it prints that path plus the record count *before* deleting. There is no ambiguity about
  which file a destructive command is aimed at, and there is no way to aim it at a copy —
  which is why the deletion test in part 3 redirects `XDG_DATA_HOME` instead.
- `clear` refuses to run without both an explicit scope (`--all`, `--older-than`, `--tool`,
  `--labels`) and `--yes`.
- Session labels survive every scope but `--labels`, and the command says how many it kept.
  They are the one thing in the store no re-import can rebuild.
- `ingest_file` is the only table holding a full local path. It is never synced, exported,
  or rendered.

**Availability:** the store grows without an upper bound except for the step timeline, which
`trace.horizon_days` prunes (`ingest.pruneTrace`, counted and reported). `doctor` states
size, growth rate and reclaimable space every run.

### The team server — network

This is the largest exception to assaio's offline posture, and it is opt-in twice: you run
the server, and you point `sync` at it.

**Trusted:** the network it sits on. `internal/server` ships **no TLS of its own** — put a
reverse proxy in front of it, on a network you trust. `sync` warns when `--server` is
plaintext `http://` to a non-localhost host, because the token and the usage data are then
in cleartext.

**Checked at the boundary:**

| Check | Enforced by |
|---|---|
| Every route but `/healthz` requires a bearer token, the dashboard included | `Server.Handler`, `authorizedReader` |
| The secret is verified **before a byte of the body is read** | `handleUsage` |
| Constant-time comparison; an empty configured secret never matches | `constantTimeEqual` |
| A secret is at least 16 characters; two members may not share one | `Members.Validate`, `MinTokenBytes` |
| Request body capped at 128 MiB | `maxUsageBodyBytes`, `http.MaxBytesReader` |
| Per-secret fixed-window rate limit (120/min default) | `rateLimiter.allow` |
| Header, read, write and idle timeouts on every connection | `readHeaderTimeout`, `readTimeout`, `writeTimeout`, `idleTimeout` |
| Every record range- and magnitude-checked, same bounds as the plugin boundary | `validateRecord` → `usage.CheckTimestamp`, `usage.CheckCounts` |
| Tool must be a known source or a well-formed `plugin:<name>` | `knownTools`, `pluginToolPattern` |
| A push failing validation is rejected **whole**, never partially inserted | `handleUsage` |
| Every dedupe key prefixed `<member>:`, so a row has one possible writer | `handleUsage` → `InsertSynced` |
| Method and path quoted before logging, so a crafted path cannot forge log lines | `logRequests` |
| Error responses describe the client's own data, never internal detail | `handleUsage` |

**An attacker holding a valid token can:** read the whole team dashboard. There are no
roles — any configured secret grants a read, which the code says out loud rather than
inventing a role model the product does not have. In **shared-token** mode
(`Identity.ClientAsserted`, one secret for everyone) they can also push as *any* member,
because the member name comes from the request body. Configuring per-member tokens
(`server.members`) switches to `ServerDerived`, where the member is whoever holds the secret
and cannot be asserted; `doctor` names which mode a deployment is in.

**They cannot reach:** another member's rows in `ServerDerived` mode (the dedupe prefix gives
each row exactly one writer); prompts, code, or file paths, because none of those ever leave
a machine; or a metric plugin — the server never executes one ([ADR 0004](adr/0004-exec-metric-plugin-protocol.md)).

**An attacker on the wire, without TLS, can:** read the token and the usage data, and replay
a push. This is the MVP boundary, stated in `internal/server`'s package doc, in
[`PRIVACY.md`](../PRIVACY.md), and by `sync` itself.

### Exec plugins — running someone else's code

This is the one surface where assaio executes code it did not write. **The process is not
the boundary; the config file is.**

**Trusted:** completely, once declared. A plugin runs as a subprocess with your user's
privileges and reads whatever your user can read. There is no sandbox, and this document
does not pretend there is one.

**Checked before it runs** (`plugin.Resolve` / `plugin.ResolveMetric`, `config.PluginConfig`):
opt-in from `config.yaml` **only** — never discovered on `PATH`, never downloaded, never
auto-registered; name must match `[a-z0-9-]+`; a relative command is resolved through
`exec.LookPath`; every capability name — the plugin's declaration and your `needs:` veto alike
— must be in assaio's closed capability set, and a declaration naming an unknown section,
column or predicate is refused whole before any window is serialized; timeout parses
(60s default).

**Checked while it runs:** the run is bounded by `Config.Timeout` with `cmd.WaitDelay =
killGrace` (2s), so a grandchild that survives the kill cannot stall assaio; stdout is capped
— 64 MiB for a parser plugin's JSONL stream (`maxStdout`), 1 MiB for the single document a
metric or rule plugin replies with (`maxMetricStdout`, `maxRuleStdout`) — and the cap breach
is checked *before* `ctx.Err()`, so a flood is not misreported as a timeout; stderr is
prefixed and passed through rather than swallowed.

**Checked on everything it emits:** a handshake line naming the expected protocol version and
the configured plugin name; then, for a parser plugin, per record line: strict JSON
(`DisallowUnknownFields` — an `outputTokens` where the protocol says `output_tokens` is a
named violation, not a silent zero), every string bounded at 512 bytes, `granularity` from a
closed set, and `usage.CheckTimestamp` + `usage.CheckCounts`. A bad record line is skipped and
counted, never fatal; a bad *handshake*, a timeout, or a non-zero exit drops the whole run. The
metric and rule protocols are stricter still — one document, and any protocol failure discards
it whole rather than acting on something partially sanitized. Everything stored is namespaced
`plugin:<name>` (`parser.PluginPrefix`), so a plugin can never impersonate a built-in source,
and it lands through `Store.Insert` (first-write-wins) rather than the restating path.

**What each protocol sends outward** matters as much as what it accepts back:

- **Parser** ([ADR 0003](adr/0003-exec-plugin-protocol.md)) — receives nothing. One
  environment variable, `ASSAIO_PLUGIN_PROTOCOL=1`, and the argument `scan`.
- **Metric** ([ADR 0004](adr/0004-exec-metric-plugin-protocol.md)) — receives your stored
  aggregates on stdin: project names, model names, member pseudonyms, token and line counts.
  Never prompts or code, which are never collected at all. Since protocol 4 it receives only
  what it declared in its own `describe` run — the sections, the columns inside them, and the
  rows its predicates admit — so the default disclosure is narrower than the plugin's
  self-declared reading list, not wider. `needs:` in `config.yaml` is your **veto** over that
  list: anything you exclude is left out of the envelope and named to the plugin in
  `withheld`, rather than sent as an empty array it could mistake for a window with none.
- **Rule** ([ADR 0005](adr/0005-exec-rule-plugin-protocol.md)) — receives strictly less:
  the window's verdicts with their ranked `Bars` **stripped before the call**
  (`buildRuleInput`), because that is where project, skill and sub-agent names appear and a
  rule gates on a verdict, not on which repository produced it.

**A hostile plugin can reach:** everything your user can. Declaring one is the security
decision; nothing after it is. Know what a metric plugin does with the aggregates before you
declare it.

**It cannot:** write a record outside the validated shape, claim to be a built-in tool,
overwrite a stored row (first-write-wins), or alter an in-tree verdict — a rule plugin emits
alerts and cannot clear one.

### The `share` artifact — publication

`assaio-agent share` is the only surface built for strangers, and it is the one where a
mistake cannot be recalled: the image outlives the store it came from, and a re-shared image
arrives with the caption stripped off ([ADR 0014](adr/0014-public-artifact-rules.md)).

**Trusted:** nothing. Redaction here is **structural, not a flag** — there is no
`--no-anonymize` equivalent because no field `share.Assay` carries can hold a repository,
member, path, branch, skill or sub-agent name. There is no code path that renders one.

**Checked:** `internal/share`'s own `TestNoUserChosenNameReachesAnySurface` plants a secret
into every user-chosen string on every row — project, member, entrypoint, and the
`plugin:<name>` tool label — renders every output the command can produce, and fails if the
secret appears in the HTML, the text, the post, or the marshalled JSON payload. It walks the
payload as well as the rendered text, because the preview page ships the whole `Assay` as
JSON and a field nothing draws today is still a field that left the machine.

Repositories appear only as a count, labelled *never named*. An out-of-tree parser's tool
label collapses to `a plugin source`, since that name comes from your own config rather than
from a vendor. Tools and models **are** named, deliberately: which agent and which model ran
is a fact about a vendor, not about you.

**No figure originates in the renderer.** Every number is read from what `internal/analyze`
published for the same window, including its published *string*, so a card cannot disagree
with the report it came from.

**The artifact makes no request.** The preview page is self-contained — inline CSS, one
inline script, no external font, image or request — and the PNG, MP4 or WebM it offers are
produced in your own browser from a canvas and saved by your own download. `share` writes a
file and asks the desktop to open it (`open`, `xdg-open`, `rundll32`), which is the only
program assaio ever launches; `--no-open` writes the file and stops.

**Someone who receives a posted card learns:** which tools and models you use, aggregate
token, line and cost magnitudes, a session-shape fingerprint, and how many repositories were
active. That is real information about an organization's scale, and posting is a choice. They
learn no name, path, branch, prompt, or line of code.

### `runtime inspect` — an outbound connection

Experimental, opt-in per invocation, and the only place assaio connects to something that is
not a team server ([`runtime-inspect.md`](runtime-inspect.md)).

**Trusted:** the URL you type. It is not derived from any data assaio read.

**Checked** (`runtime.Fetch`, `runtime.FetchLimits`): a plain `GET` and nothing else — no
header a caller can inject, no body, and **no credential**, because an inspection that could
carry a secret would need a threat model this experiment does not have. `--timeout` (5s),
`--max-bytes` (8 MiB) and `--max-redirects` (2; `0` forbids redirects) bound the read. A
non-200 is an error. A body over the budget is an error rather than a truncated read,
because a partial exposition would make every metric it did not reach look absent. Nothing
fetched is stored. `--vllm-file` / `--dcgm-file` read a saved snapshot and touch no network
at all.

**A hostile endpoint can:** serve a bounded body that assaio parses, and attempt a redirect
chain within the limit. **It cannot** obtain a credential, because none is ever sent.

**Residual:** assaio will connect to whatever address you name, including a loopback or
link-local one. The URL comes from your command line, never from parsed data, so this is not
a request-forgery sink — but it is your responsibility to point it at your own exporter.

## 2. Data map

### What never leaves the process at all

Prompt text, model responses, and the content of any file, diff or code. These are not
redacted — they are never decoded. A diff hunk contributes a `+1` or a `-1` and the text
after the prefix is not read. A file path is read into memory to group edits within one
transcript and to resolve a repository root, and discarded before parsing ends.

### What never leaves the machine

The store at `~/.local/share/assaio/assaio.db` (`XDG_DATA_HOME`-aware), including the
`ingest_file` table's full local paths, the `session_label` annotations you type with
`mark` — `sync` does not send them, and no other path does — and the per-install
pseudonymization key at `pseudonym.key` (mode `0600`), which is what makes a `project-xxxx`
label unreproducible by anyone who does not hold it.

### Where the network is, exactly

Three code paths in the whole binary can open a socket, and this is checkable rather than
asserted:

```sh
$ grep -rln '"net/http"' internal/ cmd/ | grep -v _test | sort
internal/cli/sync_push.go
internal/runtime/fetch.go
internal/server/auth.go
internal/server/handlers.go
internal/server/ratelimit.go
internal/server/server.go
```

`sync_push.go` is `sync`, the four `server/` files are `serve`, and `fetch.go` is
`runtime inspect`. Everything else — `backfill`, `report`, `effectiveness`, `analyze`,
`status`, `check`, `doctor`, `dashboard`, `share`, `reconcile`, `digest`, `mark`, `compact`,
`clear` — has no way to reach a network. The model price table is embedded at build time
(`//go:embed litellm.json`), so pricing is offline too. There is no telemetry, no analytics,
and no crash reporting anywhere in the repository.

### What crosses the machine boundary, and under whose control

| Path | Trigger | Destination | Carries | Redaction |
|---|---|---|---|---|
| `sync` | you run it | a server **you** operate | raw `usage.Record` rows from `Store.Export`: tool, session id, timestamp, model, token counts, dedupe key, **project**, **subpath**, **branch**, entrypoint, granularity, activity counts, **skill**, **agent** | member is a pseudonym unless `--member` is passed; the other names travel as they are stored |
| `serve` | you run it | whoever holds a token | the aggregated Assay dashboard | member and project pseudonymized (`anonymize = true`, not overridable over HTTP); raw names only via `report --identify` against the same store, which says so in its own output |
| `runtime inspect --*-url` | per invocation | an endpoint you name | one `GET`, no credential | n/a — nothing is sent |
| metric plugin | declared in config | a local subprocess you chose | stored aggregates: projects, models, member pseudonyms, token/line counts — only the sections, columns and rows the plugin declares, and only those your `needs:` veto allows | none beyond member pseudonyms — this is a local program you trusted by declaring it |
| rule plugin | declared in config | a local subprocess you chose | verdicts only, `Bars` stripped | structural |
| `share` | you run it | a file you then post | figures quoted from `analyze`, tools, models, counts | structural; no name can be rendered |
| `dashboard` | you run it | a file you then share | verdicts, figures, a project drill-down | project names pseudonymized by default; `--no-anonymize` opts out |

Two rows deserve emphasis because they are easy to read the other way round:

- **`sync` sends real project, subpath, branch, skill and sub-agent names.** Pseudonymization
  on the team server is applied when a figure is *rendered* (`dashboard.Build(..., anonymize
  = true, ...)`), not when a record is stored. That is consistent with
  [`PRIVACY.md`](../PRIVACY.md) — "you run the server on your own infrastructure and control
  what reaches it" — and it is the thing to know before pointing `sync` at a host you do not
  control.
- **Member identity is pseudonymous by default in both directions.** `sync` derives a stable
  `member-xxxx` from hostname and OS user unless `--member` opts in; `report` renders a
  pseudonym in table, JSON and CSV alike, and `report --identify` is the single door to raw
  names.

### Deletion and retention

Everything is local and self-contained, so deletion is a local operation with no request to
file and no third party to ask. `clear` scopes it, `compact` returns the space, deleting the
database file removes everything at once. The step timeline is bounded by
`trace.horizon_days` (30 by default) and pruned on ingest. Part 3 runs the procedure.

One retention fact points the other way and `doctor` states it: Claude Code deletes its own
transcripts after 30 days by default, so a store older than that holds history the source no
longer has. A `clear` on those days is irreversible by any means, including a re-import.

## 3. The deletion test

The mechanics exist in `clear` and `compact`. This is the written procedure, and the
transcript below is a real run of it on 2026-09-02, not an illustration.

**Run it on a throwaway store.** `clear` has no `--db` flag and always opens
`paths.DBPath()`, so the only way to point it somewhere safe is to redirect the data
directory for the whole shell:

```sh
export XDG_DATA_HOME="$(mktemp -d)"     # the store clear will open
export XDG_CONFIG_HOME="$(mktemp -d)"   # so the config below is the only one in effect
FIXTURES="$(mktemp -d)"
DB="$XDG_DATA_HOME/assaio/assaio.db"
```

Seed it from the repository's own parser fixture rather than from real data — 300 copies of
`internal/parser/claude/testdata/session.jsonl`, each with distinct session, line and agent
ids so the copies are separate sessions rather than duplicates the store would dedupe away:

```sh
mkdir -p "$FIXTURES/claude" "$FIXTURES/none"
i=1
while [ "$i" -le 300 ]; do
  mkdir -p "$FIXTURES/claude/project-$i"
  sed -e "s/\"uuid\":\"/\"uuid\":\"$i-/g" \
      -e "s/\"sessionId\":\"/\"sessionId\":\"$i-/g" \
      -e "s/\"agentId\":\"/\"agentId\":\"$i-/g" \
      internal/parser/claude/testdata/session.jsonl \
      > "$FIXTURES/claude/project-$i/session.jsonl"
  i=$((i + 1))
done

mkdir -p "$XDG_CONFIG_HOME/assaio"
cat > "$XDG_CONFIG_HOME/assaio/config.yaml" <<YAML
sources:
  claude: ["$FIXTURES/claude"]
  codex: ["$FIXTURES/none"]
  gemini: ["$FIXTURES/none"]
  cline: ["$FIXTURES/none"]
  copilot: ["$FIXTURES/none"]
  agy: ["$FIXTURES/none"]
trace:
  horizon_days: 0
YAML
```

Every source is overridden, including the five with no fixtures: `paths.Resolve` replaces
the built-in roots rather than merging with them, so this is what keeps `backfill` from
reading the real home directory. `horizon_days: 0` keeps the step timeline unpruned, so the
deletion has to reach it too.

### Seed, and confirm what is there

```
$ assaio-agent backfill
claude-code   files=300  records=1500  inserted=1500  steps=1507
codex         files=0  records=0  inserted=0
gemini-cli    files=0  records=0  inserted=0
copilot-cli   files=0  records=0  inserted=0
cline         files=0  records=0  inserted=0
agy           files=0  records=0  inserted=0

$ assaio-agent doctor --db "$DB" 2>&1 | grep -E "^(store|size|timeline):"
store:        ok, 1500 record(s) at /var/folders/.../assaio/assaio.db
size:         768.0 KB
timeline:     1.5K step(s) · 2026-07-01 → 2026-07-01

$ assaio-agent report --db "$DB" --since 3650d
+------------+---------------+---------------------------+---------+--------+---------+---------+--------+--------+
| DAY        | TOOL          | MODEL                     |      IN |    OUT | CACHE R | CACHE W | CACHE% | COST $ |
+------------+---------------+---------------------------+---------+--------+---------+---------+--------+--------+
| 2026-07-01 | claude-code   | claude-haiku-4-5          |   3,000 |  6,000 |       0 |       0 |    0.0 |  0.033 |
| 2026-07-01 | claude-code ‡ | claude-haiku-4-5-20251001 | 150,000 |  9,000 |       0 |       0 |    0.0 |   0.20 |
| 2026-07-01 | claude-code   | claude-opus-4-5           | 135,000 | 96,000 | 810,000 |  15,000 |   85.7 |   3.61 |
+------------+---------------+---------------------------+---------+--------+---------+---------+--------+--------+
|            |               |                           |         |        |         |         |  TOTAL |   3.84 |
+------------+---------------+---------------------------+---------+--------+---------+---------+--------+--------+
```

A session label is added too, because the deletion contract is as much about what survives
as about what goes:

```
$ assaio-agent mark --last --db "$DB" --task refactor --outcome done --difficulty medium
marked 1-s1 · app · 2026-07-01 12:03
  task=refactor outcome=done difficulty=medium

$ ls -l "$DB"
-rw-r--r--@ 1 soon  staff  786432  2 wrz 12:31 /var/folders/.../assaio/assaio.db
```

### Delete

```
$ assaio-agent clear --all --yes
clearing /var/folders/.../assaio/assaio.db (1,500 records)
kept 1 session label(s) -- no re-import can rebuild them; use --labels to delete them too
deleted 1500 record(s)
the next 'assaio-agent backfill' re-reads those inputs from scratch
680.0 KB still held by the store — run 'assaio-agent compact' to reclaim it
```

The path and the count are printed before anything is deleted, which is the whole reason
this command has no `--db`.

### Verify absent

```
$ assaio-agent doctor --db "$DB" 2>&1 | grep -E "^(store|size|timeline):"
store:        ok, 0 record(s) at /var/folders/.../assaio/assaio.db
size:         768.0 KB · 680.0 KB reclaimable — run 'assaio-agent compact'

$ assaio-agent report --db "$DB" --since 3650d
No usage found. This store has no usage records.

$ assaio-agent mark --list --db "$DB" --since 3650d
No sessions in this window. This store has no usage records.

$ ls -l "$DB"
-rw-r--r--@ 1 soon  staff  786432  2 wrz 12:31 /var/folders/.../assaio/assaio.db
```

Three things to read here. Records are gone. The **`timeline:` line is absent** — `doctor`
prints it only when the store holds steps, so all 1,507 went with the records, which is what
`clear` erasing the timeline under the same scope means in practice. And **the file is still
786,432 bytes**, byte for byte what it was before the deletion: SQLite frees pages *inside*
the file and never shrinks it on `DELETE`. Anyone who ran `clear` to reclaim disk has not
reclaimed any yet, which is why `clear` says so on its last line.

### Compact, and verify the size

```
$ assaio-agent compact --db "$DB"
store: 768.0 KB -> 88.0 KB (reclaimed 680.0 KB)

$ ls -l "$DB"
-rw-r--r--@ 1 soon  staff  90112  2 wrz 12:31 /var/folders/.../assaio/assaio.db
```

786,432 → 90,112 bytes. The 88 KB that remain are the schema, its indexes, and the empty
bookkeeping tables. `compact` needs roughly twice the store's size in temporary disk space,
which is why it is a separate command rather than a step inside `clear`.

### The part `--all` deliberately does not delete

```
$ assaio-agent clear --labels --yes
clearing /var/folders/.../assaio/assaio.db (0 records)
--labels with no other scope: deleting all 1 session label(s); no re-import can rebuild them
deleted 1 session label(s)

$ assaio-agent compact --db "$DB"
store: 88.0 KB, nothing to reclaim
```

The label outlived `clear --all`, exactly as the command said it would, and needed its own
flag. Nothing rebuilds a label a person typed, so deleting it is a second decision rather
than a side effect of the first.

### What this establishes

- A scoped deletion removes usage records **and** the step timeline in the same scope.
- No surface — `report`, `doctor`, `mark --list` — can still see the deleted data.
- Deleting rows does not shrink the file; `compact` is what returns the space, and it
  returned 86% of it here.
- Session labels are excluded from every scope but `--labels`, and the exclusion is
  announced rather than silent.
- Deleting the database file removes everything at once and needs no command at all.

Re-run this whenever `clear`, `compact`, the schema, or the retention rules change. The
procedure is the test; the numbers above are one run of it.
