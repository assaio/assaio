<div align="center">

# assaio

**Vendor-neutral evidence for AI-assisted software engineering.**

Measure cost, usage and code-producing activity from local coding-agent logs today;
connect that evidence to review, CI and durable outcomes next. Offline-first, reproducible
and designed to measure systems rather than rank people.

[![CI](https://github.com/assaio/assaio/actions/workflows/ci.yml/badge.svg)](https://github.com/assaio/assaio/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/assaio/assaio)](https://goreportcard.com/report/github.com/assaio/assaio)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/assaio/assaio/badge)](https://scorecard.dev/viewer/?uri=github.com/assaio/assaio)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Latest release](https://img.shields.io/github/v/release/assaio/assaio)](https://github.com/assaio/assaio/releases)

[Website](https://assaio.dev) · [Roadmap](ROADMAP.md) · [Features](FEATURES.md) ·
[Documentation](docs/README.md) · [Privacy](PRIVACY.md)

</div>

---

AI coding vendors now provide useful usage dashboards of their own. They are the right
place for plan limits and vendor-specific administration. `assaio` is for the questions
that cross those boundaries:

- What did Claude Code, Codex, Gemini CLI, Copilot CLI and Cline cost on one basis?
- Which repositories turn that spend into accepted edits, and where is there friction?
- How complete is each figure, and which source could not supply it?
- Did a change survive review and CI, rather than merely produce more lines?
- Can the result be reproduced without uploading prompts, code or conversations?

The first three are available now. The local `evidence` command now makes the first narrow
part of the fourth visible: content-free session→commit candidates with explicit confidence,
ambiguity, abstention and population coverage. PR, review, CI, merge and durable-outcome
correlation remain the next product milestone; they are not claimed as shipped. The research
and product choices behind that focus are in the [roadmap](ROADMAP.md).

<p align="center">
  <img src="docs/assets/report-by-project.svg" alt="assaio effectiveness report by project" width="720">
</p>

## Is assaio a fit?

Use it when you are an individual, platform team or engineering-enablement group that:

- uses more than one coding assistant;
- needs a local, inspectable baseline before granting a server access to repositories;
- wants cost and output evidence with explicit provenance and missing-data handling;
- wants to test an improvement without turning telemetry into an employee leaderboard.

It is not a fit when you need live quota bars, prompt replay, a general LLM observability
backend, or a production-ready multi-tenant service. Vendor tools cover quota data better;
`assaio` deliberately does not extract or store prompt or response bodies; the team server
is an MVP.

## Install

`assaio-agent` is a single binary with embedded SQLite. Releases cover macOS, Linux and
Windows on amd64 and arm64.

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

## A 60-second first look

See the product without reading your logs:

```console
$ assaio-agent demo
```

Then run the guided import:

```console
$ assaio-agent init
```

`init` shows what it will read, imports the history and writes the first report. It does
not send anything over the network.

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

## What is measured

| Layer | Available now | Claim |
| --- | --- | --- |
| Activity | sessions, turns, tool calls, model and entrypoint mix | an observed action happened |
| Output | accepted edits, AI line activity, rework and rejection where recorded | an artifact was produced |
| Outcome | directional local `survival` check only | a defined test was met |
| Impact | not shipped | a delivery, quality or business result changed |

Every metric result carries its source coverage, sample size, freshness and parser version. A
source that does not record a field is excluded from that denominator; it is never counted
as zero. A missing model price renders `—`/`null`, not `$0`. Run:

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

## Sources

`assaio` discovers the existing local logs of:

| Source | Tokens/cost | Activity | Important limit |
| --- | --- | --- | --- |
| Claude Code | yes | yes | local transcripts follow the tool's retention policy |
| Codex CLI | yes | yes | local rollout history is not the account-wide `/usage` history |
| Gemini CLI | yes | limited | calibration still needs an external real capture |
| GitHub Copilot CLI | yes | limited | records are session-grained after completion |
| Cline | yes | limited | calibration still needs an external real capture |
| Antigravity CLI (`agy`) | no | yes | its format publishes neither tokens nor working directory |

The [source-depth matrix](https://assaio.dev/docs/reference#sources) names every field and
its provenance. Log formats are vendor-internal and can change; golden files, fuzzing,
calibration checks and `doctor` canaries make drift visible, but cannot prevent it.

Costs are API-equivalent estimates from a vendored LiteLLM price snapshot. They are not a
vendor invoice or a reconstruction of subscription quota consumption. `reconcile` compares
an export you downloaded with the local estimate and leaves the unexplained remainder
visible.

## Privacy model

The normal offline analysis path:

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

`serve` and `sync` can pool pseudonymous usage on infrastructure you operate. This is a
tested MVP, not a production claim. It has authentication, request bounds and an aggregated
dashboard, but still needs RBAC, token rotation, resumable sync, retention controls, a
backup/restore drill and a measured operating envelope. Those are explicit gates in the
[roadmap](ROADMAP.md).

## Extension points

Out-of-tree executables can add a parser, metric or `check` rule in any language. Each
protocol has a handshake, a versioned JSON contract, boundary validation and a `verify`
command. The core never imports plugin internals.

Start with [docs/extending.md](docs/extending.md). The complete machine-readable reference
can also be exported by the binary:

```console
$ assaio-agent docs export
```

The in-process Go packages remain under `internal/` until the public contracts freeze.

## Project status

The local CLI is suitable for evaluation and design-partner pilots. It has cross-platform
CI, race tests, native parser fuzzers, an 86% statement-coverage snapshot at v0.26 audit
time, vulnerability scanning and correction lineage. It remains pre-1.0 because two things
are not yet true:

1. the outcome join beyond local session→commit candidates — PR/review/CI and durable
   outcomes — is not shipped;
2. the contracts and calibration have not been proven across several external teams and
   release cycles.

The [roadmap](ROADMAP.md) defines the evidence needed for v1.0. The detailed candidate pool
is in [BACKLOG.md](BACKLOG.md); it is not a schedule.

## Contributing and security

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a PR. Changes use one Conventional
Commit with DCO sign-off and must pass `make fmt`, `make lint` and `make test`. Parser changes
also run `make fuzz`.

Report vulnerabilities privately through [GitHub Security Advisories](https://github.com/assaio/assaio/security/advisories/new),
not a public issue; see [SECURITY.md](SECURITY.md).

## License

Apache-2.0 — see [LICENSE](LICENSE). The embedded LiteLLM pricing snapshot is MIT-licensed;
attribution is in [NOTICE](NOTICE).
