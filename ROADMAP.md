# Roadmap

**Direction, not commitment.** No dates and no guarantee any candidate here ships as described
or ships at all; the order below is the intended sequence, not a schedule. `assaio` is pre-1.0 and the most useful input to what comes next
is feedback from people running it against their own repositories and teams.

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

Ordered. A stage may overlap the next only once its exit criteria are **measured**, not
assumed.

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

### 2. Analytics kernel — **next**, and the one stage 3 waits on where they touch

Make new evidence domains possible without weakening what exists.

- `analyze.Capability` and `Needs` shipped in v0.24; four validators declare. Retiring the
  store row types from `analyze.Input` is the rest of `B102` and is a breaking change to the
  metric-plugin wire, so it lands with the contract freeze.
- Decide `internal/event`'s future: make the observation contract the persisted, correction-
  aware ingestion spine, or delete the unused half. Two canonical models is the thing not to
  keep (`B104`).
- Correction lineage on the envelope; a typed delivery-outcome payload.
- Version signal definitions independently of SQLite migrations.

**Exit:** current reports are equivalent on the calibration corpus; a fixture plugin can add a
payload, a signal and a recommendation without importing internal packages; a missing capability
produces an explicit reason; corrections are idempotent and keep their lineage.

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

**Kill criteria.** If no repeatable decision emerges, or nobody is willing to run it a second
time, `runtime inspect` is **removed** rather than kept as a feature nobody uses. Deleting it is
the expected outcome of a gate that fails, not a failure of the gate.

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
none is claimed.

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
