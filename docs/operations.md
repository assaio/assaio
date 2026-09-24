# Operations catalogue

Every entry point a maintainer or an agent can run, with what it writes and what guards it.
Grouped by flow, in the order they are run. `python3 .claude/checks/ops_catalogue.py` fails when
a Makefile target, workflow, script, hook or check has no row here, or a row names nothing in
the tree. ⚠ marks what writes to a real store, the network, or a published surface.

Column key — **writes**: what changes on disk or remotely · **guard**: the flag, gate or hook that
stops a mistake · **paid**: whether it spends a model subscription · **needs**: credentials or
tools beyond Go.

## Verify (read-only, run freely)

| Entry point | Does | Writes | Guard | Paid | Needs |
|---|---|---|---|---|---|
| `make lint` | `gofmt -l`, `go vet`, `golangci-lint run` | nothing | — | no | golangci-lint |
| `make test` | `go test ./...` (CI adds `-race`, coverage) | test caches | — | no | — |
| `make build` | static binary to `bin/assaio-agent`, version from `git describe` | `bin/` (ignored) | — | no | — |
| `make fuzz` | every parser, reconciler, plugin and openmetrics fuzzer for `FUZZTIME` (20s) | a crasher under `testdata/fuzz/` — keep it | — | no | — |
| `make vuln` | `govulncheck ./...` | nothing | — | no | network (vuln DB) |
| `.claude/checks/gate.sh` | the quiet local gate: fmt `--diff`, lint, test, build, docs drift, harness checks, fuzz when a parser changed, vuln with `full` | logs under `~/.cache/assaio-gate/` | — | no | jq, python3 |
| `.claude/checks/setup.py` | harness well-formed: frontmatter, rule globs, cited paths, `AGENTS.md` budget, no Fable id | nothing | — | no | python3 |
| `.claude/checks/ops_catalogue.py` | this file ↔ the tree, both directions | nothing | — | no | python3 |
| `.claude/checks/checks_test.sh` | negative controls for the two checks above | temp dirs | — | no | python3 |
| `.claude/hooks/guard_test.sh` | the deny/ask/allow table for the guard hook | temp dirs | — | no | jq |

## Change the tree (idempotent; commit the result)

| Entry point | Does | Writes | Guard | Paid | Needs |
|---|---|---|---|---|---|
| `make fmt` | `golangci-lint fmt` (gofumpt + goimports) | source files in place | lefthook `format` job checks `--diff` | no | golangci-lint |
| `make tidy` | `go mod tidy` | `go.mod`, `go.sum` | — | no | network (module proxy) |
| `make prices` | downloads LiteLLM prices, sets `SnapshotDate` to today (UTC), folds unprefixed ids into `retained.json` | `internal/pricing/litellm.json`, `internal/pricing/snapshot.go`, `internal/pricing/retained.json` | `make test` fails on missing or mispriced ids | no | network (raw.githubusercontent.com), curl |
| `make docs` | regenerates `docs/reference.json`, `site/reference.html`, `site/docs.html`, `site/docs/` from the binary's registries | those files | `make test` fails on drift | no | — |
| `docs/assets/make-og.py` | redraws `site/og.png`; upload the same image by hand to the repository's Social preview | the PNG named as argument | `sharecard` job | no | python3 + Pillow, macOS fonts |
| `make hooks` | installs the opt-in lefthook hooks | `.git/hooks/` | — | no | lefthook |
| `.lefthook/commit-msg/check.sh` | Conventional Commit subject, `Signed-off-by`, no AI-author trailer | nothing (rejects the commit) | — | no | — |

## The store (never the maintainer's real one)

`backfill`, `clear`, `compact`, `serve` and `statusline` have no `--db`: they open
`$XDG_DATA_HOME/assaio/assaio.db`. Every agent run sets `XDG_DATA_HOME=$(mktemp -d)` first.

| Entry point | Does | Writes | Guard | Paid | Needs |
|---|---|---|---|---|---|
| `assaio-agent init` | first run: shows what it reads, imports, writes the first report | ⚠ the store | `XDG_DATA_HOME` | no | local logs |
| `assaio-agent backfill` | imports all historical local logs | ⚠ the store | `XDG_DATA_HOME` | no | local logs |
| `assaio-agent clear` | deletes stored usage (all, older-than, per-tool) | ⚠ the store | `--yes` required; hook denies without `XDG_DATA_HOME` | no | — |
| `assaio-agent compact` | reclaims freed space (SQLite never shrinks on DELETE) | ⚠ the store file | `XDG_DATA_HOME` | no | — |
| `assaio-agent mark` | labels a session | ⚠ label rows (the only rows no re-import rebuilds) | `--suggest` previews, `--accept-suggested` writes | no | — |
| `assaio-agent digest` | what moved since the last digest | ⚠ the digest snapshot | `--dry-run` | no | — |
| `assaio-agent dashboard` | the offline Assay report | `assaio-dashboard.html` in cwd (ignored) | `--output` | no | — |
| `assaio-agent share` | the postable card/reel/text | `assaio-share.html`, media files (ignored) | — | no | — |

