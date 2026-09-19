<div align="center">

# assaio: offline AI coding usage and cost analysis

**Compare AI coding cost, token usage and code-producing activity from local logs.**

`assaio-agent` is an offline-first Go CLI with embedded SQLite for Claude Code, Codex CLI, Gemini CLI, GitHub Copilot CLI, Cline and Antigravity CLI. It reports available cost, usage and activity evidence without collecting prompts, responses or code. It measures systems, never ranks people.

[![CI](https://github.com/assaio/assaio/actions/workflows/ci.yml/badge.svg)](https://github.com/assaio/assaio/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/assaio/assaio)](https://goreportcard.com/report/github.com/assaio/assaio)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/assaio/assaio/badge)](https://scorecard.dev/viewer/?uri=github.com/assaio/assaio)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Latest release](https://img.shields.io/github/v/release/assaio/assaio)](https://github.com/assaio/assaio/releases)

[assaio website](https://assaio.dev) · [Product roadmap](ROADMAP.md) · [Shipped features](FEATURES.md) · [Documentation](docs/README.md) · [Privacy model](PRIVACY.md)

</div>

---

Vendor dashboards remain the right place for plan limits and vendor-specific administration. `assaio` adds a local, cross-vendor view for these questions:

- What did Claude Code, Codex, Gemini CLI, Copilot CLI and Cline cost on one basis?
- Which repositories turn that spend into accepted edits, and where is there friction?
- How complete is each figure, and which source could not supply it?
- Did a change survive review and CI, rather than merely produce more lines?
- Can the result be reproduced without uploading prompts, code or conversations?

Today, `assaio` answers the first three questions. The local `evidence` command also finds content-free session→commit candidates and reports confidence, ambiguity, abstention and population coverage. PR, review, CI, merge and durable-outcome correlation are the next product milestone and are not shipped. See the [roadmap](ROADMAP.md) for the research and product choices behind that focus.

<p align="center">
  <img src="docs/assets/report-by-project.svg" alt="assaio AI coding effectiveness report by project" width="720">
</p>

## Who assaio is for

`assaio` is for individual developers, platform teams and engineering-enablement groups that:

- use more than one coding assistant
- need a local, inspectable baseline before granting a server access to repositories
- want cost and output evidence with explicit provenance and missing-data handling
- want to test an improvement without turning telemetry into an employee leaderboard

`assaio` is not a fit if you need live quota bars, prompt replay, a general LLM observability backend or a production-ready multi-tenant service. Vendor tools are better for quota data. `assaio` does not extract or store prompt or response bodies, and the team server is an MVP.

## Install assaio-agent

`assaio-agent` is a single Go binary with embedded SQLite. Prebuilt releases support macOS, Linux and Windows on amd64 and arm64.

Homebrew:

```sh
brew install assaio/tap/assaio-agent
```

With Go 1.25 or newer:

```sh
go install github.com/assaio/assaio/cmd/assaio-agent@latest
```

Or download an archive from [GitHub Releases](https://github.com/assaio/assaio/releases).
Each release includes checksums, an SPDX SBOM and build-provenance attestations. See
[RELEASING.md](RELEASING.md) for verification.

## Try assaio in 60 seconds

Preview `assaio` without reading your logs:

```console
$ assaio-agent demo
```

Then run the guided import:

```console
$ assaio-agent init
```

`init` shows which local logs it will read, imports their history and writes the first report. It does not send anything over the network.

The normal loop is intentionally short:

```console
$ assaio-agent dashboard --since 30d --output assay.html
$ assaio-agent evidence --repo . --since 30d
$ assaio-agent digest --weekly --dry-run
$ assaio-agent doctor --strict
```

- `dashboard` creates a self-contained offline HTML report.
- `evidence` compares local sessions with local commit observations without storing an edge.
- `digest` says what moved since the previous run and whether the comparison is sound.
- `doctor` reports source coverage, format drift, store health and unpriced usage.

For automation or a narrower question, use `report`, `effectiveness`, `analyze`,
`recommend`, `reprice`, `reconcile`, `check` and `signals`. The generated
[command reference](https://assaio.dev/docs/reference) is authoritative; the README no
longer duplicates every flag.

## What assaio measures

| Layer | Available now | Claim |
| --- | --- | --- |
| Activity | sessions, turns, tool calls, model and entrypoint mix | an observed action happened |
| Output | accepted edits, AI line activity, rework and rejection where recorded | an artifact was produced |
| Outcome | directional local `survival` check only | a defined test was met |
| Impact | not shipped | a delivery, quality or business result changed |

Every metric includes source coverage, sample size, freshness and parser version. If a source does not record a field, `assaio` excludes that source from the denominator instead of counting it as zero. A missing model price appears as `—`/`null`, not `$0`.

Run:

```console
$ assaio-agent signals coverage
```

for the capabilities of your own data. [FEATURES.md](FEATURES.md) records what is shipped;
[docs/corrections.md](docs/corrections.md) records every published figure later found to be
wrong.

`evidence` is an attribution observation beside these layers, not an outcome metric. It uses
only a stored project basename and bounded time proximity, labels its answers `matched`,
`ambiguous` or `unmatched`, and keeps competing commits visible. A match does not prove that
the session caused the commit. See [how to read the result](docs/evidence.md).

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

The [source-depth matrix](https://assaio.dev/docs/reference#sources) lists every field and where it comes from. Vendor log formats can change. Tests, calibration checks and `doctor` make format drift visible, but cannot prevent it.

Cost figures are API-equivalent estimates based on a vendored LiteLLM price snapshot. They are not vendor invoices and do not reconstruct subscription quota use. `reconcile` compares a downloaded export with the local estimate and shows any unexplained remainder.

## Privacy: local and offline

On the normal offline analysis path, `assaio`:

- makes no network request and has no telemetry;
- does not extract or store prompt text, response text or repository file contents;
- reads commit hashes, times and content-free change counts only when `evidence` or `survival`
  is invoked, and stores none of them;
- stores token counts, model names, timestamps, pseudonymous identity and content-free
  activity counts in local SQLite;
- pseudonymizes project and member names by default at export boundaries;
- refuses per-person output, spend and productivity leaderboards.

Line activity comes from counts and diff markers; the code on those lines is not stored.
The exact field map, retention behavior and deletion commands are in [PRIVACY.md](PRIVACY.md).

## Team mode

`serve` and `sync` can pool pseudonymous usage on infrastructure you operate. Team mode is a tested MVP, not a production-ready service.

It includes authentication, request bounds and an aggregated dashboard. It still needs RBAC, token rotation, resumable sync, retention controls, a backup/restore drill and a measured operating envelope. These are explicit gates in the [roadmap](ROADMAP.md).

## Extension points

External executables written in any language can add a parser, metric or `check` rule. Each protocol provides a handshake, versioned JSON contract, boundary validation and `verify` command. The core does not import plugin internals.

Start with [docs/extending.md](docs/extending.md). The complete machine-readable reference
can also be exported by the binary:

```console
$ assaio-agent docs export
```

The in-process Go packages remain under `internal/` until the public contracts freeze.

## Project status

The local CLI is suitable for evaluation and design-partner pilots. At the v0.26 audit, the project had an 86% statement-coverage snapshot. It also has cross-platform CI, race tests, native parser fuzzers, vulnerability scanning and a published correction record.

It remains pre-1.0 because:

1. PR, review, CI and durable-outcome correlation beyond local session→commit candidates is not shipped;
2. the contracts and calibration have not yet been proven across several external teams and release cycles.

The [roadmap](ROADMAP.md) defines the evidence needed for v1.0. The detailed candidate pool
is in [BACKLOG.md](BACKLOG.md); it is not a schedule.

## Collaboration

To discuss a design partnership, pilot or consulting on measuring AI-assisted engineering, email [contact@assaio.dev](mailto:contact@assaio.dev) or use the [contact form](https://karauda.com/contact).

## Contributing and security

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a PR. Each change must use one Conventional Commit, include DCO sign-off, and pass `make fmt`, `make lint` and `make test`. Parser changes must also pass `make fuzz`.

Report vulnerabilities privately through [GitHub Security Advisories](https://github.com/assaio/assaio/security/advisories/new),
not a public issue; see [SECURITY.md](SECURITY.md).

## License

Apache-2.0 — see [LICENSE](LICENSE). The embedded LiteLLM pricing snapshot is MIT-licensed;
attribution is in [NOTICE](NOTICE).
