package signal

import "sort"

var perTurn = []string{GrainTurn, GrainSession, GrainDay, GrainWindow}

// catalog is every signal assaio can report today, declared once. An entry claims only what
// the code actually derives: a claim that outruns the derivation shows up as support the
// metrics do not really have the moment `signals coverage` runs on real data.
var catalog = []Signal{
	{
		ID: "ai.tokens.total", Title: "Total tokens", Unit: "tokens",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "no billable usage in the window",
		Describe:  "Input, output and both cache directions summed. Reasoning is inside output and never re-added.",
	},
	{
		ID: "ai.tokens.input", Title: "Input tokens", Unit: "tokens",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "no billable usage in the window",
		Describe:  "Uncached prompt tokens. Claude's own figure can be a streaming placeholder, so totals may drift from the vendor console.",
	},
	{
		ID: "ai.tokens.output", Title: "Output tokens", Unit: "tokens",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "no billable usage in the window",
		Describe:  "Tokens the model produced, including any reasoning portion.",
	},
	{
		ID: "ai.tokens.cache_read", Title: "Cache-read tokens", Unit: "tokens",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "nothing was served from the prompt cache -- real for one-shot work",
		Describe:  "Prompt tokens billed at the cache-read rate.",
	},
	{
		ID: "ai.tokens.cache_write", Title: "Cache-write tokens", Unit: "tokens",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "nothing was written to the prompt cache",
		Describe:  "Prompt tokens billed at the cache-write rate; wasted when never read back.",
	},
	{
		ID: "ai.tokens.reasoning", Title: "Reasoning tokens", Unit: "tokens",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "this source does not report reasoning separately -- not that none was spent",
		Describe:  "The thinking portion already counted inside output, for the sources that surface one.",
	},
	{
		ID: "ai.cost.estimated", Title: "Estimated cost", Unit: "usd",
		Status: Estimated, Grains: perTurn,
		ZeroMeans: "no priced usage -- a model absent from the price table is excluded, never counted as free",
		Describe:  "Tokens priced at public pay-as-you-go rates. Not your bill: a flat plan bills differently, and long-context and cache tiers are not modelled.",
	},
	{
		ID: "ai.lines.added", Title: "AI lines added", Unit: "lines",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "this source records no edit activity -- absent, not zero",
		Describe:  "Added lines counted from the tool's own edit records. The code itself is never stored.",
	},
	{
		ID: "ai.lines.removed", Title: "AI lines removed", Unit: "lines",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "this source records no edit activity -- absent, not zero",
		Describe:  "Removed lines counted from the tool's own edit records.",
	},
	{
		ID: "ai.edits.count", Title: "Edit operations", Unit: "count",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "this source records no edit activity -- absent, not zero",
		Describe:  "How many edit-type tool calls ran, regardless of how many lines each moved.",
	},
	{
		ID: "ai.tool_calls.count", Title: "Tool calls", Unit: "count",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "this source does not name its tool calls -- absent, not zero",
		Describe:  "Every tool invocation in a turn, split by purpose where the source names them.",
	},
	{
		ID: "ai.tool_errors.count", Title: "Failed tool calls", Unit: "count",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "this source does not mark call outcomes -- absent, not a clean run",
		Describe:  "Calls that came back an error, counted only where a source marks the outcome of every call.",
	},
	{
		ID: "ai.rejected.count", Title: "Declined tool calls", Unit: "count",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "this source does not record a human declining a call -- absent, not a run nobody stopped",
		Describe:  "Tool proposals a person refused, counted only where a source records the refusal at all.",
	},
	{
		ID: "ai.compactions.count", Title: "Context compactions", Unit: "count",
		Status: Observed, Grains: perTurn,
		ZeroMeans: "this source does not mark a context overflow -- absent, not a session that never overflowed",
		Describe:  "Times a session's context overflowed and was auto-summarized, counted from the tool's own boundary marker.",
	},
	{
		ID: "ai.rework.lines", Title: "Rework lines", Unit: "lines",
		Status: Derived, Grains: perTurn,
		ZeroMeans: "no added line was undone later in the same transcript, or the source records no edits",
		Describe:  "Added lines a later edit in the same session removed again. A churn proxy; healthy iteration churns too.",
	},
	{
		ID: "ai.sessions.count", Title: "Sessions", Unit: "sessions",
		Status: Observed, Grains: []string{GrainDay, GrainWindow},
		ZeroMeans: "no sessions in the window",
		Describe:  "Distinct sessions seen. A resumed session keeps its id, so one session can span days.",
	},
	{
		ID: "ai.turns.count", Title: "Turns", Unit: "count",
		Status: Observed, Grains: []string{GrainSession, GrainDay, GrainWindow},
		ZeroMeans: "no turns recorded -- a session-total source contributes none",
		Describe:  "Model requests. A source that totals a whole session cannot contribute here, which is what turn-level coverage reports.",
	},
	{
		ID: "ai.session.active_minutes", Title: "Focused minutes", Unit: "minutes",
		Status: Derived, Grains: []string{GrainSession, GrainWindow},
		ZeroMeans: "no measurable gap-free activity",
		Describe:  "Time between turns with gaps over 30 minutes excluded, so a resumed session's idle time is not counted as work.",
	},
	{
		ID: "ai.skill.tokens", Title: "Tokens per skill", Unit: "tokens",
		Status: Observed, Grains: []string{GrainWindow},
		ZeroMeans: "this source does not label turns with a skill -- absent from the split, not zero",
		Describe:  "Tokens the tool itself attributed to a named skill.",
	},
	{
		ID: "ai.agent.tokens", Title: "Tokens per sub-agent", Unit: "tokens",
		Status: Observed, Grains: []string{GrainWindow},
		ZeroMeans: "this source has no sub-agents, or does not label them -- absent, not zero",
		Describe:  "Tokens spent inside a named sub-agent type.",
	},
}

// Catalog returns every declared signal, sorted by id.
func Catalog() []Signal {
	out := make([]Signal, len(catalog))
	copy(out, catalog)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns the signal declared under id.
func Get(id string) (Signal, bool) {
	for i := range catalog {
		if catalog[i].ID == id {
			return catalog[i], true
		}
	}
	return Signal{}, false
}
