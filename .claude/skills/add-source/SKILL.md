---
name: add-source
description: Checklist for adding a data source (a new parser under internal/parser/) or a metric validator (internal/analyze/) with every trap this repo has hit — the same-commit surfaces, the fuzz and golden contract, the capability matrix, the published counts. Use "new parser", "support tool X", "add a validator", "new metric", "connector".
argument-hint: "<source|metric> <name>"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

Kind and name: $ARGUMENTS

The walkthroughs are authoritative — read the one you need before writing code:
`docs/extending/data-source.md` (parser) or `docs/extending/metric-validator.md` (validator,
with `metric-validator-example.md`). This skill is the list of what those pages cannot enforce.

## Before starting (a source)

- `ROADMAP.md` gates a new in-tree parser: a stable discoverable source, a real redacted
  corpus and a user who will rerun it. Without all three, propose an exec plugin
  (`docs/extending/parser-plugin.md`) instead — no fork, any language.
- A connector issue first (`.github/ISSUE_TEMPLATE/connector.yml`), per the intake path.

## The same-commit surfaces (each has been missed before)

| Surface | What changes |
|---|---|
| `PRIVACY.md` | directories opened, fields extracted — also when the source reads *less* |
| `README.md` sources table, `site/llms.txt` | the source named with its limit |
| `AGENTS.md` layout, `CITATION.cff` abstract | the package; the "across …" tool list (`consistency.yml`) |
| `internal/pricing/litellm.json` | every model the source names has a price, or the unpriced share widens |
| `docs/extending/source-fields.md` | the capability matrix row (ADR 0011): what the source can answer |
| `FEATURES.md`, `CHANGELOG.md`, `BACKLOG.md` | the lifecycle (`.claude/rules/surfaces.md`) |
| `Makefile` `fuzz` target, `fuzz.yml` | the new `FuzzParse` (the workflow diffs discovered fuzzers against the recipe) |
| `doctor` | the source detected, its canaries (`internal/drift`) |

`make docs` regenerates the reference and the site's source count in both directions.

## The contract (parser)

Single-root `Discover`; skip-and-count on corrupt lines; shared scanner with `MaxLineBytes`;
`NonNeg`; deterministic `DedupeKey`; hermetic (no git, no network); golden files from a real
redacted capture; `FuzzParse` with a seed corpus; granularity and tier declared honestly
(ADR 0017: a source without a token counter is `activity-only`, never zeros).

## The contract (validator)

One file, self-registered; reads only the sources that record its field (ADR 0011); states its
layer (ADR 0013); `—` for absence; a threshold with a source or derived from the window; what
its pattern cannot be told apart from; pseudonymized bars. It appears in the CLI, the dashboard
and the reference automatically — check all three, then `/ui-check`.

Done means: gate full green, `corpus-prover` on the real corpus (or the honest statement that
this machine holds no such log — `B144`), `/review`, and the table above ticked in the work file.
