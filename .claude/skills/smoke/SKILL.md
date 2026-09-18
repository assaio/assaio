---
name: smoke
description: Boots the local stack the way a first-time user meets it — build the binary, demo, init on a throwaway store, doctor --strict, dashboard, evidence on this repo, and the team server round trip — and reports what a reader sees. Use "does it still run", "smoke test", "first-run path", "try the binary", after any CLI or ingest change.
argument-hint: "[first-run|team|all]"
allowed-tools: Bash, Read, Glob, Agent
---

Scope: $ARGUMENTS (empty = `first-run`).

Everything runs against a throwaway store — `export XDG_DATA_HOME=$(mktemp -d)` once, then
every command below inherits it. The maintainer's real store is never opened here.

## first-run (the documented path: demo → init → dashboard → digest)

```sh
make build
export XDG_DATA_HOME=$(mktemp -d)
bin/assaio-agent demo
bin/assaio-agent init --non-interactive    # reads the real logs into the throwaway store
bin/assaio-agent doctor --strict
bin/assaio-agent dashboard --since 30d --output "$XDG_DATA_HOME/assay.html"
bin/assaio-agent evidence --repo . --since 30d
bin/assaio-agent digest --weekly --dry-run
bin/assaio-agent signals coverage
```

Hand the long ones to `runner` and read the verdicts. What to look for: exit codes; a source
detected that `doctor` did not name; a `$0` or `0%` where a `—` belongs; an unpriced share
above `pricing.max_unpriced_share`; a dashboard that opens (`/ui-check` for the browser view).

## team (serve + sync round trip on loopback)

```sh
ASSAIO_SERVER_TOKEN=$(head -c 24 /dev/urandom | base64) bin/assaio-agent serve --addr 127.0.0.1:8787 &
bin/assaio-agent sync --server http://127.0.0.1:8787 --since 7d --token "$ASSAIO_SERVER_TOKEN"
curl -fsS -H "Authorization: Bearer $ASSAIO_SERVER_TOKEN" http://127.0.0.1:8787/ | head -c 400
kill %1
```

The token is generated here and printed nowhere; `serve` writes its own `assaio-server.db`
under the throwaway data dir.

Report: each command, its exit code, and the one line of its output that answers "did it work",
plus anything the documented first-run path (`README.md`, "A 60-second first look") promises
and the binary did not deliver.
