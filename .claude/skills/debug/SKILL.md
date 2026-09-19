---
name: debug
description: Root-causes a wrong figure, a failing test, a parser drift or a crash in this repo with the traps that have bitten here — double counting, wrong denominators, windows one day too wide, golden files that hid a bug, a store already migrated. Use "why is this number wrong", "test fails", "doctor reports drift", "reproduce this".
argument-hint: "<symptom>"
allowed-tools: Read, Grep, Glob, Bash, Agent
---

Symptom: $ARGUMENTS

## Reproduce before reasoning

- A wrong figure: pin the window (`--since`/`--until` ending before today, never `7d` — the
  corpus grows while you look), build the binary, run against a copy of the store
  (`XDG_DATA_HOME=$(mktemp -d)`, then `backfill`) and capture the exact output. Compare
  `analyze --format json` with the dashboard and the report: the same number rendered
  differently is a renderer bug, a different number is a derivation bug.
- A failing test: run it alone with `-run`, read the golden diff in full, and ask which of the
  two sides is the truth before touching either. Golden files have hidden a 2× parser error for
  eleven releases (`docs/corrections.md`).
- A parser drift: `doctor --strict` names the canary; find the newest transcript that trips
  it, cut a minimal redacted sample, and add it to `testdata/` as the reproduction first.

## The usual suspects here

- counted once per content block instead of once per response;
- a unit chosen before rounding; a share totalled in the wrong dimension; a rate above 100%;
- a window one day wider than its name; timezone at the day boundary;
- a denominator spanning main-loop, sub-agent and SDK sessions at once (`internal/trace`);
- a path that breaks on non-ASCII or a worktree collapsing to `..` (`internal/projectid`);
- a shipped migration edited instead of a new one — an upgraded store skips it silently;
- a restate path that takes `MAX` and cannot lower a figure (`B116`);
- a price missing from `internal/pricing/litellm.json` rounding to zero instead of widening
  the unpriced share.

## Fix and prove

Write the failing test first (table row or golden sample), fix, then `corpus-prover` shows the
figure moved on the real corpus by the expected amount and nothing else did. A correction of a
figure an earlier release published gets its `docs/corrections.md` entry and changelog line
together (`RELEASING.md`, "The changelog flow").
