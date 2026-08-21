package i18n

// The pages about how the work went: how broadly AI is used, what the sessions were doing,
// how much friction they hit, and how much of what they wrote was undone again.

const explainAdoption = `Adoption & Usage Breadth

What it measures
  Sessions, active days, and how many distinct projects and tools the usage spans, plus
  whether that breadth is growing.

How to read it
  Breadth shows how far AI usage has spread, not how good any of it is. Narrow usage
  concentrated in one project is a normal early state, not a failure.

What to do about it
  A narrow or flat trend is a cue to invest in onboarding -- examples, shared skills,
  someone to pair with -- rather than a mark against anyone. Breadth is a team-health
  signal, and pushing it as a target turns it into a number people chase.

Limits
  Active days are bucketed in UTC, so late local-evening work can land on the next day.
  This never becomes a per-person view: it is aggregate by design.`

const explainContext = `Context Health

What it measures
  Conversation depth, peak context size, active time per session, and how often sessions
  hit a context compaction.

How to read it
  Frequent compaction means sessions are outgrowing their context window mid-task. That
  is a workflow signal, not a quality problem -- but it does have a real cost, since
  everything after a compaction has to be re-established.

What to do about it
  Try shorter, more focused sessions, or split a long task at a natural boundary rather
  than letting it compact. Starting fresh with a short written handoff is usually cheaper
  than paying to rebuild context implicitly.

Limits
  Peak context is read from what the tools record and is not compared against the model's
  real limit -- a vendored context-window table would be needed for that, and does not
  exist yet.`

const explainExploreProduce = `Explore vs Produce

What it measures
  What the tool calls were for: reading and searching the codebase versus writing in it,
  with reads per write. The tool's name is classified during parsing and then dropped --
  neither it nor any tool input is ever stored.

How to read it
  Exploring is how an agent earns the right to write, so a high read share is not waste.
  Unfamiliar or large codebases legitimately need more of it.

What to do about it
  The case worth opening is the extreme: a window that reads and searches endlessly and
  almost never writes. That usually means the agent cannot find what it needs -- poor
  structure, missing conventions, or a task described too vaguely to act on. assaio does not
  say where that line is, because no published definition sets one.

Limits
  Only tools that name their tool calls contribute; everything else is excluded rather
  than counted as no activity, and 'assaio-agent signals coverage' names which sources
  those are on your machine. A Claude sub-agent's aggregate record carries no tool-call
  split, so delegated work is under-represented here.`

const explainFriction = `Friction

What it measures
  How often tool calls come back an error, reported separately from how often a human
  declines one. The two are deliberately not merged: they have different fixes.

How to read it
  Some failure is normal -- an agent probes, a command exits non-zero, it adapts. A rising
  error rate is the useful signal, and every failed call is paid for twice: once to make
  it, once to recover.

What to do about it
  A persistent error rate usually means something concrete and fixable: a broken command
  in the project's instructions, a missing dependency, a path that does not exist, or a
  permission the agent keeps hitting. Human rejections point elsewhere -- usually at the
  agent proposing work you did not want.

Limits
  Only tools that record tool-call outcomes contribute. A session ingested while still
  being written can under-count errors, because a failed result is attributed from a later
  line that may not have been written yet.`

const explainRework = `Rework & Rejection

What it measures
  Within-session code churn -- lines written and then rewritten in the same session -- and
  how often a human rejected a tool call.

How to read it
  Some churn is healthy iteration; that is what drafting looks like. Nothing published says
  where a rework or rejection rate becomes a problem, so assaio reports both and grades
  neither -- a high one is worth a closer look at specific sessions, not a verdict on the
  window.

What to do about it
  Read it as a lead, not a verdict. Open a high-rework session and ask whether the agent
  was iterating productively or thrashing against a misunderstanding. If it is the latter,
  the fix is usually clearer up-front constraints.

Limits
  The link between AI churn and real bugs is contested, and assaio deliberately does not
  claim one. Real quality signals need git and issue-tracker correlation -- the server
  stage -- and bug density is only ever compared against age-matched human code.`
