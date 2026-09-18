---
paths:
  - ".claude/**"
---

# The harness itself

- **Every file here is a fixed cost or a guard.** Skill and agent descriptions load into every
  session (~250–400 chars each, with the phrases that should trigger them); bodies load on use.
  `AGENTS.md` loads everywhere, including subagents: 150 lines is the budget, checked.
- **Agents name a model: `haiku`, `sonnet` or `opus`**, with `effort`; never `inherit` (the session
  may be running Fable, which is never a working model) and never a Fable/Mythos id — the hook
  denies the assignment, `setup.py` re-checks.
- **Frontmatter pitfalls**: quote `argument-hint` when it holds `[`; no `: ` inside an unquoted
  value; rules always carry `paths:` (a rule without them loads always) and each glob must hit a
  tracked file. `python3 .claude/checks/setup.py` is the check; `checks_test.sh` its negative control.
- **Hooks fail open** (internal error → exit 0), deny only the irreversible, ask for what spends
  money or publishes. Every new case goes into `guard_test.sh` with its negative twin.
- **Cited paths must exist**; `setup.py` reads every backticked path in `AGENTS.md`, the rules,
  skills, agents and `docs/operations.md`.
- Local rules only tighten: nothing here may relax a safety rule from `~/.claude/CLAUDE.md`
  (model family, attribution, `claude -p` in loops) or from `CONTRIBUTING.md`.
- Commit `.claude/` changes with the rule they implement; `settings.local.json` stays untracked.
