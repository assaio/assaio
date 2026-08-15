# `.claude/` — the harness config for this repo

Checked in, because a release can change it and because the rules below are the repo's, not
one machine's. `settings.local.json` stays untracked for personal overrides.

`AGENTS.md` says *what* the rules are; this directory is the part of them a harness can
execute. Nothing here restates guidance a capable model already follows — each file exists
because something in this project failed without it.

## settings.json

- **`attribution: {commit: "", pr: ""}`** — turns "never credit an AI assistant as an author"
  (`CONTRIBUTING.md` rule 4) from prose into the harness default. The `dco` CI job rejects
  those trailers; this stops them being written in the first place.
- **`permissions.ask`** on `git commit`, `git push`, `git tag`, `make release*`, `gh pr
  create|merge`, `gh release` — commits are made by hand after a milestone, so every one of
  them is a deliberate keystroke.
- **`permissions.allow`** on the read-only and verify commands (`make test|lint|fmt|docs`,
  `go test|vet|build|list`, read-only `git` and `gh`) — friction on the gate is friction on
  running the gate.
- **No `permissions.deny`.** It carried `Bash(assaio-agent clear:*)`, which reads as protection
  for the most destructive command here and is not: the prefix never matches
  `go run ./cmd/assaio-agent clear`, which is how the command is actually run from this tree,
  and it does match the *correct* redirected form no better than the wrong one. The hook below
  matches every spelling, distinguishes the safe invocation from the unsafe one, and says what
  to type instead. Two mechanisms for one rule is one that can drift; the weaker one went.

## hooks/guard.sh

One `PreToolUse(Bash)` hook. It denies four things and nothing else:

| Denied | Why it is not just a prompt |
|---|---|
| `assaio clear` without a redirected `XDG_DATA_HOME` | The real store is 170 MB and holds days the sources themselves have already deleted. There is no `--db` to point elsewhere. |
| `git tag -d/-f`, deleting or force-pushing a tag or `main` | Published tags are immutable by repository ruleset; a bad release is fixed by the next one. |
| A commit message carrying an AI-authorship trailer | The `commit-msg` hook is opt-in (`make hooks`); this is not. |
| `git commit --no-verify` | It skips the Conventional-Commits and `Signed-off-by` check. |

`bash .claude/hooks/guard_test.sh` runs its 27 cases, allow and deny alike. Not a CI gate — a
guard that silently stopped matching looks exactly like a guard that was never needed.

## agents/

Each one holds a line that is otherwise nobody's job, and knows a failure this repo has
actually shipped.

| Agent | Holds |
|---|---|
| `honesty-auditor` | Layer, provenance, confidence, scope denominators, the refusals. The v0.12 class of defect. |
| `go-reviewer` | The norms `.golangci.yml` omits on purpose — file size, one responsibility, comment policy, the parser contract. |
| `surface-auditor` | Whether the published *prose* still describes this binary. The mechanical half is already a test (`B161`). |
| `store-steward` | Migration immutability, and a size bound measured in bytes on a real corpus rather than in rows. |
| `corpus-prover` | Proof on real logs that the figure which was supposed to move, moved — and nothing else did. |
| `release-captain` | `RELEASING.md` end to end, stopping before the tag. |

## commands/

- `/gate` — the full local gate, reporting only what failed.
- `/release [patch|minor|vX.Y.Z]` — prepare a release, stop at the tag command.
- `/selfreview [diff|repo|<path>]` — the four reviewers in parallel, findings merged and
  ranked, nothing edited.
