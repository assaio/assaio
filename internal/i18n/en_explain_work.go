package i18n

// The work-shape pages: how broadly AI is used, what the sessions were actually doing,
// how much friction they hit, and how much code came out.

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
  What this flags is the extreme: a window that reads and searches endlessly and almost
  never writes. That usually means the agent cannot find what it needs -- poor structure,
  missing conventions, or a task described too vaguely to act on.

Limits
  Only tools that name their tool calls contribute (Claude Code and Codex today);
  everything else reports zeros and is excluded rather than counted as no activity. A
  Claude sub-agent's aggregate record carries no tool-call split, so delegated work is
  under-represented here.`

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
  Some churn is healthy iteration; that is what drafting looks like. Elevated rework flags
  friction worth a closer look at specific sessions.

What to do about it
  Read it as a lead, not a verdict. Open a high-rework session and ask whether the agent
  was iterating productively or thrashing against a misunderstanding. If it is the latter,
  the fix is usually clearer up-front constraints.

Limits
  The link between AI churn and real bugs is contested, and assaio deliberately does not
  claim one. Real quality signals need git and issue-tracker correlation -- the server
  stage -- and bug density is only ever compared against age-matched human code.`

const explainRhythm = `Work Rhythm

What it measures
  When AI sessions run: the off-hours and weekend share, the time-of-day shape of the
  work, and how long the longest focused sessions last.

How to read it
  This is a question about how work is scheduled, read across the whole window. A high
  off-hours share or frequent marathon sessions is worth a conversation about workload.

What to do about it
  Treat a persistent off-hours pattern as a scheduling and capacity signal. Long focused
  sessions are worth pairing with Context Health: marathons and compaction usually show
  up together.

Limits
  This is explicitly never an attendance view and never attributed to an individual.
  Session starts are read in the reporting machine's local time, while day buckets
  elsewhere are UTC, so the two can disagree at the edges of a day.`

const explainSessionTaxonomy = `Session Taxonomy

What it measures
  What kind of work sessions were: conversational (no edits), light-edit, or heavy-edit,
  and how the mix splits.

How to read it
  Conversational sessions are real work. Design, debugging, and planning rarely touch
  files, and a high conversational share is not waste.

What to do about it
  Read the mix to understand how you actually use AI, not to push every session toward
  edits. If the mix surprises you, that is the finding -- for instance a tool you assumed
  was writing code turning out to be mostly a rubber duck.

Limits
  This is a mix, not a scorecard. A thrash bucket would need per-session rework, which is
  not stored yet. Sessions from tools without line signals cannot be classified by edit
  weight at all.`

const explainSkillEconomics = `Skill & Agent Economics

What it measures
  Which skills and sub-agents the tokens went to, and how much code each produced. Only
  the category labels the tool itself recorded are stored -- never a prompt, a path, or
  any content.

How to read it
  Shared skills and sub-agents are where a team's AI spend quietly concentrates. A skill
  burning a large share of the tokens is not a fault: research and review legitimately
  read a lot without writing much.

What to do about it
  The point is that it should be a deliberate choice rather than a surprise. If one skill
  dominates, decide whether that matches its value, and whether it could run on a cheaper
  model tier.

Limits
  Attribution only exists where the tool records it, so a window with none reports that
  rather than inventing a zero. Skill and agent names are user-authored, so they are
  pseudonymized in shareable exports.`

const explainThroughput = `Throughput

What it measures
  Total AI-added lines, lines per active day, the top projects by lines, and the
  week-over-week trend.

How to read it
  Lines added is an output-volume signal, not a quality score. More lines is not better
  code, and assaio will not present it as such.

What to do about it
  Read the trend alongside Rework and Model Fit before concluding that more code means
  more value. A rising line count with rising rework is churn, not throughput.

Limits
  Only Claude Code and Codex contribute line counts today; Gemini CLI and Cline show cost
  without lines, so any per-project reading depends on which tool was used. This is never
  ranked per named individual -- that refusal holds regardless of demand.`

const explainIntent = `Task Intent Coverage

What it measures
  How many of the window's sessions carry a label -- task class, outcome, difficulty --
  and whether those labels are spread across enough task classes to read any other metric
  per kind of work.

How to read it
  Intent is the one fact a session log never contains: what the work was for. It cannot be
  recovered by reading prompts, so it comes from an explicit, optional label you attach
  with 'assaio-agent mark'. Once two task classes have a few sessions each, every metric
  can be filtered -- 'assaio-agent analyze --task refactor', 'report --by task' -- and
  questions like "is AI-written code cheaper per line on bugfixes than on features"
  become answerable instead of being averaged into one meaningless number.

  This reads readiness to compare, never diligence. It has no unfavorable verdict: an
  unlabeled session is fully counted in every other metric and is not a failure, and a
  window whose work genuinely is all one kind is not doing anything wrong.

What to do about it
  Label a session right after finishing it: 'assaio-agent mark --task bugfix --outcome
  done'. With no session named, that marks the most recent session in the repository you
  are standing in. 'assaio-agent mark --list' shows what is labeled and what is not.

Limits
  Labels are category values only -- never free text, never a prompt -- and they stay on
  this machine: 'sync' does not send them. A session id is reused across --resume, so a
  single label can span work that changed character later in the same session. A filtered
  view covers only the sessions you labeled, which is why its confidence envelope reports
  a smaller sample than the unfiltered run.`
