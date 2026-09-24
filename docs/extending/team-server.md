# The team server

*Part of [Extending assaio](../extending.md).*

`assaio-agent serve` runs a self-hosted team server. Teammates use `assaio-agent sync` to push local
usage over HTTP. At `GET /`, the server returns an aggregated Assay dashboard, pseudonymized by
default, for the whole team (`internal/server`). This is an MVP with no TLS. Run `serve` behind a
reverse proxy on a trusted network; do not expose it to the open internet.

**Every route except `/healthz` requires the bearer token, including the dashboard** (since v0.24;
it was previously open despite showing the whole team's usage). `/healthz` reads nothing and stays
open for orchestrators. It is exempt from rate limits so unrelated traffic cannot fail liveness
checks.

Identity has two modes; `serve` prints the active mode at startup:

- **Server-derived** — set `server.members` to one secret per member. The secret holder determines
  the member; the request body's name is ignored. Prefer this mode. The dedupe-key prefix separating
  members' rows assumes one possible writer per row, and this mode enforces that.
- **Client-asserted** — use one shared `server.token`. Any holder can push usage under any member
  name. `serve` reports this at startup.

Requests are rate limited per secret (`server.rate_limit_per_minute`, default 120). A negative value
disables the limit for deployments that bound traffic elsewhere. Keying by secret instead of address
keeps one member's runaway loop from locking out colleagues.

Run `assaio-agent doctor --db <central store>` against the server database to see its size,
reclaimable space, measured growth, and projected year.

The same extension mechanism applies to the server. `server.BuildDashboard`
(`internal/server/dashboard.go`) calls the same `dashboard.Build` as local `assaio-agent dashboard`,
using the same process-wide `analyze.Validators()` registry where validators self-register. There is
no separate server validator list. A custom validator compiled into the team's `assaio-agent` build
(see [Adding a metric validator](metric-validator.md)) appears automatically on the team dashboard
with the same faceplate cell, ledger entry, and anonymization rules; no server config is needed.
**Exec plugins** are excluded: `serve` runs neither [metric plugins](metric-plugin.md) nor [rule
plugins](rule-plugin.md). The dashboard rebuilds for each request, and spawning a subprocess per
view would create a denial-of-service risk without a server budget. They run on local CLI surfaces:
`analyze`, `dashboard`, `metrics verify`, and `check` for rules (see [ADR
0004](../adr/0004-exec-metric-plugin-protocol.md) and [ADR
0005](../adr/0005-exec-rule-plugin-protocol.md)). Served dashboards always anonymize:
`BuildDashboard` hardcodes `anonymize = true`. Real names are available only locally, through an
explicit `--no-anonymize` run against a store copy
(`assaio-agent dashboard --db <path-to-central-db> --no-anonymize`), never through the served
dashboard.

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
