# Roadmap

This is a direction with exit criteria, not a delivery calendar. [FEATURES.md](FEATURES.md)
records what exists, [CHANGELOG.md](CHANGELOG.md) records what shipped, and
[BACKLOG.md](BACKLOG.md) holds implementation candidates. Public compatibility is defined
only in [docs/compatibility.md](docs/compatibility.md).

## Product decision

`assaio` will be the open, privacy-first evidence layer for AI-assisted software
engineering: it observes how coding agents are used, joins that evidence to the changes
they produce, and helps a team verify whether an intervention improved a durable outcome.

The project will not compete as another quota bar, token dashboard or general engineering
intelligence suite. Usage and cost remain the required first layer, not the destination.
The next release line prioritizes one chain:

```text
agent session -> change -> pull request -> review and CI -> durable outcome -> verified experiment
```

Each edge must publish its provenance, coverage, ambiguity and unmatched population. If an
edge cannot be defended, the product shows the two observations separately.

## What the market evidence changes

The evidence is unusually consistent about the measurement problem even when estimates of
AI's benefit disagree:

| Evidence | Product implication |
| --- | --- |
| GitHub, Anthropic, OpenAI and Cursor already expose adoption, usage, spend, accepted-line or contribution analytics in their own admin products.[^1][^2][^3][^4] | Vendor-specific usage reporting is a baseline feature. Rebuilding it with fewer first-party signals is not a defensible centre. |
| A 2026 survey of 636 engineering professionals found a fragmented tool market, rising cost concern and materially different outcomes between ad-hoc and highly enabled adopters.[^5] | Cross-tool normalization matters, but the valuable question is which operating practice works, not which vendor has more activity. |
| DORA's study of nearly 5,000 professionals describes AI as an amplifier: throughput can improve while delivery stability worsens, and small batches, fast feedback and strong version control determine whether speed becomes value.[^6] | Lines and tokens are output signals. The engine must connect them to review, CI, batch size and stability before making outcome claims. |
| DX observed AI-tool use rise 65% across 400+ organizations while median PR throughput rose 7.76%; Swarmia observed median PR batch size roughly double across 1,450+ organizations.[^7][^8] | Large individual gains do not imply system-level gains. Baselines need matched windows and guardrails for review load and change risk. |
| METR found a 19% slowdown for experienced open-source developers using early-2025 tools. Its 2026 follow-up suggested later tools may help, but selection effects made the size unreliable.[^9][^10] | A universal ROI coefficient or "hours saved" headline would be invented. Assaio should measure a team's own intervention and retain an inconclusive result. |
| Open-source tools such as `ccusage` already make local token and cost reporting easy and widely available.[^11] | The local store wins on reproducibility, multi-tool provenance and outcome linkage—not on another spend chart. |

These sources include vendor and platform-provider research, so their estimates are not
treated as interchangeable causal evidence. They support the direction; they do not supply
a threshold for grading an individual team.

## Primary user and decision

The primary user is an engineering-enablement or platform lead responsible for a team using
more than one coding agent. Their recurring decision is:

> Which workflow, model or enablement change should we keep, expand, revise or stop, given
> its cost, review burden, delivery result and data quality?

The local developer remains the adoption wedge: one binary, no account, no prompt capture,
and useful evidence before any repository credential or server is introduced. The team view
exists to pool that evidence and run governed experiments, never to evaluate employees.

The product loop is:

1. **Observe** activity, cost and output without content.
2. **Join** sessions to repository outcomes with explicit uncertainty.
3. **Compare** matched windows or cohorts rather than global benchmarks.
4. **Act** through a reversible recommendation.
5. **Verify** the result and keep failed or inconclusive experiments visible.

## Engine direction

One engine serves local and team modes. Its boundaries stay narrow:

- **Observations:** immutable, versioned facts with source, time, subject, provenance and
  confidence. Usage remains the canonical store row; other domains use `internal/event`.
- **Attribution:** deterministic links with a method, score or ambiguity class. No
  timestamp-only join is silently promoted to identity.
