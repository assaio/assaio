package i18n

// The pages about the shape of the work rather than how it went: when it ran, what kind of
// session it was, where attributed spend concentrated, how much came out and what any of it was
// for. The split is by subject rather than by line count -- a translator working on one of
// these is working on the others, and cutting the catalog at an arbitrary line would leave them
// switching files mid-thought.

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
  Only sources that record changed lines contribute here; the rest show cost without
  lines, so any per-project reading depends on which tool was used --
  'assaio-agent signals coverage' says what your own data supports. This is never ranked per named individual -- that refusal holds
  regardless of demand.`

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
