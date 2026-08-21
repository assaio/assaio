# The team server

*Part of [Extending assaio](../extending.md).*

`assaio-agent serve` runs a self-hosted team server: teammates' `assaio-agent sync` runs
push their local usage to it over HTTP, and it serves back one aggregated,
pseudonymized-by-default Assay dashboard for the whole team at `GET /`
(`internal/server`). This is still an MVP — there is no TLS, so run `serve` behind a reverse
proxy on a trusted network rather than exposing it to the open internet.

**Every route requires the bearer token, the dashboard included** (since v0.24; it was open
before, which protected the wrong direction — the page carries a whole team's usage).

Identity has two modes and `serve` prints which one it started in:

- **Server-derived** — set `server.members` to one secret per member. The member is whoever
  holds the secret, and the name in the request body is ignored. Prefer this: the dedupe-key
  prefix that keeps two members' rows apart has always assumed exactly one possible writer per
  row, and this is what enforces it.
- **Client-asserted** — a single shared `server.token`. It still works, and any holder can push
  usage under any member name. `serve` says so at startup.

Requests are rate limited per secret (`server.rate_limit_per_minute`, default 120; a negative
value disables it for a deployment that bounds traffic elsewhere), keyed by secret rather than
by address so one member's runaway loop cannot lock out their colleagues.

Point `assaio-agent doctor --db <central store>` at the server's database to see its size,
reclaimable space, measured growth and projected year.

The extension mechanism does not change at that boundary. `server.BuildDashboard`
(`internal/server/dashboard.go`) calls the exact same `dashboard.Build` the local
`assaio-agent dashboard` command calls, over the exact same process-wide
`analyze.Validators()` registry every validator self-registers into — there is no
separate server-side validator list. That means a custom validator compiled into your
team's `assaio-agent` build (see [Adding a metric validator](metric-validator.md))
shows up on the team server's dashboard automatically: same faceplate cell, same ledger
entry, same anonymization rules, with nothing to configure on the server side. The
deliberate exception is **exec plugins**: `serve` executes neither [metric
plugins](metric-plugin.md) nor [rule
plugins](rule-plugin.md), because its dashboard endpoint is rebuilt per request and a
subprocess per view is a denial-of-service surface the server has no budget for — they are
local-CLI surfaces (`analyze`,
`dashboard`, `metrics verify`, and `check` for rules; see [ADR
0004](../adr/0004-exec-metric-plugin-protocol.md) and [ADR
0005](../adr/0005-exec-rule-plugin-protocol.md)). The one
difference from the local CLI is that the served dashboard's anonymization is not
optional — `BuildDashboard` hardcodes `anonymize = true`, so a real-name view is only
ever available locally, as an explicit `--no-anonymize` run against a copy of the store
(`assaio-agent dashboard --db <path-to-central-db> --no-anonymize`), never as the
served default.

```yaml
# on the server
server:
  addr: 127.0.0.1:8787    # loopback by default; widen deliberately
  token: ""    # required; override with ASSAIO_SERVER_TOKEN, do not commit a real one

# on each teammate's machine
sync:
  server: "http://assaio.internal:8787"
  token: ""    # override with ASSAIO_SYNC_TOKEN
  member: ""   # opt-in self-identification; default: an auto pseudonym from hostname+OS-user
```

---
