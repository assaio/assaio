# The team server

*Part of [Extending assaio](../extending.md).*

`assaio-agent serve` runs an optional self-hosted aggregate usage MVP. Teammates use
`assaio-agent sync` to push local usage for observed adoption and API-equivalent cost estimates.
Adoption covers synced sessions, active days and tool/project breadth, not every employee.
At `GET /`, the server returns an aggregated Assay dashboard, pseudonymized by default, for the
synced team usage (`internal/server`). Local PR, review and CI evidence stays on the invoking
clone and is absent from sync and the server. This is an MVP with no TLS or user roles. Put it
behind a TLS reverse proxy on a trusted network.

**Every route except `/healthz` requires the bearer token, including the dashboard** (since v0.24;
it was previously open despite showing the whole team's usage). `/healthz` reads nothing and stays
open for orchestrators. It is exempt from rate limits so unrelated traffic cannot fail liveness
checks.

`serve` prints its identity mode at startup:

- **Server-derived** — set `server.members` to one bearer token per keyed member digest. The token
  determines the member; the request's digest must match it. This mode accepts sync v2 writes.
- **Shared token** — one `server.token` permits dashboard reads but cannot accept sync v2 writes;
  a push receives 409. The old `/v1/usage` write route returns 426.

Requests are rate limited per secret (`server.rate_limit_per_minute`, default 120). A negative value
disables the limit for deployments that bound traffic elsewhere. Keying by secret instead of address
keeps one member's runaway loop from locking out colleagues.

Configure the server with each client's locally printed digest and a separate token:

```yaml
server:
  addr: 127.0.0.1:8787
  members:
    member-v2-0123456789abcdef0123456789abcdef: "replace-with-member-bearer-token"
```

On the matching client, set a stable local input; keep real secrets outside this file:

```yaml
sync:
  server: "https://assaio.example.com"
  member: alice
```

Provision `ASSAIO_SYNC_TOKEN` with the matching bearer token and `ASSAIO_SYNC_IDENTITY_KEY` with a
private 32-byte key encoded as 64 hex digits. Keep the identity key on clients, outside server
config, mapping files, repositories and requests. Back it up and preserve it across runs. Devices
representing one member need the same key and stable local `--member` or `sync.member` input.
That input is never sent as a cleartext member name. Print the digest without a network request:

```sh
assaio-agent sync --print-member-digest --member alice
```

For an existing central database with synced member rows, upgrade in this order:

1. Stop `serve` and back up the central database. The migration command cannot prove that the
   server process is stopped.
2. Print each member's v2 digest using that client's retained identity key and stable input.
   Prepare a complete one-to-one map from every old stored member label to its new digest in a
   local file outside the repository. For example:

   ```json
   {"members":[{"from":"alice","to":"member-v2-0123456789abcdef0123456789abcdef"}]}
   ```

3. While `serve` is stopped, run:

   ```sh
   assaio-agent serve migrate-sync --db /path/server.db --map /private/path/mapping.json
   ```

   The command checks that the map covers every stored member, that targets are distinct, and
   that existing dedupe keys have the expected prefix. One transaction rekeys usage rows, the
   legacy archive, session labels and steps, changes dedupe-key prefixes, clears branch values
   on synced usage and archive rows, verifies the result, and marks the store protocol 2. An
   error rolls back the transaction.
4. Configure `server.members` with the resulting digests, start the new server, and run v2
   clients with their matching tokens and identity keys. Do not run an older server binary
   against the migrated database.

Fresh stores use protocol 2. Existing stores with synced member rows reject v2 writes until
migrated. The map is not sent or stored in the database; its local file remains until you remove
it. Old backups, SQLite free pages and WAL files may retain old values. A lost or rotated identity
key changes the digest and has no automatic rekey procedure. Historical collisions in the old
40-bit labels cannot be separated. Sync v2 excludes `GitBranch` and cleartext `Member` from the
wire payload, but still sends `Project`, `Subpath`, `SessionID`, timestamps and other allowed
usage fields, which can support external joins.

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
`BuildDashboard` hardcodes `anonymize = true`. To show project names, run
`assaio-agent dashboard --db <path-to-central-db> --no-anonymize` locally against a store copy.
`report --identify` reveals the stored member identifier, which may be a legacy name or a v2
digest.
