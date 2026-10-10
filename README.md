<div align="center">

# assaio: AI adoption, estimated costs and local delivery evidence

**See how AI coding tools are used, compare estimated costs and inspect local delivery evidence.**

assaio is an open-source, offline-first Go CLI (`assaio-agent`) with embedded SQLite for Claude
Code, Codex CLI, Gemini CLI, GitHub Copilot CLI, Cline and Antigravity CLI. Platform and engineering
teams can inspect observed usage breadth, API-equivalent cost estimates and bounded local PR,
review and CI observations. Antigravity contributes activity only, excluded from token and cost
figures. assaio collects no prompts, responses or code and never ranks people. Optional self-hosted
team mode pools usage; delivery evidence remains on the invoking clone.

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

- Which tools and projects appear in observed sessions, and on how many active days?
- What are the API-equivalent cost estimates for Claude Code, Codex, Gemini CLI, Copilot CLI and Cline?
- Where do logs record edits, rework or failed tool calls?
- What PR, review and CI evidence is available around candidate commits in this clone?
- How complete is each figure, and which source could not supply it?

Adoption describes observed sessions, active days and tool/project breadth, not the percentage of
all employees using AI. The local `evidence` command reports session→commit candidates with
confidence, ambiguity, abstention and population coverage. Optional `--github` adds bounded PR,
review, latest-head check and historical suite/run observations through your own `gh`. They are
facts about PRs, not proof of AI productivity or session outcomes. Exact review rounds, comparable
PR pipeline CI rates, authoritative squash-versus-rebase methods and PR revert relations remain
outside the shipped scope. See the [roadmap](ROADMAP.md).

<p align="center">
  <img src="docs/assets/report-by-project.svg" alt="assaio AI coding effectiveness report by project" width="720">
</p>

## Who assaio is for

`assaio` is for platform teams, engineering-enablement groups and individual developers that:

- need to understand observed AI usage across tools and projects
- want estimated costs on a common basis, with unpriced usage visible
- need local PR, review and CI evidence before choosing what to investigate
- want an inspectable baseline with clear sources and missing-data handling
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

With Go 1.26 or newer:

```sh
go install github.com/assaio/assaio/cmd/assaio-agent@latest
```

