<div align="center">

# assaio: offline AI coding usage and cost analysis

**Compare AI coding cost, token usage and code-producing activity from local logs.**

assaio is an open-source, offline-first Go CLI (`assaio-agent`) with embedded SQLite for Claude
Code, Codex CLI, Gemini CLI, GitHub Copilot CLI, Cline and Antigravity CLI. It reports available
cost, usage and activity evidence without collecting prompts, responses or code. It measures systems
and never ranks people.

[![CI](https://github.com/assaio/assaio/actions/workflows/ci.yml/badge.svg)](https://github.com/assaio/assaio/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/assaio/assaio)](https://goreportcard.com/report/github.com/assaio/assaio)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/assaio/assaio/badge)](https://scorecard.dev/viewer/?uri=github.com/assaio/assaio)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Latest release](https://img.shields.io/github/v/release/assaio/assaio)](https://github.com/assaio/assaio/releases)

[assaio website](https://assaio.dev) · [Roadmap](ROADMAP.md) · [Shipped features](FEATURES.md) ·
[Documentation](docs/README.md) · [Privacy model](PRIVACY.md)

</div>

---

Use vendor dashboards for plan limits and vendor administration. `assaio` gives a local view across
vendors for these questions:

- What did Claude Code, Codex, Gemini CLI, Copilot CLI and Cline cost on the same basis?
- Which repositories turn that spend into accepted edits, and where is there friction?
- How complete is each figure, and which source could not supply it?
- Did a change survive review and CI, beyond producing more lines?
- Can the result be reproduced without uploading prompts, code or conversations?

`assaio` answers the first three questions. The local `evidence` command finds content-free
session→commit candidates and reports confidence, ambiguity, abstention and population coverage. PR,
review, CI, merge and durable-outcome correlation are not shipped. See the [roadmap](ROADMAP.md) for
the research and product choices behind this focus.

<p align="center">
  <img src="docs/assets/report-by-project.svg" alt="assaio AI coding effectiveness report by project" width="720">
</p>

## Who assaio is for

`assaio` is for individual developers, platform teams and engineering-enablement groups that:

- use more than one coding assistant
- need a local, inspectable baseline before giving a server repository access
- want cost and output evidence with clear sources and handling of missing data
- want to test an improvement without making telemetry an employee leaderboard

`assaio` is not a fit for live quota bars, prompt replay, a general LLM observability backend or a
production-ready multi-tenant service. Use vendor tools for quota data. `assaio` does not extract or
store prompt or response bodies. The team server is an MVP.

## Install assaio-agent

`assaio-agent` is one Go binary with embedded SQLite. Prebuilt releases support macOS, Linux and
Windows on amd64 and arm64.

Homebrew:

```sh
brew install assaio/tap/assaio-agent
```

With Go 1.25 or newer:

```sh
go install github.com/assaio/assaio/cmd/assaio-agent@latest
```

Or download an archive from [GitHub Releases](https://github.com/assaio/assaio/releases). Releases
include checksums, an SPDX SBOM and build-provenance attestations. See [RELEASING.md](RELEASING.md)
to verify them.

## Try assaio in 60 seconds

Preview `assaio` without reading your logs:

```console
$ assaio-agent demo
```

Then run the guided import:

```console
$ assaio-agent init
```

`init` shows which local logs it will read, imports their history and writes the first report. It
sends nothing over the network.

The usual loop is short:

```console
$ assaio-agent dashboard --since 30d --output assay.html
$ assaio-agent evidence --repo . --since 30d
$ assaio-agent digest --weekly --dry-run
$ assaio-agent doctor --strict
```

- `dashboard` creates a self-contained offline HTML report.
- `evidence` compares local sessions with local commit observations without storing an edge.
- `digest` reports what changed since the previous run and whether the comparison is sound.
- `doctor` reports source coverage, format drift, store health and unpriced usage.

For automation or a narrower question, use `report`, `effectiveness`, `analyze`, `recommend`,
`reprice`, `reconcile`, `check` and `signals`. The generated [command
reference](https://assaio.dev/docs/reference) is authoritative; the README omits the full flag list.

## What assaio measures

| Layer | Available now | Claim |
| --- | --- | --- |
| Activity | sessions, turns, tool calls, model and entrypoint mix | an observed action happened |
| Output | accepted edits, AI line activity, rework and rejection where recorded | an artifact was produced |
| Outcome | directional local `survival` check only | a defined test was met |
| Impact | not shipped | a delivery, quality or business result changed |

Every metric includes source coverage, sample size, freshness and parser version. If a source lacks
a field, `assaio` leaves it out of the denominator instead of counting it as zero. A missing model
price appears as `—`/`null`, not `$0`.

Run:

```console
$ assaio-agent signals coverage
```

to see what your data supports. [FEATURES.md](FEATURES.md) lists shipped features;
[docs/corrections.md](docs/corrections.md) records published figures later found to be wrong.

`evidence` observes attribution; it is not an outcome metric. It uses a stored project basename and
bounded time proximity, labels results `matched`, `ambiguous` or `unmatched`, and shows competing
commits. A match does not show that the session caused the commit. See [how to read the
result](docs/evidence.md).

## Supported AI coding tools

`assaio` reads existing local logs from these tools:

| Source | Tokens/cost | Activity | Important limit |
| --- | --- | --- | --- |
| Claude Code | yes | yes | local transcripts follow the tool's retention policy |
| Codex CLI | yes | yes | local rollout history is not the account-wide `/usage` history |
| Gemini CLI | yes | limited | calibration still needs an external real capture |
| GitHub Copilot CLI | yes | limited | records are session-grained after completion |
| Cline | yes | limited | calibration still needs an external real capture |
| Antigravity CLI (`agy`) | no | yes | its format publishes neither tokens nor working directory |

The [source-depth matrix](https://assaio.dev/docs/reference#sources) lists each field and its
source. Vendor log formats can change. Tests, calibration checks and `doctor` show format drift but
cannot prevent it.

Costs are API-equivalent estimates from a vendored LiteLLM price snapshot, not vendor invoices or
reconstructed subscription quota use. `reconcile` compares a downloaded export with the local
estimate and shows any unexplained remainder.

## Privacy: local and offline

On its normal offline analysis path, `assaio`:

- makes no network request and has no telemetry;
- does not extract or store prompt text, response text or repository file contents;
- reads commit hashes, times and content-free change counts only when `evidence` or `survival` runs,
  and stores none of them;
- stores token counts, model names, timestamps, pseudonymous identity and content-free activity
  counts in local SQLite;
- pseudonymizes project and member names by default at export boundaries;
- refuses per-person leaderboards for output, spend and productivity.

Line activity comes from counts and diff markers; code on those lines is not stored. See
[PRIVACY.md](PRIVACY.md) for the exact fields, retention and deletion commands.

## Team mode

`serve` and `sync` can pool pseudonymous usage on infrastructure you operate. Team mode is a tested MVP, not a production-ready service.

It has authentication, request bounds and an aggregated dashboard. It still needs RBAC, token
rotation, resumable sync, retention controls, a backup/restore drill and a measured operating
envelope. The [roadmap](ROADMAP.md) lists these gates.

## Extension points

Executables in any language can add a parser, metric or `check` rule. Each protocol has a handshake,
versioned JSON contract, boundary validation and `verify` command. The core does not import plugin
internals.

Start with [docs/extending.md](docs/extending.md). The binary can also export the full
machine-readable reference:

```console
$ assaio-agent docs export
```

The in-process Go packages stay under `internal/` until the public contracts freeze.

## Project status

The local CLI is suitable for evaluation and design-partner pilots. At the v0.26 audit, the project
had an 86% statement-coverage snapshot. It has cross-platform CI, race tests, native parser fuzzers,
vulnerability scanning and a published correction record.

It remains pre-1.0 because:

1. PR, review, CI and durable-outcome correlation beyond local session→commit candidates is not
   shipped;
2. the contracts and calibration have not been tested across several external teams and release
   cycles.

The [roadmap](ROADMAP.md) sets the evidence needed for v1.0. [BACKLOG.md](BACKLOG.md) lists
candidate work, not a schedule.

## Collaboration

For a design partnership, pilot or consulting on measuring AI-assisted engineering, email
[contact@assaio.dev](mailto:contact@assaio.dev) or use the [contact
form](https://karauda.com/contact).

## Contributing and security

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a PR. Each change needs one Conventional
Commit, DCO sign-off, and passing `make fmt`, `make lint` and `make test`. Parser changes must also
pass `make fuzz`.

Report vulnerabilities privately through [GitHub Security
Advisories](https://github.com/assaio/assaio/security/advisories/new), not a public issue. See
[SECURITY.md](SECURITY.md).

## License

Apache-2.0; see [LICENSE](LICENSE). The embedded LiteLLM pricing snapshot is MIT-licensed; see
[NOTICE](NOTICE) for attribution.
