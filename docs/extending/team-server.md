# The team server

*Part of [Extending assaio](../extending.md).*

`assaio-agent serve` runs a self-hosted team server: teammates' `assaio-agent sync` runs
push their local usage to it over HTTP, and it serves back one aggregated,
pseudonymized-by-default Assay dashboard for the whole team at `GET /`
(`internal/server`). This is an MVP — a single shared bearer token gates the *write*
endpoint (`POST /v1/usage`) only, there is no TLS, and **the served dashboard itself has
no auth at all** in this version — anyone who can reach the port sees it; run `serve`
behind a reverse proxy on a trusted network, not exposed to the open internet (see
`internal/server`'s package doc and the security note `serve` prints on startup).

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
plugins](rule-plugin.md), because its dashboard endpoint is
unauthenticated and rebuilt per request — they are local-CLI surfaces (`analyze`,
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
