---
name: ui-check
description: Checks the offline dashboard and the site in a real browser through the browser agent — every section rendered, empty and unmeasured states as dashes, zero console and network activity, 400 px, keyboard, both themes — with screenshots kept as evidence outside the main context. Use after any dashboard, i18n, validator-text or site change, "check the dashboard", "look at the page".
argument-hint: "[dashboard|site|<html file>]"
allowed-tools: Bash, Read, Glob, Agent
---

Target: $ARGUMENTS (empty = both).

## Produce the artifact first

- Dashboard on a throwaway store with the bundled sample, never the real one:
  `XDG_DATA_HOME=$(mktemp -d) go run ./cmd/assaio-agent demo` prints the reports; for the file,
  `XDG_DATA_HOME=$(mktemp -d) sh -c 'go run ./cmd/assaio-agent backfill >/dev/null 2>&1; go run ./cmd/assaio-agent dashboard --since 90d --output "$0"' "$PWD/assaio-dashboard.html"`
  on a store filled from the real logs when the change concerns real data (the file is
  gitignored). An empty store is also a state to check: the page must say unmeasured, not zero.
- Site: `site/index.html` and `site/docs/*.html` straight from disk after `make docs`.

## Delegate to `browser`

Playwright refuses `file://`: tell the agent to serve the directory over localhost
(`python3 -m http.server`) and to ignore the favicon 404 that causes; one artifact per run.

Give it the file paths, the sections the change touched, and the checklist it carries: every
section present, `—`/unmeasured for absence, no console errors, no network request, 400 px and
default width, keyboard focus on the toggle and links, both themes, the faceplate scrolled into
view before its screenshot. Ask for a pass/fail table with the exact observation and the
screenshot paths.

## Read the return

Screenshots stay where the agent put them; open only the ones a failing row points at. A
failing row is a finding for `/build` or `/debug`; a passing table plus the screenshot paths
goes into the work file's Journal as QA evidence and into the PR description when the change
is user-facing. Nothing here edits the page.
