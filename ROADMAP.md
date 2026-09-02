# Roadmap

**Direction, not commitment.** No delivery dates and no guarantee any candidate here ships as
described or ships at all; the order below is the intended sequence, not a schedule. One kind of
version is named on purpose and points the other way: two experimental surfaces have their
**kill criteria read at v0.30** — a promise to decide by then, not to deliver by then, and the
only kind of date a file like this can keep alone. `assaio` is pre-1.0 and the most useful input
to what comes next is feedback from people running it against their own repositories and teams.

Three companions hold the specifics and this file deliberately does not repeat them:
[FEATURES.md](FEATURES.md) (what exists today, and since when), [CHANGELOG.md](CHANGELOG.md)
(the per-release delta), and [BACKLOG.md](BACKLOG.md) (concrete candidate items, each with an
id like `B18`). What v1.0 freezes is [docs/compatibility.md](docs/compatibility.md) — one file,
because this one, the release guide and the extension docs each carried a different answer for
several releases.

## The north star

`assaio` answers one question: **is an organization's use of AI coding tools actually worth
it** — in cost, in code that reaches and survives production, in quality, in developer
experience? Today it measures **output and efficiency** honestly. It does not yet measure
**outcome**: whether that code was any good. Closing that gap without fabricating a number to
do it is the throughline of everything below.

Stated once, so the rest has a subject: **the open-source, privacy-first evidence layer for
AI-assisted software engineering — reading what the agents on your machine already recorded,
and connecting it to commits, review, CI and durable outcomes.**

Two halves, not equal. Correlation makes an outcome claim possible; **the analysis of the logs
themselves is what makes any of it worth reading**, and it is the half that has to be
strongest. A perfect link from a session to a merged pull request says nothing if the session
was parsed shallowly or a format change silently halved the numbers. The local half also works
with no server, no credentials, no network and no repository access — and it is the part a
competitor cannot copy by wiring up one more API. When the two compete for attention, depth of
analysis wins.

`$ per 100 AI lines` is a useful ratio and it stays, as a **directional diagnostic** rather
than the headline. Lines are an *output* measure; promoting one to an outcome claim is the most
likely way this project starts lying.

## Three modules

| Module | What it is | Status |
| --- | --- | --- |
| **Assaio Usage** | The core: how a person or team uses AI coding tools, on what model, at what cost, and producing what — plus the small number of experiments that evidence supports. **Outcome is the milestone, not a shipped layer**: `survival` is a directional local check and nothing else joins a session to what shipped. | **shipped** through the output layer; outcome is stage 3 |
| **Assaio Team & Cloud** | Aggregation and the commercial path: self-hosted or managed collection, budgets, governance, retention, access control, shared recommendations. Never an employee ranking. | self-hosted is an **MVP**; managed is **later** |
| **Assaio Runtime Insights** | Optional, for people running their own models: import request, runtime and accelerator evidence to explain self-hosted cost, latency and capacity. | **experimental** feasibility slice only, behind a demand gate |

Runtime Insights is **not** a second product centre and **not** a v1 requirement. assaio is not
a GPU monitor, a Prometheus, an inference gateway or a cluster manager. A hosted Claude, Codex,
Gemini or Cursor session runs on the vendor's accelerators, which no local signal reveals:
those stay `unknown` and are never estimated.

## Three deployment modes, one engine

**Local** (the open-source binary, embedded SQLite, permanently offline-capable) ·
**Team self-hosted** (`serve` + `sync` on your own infrastructure) · **Assaio Cloud** (managed
operation, not built).

They share ingestion, analytics, recommendation and export contracts. They do not share one
mandatory storage implementation, and the open-source binary must never require the commercial
service to analyze or export its own data. Monetization, when it comes, is managed operation,
scale, governance, retention and support — never withholding local results or making data hard
to get out.

## Four layers, never relabeled

Every figure states which of these it is a claim about, and no surface may promote one to the
next ([ADR 0013](docs/adr/0013-measurement-layers.md)).

| Layer | Coding example | Inference example | Claim allowed |
| --- | --- | --- | --- |
| Activity | a session ran | a request was served | an observed action happened |
| Output | an accepted edit | a served response | an artifact was produced |
| Outcome | the change survived, CI passed | an eval score, an SLO met | the output met a defined test |
| Impact | delivery or defect change | a business or user KPI | an age-matched or controlled effect |

## The next milestones

Ordered from stage 1. A stage may overlap the next only once its exit criteria are
**measured**, not assumed. Stage 0 stands outside that order: it runs alongside all of them,
nothing waits on it, and it is the only one this repository cannot finish by itself.

### 0. Somebody outside this repository — **parallel to every stage below**