- **Signals:** versioned definitions separate from schema migrations. Every denominator
  declares which source capabilities it needs.
- **Evaluations:** comparisons with baseline, intervention, guardrails and a comparability
  verdict. Absence remains different from zero.
- **Recommendations:** evidence, risk, rollback and follow-up, plus lifecycle state and the
  measured result.
- **Connectors:** content-free adapters at the edge. The core consumes contracts and never
  imports a vendor package.

SQLite remains the local implementation, not a public API. Exec parser, metric and rule
protocols remain the supported extension boundary until the v1 contract freeze.

## The next milestones

Milestones are ordered by dependency. Work may overlap only when it does not weaken the
earlier exit criteria.

### 1. Trust and activation

Make the current local product easy to evaluate and hard to misunderstand.

- Keep one documented first-run path: `demo`, `init`, `dashboard`, then `digest`.
- Keep source coverage, parser version, restatements and unpriced share visible.
- Calibrate Gemini CLI, Cline and reconciliation against contributed real captures.
- Add SAST, keep dependency and fuzzing automation quiet enough that alerts retain meaning.
- Stop adding validators that do not change a named decision.

**Exit:** three external teams complete a first report; at least two return for a later
digest or provide a second capture; no known wrong figure is presented as complete; the
first-run path is tested on a clean machine.

### 2. Evidence graph: session to shipped change

Build the differentiating outcome path before expanding the UI or connector count.

- Add content-free commit, pull-request, review and check-run observations, with GitHub as
  the first end-to-end connector and an importable contract for other forges.
- Link a session to a change only through evidence the attribution corpus can grade.
- Report coverage and competing candidates for every join.
- Add outcome signals only when their populations are comparable: merged changes, review
  rounds, CI result, batch size and survival at a fixed age.
- Prefer cost per merged or surviving change over cost per generated line.

**Exit:** a local report can account for matched and unmatched sessions and changes; a
known-ambiguous corpus stays ambiguous; re-import is idempotent; no prompt, code, diff or PR
body is stored.

### 3. Verified experiments

Turn diagnostics into learning rather than permanent advice.

- Persist recommendation states: proposed, accepted, running, verified, inconclusive,
  harmful, rolled back and closed.
- Compare an intervention with a declared baseline and guardrails for quality, review load
  and delivery stability.
- Detect history, parser and population drift before comparing windows.
- Track precision by recommendation family and disable a family that repeatedly fails.
- Support team-defined thresholds without presenting them as universal research findings.

**Exit:** every actionable recommendation has a rollback and follow-up; the product can show
that an intervention helped, hurt or remained inconclusive; rerunning the same evidence gives
the same answer.

### 4. Production team pilot

Harden `serve` and `sync` only alongside design partners who need the shared loop.

- RBAC, scoped and rotatable credentials, audit events and minimum cohort sizes.
- Chunked resumable sync with server-advertised limits and idempotency.
- Retention, deletion, backup and a restore drill.
- Cached team views and a measured single-node storage/query envelope.
- TLS deployment guidance and documented failure/recovery procedures.

**Exit:** at least three external teams use the relevant surfaces across two releases; an
interrupted sync resumes without loss or duplication; a restore drill succeeds; an
unauthorized caller cannot read a report or submit another member's identity.

### 5. v1 contract and operations freeze

- Freeze the machine-readable outputs, observation and signal contracts, exec protocols,
  sync protocol and recommendation record as defined in `docs/compatibility.md`.
- Test upgrades and rollback from the oldest supported version.
- Reconcile a real vendor export and retain unexplained deltas.
- Keep release artifacts reproducible, attested, scanned and built on a supported toolchain.
- Complete the local and team threat model, privacy map and deletion tests.

**Exit:** the claims and public contracts are dependable, not merely feature-rich. Runtime
Insights is not a v1 requirement.

## Explicit gates and deferrals

