package parser

// The depth tiers, ordered by what a source can support. Deep is separated from Standard by
// attribution; Standard from ImportOnly by whether the figures can be attributed to a
// session at all. Granularity is a documented gap within Standard rather than a tier of its
// own: a source that totals per session still answers session questions honestly, it just
// cannot answer one asked per turn.
const (
	// Deep carries tokens, per-turn activity, and the labels that say what the work was.
	Deep = "deep"
	// Standard carries reliable usage whose gaps -- missing activity signals, or a coarser
	// record granularity -- are documented rather than implied.
	Standard = "standard"
	// ImportOnly carries billing or aggregate figures that cannot support session-level
	// conclusions.
	ImportOnly = "import-only"
)

// Depth is what one source can actually tell you. "Supports Gemini CLI" is misleading when
// it means tokens but no edits, so every source publishes its per-field capability and,
// below Deep, the specific gaps behind the label.
type Depth struct {
	Tool string
	Tier string
	// Tokens, Activity and Attribution are the three capability axes: what it cost, what
	// was produced, and what the work was labeled as.
	Tokens, Activity, Attribution bool
	// Answers lists the ids of the signals this source can actually answer (internal/signal).
	// The three axes above summarise a source for the tier table; this is the per-signal
	// truth, and the two are not interchangeable -- "has activity" is a single bit over a
	// source that reports changed lines but no edit count, no tool calls and no turns. What
	// Gaps says in prose, this says in a form the signal catalog can compute with.
	Answers []string
	// Gaps names what this source does not carry, in a reader's terms. Required below Deep.
	Gaps []string
}

// depths is the matrix, deepest first. A parser that gains a capability updates its row
// here in the same change -- this is the one place the answer lives, and doctor, the
// coverage validator and the docs all read it rather than each keeping their own list.
// The signal groups sources are built from, so a row states what it adds rather than
// repeating a list. Ids are internal/signal's; a test there asserts every one is declared.
var (
	costSignals = []string{
		SignalTokensTotal, SignalTokensInput, SignalTokensOutput,
		SignalTokensCacheRead,
		SignalCostEstimated, SignalSessionsCount,
	}
	// cacheWriteSignals are declared per source rather than bundled with cost, for the same
	// reason reasoning is: Codex and Gemini CLI publish no cache-write counter at all, so a
	// bundled claim promised a figure their records can only ever leave at zero -- a silence
	// reported as a measurement, which is the one thing the depth matrix exists to prevent.
	cacheWriteSignals = []string{SignalTokensCacheWrite}
	// reasoningSignals are declared per source rather than bundled with cost: Claude Code
	// and Cline never surface a thinking count, so claiming it for them would report support
	// for a figure their records can only ever leave at zero.
	reasoningSignals = []string{SignalTokensReasoning}
	// cacheDetailSignals say what a cache write bought and why one could not be read back.
	// Claude Code is the only source publishing either today, which is why they are declared
	// per source rather than folded into costSignals: a source that reports a cache write
	// and no tier would otherwise claim every write was the cheap one.
	cacheDetailSignals = []string{SignalTokensCacheWrite1h, SignalCacheMissReason}
	// perTurnSignals need records at turn grain: a source that totals a whole session has
	// no second timestamp to measure a gap or a turn against.
	perTurnSignals = []string{SignalTurnsCount, SignalActiveMinutes}
	lineSignals    = []string{SignalLinesAdded, SignalLinesRemoved}
	editSignals    = []string{SignalEditsCount, SignalToolCallsCount, SignalReworkLines}
	// compactionSignals need the tool to mark a context overflow of its own accord. A source
	// that writes no boundary line reports none, and a rate computed over it would read that
	// silence as a session whose context never overflowed.
	compactionSignals = []string{SignalCompactionsCount}
)

func answers(groups ...[]string) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

var depths = []Depth{
	{
		Tool: "claude-code", Tier: Deep,
		Tokens: true, Activity: true, Attribution: true,
		Answers: answers(costSignals, cacheWriteSignals, perTurnSignals, lineSignals,
			editSignals, compactionSignals, cacheDetailSignals,
			[]string{SignalToolErrorsCount, SignalRejectedCount, SignalSkillTokens, SignalAgentTokens}),
	},
	{
		Tool: "codex", Tier: Standard,
		Tokens: true, Activity: true, Attribution: false,
		// tool_errors is absent deliberately: Codex marks failures only for file edits, so
		// counting it would read partial coverage as a clean run -- the same reason the
		// friction validator excludes it rather than treating silence as success.
		Answers: answers(costSignals, reasoningSignals, perTurnSignals, lineSignals, editSignals, compactionSignals),
		Gaps: []string{
			"no cache-write counter, so a written cache is invisible where a read one is not",
			"no skill or sub-agent labels, so its turns are absent from the attribution split",
			"tool-use denials are not recorded, and call failures only for file edits",
		},
	},
	{
		Tool: "gemini-cli", Tier: Standard,
		Tokens: true, Activity: false, Attribution: false,
		Answers: answers(costSignals, reasoningSignals, perTurnSignals),
		Gaps: []string{
			"no line, edit or tool-call signals, so it contributes cost but no output figures",
			"no cache-write counter, so a written cache is invisible where a read one is not",
			"tool-use tokens are folded into output, and ~/.gemini may be shared with other tools",
		},
	},
	{
		Tool: "copilot-cli", Tier: Standard,
		Tokens: true, Activity: true, Attribution: false,
		// Lines but nothing else: a whole-session total carries no turn, edit or tool-call
		// count, so every per-turn signal is absent rather than zero.
		Answers: answers(costSignals, cacheWriteSignals, reasoningSignals, lineSignals),
		Gaps: []string{
			"totals exist only when a session ends, so one record covers a whole session and per-turn figures exclude it",
			"code changes are counted once per session with no per-model split, so they are credited whole to the model that made the most requests",
		},
	},
	{
		Tool: "cline", Tier: Standard,
		Tokens: true, Activity: false, Attribution: false,
		Answers: answers(costSignals, cacheWriteSignals, perTurnSignals),
		Gaps: []string{
			"no line, edit or tool-call signals, so it contributes cost but no output figures",
			"its own per-request cost is recorded but recomputed from tokens for cross-tool consistency",
		},
	},
}