## Team (network, opt-in)

| Entry point | Does | Writes | Guard | Paid | Needs |
|---|---|---|---|---|---|
| `assaio-agent serve` | the team server MVP on loopback, no TLS | ⚠ `assaio-server.db`; listens on `--addr` | `--token` ≥16 bytes or `server.members` required | no | `ASSAIO_SERVER_TOKEN` |
| `assaio-agent sync` | pushes local records to a server | ⚠ HTTP POST to `--server` | warns on cleartext off-loopback; 401 stops | no | `ASSAIO_SYNC_TOKEN`, `ASSAIO_SYNC_SERVER` |
| `assaio-agent runtime` | reads a self-hosted vLLM/DCGM endpoint | nothing | only with `--vllm-url` / `--dcgm-url` | no | the endpoint |

## Release and publish (asks first)

| Entry point | Does | Writes | Guard | Paid | Needs |
|---|---|---|---|---|---|
| `make release-patch` | computes the next patch version and runs `release` | see `release` | `CONFIRM=yes` | no | — |
| `make release-minor` | computes the next minor version and runs `release` | see `release` | `CONFIRM=yes` | no | — |
| `make release` | clean tree, changelog section present and `[Unreleased]` empty, gate, then `git tag -a` and prints the push command | ⚠ a local tag | `CONFIRM=yes`; settings `ask`; tags immutable | no | — |
| `make snapshot` | goreleaser snapshot build, publishes nothing | `dist/` (ignored) | — | no | goreleaser, syft |
| `.github/workflows/release.yml` | on a `v*` tag push: goreleaser archives, SBOMs, ⚠ GitHub Release, ⚠ Homebrew formula to `assaio/homebrew-tap`, provenance attestation | ⚠ public artifacts | the tag push is the trigger; `RELEASING.md` | no | `HOMEBREW_TAP_GITHUB_TOKEN` (repo secret) |
| `.github/workflows/site.yml` | guards only: no version named, nothing fetched at render, share card exists, served paths resolve | nothing | — | no | — |
| Cloudflare Workers Builds | ⚠ `npx wrangler deploy` of `site/` on every push to `main`, ungated (`docs/site.md`) | ⚠ https://assaio.dev/ | none upstream; verify with the live-page diff in `RELEASING.md` | no | Cloudflare Git connection |

## CI on every push and pull request (read-only)

| Entry point | Does | Writes | Guard | Paid | Needs |
|---|---|---|---|---|---|
| `.github/workflows/ci.yml` | fmt, vet, lint, `go test -race` on Linux/macOS/Windows, build, govulncheck | nothing | required for merge | no | — |
| `.github/workflows/consistency.yml` | backlog hygiene, changelog ↔ tags, CITATION.cff ↔ parsers | nothing | required for merge | no | — |
| `.github/workflows/dco.yml` | `Signed-off-by` on every commit, no AI-author trailer | nothing | required for merge | no | — |
| `.github/workflows/fuzz.yml` | 30s per fuzzer on PRs touching parsers, 10 min nightly; opens an issue on a nightly failure | ⚠ a GitHub issue (nightly) | — | no | — |
| `.github/workflows/codeql.yml` | CodeQL for Go, Actions, Python | SARIF to code scanning | — | no | — |
| `.github/workflows/scorecard.yml` | OpenSSF Scorecard, results published | the public score | — | no | — |

## Agent harness (this repository's `.claude/`)

| Entry point | Does | Writes | Guard | Paid | Needs |
|---|---|---|---|---|---|
| `.claude/hooks/guard.sh` | PreToolUse: denies the real store, tag rewrites, AI trailers, `git add .`, secrets on screen, Fable ids; asks for `claude -p`, direct codex/agy, force-push, a push to `main` with open work | nothing | its own test table | no | jq |
| `.claude/hooks/session_start.sh` | three lines of state at session start | nothing | — | no | — |
| `.claude/scripts/text_model.py` | the GPT/Gemini door: `--detect`, or one prompt + inputs → validated output, `--engine auto\|gpt\|gemini\|both` | the `--out` file(s) | schema validation; exit 2 when no engine | ⚠ yes — one codex/agy session per call | codex and/or agy logged in |

## To decide

- `docs/assets/make-og.py` needs Pillow and macOS system fonts; it runs by hand and nothing
  checks that the PNG still matches the palette in `site/index.html`.
- `make snapshot` and `make hooks` need goreleaser, syft and lefthook, none of which this
  machine has; `make vuln` fetches `govulncheck@latest` on every run.
- The site deploy has no upstream gate. The alternative — deploying from GitHub Actions with two
  repository secrets — is recorded in `docs/site.md` and not taken.