| Workstream | Decision |
| --- | --- |
| More local token dashboards or visualizations | Defer unless a design partner names a decision the existing dashboard cannot support. |
| New coding-tool parsers | Require a stable discoverable source, a real redacted corpus and a user who will rerun it. Prefer out-of-tree plugins before in-tree ownership. |
| `runtime inspect` beyond its current slice | At v0.30, remove it unless three self-hosted-model operators identify the same recurring decision, one contributes a real exposition and one returns for a second use. |
| Production team server | Starts with milestone 4 design partners, not in anticipation of them. |
| Managed cloud | Starts only after the self-hosted team loop is repeatable and its operating envelope is measured. |
| In-process Go plugin API | Waits for the v1 contract freeze; exec protocols remain the stable path. |

## What v1.0 has to mean

The readiness scorecard below is the compact form of milestone 5. `v1.0` is a promise
about dependable claims and contracts, not the point at which every backlog item ships.

The release decision should use these gates, not feature count:

| Surface | Current posture | Ready when |
| --- | --- | --- |
| Local usage and cost | pilot-ready, pre-1.0 | external captures confirm source shapes and first-run activation repeats |
| Local output diagnostics | directional and useful | outcome language never leaks into output-only metrics |
| Evidence graph | not shipped | attribution and unmatched populations pass conformance corpora |
| Recommendations | proposed-only | interventions can be accepted, measured and closed |
| Team server | MVP | milestone 4 security, recovery and scale gates pass |
| Managed cloud | not built | repeated self-hosted demand justifies operating it |

Product success is a completed decision loop, not stars, releases, connector count, tokens or
generated lines. Before v1, the useful adoption signals are clean first-run completion, a
second digest, join coverage, experiments closed with a result, and returning design partners.
Collect them through explicit pilot feedback; the offline agent does not gain telemetry.

## Non-goals and invariants

- No employee performance system or named-person output/spend leaderboard.
- No estimated time-saved or universal ROI headline.
- No bug-density comparison without age-matched human code.
- No prompt, response, source-code or diff collection in the default product.
- No general trace backend, inference gateway, model router, GPU dashboard or Prometheus
  replacement.
- No autonomous production optimizer.
- Every fact carries provenance and confidence; absence is never zero.
- Pseudonymized and aggregated is the default at every export boundary.
- Local analysis remains offline-capable and independent of a commercial service.

## Sources

[^1]: GitHub, “[GitHub Copilot usage metrics](https://docs.github.com/en/copilot/concepts/copilot-usage-metrics/copilot-metrics),” accessed September 2026.
[^2]: Anthropic, “[Claude Code usage analytics](https://support.claude.com/en/articles/12157520-claude-code-usage-analytics),” July 2026.
[^3]: Cursor, “[Analytics](https://docs.cursor.com/account/teams/analytics)” and “[Admin API](https://docs.cursor.com/en/account/teams/admin-api),” accessed September 2026.
[^4]: OpenAI, “[New usage analytics and updated spend controls for enterprises](https://openai.com/index/chatgpt-enterprise-spend-controls/),” June 2026.
[^5]: Jellyfish, “[2026 State of Engineering Management Report](https://jellyfish.co/2026-state-of-engineering-management/),” 636 respondents, March 2026 survey.
[^6]: DORA, “[State of AI-assisted Software Development 2025](https://dora.dev/research/2025/dora-report/),” based on survey responses from nearly 5,000 technology professionals.
[^7]: DX, “[AI and engineering velocity: A longitudinal analysis](https://getdx.com/report/ai-and-engineering-velocity-a-longitudinal-analysis/),” 400+ organizations, November 2024–February 2026.
[^8]: Swarmia, “[Measuring the productivity impact of AI coding tools](https://www.swarmia.com/blog/productivity-impact-of-ai-coding-tools/),” 1,450+ organizations, June 2026.
[^9]: METR, “[Measuring the Impact of Early-2025 AI on Experienced Open-Source Developer Productivity](https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/),” randomized controlled trial, July 2025.
[^10]: METR, “[We are Changing our Developer Productivity Experiment Design](https://metr.org/blog/2026-02-24-uplift-update/),” February 2026.
[^11]: ccusage, “[ccusage](https://github.com/ccusage/ccusage),” open-source local usage analysis for coding agents.
