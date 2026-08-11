# Label rules you can paste in

*Part of [Extending assaio](../extending.md). What labels are for: [`mark`](../extending.md#the-surfaces) and the [task/outcome/difficulty vocabularies](https://github.com/assaio/assaio/blob/main/docs/adr/0006-session-annotations.md).*

A label is the one fact a session log cannot contain: what the work *was*. `mark --suggest`
derives it from what the store already recorded — the branch a session ran on, the skill or
sub-agent it used, the entrypoint it came through — and `--accept-suggested` writes what it
derived, never replacing a label made by hand.

The built-in rules cover branch conventions common enough to be worth shipping, and nothing
else. On the maintainer's own store they reach **4.2% of sessions and 5.5% of tokens** — which
is the honest yield of a convention assaio did not invent, not a failure. Your repository's own
conventions are worth more than any default, and each is four lines.

Every recipe below is loaded and exercised by `TestRecipeLabelRules`, which asserts the labels
it derives. A recipe that stopped working would fail the build rather than sit here reading
plausibly.

## How a rule is evaluated

Each entry names a **source** (`branch`, `skill`, `agent`, `entrypoint`), an **RE2 pattern**
over that source's value, and the **axis** and **value** a match implies. Every rule is tried;
if two rules imply different values on the same axis, the session gets **nothing** on that axis
— a disagreement is not a tie to be broken. The axis vocabularies are closed, so a rule
proposing a value outside one fails at load rather than at midnight.

Rules are *added* to the built-ins. Set `labels.defaults: false` when a default reads a
convention your repository uses for something else — an `audit/` branch that is not a review,
say — so only yours apply.

## Conventional-commit branches

The most common convention there is, and the one the defaults already cover; spelled out here
because it is the template every other recipe copies.

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

Where a branch is named after a tracker key rather than the work, the type usually lives in the
project prefix. This is the shape to copy when your keys look like `PLAT-1234` or `BUG-77`.

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

A spike is research, and its cost per line is meaningless — which is exactly why labelling it
matters: `analyze --task research` takes it out of the numbers everything else is judged by.

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

Claude Code records which skill and which sub-agent a turn ran under, and both are stronger
evidence than a branch name: a session that spent its tokens under a review sub-agent was a
review, whatever the branch was called. No other source records either today, so a repository
whose work runs elsewhere derives nothing from these — which is the correct answer, not a gap.

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

An entrypoint says how the session was started — a hook, a scheduled run, an editor. It answers
*difficulty* and *outcome* far better than it answers *task*: an unattended run that nobody was
watching is not the same work as one someone sat through.

```yaml #entrypoints
labels:
  rules:
    - source: entrypoint
      match: '(?i)(cron|schedule|ci)'
      axis: difficulty
      value: low
```

## Reading what you get before you write it

`mark --suggest` shows the evidence for each session and writes nothing, so a rule set can be
read before it is trusted:

```console
$ assaio-agent mark --suggest --since 30d
$ assaio-agent mark --accept-suggested --since 30d
```

The second command writes only what the first showed, and never replaces a label made by hand.
If the output is empty, the honest reading is that your repository's convention is not in this
file yet — not that the sessions were unlabelable.
