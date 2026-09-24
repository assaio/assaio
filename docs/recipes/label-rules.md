# Label rules you can paste in

*Part of [Extending assaio](../extending.md). Label uses: [`mark`](../extending.md#the-surfaces) and
the [task/outcome/difficulty
vocabularies](https://github.com/assaio/assaio/blob/main/docs/adr/0006-session-annotations.md).*

A session log cannot record what the work *was*; a label can. `mark --suggest` uses stored
evidence—the session's branch, skill or sub-agent, and entrypoint—to suggest it.
`--accept-suggested` writes those suggestions without replacing hand-written labels.

Built-in rules cover only common branch conventions. In the maintainer's store they label **4.2% of
sessions and 5.5% of tokens**. That is the reach of a convention assaio did not create. Your
repository's conventions can cover more, and each rule takes four lines.

`TestRecipeLabelRules` loads every recipe below and checks its derived labels. If a recipe stops
working, the build fails.

## How a rule is evaluated

Each entry specifies a **source** (`branch`, `skill`, `agent`, `entrypoint`), an **RE2 pattern** for
its value, and the resulting **axis** and **value**. Every rule runs. If two rules give different
values on one axis, the session gets **nothing** on that axis. Axis vocabularies are closed; rules
with invalid values fail when loaded.

Your rules are *added* to the built-ins. Set `labels.defaults: false` if a default misreads your
convention—for example, an `audit/` branch that is not a review—so only your rules apply.

## Conventional-commit branches

The defaults already cover this common convention. Use it as the template for other recipes.

```yaml #conventional-branches
labels:
  rules:
    - source: branch
      match: '^(feat|feature)/'
      axis: task
      value: feature
    - source: branch
      match: '^(fix|bugfix|hotfix)/'
      axis: task
      value: bugfix
    - source: branch
      match: '^(test|tests)/'
      axis: task
      value: test
    - source: branch
      match: '^refactor/'
      axis: task
      value: refactor
    - source: branch
      match: '^docs?/'
      axis: task
      value: docs
```

## Ticket keys that carry the type

If a branch uses a tracker key instead of describing the work, the project prefix often identifies
the task type. Use this pattern for keys such as `PLAT-1234` or `BUG-77`.

```yaml #ticket-keys
labels:
  rules:
    - source: branch
      match: '(?i)^(bug|def|inc)-[0-9]+'
      axis: task
      value: bugfix
    - source: branch
      match: '(?i)^(feat|story|us)-[0-9]+'
      axis: task
      value: feature
    - source: branch
      match: '(?i)^(td|debt|chore)-[0-9]+'
      axis: task
      value: refactor
```

## Spikes and throwaway branches

A spike is research, so its cost per line is meaningless. Label it so `analyze --task research`
excludes it from figures used to judge other work.

```yaml #spikes
labels:
  rules:
    - source: branch
      match: '^(spike|poc|prototype|experiment|scratch)/'
      axis: task
      value: research
    - source: branch
      match: '^(spike|poc|prototype)/'
      axis: difficulty
      value: high
```

## Skills and sub-agents

Claude Code records the skill and sub-agent used for each turn. Both provide stronger evidence than
a branch name: tokens spent under a review sub-agent indicate review work regardless of branch. No
other source records either today, so work done elsewhere gets no label from these rules.

```yaml #skills-and-agents
labels:
  rules:
    - source: skill
      match: '(?i)(review|audit)'
      axis: task
      value: review
    - source: agent
      match: '(?i)(reviewer|critic)'
      axis: task
      value: review
    - source: skill
      match: '(?i)(debug|systematic-debugging)'
      axis: task
      value: bugfix
    - source: skill
      match: '(?i)(brainstorm|design|plan)'
      axis: task
      value: research
```

## Entrypoints: what ran it, not what it was

An entrypoint shows whether a hook, scheduled run, or editor started the session. It provides better
evidence for *difficulty* and *outcome* than for *task*: an unattended run differs from one a person
watched.

```yaml #entrypoints
labels:
  rules:
    - source: entrypoint
      match: '(?i)(cron|schedule|ci)'
      axis: difficulty
      value: low
```

## Reading what you get before you write it

`mark --suggest` shows each session's evidence without writing anything, so you can review the
rules' results:

```console
$ assaio-agent mark --suggest --since 30d
$ assaio-agent mark --accept-suggested --since 30d
```

The second command writes only what the first showed and never replaces hand-written labels. Empty
output means these rules do not cover your repository's convention yet; it does not mean the
sessions cannot be labeled.
