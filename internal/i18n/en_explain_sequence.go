package i18n

// The sequence pages: what a session did in order, rather than what it added up to. Both read the
// stored step timeline, and both are scoped to one population because the populations behave
// differently enough that a blended figure describes neither.

const explainEditLoops = `Repeat Edits

What it measures
  How often a session goes back to a file it has already edited, with at least one command
  run in between. That definition is not assaio's own: it is the one CodeBurn publishes --
  "a retry is when the same file is re-edited after a shell command in between (Edit
  foo.ts, Bash, Edit foo.ts). Editing different files across shell steps is not a retry" --
  adopted so the figure means the same thing in both tools.

  The finding is not the rate. It is the session whose rate stands far outside this
  window's own, found the same way an unusual spend day is: against the window's median and
  spread, never against a fixed number. There is no published threshold for a healthy
  repeat rate, and inventing one would ship one machine's habits as everyone's line.

How to read it
  Write, run, fix is how the work gets done, so a repeat rate well above zero is the normal
  state of a productive session. Read the outliers as a shortlist of sessions to open, and
  the per-project bars as which repository the passes concentrate in.

What to do about it
  Open one of the named sessions and ask what the second and third pass were for. A
  recurring answer -- a missing constraint, a test the agent could not run, a file it could
  not see the whole of -- is a fixable input. If the answer is "the change was genuinely
  hard", there is nothing to fix and nothing was wrong.

Limits
  A hard bug looks exactly like a loop. So does a deliberate refactor of one file, a hub
  file every change has to touch, and a red-green test cycle where every pass was
  intended. A step carries no command identity either, so "a command ran" is all this sees
  -- never that it was a test.

  No cost is claimed, and that is a measurement rather than caution: on the maintainer's
  own store the stretches between a file's first and last edit hold 70.2% of the window's
  tokens across 67.5% of its steps, i.e. the window's own rate. Repeats cost what the work
  around them costs.

  Only sessions someone ran from a terminal are counted. On the audited store they repeat at
  25.0% against 15.5% inside sub-agents, which is why the two are never blended; an SDK
  caller's one-shot has no repeat behaviour at all. The excluded share is printed beside the
  figure.`

const explainRecovery = `Failure Recovery

What it measures
  What happens after a call fails, a person declines one, or the context is compacted
  away: what the turns following it cost against what a turn costs anywhere else in the
  window, how much of the work ran on a summarized context, and how many sessions stopped
  on a failure instead of getting past one.

How to read it
  The cost figure is a multiple of what a turn costs elsewhere in the same window -- turns
  that follow no failure -- so 1.0x means a failure changed nothing about what came next.
  There is no published multiple at which recovery becomes a problem, and assaio does not
  invent one: the figure is reported and the call is yours. The only non-arbitrary point on
  the scale is 1.0, and with thousands of turns on each side a 3% difference clears any
  noise test while meaning nothing anyone would act on.

  The baseline excludes the aftermath it is compared against. Leaving it in pulls the ratio
  toward 1.0 -- toward "nothing to see" -- by exactly the share of the window that follows a
  failure, which is a bias in the direction that flatters.

  Read over turns, not steps, deliberately. A tool call carries no tokens, and the steps
  right after a failure are more heavily model turns than the window is -- 49.2% against
  47.7% over a ten-step window on the audited corpus, 62.2% over a three-step one. Per
  step that reads 1.06x and 1.37x; per turn, 1.03x and 1.04x. The per-step number is the
  sample's composition, not a cost, and it is the number most likely to be recomputed
  wrongly from the same data.

What to do about it
  A multiple well above 1.0 points at a failure the agent could not resolve cheaply: a
  command that never works, a permission it keeps hitting, a dependency that is not
  installed. The sessions that stopped on a failure are a shortlist, not a verdict -- some
  stopped because the work was finished.

Limits
  A failure the agent expected -- probing for a file, testing a guess -- is indistinguishable
  here from one that cost it the thread, and a compaction at a natural boundary from one
  that lost something needed. The last visible step is only the last one stored: a session
  whose transcript was deleted, or whose steps fell past the retention horizon, can end
  anywhere. Sessions still running when the report was read are excluded from the ending
  figure rather than counted as finished.

  Scoped to sessions someone ran from a terminal, for the reason repeat edits are: a
  sub-agent returns to whoever launched it rather than abandoning a run of its own.`