Or download an archive from [GitHub Releases](https://github.com/assaio/assaio/releases). Releases
include checksums, an SPDX SBOM and build-provenance attestations. See [RELEASING.md](RELEASING.md)
to verify them.

## First run

Preview `assaio` without reading your logs:

```console
$ assaio-agent demo
```

Then run the guided import:

```console
$ assaio-agent init
```

`init` shows which local logs it will read and which configured parser plugins it will run, imports
their history and writes the first report. assaio itself sends nothing over the network; a parser
plugin from your config is your own program (see [PRIVACY.md](PRIVACY.md)).

The usual loop is short:

```console
$ assaio-agent dashboard --since 30d --output assay.html
$ assaio-agent evidence --repo . --since 30d
$ assaio-agent evidence --repo . --since 30d --github
$ assaio-agent digest --weekly --dry-run
$ assaio-agent doctor --strict
```

- `dashboard` creates a self-contained offline HTML report.
- `evidence` compares local sessions with local commit observations without storing an edge;
  optional `--github` uses your own `gh` to read bounded PR, review, latest-head check and separate
  historical suite/run observations. Only PRs named by candidates or alternatives get details.
- `digest` reports what changed since the previous run and whether the comparison is sound.
- `doctor` reports source coverage, format drift, store health and unpriced usage.

For automation or a narrower question, use `report`, `effectiveness`, `analyze`, `recommend`,
`reprice`, `reconcile`, `check` and `signals`. The generated [command
reference](https://assaio.dev/docs/reference) is authoritative; the README omits the full flag list.

## What assaio measures

| Layer | Available now | Claim |
| --- | --- | --- |
| Activity / adoption | observed sessions, active days, tool/project breadth, turns and model mix | usage was observed in the available logs |
| Output | accepted edits, AI line activity, rework and rejection where recorded | an artifact was produced |
| Local delivery evidence | session→commit candidates; optional bounded PR, review and CI observations | a candidate relation or PR state was observed, without causal attribution |
| Outcome | directional local `survival` check | recent commit changes remain in `HEAD`, without assigning lines to AI sessions |
| Impact | not shipped | no causal delivery, quality or business result is established |

Every metric includes source coverage, sample size, freshness and parser version. If a source lacks
a field, `assaio` excludes it from the denominator rather than counting it as zero. For an exec
parser plugin, omitted token counters are still stored as 0 (`B212`). A missing model price appears
as `—`/`null`, not `$0`.

Run:

```console
$ assaio-agent signals coverage
```

to see what your data supports. [FEATURES.md](FEATURES.md) lists shipped features;
[docs/corrections.md](docs/corrections.md) records published figures later found to be wrong.

`evidence` observes attribution; it is not an outcome metric. It joins a repository's commits
only with sessions whose rows resolved to that repository, uses bounded time proximity, labels
results `matched`, `ambiguous` or `unmatched`, and shows competing commits. A match does not
show that the session caused the commit. With `--github`, independent bounded repository reads
add PR and review snapshots, latest-head checks and historical suites/runs on currently listed
commits. Details appear only for PRs named by candidates or alternatives. Counts expose missing
and truncated layers; null means unavailable, while an explicit empty connection means known
empty. Reviewed-revision counts require complete usable review populations. A request-change snapshot
share uses the entire named merged PR population, including alternatives, as its denominator
and is withheld unless every PR has a nonempty complete usable review population. Exact rounds and comparable PR pipeline CI rates are withheld. Merge topology
can prove a multi-parent merge, but one parent cannot distinguish squash from rebase. Observations
stay local and in memory for one command; offline evidence stays unchanged. See
[how to read the result](docs/evidence.md).

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
reconstructed subscription quota use. Since v0.29.0, when LiteLLM stops listing a model, assaio
keeps the last price LiteLLM published for it; the vendor may no longer offer that rate, and
`doctor` counts such models. `reconcile` compares a downloaded export with the local estimate and
shows any unexplained remainder.

## Privacy: local and offline

On its normal offline analysis path, `assaio`:

- makes no network request and has no telemetry; `evidence --github` asks GitHub for pull
  requests through your own `gh` only when you run it;
- does not extract or store prompt text, response text or repository file contents;
- reads commit hashes, times and content-free change counts only when `evidence` or `survival` runs,
  and stores none of them;
- stores token counts, model names, timestamps, pseudonymous identity and content-free activity
  counts in local SQLite;
- pseudonymizes project and member names by default in dashboard and share output; team sync
  transmits `Project`, `Subpath` and session ID to the server;
- refuses per-person leaderboards for output, spend and productivity.

Line activity comes from counts and diff markers; code on those lines is not stored. See
[PRIVACY.md](PRIVACY.md) for the exact fields, retention and deletion commands.

## Team mode

`serve` and `sync` can pool usage on infrastructure you operate. Sync v2 sends a keyed member
digest and usage records to the team server; project, subpath and session ID also reach it. Each
writer needs a distinct bearer token. Existing central stores require an offline migration and
backup; see the [team-server guide](docs/extending/team-server.md). Team mode is a tested MVP, not
a production-ready service.

It has authentication, request bounds and an aggregated usage dashboard for observed adoption
and estimated costs. PR/review/CI evidence is absent from sync and the server. It still needs RBAC, token
rotation, resumable sync, retention controls, a backup/restore drill and a measured operating
envelope. The [roadmap](ROADMAP.md) lists these gates.

## Extension points

Executables in any language can add a parser, metric or `check` rule. Each protocol has a handshake,
a versioned JSON contract and boundary validation. Parser and metric plugins have a `verify`
command; a rule plugin is checked by running `check`. The core does not import plugin internals.

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

1. GitHub delivery evidence is bounded and local: exact review rounds, comparable PR pipeline CI
   histories and rates, authoritative squash-versus-rebase methods and trustworthy PR revert
   relations still need more source evidence and conformance cases; causal AI outcomes are not derived;
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
