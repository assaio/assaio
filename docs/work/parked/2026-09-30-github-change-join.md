# B92 + B100 · GitHub pull-request observations and the session→change join

Work file for one task: spec, plan, journal and handoff in one place. Deleted by `/ship`.

## Goal

`evidence` stops at commits: it says which local commits sit near a session, not whether that
work reached a pull request, was reviewed, passed CI or merged. ROADMAP milestone 2 asks for
exactly that next step — "extend the shipped local session→commit slice to shipped changes" with
GitHub as the first connector — and `B100` says its privacy policy ships with the connector, not
after it.

After this change, `evidence --github` asks the user's own `gh` for the repository's pull
requests, content-free, and joins each candidate commit to the pull request(s) that carry it
(as a branch commit or as the squash/merge commit). Each candidate shows its change's state,
review-state counts and check rollup; the summary states how much of the population reached a
change and how much reached a merged one, over the same denominator. Nothing is stored, synced
or shared, and no identity is fetched.

Layer: the join is an **outcome-layer observation** (a merge, a review, a check happened), not a
metric. No rate, score or comparison is derived here (`B94` owns the funnel, and only over
comparable populations). A session with no candidate, or a candidate commit with no observed
change, stays unmatched and is counted; absence is never "not merged".

## Open questions (with the default the work proceeds on)

- Q1: How does assaio reach GitHub? — default: **by executing the user's `gh`** with one
  constant, content-free GraphQL query (D1). assaio never reads, stores or passes a token.
- Q2: Is the network call opt-in? — default: **yes, per invocation**: only `evidence --github`
  runs `gh`. Without the flag `evidence` is byte-identical to today. PRIVACY.md names it as the
  third network exception beside `sync` and `serve`.
- Q3: Is anything stored? — default: **no**. Like commits (ADR 0009/0018), pull-request
  observations and session→change links are recomputed in one pass. No migration.
- Q4: Revert relations (B92 lists them)? — default: **deferred**. Detecting a revert PR needs its
  title; the commit-level revert flag already exists. Kept on the backlog item.
- Q5: An import path for other forges? — default: **contract now, importer later**. The payload
  is the importable contract (ADR 0007 envelope); a `--changes <file>` importer for GitLab/GHES
  exports is a follow-up item once a real export exists.
- Q6: Enterprise Server? — default: through `gh`'s own host resolution for the repository's
  remote; stated as untested.

## Decisions

- D1 (fetch): **A — exec `gh api graphql`**, the way `internal/vcs` execs `git`: the query is a
  compile-time constant that names only allowlisted fields; `gh` owns authentication (Cloud and
  Enterprise), assaio passes no token on argv or env, output is size-bounded, the call has a
  timeout, and a missing or unauthenticated `gh` is an error naming the fix.
  - B — an HTTPS client in the binary with a token: credential discovery, storage and redaction
    become assaio's problem ("credentials never reach reports or plugins" gets a code path to
    get wrong), plus TLS, retries and pagination code; rejected.
  - C — import-only (user runs a `gh` recipe, assaio reads the file): keeps the binary off the
    network, but moves the content boundary to a file the user writes with whatever fields they
    asked for, and every user writes a recipe; kept as the future importer (Q5), not the v1 path.
- D2 (contract): `scm.pull_request.observed` (the name ADR 0007 reserved), grain `change`, id
  = `<host>/<owner>/<repo>#<number>`, privacy `local-only`. Payload: number, state
  (open/merged/closed), draft, created/merged/closed times, merge commit, commit oids (with a
  truncation flag past the page the query reads), review-state counts (approved,
  changes-requested, commented, dismissed), check rollup of the head commit (state + context
  count). No title, body, branch name, author, reviewer, label, comment or check name — the
  query cannot ask for them and a contract test fails if a field is added that could carry one.
- D3 (window): pull requests updated at or after `--since`, newest first, paginated until older;
  a PR that carries a candidate commit but was last updated before the window is not observed,
  and the summary says how many candidate commits found no change (never "not merged").
- D4 (join): a candidate commit links to every observed PR whose commit list contains it or whose
  merge commit it is (squash and merge-commit flows). Many-to-many stays many-to-many. Per result:
  the linked changes with number, state, times, review counts, check rollup, and how the commit
  links (`branch-commit` / `merge-commit`). Summary: `changeCovered` (population sessions with at
  least one candidate linked to a change), `changeMerged` (… to a merged change), and the counts
  of pull requests observed and truncated. Denominator = the existing population.
- D5 (privacy, B100): ADR 0020. Local-only: never stored, synced, exported, shared, sent to a
  plugin or shown on a dashboard; `evidence` keeps no `--db`. No person is fetched, so no person
  can be ranked; a structural test asserts no author/member/login/reviewer/rank/score field on
  the payload or the evidence document. Retention: none. Server views of pull-request data or
  edges: none exist; any future one needs its own ADR, a keyed digest instead of numbers and
  hashes (`B103`), and a minimum cohort; it never shows one member's changes. Re-identification:
  PR numbers, commit hashes and times identify work, which is why the class is `local-only`.
