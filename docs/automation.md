# Automating assaio — hooks, scheduled sync, and survival

This is the operating recipe for keeping an organization's AI-usage picture fresh without a
daemon: refresh the local store, push it to a self-hosted team server, and run the
directional `survival` outcome check on a schedule. Everything here is opt-in, and the only
network step is `sync` (pseudonymized by default). A managed cloud is roadmap; today the
"company server" is one you run yourself with `serve` (see the [team server](../README.md)).

## The pieces

- **`backfill`** — re-imports local session logs into the store. Idempotent (deterministic
  dedupe keys), so running it repeatedly never double-counts, and incremental since v0.4:
  an input already parsed unchanged is skipped, so a repeat run costs a fraction of a
  second rather than a minute. Safe — and now cheap — to run often.
- **`statusline`** — one ambient line for a status bar. It only reads the store, so it is
  as fresh as your last `backfill`; that is why the line always ends with the data's age.
- **`sync`** — pushes local usage to a team server over one bearer-token HTTPS call,
  **pseudonymized by default** (`--member` is an explicit opt-in to a real name). The only
  network path; nothing else leaves the machine.
- **`survival`** — the directional, local outcome check: how much of a repo's window
  survives in `HEAD`, beside the AI lines the store recorded. It shells out to `git blame`,
  so it is heavier than the others — run it periodically (e.g. weekly), not per commit.

## Option A — scheduled refresh + push (recommended)

A small timer keeps the store fresh and pushes it to the company server. This is more
robust than a git hook: it runs whether or not you committed, and never blocks anything.

A wrapper script the timer runs — `~/.local/bin/assaio-refresh`:

```sh
#!/bin/sh
set -eu
assaio-agent backfill >/dev/null
assaio-agent sync --server "https://assaio.example.com" --token "$ASSAIO_SYNC_TOKEN"
```

Keep the token out of the script — read it from the environment (`ASSAIO_SYNC_TOKEN`) or a
secret manager.

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

If you'd rather refresh right after each commit, do it **in the background** so the commit
never waits on a full backfill. `.git/hooks/post-commit` (make it executable):

```sh
#!/bin/sh
# Refresh assaio in the background; never block the commit.
( assaio-agent backfill >/dev/null 2>&1 \
  && assaio-agent sync --server "https://assaio.example.com" --token "$ASSAIO_SYNC_TOKEN" ) &
```

To install it for every repo you clone, point git at a hooks template once:
`git config --global init.templateDir ~/.git-template` and drop the hook under
`~/.git-template/hooks/`.

## Option C — Claude Code session hooks (for `statusline`)

If you want the status line to track the day as you work, refresh at the two points where
a transcript has settled. In `~/.claude/settings.json`:

```json
{
  "statusLine": { "type": "command", "command": "assaio-agent statusline" },
  "hooks": {
    "SessionEnd": [{ "hooks": [{ "type": "command", "command": "assaio-agent backfill" }] }],
    "PreCompact": [{ "hooks": [{ "type": "command", "command": "assaio-agent backfill" }] }]
  }
}
```

**A per-turn (`Stop`) hook is safe since v0.4**, and it is the closest thing to live data
assaio has. It used to be the wrong recommendation for a real reason: Claude Code attributes
a turn's failed tool calls from a *later* line in the transcript, so ingesting at the end of
a turn wrote a row before that line existed, with zero errors recorded — and nothing ever
corrected it, so `friction` under-reported from then on. v0.4 made a re-read restate those
columns upward (`B68`), which removed the objection.

```json
"Stop": [{ "hooks": [{ "type": "command", "command": "assaio-agent backfill" }] }]
```

What it costs, measured on a 4.5 GB / 5710-transcript history:

| The hook fires and… | Time |
|---|---|
| nothing changed since the last run | 0.07 s |
| one typical transcript (112 KB) was appended to | 0.17 s |
| the largest transcript on the machine (89 MB) was appended to | 0.99 s |

That is per turn, on the largest local history we have measured; a fresh install is faster.
The trade is latency against freshness, and it is yours to make — `SessionEnd` and
`PreCompact` alone keep the numbers just as honest, at the cost of a status line that lags
inside a long session, which is exactly why it always shows its age.

## Survival on a schedule

`survival` is the outcome signal — run it per repo, weekly, and log or publish the result.
Because it blames the window's touched files, give it its own (less frequent) timer:

```sh
#!/bin/sh
# ~/.local/bin/assaio-survival-report — one repo, appended to a log.
cd "$1" || exit 1
assaio-agent survival --since 90d >> "$HOME/assaio-survival.log"
```

A weekly cron entry per repo:

```cron
0 8 * * 1 /home/you/.local/bin/assaio-survival-report /home/you/src/acme-web
```

Read it as a trend over weeks: a survival rate that stays high as a repo ages is the honest,
local answer to "is the AI-written code sticking?" — directional, never a per-line
attribution (assaio counts lines, it does not store code).

## The company server

Run one `serve` instance on trusted infrastructure and point every agent's `sync` at it:

```sh
assaio-agent serve --addr :8787 --token "$ASSAIO_SERVER_TOKEN"   # behind a TLS reverse proxy
```

The server collects pushed usage and serves the aggregated, always-anonymized team
dashboard at `/`. It is a deliberate MVP (one shared token, no TLS of its own — run it
behind a reverse proxy on a trusted network); see the server package doc and `ROADMAP.md`
for the hardening path (per-member auth, retention, resumable sync).

## What is and isn't automated to the server today

- **Usage** (tokens, cost, activity) syncs to the server and aggregates into the team
  dashboard. This is the live picture.
- **Survival** runs locally and prints its result; it is **not** pushed to the server yet.
  Server-side survival — correlating synced usage against git and issue trackers across the
  whole team — is the roadmap's "Outcome & quality" stage, and the future managed cloud.
```
