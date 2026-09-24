# Automating assaio — hooks, scheduled sync, and survival

Keep an organization's AI usage data fresh without a daemon: refresh the local store, push it to a
self-hosted team server, and schedule the directional `survival` outcome check. All steps are
opt-in. Only `sync` uses the network, and it pseudonymizes by default. Managed cloud is on the
roadmap; today you run the company server with `serve` (see the [team server](../README.md)).

## The pieces

- **`backfill`** — imports local session logs into the store. Deterministic dedupe keys make repeats
  idempotent. Since v0.4, it skips unchanged inputs, cutting a repeat run from about a minute to a
  fraction of a second. Run it often without double-counting.
- **`statusline`** — shows one line in a status bar. It only reads the store, so it reflects your
  last `backfill` and always shows the data's age.
- **`sync`** — pushes local usage to a team server in one bearer-token HTTPS call. It is
  **pseudonymized by default**; `--member` explicitly opts in to a real name. This is the only
  network path; nothing else leaves the machine.
- **`survival`** — checks locally how much of a repo's window survives in `HEAD`, alongside stored
  AI lines. It calls `git blame`, so run it periodically, such as weekly, rather than per commit. It
  reports merge commits separately from the rate: git gives no line counts for merges, so a
  hand-resolved conflict counts as neither added nor surviving. In a merge-heavy repo, expect a
  large merge line and read the rate as covering ordinary commits only.

## Option A — scheduled refresh + push (recommended)

A timer refreshes the store and pushes it to the company server. It runs even without a commit and
blocks nothing, unlike a git hook.

Create the timer's wrapper script at `~/.local/bin/assaio-refresh`:

```sh
#!/bin/sh
set -eu
assaio-agent backfill >/dev/null
assaio-agent sync --server "https://assaio.example.com" --token "$ASSAIO_SYNC_TOKEN"
```

Keep the token out of the script. Read it from the environment (`ASSAIO_SYNC_TOKEN`) or a secret
manager.

**launchd (macOS)** — `~/Library/LaunchAgents/com.assaio.refresh.plist`, every 30 minutes:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
  <key>Label</key><string>com.assaio.refresh</string>
  <key>ProgramArguments</key><array><string>/Users/you/.local/bin/assaio-refresh</string></array>
  <key>StartInterval</key><integer>1800</integer>
  <key>EnvironmentVariables</key><dict><key>ASSAIO_SYNC_TOKEN</key><string>…</string></dict>
</dict></plist>
```

`launchctl load ~/Library/LaunchAgents/com.assaio.refresh.plist`.

**cron (Linux)** — every 30 minutes:

```cron
*/30 * * * * ASSAIO_SYNC_TOKEN=… /home/you/.local/bin/assaio-refresh >/dev/null 2>&1
```

## Option B — a git post-commit hook

To refresh after each commit, run it **in the background** so backfill does not delay the commit.
Make `.git/hooks/post-commit` executable:

```sh
#!/bin/sh
# Refresh assaio in the background; never block the commit.
( assaio-agent backfill >/dev/null 2>&1 \
  && assaio-agent sync --server "https://assaio.example.com" --token "$ASSAIO_SYNC_TOKEN" ) &