- decided alone: flag name `--github` — reverse by renaming before release.
- decided alone: package `internal/github` (one source, one package) — ADR 0020 names it.

## Blast radius (each box is a completion criterion)

- [ ] `internal/event`: payload + type + validation + content-free contract test
- [ ] `internal/github`: query constant, gh runner (injectable), parser, pagination, bounds;
      golden from a real capture of this public repository; fuzz on the parser
- [ ] `internal/attribution`: change links on results, summary fields, render
- [ ] `internal/cli/evidence.go`: `--github`, errors, notes
- [ ] tests: squash and branch-commit links, many-to-many, truncated commits, window edge,
      gh missing / unauthenticated / malformed output, no identity field anywhere
- [ ] ADR 0020 (+ ADR 0018 status note, docs/README index)
- [ ] PRIVACY.md (network exception, fields), docs/threat-model.md (correlation), docs/evidence.md
- [ ] CHANGELOG (Added), FEATURES (evidence row), BACKLOG (B100 and B92 deleted or narrowed;
      follow-ups: importer, revert relations), ROADMAP §2 line, README/llms.txt/site evidence lines
- [ ] `make docs`; gate full; reviewers; mutation check

## Tracks

| # | Files | Behaviour change | Tests | Verify | Executor | Depends on |
|---|---|---|---|---|---|---|
| 1 | `internal/event/payload_scm.go` + test | new payload | validate, content-free | `go test ./internal/event/` | this session | — |
| 2 | `internal/github/*` + testdata | collector | golden, pagination, errors, fuzz | `go test ./internal/github/` | this session | 1 |
| 3 | `internal/attribution/{changes,result,render}.go` + tests | join + summary | link kinds, coverage | `go test ./internal/attribution/` | this session | 1 |
| 4 | `internal/cli/evidence.go` + tests | `--github` | fake gh runner | `go test ./internal/cli/ -run Evidence` | this session | 2, 3 |
| 5 | docs + ADR + surfaces | — | `make docs`, `make test` | gate full | this session + GPT door | 1–4 |

## Journal

- 2026-09-30: premise checked: `event.TypeCommit` is the only type; `scm.pull_request` reserved by
  name (`internal/event/payload.go:11`). Probe of this repository's merged PRs through
  `gh api graphql` returned state, times, merge commit, commit oids, review states and a check
  rollup with no content field requested. Squash merges mean the commit on `main` is the PR's
  merge commit, not its branch commits — the join needs both.

- 2026-09-30: `associatedPullRequests` per commit works for squash and branch commits (null for
  an unpushed commit), but querying by candidate oid tells GitHub which commits AI sessions
  touched — rejected on privacy grounds (honesty-auditor H5). Listing PRs by repository and
  window stays the request shape.
- 2026-09-30: challengers (go-reviewer G1–G12, honesty-auditor H1–H9). The design does not hold as
  written:
  - G1/H2: `evidence` candidates are commits reachable from HEAD matched by committer time. On a
    squash or rebase repository the commit on `main` carries the merge time, so the session that
    did the work days earlier links to nothing and whichever session overlapped the merge click
    (or a teammate's) gets a medium-confidence link to the whole PR. Joining PRs onto those
    candidates amplifies a mis-credit that already exists in session-commit/v1. Fix needs a new
    candidate population (the PR's branch commits present locally, with their own times), a new
    algorithm version, and squash/rebase/merge/stacked scenarios in the ADR 0010 corpus first.
  - H1/G8: `changeMerged` tracks the checked-out branch, not the work; it reads as B94's rate.
  - H3/G2: "no change" has at least six causes; absence vs zero in the frozen JSON needs a nested
    object that is nil without `--github`.
  - G3: repository and host must come from `gh repo view`, never from parsing the remote (forks,
    SSH aliases, `GH_REPO`, credentials in URLs).
  - H4: B100 cannot be closed by this: `sync` already ships branch names that join to PR authors;
    a PR number is one lookup from its author; a user-owned repo's id contains a login.
  - G6/H8: review counts and the check rollup pre-empt `scm.review`/`ci.check` reserved by ADR
    0007; cut them or amend 0007. The payload cannot hold a slice of oids under the contract test.
  - Network surfaces (PRIVACY, threat model, evidence help and footer, ADR 0018, cli rule) and
    limits (timeout, output cap, helper-process tests for Windows, Makefile fuzz list) listed.
- 2026-09-30: the maintainer chose the order B100 → corpus scenarios → B92. Parked here.

## Handoff

- Done: plan and challenge; no code.
- Waits for: (1) B100 alone (ADR 0020, threat model, PRIVACY residual risks); (2) squash, rebase,
  merge and stacked scenarios in the ADR 0010 corpus, with session-commit/v1 measured on them.
- Then: rebuild this plan with the PR's locally-present branch commits as candidates, a new
  algorithm version, the nested `changes` object that is nil without `--github`, repository and
  host from `gh repo view`, no merge rate, review/check left to their reserved types.
