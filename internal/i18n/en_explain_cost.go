package i18n

// The cost-and-efficiency pages: what the tokens went to, what they cost, and where a
// cheaper shape would have bought the same output.

const explainBurnAnomaly = `Burn Anomaly

What it measures
  The days whose token burn stands far outside this window's typical day, found with a
  median/MAD outlier test rather than a mean. A median-based test is used because one
  runaway day would drag a mean far enough to hide itself.

How to read it
  A spike is a prompt to go look, not a fault. A migration, a long refactor, or a big
  one-off analysis legitimately burns more than a normal day, and none of that is waste.

What to do about it
  Open the flagged day and ask what ran. The findings worth catching are the ones nobody
  intended: a loop that retried forever, a backfill re-ingested twice, an agent left
  running against an empty task, or a script that called the model in a tight loop.

Limits
  Days are bucketed in UTC, so late local-evening work can land on the next day and split
  one working session across two buckets. A window with very few active days has no
  stable "typical day" to compare against, and the test stays quiet rather than inventing
  one.`

const explainCacheHygiene = `Cache Hygiene

What it measures
  How much of your billed input was served from the prompt cache instead of re-sent as
  fresh input, plus whether cache writes are actually being reused afterwards.

How to read it
  High reuse means repeated context -- system prompts, file contents, conversation
  history -- is being served cheaply rather than re-billed every turn. This is a cost
  signal, not a quality one. A big one-shot task legitimately shows low reuse because
  there was nothing to reuse yet.

What to do about it
  Low reuse across many sessions usually means context is being rebuilt each turn rather
  than carried. Cache writes that are never read back are the clearer waste: you paid the
  write premium and got none of the discount.

Limits
  The cache lifetime a write bought is in some logs but not read yet, so reuse is
  approximated at day grain.
  The price table is flat per model and does not model the 1h-cache premium, so heavily
  cached sessions are costed slightly low.`

const explainConcentration = `Spend Concentration

What it measures
  How token spend spreads across projects, and -- more usefully -- where a project's share
  of the tokens outruns its share of the AI-written lines.

How to read it
  Concentration by itself is neither good nor bad. One project can legitimately own most
  of the work for a whole quarter. Read the gap instead: a project holding a far larger
  share of the tokens than of the lines is where spend is not converting into code.

What to do about it
  Look at the widest gap first. Common causes are a repository the agent has to re-read
  constantly, a task that is mostly investigation, or an automation pointed at a project
  nobody is actively shipping.

Limits
  Lines are only counted for sources that report them, so a project worked on mainly
  through one that does not will show tokens with no lines and read as a false gap. Check
  Coverage, or 'assaio-agent signals coverage', before drawing conclusions.`

const explainCoverage = `Coverage & Confidence

What it measures
  How much of this window rests on complete data: the share of tokens from tools that
  capture full activity, and the share of usage on models the price table knows.

How to read it
  This is the honesty backbone of every other metric. Low activity coverage means line and
  edit signals describe only part of your usage. Low priced coverage means some cost is
  excluded from every dollar figure -- a floor, never a real zero.

What to do about it
  Treat it as a gate rather than a goal. Before acting on any lines-based figure, check
  that activity coverage is high enough for it to mean anything. If pricing coverage is
  low, a new or renamed model is probably missing from the vendored table.

Limits
  Coverage describes what assaio could read, not whether the tools themselves recorded
  the truth. All on-disk log formats are internal and may change between tool versions.`

const explainModelFit = `Model Fit

What it measures
  The token share on premium versus cheaper models, the lines-per-token contrast between
  them, and the real sub-agent delegation share read from the logs' own markers.

How to read it
  A high premium share is not wrong on its own. The question is whether the work needed
  it: routine edits, renames, and boilerplate usually come out just as well on a cheaper
  model, and that is spend you can trim without losing output.

What to do about it
  Compare lines-per-token between the tiers. If the cheaper tier is producing comparably,
  route the routine work there and keep the premium model for the parts that genuinely
  need it. Delegating to sub-agents is another lever, since sub-agent work can run on a
  different tier than the main loop.

Limits
  Task difficulty is invisible in the logs, so a high premium share can be entirely
  justified. This is a prompt to look, never a verdict.`

const explainModelRightSizing = `Model Right-Sizing

What it measures
  Turns on a premium model that produced very little output -- the small tasks a cheaper,
  faster model might handle just as well.

How to read it
  A premium-model turn emitting a handful of output tokens is often boilerplate, a short
  confirmation, or a quick lookup. On a flat subscription this is about speed and rate
  limits rather than dollars; on pay-as-you-go it is direct spend.

What to do about it
  Sample a few of the flagged turns before changing anything. If they are genuinely
  trivial, the fix is usually routing: a cheaper default model with escalation for hard
  work, rather than a premium default everywhere.

Limits
  Output size is a poor proxy for difficulty -- a one-line answer can require deep
  reasoning. This needs per-turn data, so it is reported for the window as a whole and is
  skipped in the per-project drill-down, where a window-wide constant would contradict
  the window-level verdict.`

const explainReasoningShare = `Reasoning Share

What it measures
  How much of the generated output is extended-thinking (reasoning) tokens, among the
  tools that report them, plus how much of your output that coverage even represents.

How to read it
  Reasoning tokens are billed as output. A high share means much of the model's work is
  internal deliberation rather than text or code you received, which can be overkill on
  routine tasks and exactly right on hard ones.

What to do about it
  If the share is high on work you consider routine, lower the thinking budget or pick a
  model tier whose default deliberation matches the task. Leave it alone for genuinely
  hard problems -- that is what it is for.

Limits
  Only some sources report reasoning separately, so the coverage figure is part of the
  reading, not a footnote -- 'assaio-agent signals coverage' says which do. Reasoning tokens are
  assumed to be included in output for costing.`

const explainSubscriptionFit = `Subscription Fit

What it measures
  Whether a flat monthly plan pays off against API pay-as-you-go pricing, by projecting
  this window's API-equivalent cost to a month at your active-day pace and comparing it
  with the plan price you configured.

How to read it
  A high multiple means the plan is a bargain at your volume. Below 1x means pay-as-you-go
  might be cheaper. Read it as a multiple, not as consumption: a flat plan only starts
  paying off above 100% of its price, so a small percentage early in the month is normal.

What to do about it
  Set config.pricing.monthly_subscription_cost, otherwise there is nothing to compare
  against and the metric will keep prompting you. If the multiple is persistently below
  1x, the plan is costing more than the equivalent API usage.

Limits
  The API figure is an estimate at public pay-as-you-go prices, not your actual bill. The
  projection assumes your active-day pace holds for the month, which a quiet or unusually
  busy window will distort. The price table is flat per model, so long-context and
  cache-premium usage is under-estimated.`

const explainTurnEfficiency = `Turn Efficiency

What it measures
  How much you get per prompt: the one-shot rate, the median number of turns in sessions
  that produced code, and output tokens per turn.

How to read it
  These are prompting-efficiency signals, not quality. A session that lands an edit in a
  turn or two is efficient. A long session can equally be deliberate, careful work on
  something hard.

What to do about it
  A low one-shot rate across many sessions usually points at context: the agent starting
  without the files, conventions, or constraints it needed. Front-loading that context is
  the cheapest fix.

Limits
  Task size is invisible here, so this is directional and never a per-person score. Small
  samples are gated: below the minimum session floor the metric reports neutral rather
  than grading a handful of sessions.`