```

To install it in every repo you clone, set a git hooks template once with
`git config --global init.templateDir ~/.git-template`, then put the hook in
`~/.git-template/hooks/`.

## Option C — Claude Code session hooks (for `statusline`)

To keep the status line current as you work, refresh when a transcript settles: at these two points
in `~/.claude/settings.json`:

```json
{
  "statusLine": { "type": "command", "command": "assaio-agent statusline" },
  "hooks": {
    "SessionEnd": [{ "hooks": [{ "type": "command", "command": "assaio-agent backfill" }] }],
    "PreCompact": [{ "hooks": [{ "type": "command", "command": "assaio-agent backfill" }] }]
  }
}
```

**A per-turn (`Stop`) hook is safe since v0.4** and gives assaio its freshest data. Claude Code
records a turn's failed tool calls on a later transcript line. An end-of-turn ingest can therefore
record zero errors before that line arrives. Since v0.4, a re-read raises those stored columns
(`B68`), so `friction` can include the later errors.

Since v0.14, a re-read sets an activity column to the current parse, even if that is lower than the
stored value. This remains safe during a mid-write read: each attribution rule is monotone within
the transcript prefix, so later lines attach to later turns and cannot reduce an emitted turn's
edits or rework. It also lets a corrected rule update stored rows rather than keep an old, higher
figure. Vendor token counts on `usage_record` still use the maximum because they are vendor numbers.
Derived `session_step.tokens` does not: it sums four selected fields, and a build that corrects that
selection updates stored rows.

```json
"Stop": [{ "hooks": [{ "type": "command", "command": "assaio-agent backfill" }] }]
```

Measured cost on a 4.5 GB / 5710-transcript history:

| The hook fires and… | Time |
|---|---|
| nothing changed since the last run | 0.07 s |
| one typical transcript (112 KB) was appended to | 0.17 s |
| the largest transcript on the machine (89 MB) was appended to | 0.99 s |

These costs are per turn on the largest local history measured; a fresh install is faster. Choose
between latency and freshness. `SessionEnd` and `PreCompact` alone keep the numbers honest, but the
status line may lag during a long session, so it always shows its age.

## A weekly digest, delivered by you

Schedule `digest` to write markdown showing only what **moved** since the last digest. Pipe the
output to wherever your team reads it; assaio does not deliver it.

```sh
#!/bin/sh
# ~/.local/bin/assaio-digest — one week, wherever you read things.
assaio-agent digest --weekly > "$HOME/assaio-week.md"
# …then pick one:
#   mail -s "AI usage this week" you@example.com < "$HOME/assaio-week.md"
#   curl -sS -X POST -H 'Content-type: application/json' \
#     --data "$(jq -Rs '{text: .}' < "$HOME/assaio-week.md")" "$SLACK_WEBHOOK_URL"
```

Match the schedule to the window: run a `7d` digest every 7 days:

```cron
0 9 * * 1 /home/you/.local/bin/assaio-digest
```

**The first run has nothing to compare against** and says so; comparisons start on the second run.
**The digest flags weak comparisons**: overlapping windows when runs are closer together than the
window length, windows of different lengths, or a parser change between runs that may make a
correction look like changed tool use. A single-window view cannot detect that last case. Use
`--dry-run` to print a digest without making it the basis for the next run.

Each run stores a small snapshot of totals, per-model and per-project weights, and each validator's
verdict. It stores no session content. The write transaction keeps only the newest 12, so storage
does not grow over time.

## Suggested labels, once instead of per session

`mark --suggest` uses recorded branch, skill, sub-agent, and entrypoint data to propose labels and
show evidence. It writes nothing until you run `mark --accept-suggested`, and never replaces a
manual label.

```sh
assaio-agent mark --suggest --since 30d          # look
assaio-agent mark --accept-suggested --since 30d # then write
```

Built-in rules read only branch conventions (`feat/`, `fix/`, `test/`, `refactor/`, `docs/`,
`spike/`, `review/`, and common spellings). For other conventions, set `labels.rules` in the config
with a source, RE2 pattern, and implied axis value. Set `labels.defaults: false` if a built-in rule
means something else in your repo. A repo without conventions derives no labels, as intended.

Do not put this in a hook. Derived labels are suggestions; review them before writing so the
vocabulary stays meaningful.

## Survival on a schedule

Run `survival` per repo each week, then log or publish this outcome signal. It blames files touched
in the window, so give it a separate, less frequent timer:

```sh
#!/bin/sh
# ~/.local/bin/assaio-survival-report — one repo, appended to a log.
cd "$1" || exit 1
assaio-agent survival --since 90d >> "$HOME/assaio-survival.log"
```

Add a weekly cron entry for each repo:

```cron
0 8 * * 1 /home/you/.local/bin/assaio-survival-report /home/you/src/acme-web
```

Read the rate as a trend over weeks. A rate that stays high as a repo ages suggests the window's
commits are sticking; it does not say which lines AI wrote. It is directional, never per-line
attribution: assaio counts lines but stores no code.

Each run also reports the window's commit changes by **category** (test / source / docs / config /
generated / other) and how many git labeled as reverts. These numbers come from collected commit
observations ([ADR 0009](adr/0009-local-git-evidence-collector.md)) and feed the planned evidence
graph. Categories use a naming heuristic. No path, branch name, or commit message leaves the repo.

## The company server

Run one `serve` instance on trusted infrastructure and point every agent's `sync` at it:

```sh
assaio-agent serve --addr :8787 --token "$ASSAIO_SERVER_TOKEN"   # behind a TLS reverse proxy
```

The server collects pushed usage and serves an aggregated, always-anonymized team dashboard at `/`.
This MVP has no built-in TLS; put it behind a reverse proxy on a trusted network. See the server
package doc and `ROADMAP.md` for planned per-member auth, retention, and resumable sync.

## What is and isn't automated to the server today

- **Usage** (tokens, cost, activity) syncs to the server and appears in the aggregated team
  dashboard as the live picture.
- **Survival** runs locally and prints its result; it is **not** pushed to the server yet. The
  roadmap's outcomes milestone adds server-side survival by correlating team-wide synced usage with
  git and issue trackers; managed cloud comes after that.
```