The largest risk this file names, under [what v1.0 has to mean](#what-v10-has-to-mean):
**no external user exists.** Every other v1.0 condition can be met by writing code. This one cannot, it has the longest lead time
of anything here, and until now it had no plan — no stage, no item, nobody named to ask.

**The ask is data, not adoption.** A redacted log is a smaller commitment than a pilot, it
arrives in one message, and — unlike a stated interest — the project can check it. Three
captures are wanted, and each closes a hole nothing done here can close:

| What | Why it cannot be produced here | What it fixes |
| --- | --- | --- |
| One Gemini CLI chat log carrying token counts | the maintainer's install writes none (`B110`) | `gemini-cli`'s calibration trace becomes `capture: real` (`B144`) |
| One Cline task directory | Cline is not installed here | `cline`'s calibration trace becomes `capture: real` (`B144`) |
| One vendor billing export | no real export has ever been read here | `reconcile`'s column aliases stop being a guess (`B192`) |

Stage 4 wants a fourth capture — one vLLM or DCGM exposition from a real deployment — through
the same door.

**How to send one.** A [GitHub Discussion](https://github.com/assaio/assaio/discussions) is the
door that exists today; the intake template that should exist is `B191`. Redact by the field
allowlist every checked-in trace was captured under — it is the only redaction procedure this
repository has, and each trace states it in its own `origin`
([example](internal/calibration/testdata/claude-code/session.adjudicated.json)):

- every number the tool wrote stays **verbatim** — the numbers are the entire point;
- every identifier — session id, message uuid, tool-call id — becomes a stand-in;
- every path becomes a stand-in;
- every body — prompt, response, file content — becomes the **same number of** placeholder
  lines, because a line count is one of the figures being calibrated;
- nothing the parser does not read is copied at all.

A billing export has a shorter allowlist, written down where the sample lives
([`internal/reconcile/testdata/README.md`](internal/reconcile/testdata/README.md)): date, model,
token count, amount, currency, and nothing else.

Two things a contributor should not have to ask. The file is **committed to a public,
Apache-2.0 repository**, so it is redacted for publication rather than for a private handoff.
And `assaio` transmits nothing, ever ([PRIVACY.md](PRIVACY.md)) — this is a file you send by
hand, not a setting you turn on.

**What comes back.** The trace is adjudicated by hand — counted by a second implementation that
never calls the parser — and the result is a per-figure derivation stating exactly what assaio
reads out of your tool's log, field by field. Your source stops being calibrated against a
sample in its own shape, which is the difference between *the parser reads this* and *the vendor
still writes this*. Credit in the release notes by name, or none, as you prefer.

**Decision this stage produces:** whether a figure assaio publishes is trustworthy on a machine
that is not the maintainer's. Two sources and the reconciler answer *unknown* today, and no
local work moves them.

**Exit:** three captures from three different people are checked in, `B144` and `B192` close,
and at least one of those people comes back a second time — a second capture, a bug, or a
`format-drift` report ([docs/format-resilience.md](docs/format-resilience.md)). One contribution
is a gift; a second from the same person is the first evidence of a *user*, which is what the
v1.0 gate is actually about.

**Kill criteria.** If **v0.30** arrives with no capture from anybody outside this repository and
`B193` has recorded where the ask was made, then asking again is not the answer: `B144` and
`B192` are closed as *not closable here*, the affected sources stay `constructed` and keep
saying so, and the effort returns to the half of the product that needs nobody. An open item
whose only possible input is a stranger's file is how a backlog starts lying about what is in
progress. The deadline is the same release the Runtime Insights gate is read at, and for the
same reason — see there for how the cadence picks it.

### 1. Trust reset — **shipped in v0.24**

Nothing is built on a surface known to be wrong.

- Member identity is pseudonymous at every export boundary, including the metric-plugin wire.
- Fourteen metrics gave up verdicts whose lines had no source; the figures stayed, and
  `session-taxonomy` gave up a favourable colour on a fact that has no better or worse.
- `recovery`'s baseline no longer contains the aftermath it is compared against.
- A parser fix reaches `ts`, `project`, `entrypoint`, `git_branch` and a step's `kind`, and a
  re-read that *lowers* a figure is counted and reported.
- `survival` states the age of what it measured.
- One compatibility policy, one supported toolchain, every CI action pinned.
- Positioning states what the vendors' own built-in stats do and do not do.

**Still open here:** `B185` (a line derived from your own history, which is what would earn the
withdrawn verdicts back) and `B186` (one message, two transcripts, two counts).

### 2. Analytics kernel — **thin, and stage 3 no longer waits on it**

Make new evidence domains possible without weakening what exists — and stop there.

- **Decided (`B104`, [ADR 0016](docs/adr/0016-usage-is-a-store-row-not-an-event.md)): AI usage is
  a store row, not an event.** `internal/event` is the observation contract for the domains that
  have no store row of their own — a commit today, a pull request, a review round and a check run
  next — and its unused AI half is deleted. One canonical model per fact, and one error posture:
  the parsers' skip-and-count.
- The rest of `B102` — retiring the store row types from `analyze.Input` — is a breaking change to
  the metric-plugin wire, so it **lands with the contract freeze (`B23`)**, not ahead of it.
  `analyze.Capability` and `Needs` shipped in v0.24 and are the seam that change moves through;
  nothing in stage 3 waits on it in the meantime.
- Correction lineage stays where corrections happen: on `usage_record`'s restatement path, which
  already counts the rows a re-read moved down. It is not an envelope field.
- Version signal definitions independently of SQLite migrations.

**Exit:** current reports are equivalent on the calibration corpus; a fixture plugin can add a
payload, a signal and a recommendation without importing internal packages; a missing capability
produces an explicit reason; a re-read that restates a stored figure is idempotent and reports what
it moved.

### 3. Assaio Usage outcomes and verified recommendations — **next, alongside stage 2**

The core product, and where most effort belongs.

- Content-free PR, CI, review, incident and deployment observations, through connectors chosen
  by user evidence rather than by what is easy to integrate.
- Prefer cost per accepted or surviving change over cost per raw generated line.
- The recommendation lifecycle: accept, run, verify, close (`ADR 0015` ships the record;
  the statuses beyond `proposed` need stored state and a baseline-versus-intervention
  comparison with drift checks).
- Measure recommendation precision — attempted, verified, harmful, inconclusive — and disable
  a family that keeps being wrong.
- A new coding or agent source only where a stable discoverable source and a real captured
  corpus exist. Cursor qualifies when someone contributes the corpus; it does not before.

**Exit:** every recommendation has evidence, rollback and a follow-up measurement; the product
can show an intervention helped, failed, or stayed inconclusive; joins publish their ambiguity
and their unmatched population; no individual ranking exists anywhere.

### 4. Runtime Insights feasibility gate — **experimental, shipped as a slice**

`runtime inspect` ships in v0.24 ([docs](docs/runtime-inspect.md)): one snapshot of vLLM and
DCGM endpoints, read-only, nothing stored, no cost model, no GPU advice.

**External validation gate — not yet run.** No design partner has used this. What has to be
true before anything further is built:

1. Three people operating their own models identify a **recurring decision** that their existing
   runtime dashboards and Assaio Usage do not answer separately.
2. A new user can inspect one vLLM and one DCGM deployment in under fifteen minutes.
3. Someone contributes a **captured** exposition from a real deployment; today's fixtures are
   constructed from vendor documentation and prove only that the parser reads the documented
   shape.

**Kill criteria, and the release they are read at.** If no repeatable decision emerges, or
nobody is willing to run it a second time, `runtime inspect` is **removed** rather than kept as
a feature nobody uses. Deleting it is the expected outcome of a gate that fails, not a failure
of the gate. The three conditions above are read when **v0.30** is prepared, and if they are not
all true then, the removal ships in that release. Six minors after the slice landed in v0.24,
read off the cadence in [CHANGELOG.md](CHANGELOG.md): twenty-six tags reached v0.24 in five
weeks, but the last four minors among them spanned ten days, so six more is on the order of two
months rather than two weeks — long enough for a stranger to find this, run it twice and say so,
and short enough that "still experimental" stops being an answer. This file promises no date for
anything that ships; a date for a *removal* is the opposite promise, and the one it can keep
alone.

### 5. Runtime Insights economics — **later, and only if stage 4 passes**

Amortized or rental infrastructure cost, energy, idle allocation, cost per million tokens and
per successful outcome; deployment comparison at an explicit quality target, concurrency and
latency SLO; bounded OpenMetrics ranges imported rather than Prometheus replaced.

Not started, and it does not start until the gate above passes.

### 6. Production team self-hosted — **later**

v0.24 authenticated every route, made member identity derivable from the token, bounded request
rates, and made the central store diagnosable (`doctor --db`, measured growth). That is
hardening, not production readiness.

What production means here: RBAC and token rotation; audit events; TLS deployment guidance;
chunked, resumable, idempotent sync with server-advertised limits; retention, downsampling and
deletion (`B173` is the decision this waits on); backup and a tested restore drill; cached or
materialized team views instead of rebuilding per request; a measured single-node envelope;
minimum cohort sizes on every team surface.

**Exit:** a documented threat model and a restore drill that has been run; one year of projected
data inside a bounded storage and query budget; an interrupted sync resumes without duplicates
or silent loss; an unauthorized caller can neither read a dashboard nor submit another member's
identity.

### 7. Team improvement and a shared evidence graph — **later**

Team-level enablement suggestions only where several observations support them; defect and
survival compared only against age-matched human code; joins through explicit request,
deployment, workload or artifact identity, never a guessed timestamp; interventions a team can
accept, run and close without exposing any individual.

### 8. Assaio Cloud — **later, and gated on infrastructure outside this repository**

The managed service reuses the team kernel and adds deployment adapters — identity, storage,
metering, billing, regional placement, support — rather than forking the analytics engine.

**Nothing of it is built.** A managed control plane, tenant isolation, metering and an
availability objective need infrastructure this repository does not contain, and no part of it
is claimed as shipped anywhere. The work that *does* belong here is the seam: keeping the
analytics and sync contracts free of deployment assumptions so the same conformance suite can
run against local, self-hosted and managed backends.

## What v1.0 has to mean

Not "every roadmap item is finished". `v1.0` means the claims and the public contracts are
dependable. It is declared when all of the following are true, and not before:

1. **The numbers are checked against something outside assaio.** Reconciliation against a
   vendor's own export, on a real corpus, with the unexplained remainder reported rather than
   closed (`B144` still needs a redacted capture).
2. **A wrong number fails visibly.** The calibration and drift suites catch the v0.12 class of
   defect rather than reporting green through it.
3. **A correction reaches history.** Every stored field a parser fix invalidates is rebuilt, or
   the surface says plainly which rows it cannot reach.
4. **Failure is visible.** Unsupported evidence renders `—`, an error or an unexplained-delta
   warning; never a zero and never a confident percentage.
5. **The contracts freeze**, as [docs/compatibility.md](docs/compatibility.md) defines them:
   exec protocols, the observation and signal contracts, the recommendation record, the sync
   protocol and the machine-readable outputs. The SQLite schema is deliberately **not** among
   them — it is an implementation detail with migration, export and backup guarantees, because
   v0.12 needed a migration that rewrote stored rows to fix a semantic error and it will not be
   the last.
6. **A threat model, a privacy data map and a deletion test** cover local and team modes.
7. **Upgrade and rollback are tested** from the oldest supported version, and release artifacts
   are reproducible, signed and scanned on a supported toolchain.

**External validation gate, and it has not been met:** at least three external teams running
the relevant surfaces across two release cycles. Twenty-six tags in the first five weeks
demonstrate execution; they demonstrate nothing about whether anyone repeatedly makes a better
decision with this. No design-partner result, case study or production deployment exists, and
none is claimed. **Stage 0 is the plan for it** — it asks for data rather than adoption, because
a redacted log is a smaller commitment than a pilot and is something this project can check.

Runtime Insights is **not** a v1 requirement. It joins the supported surface only if its demand
gate passes and the maintainers then choose to add its contracts.

## Non-goals

Not "not yet" — **not, regardless of demand**:

- An inference proxy, model router or gateway.
- A general trace store, metrics backend, or a replacement for Prometheus, OpenTelemetry,
  MLflow, Langfuse or Phoenix.
- A training or model-serving orchestrator.
- A GPU dashboard or a runtime control plane.
- A public model leaderboard, or any employee performance system.
- An autonomous optimizer that changes a production configuration.
- An "estimated time saved" headline, or any invented productivity value — the logs contain no
  counterfactual.
- Per-named-individual leaderboards of lines or tokens, per-person analytics outside a governed
  team opt-in, or cohort comparisons without a minimum cohort size and consent.
- Reading prompts, responses, source code or diffs. Content-free is the default and changing it
  would be a separate security decision with its own threat model, not a flag.

## Principles that do not change

- **Every figure carries its provenance and its confidence.** A signal labelled directional
  beats a precise-looking number that is wrong.
- **A verdict needs a line, and a line needs an authority** — derived from your own data, cited
  from a published definition, or set by you. Anything else reports the figure and refuses the
  grade.
- **Absence is never zero.** A source that records nothing contributes no zero to a rate.
- **Bug density on AI lines is compared only against age-matched human code.**
- **Pseudonymized and aggregated by default**, everywhere, including on the way out.
- **Offline by default.** Two surfaces can use a network, and only when invoked: the team
  server you run (`serve`/`sync`), and the experimental `runtime inspect`, which reads an
  endpoint you name. Nothing else ever does.

## How we prioritize

For every proposed source, metric or recommendation, all five:

1. a named user decision it changes;
2. a real sample or a design partner;
3. a provenance and confidence contract;
4. an expected storage and cardinality cost;
5. a testable exit or kill criterion.

Stop or narrow a workstream when users do not act on the finding, when the join cannot be made
honestly, when collection costs more than the decision is worth, or when an established
integration solves it with less ownership.

Stars, connector count, token volume, generated lines and release count are supporting signals.
They are not product outcomes.
