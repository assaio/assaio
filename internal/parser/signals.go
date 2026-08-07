package parser

// The ids of the signals a source can declare it answers; internal/signal declares what each
// one means. They are constants in the package that publishes capability so that asking
// Answers a question spells the id once: a typo is then a compile error rather than a silent
// "no source answers this", which empties a metric instead of failing it.
//
//nolint:gosec // G101 matches the word "token" in these names; they are public signal ids, not credentials.
const (
	SignalTokensTotal        = "ai.tokens.total"
	SignalTokensInput        = "ai.tokens.input"
	SignalTokensOutput       = "ai.tokens.output"
	SignalTokensCacheRead    = "ai.tokens.cache_read"
	SignalTokensCacheWrite   = "ai.tokens.cache_write"
	SignalTokensCacheWrite1h = "ai.tokens.cache_write_1h"
	SignalCacheMissReason    = "ai.cache.miss_reason"
	SignalTokensReasoning    = "ai.tokens.reasoning"
	SignalCostEstimated      = "ai.cost.estimated"
	SignalSessionsCount      = "ai.sessions.count"
	SignalTurnsCount         = "ai.turns.count"
	SignalActiveMinutes      = "ai.session.active_minutes"
	SignalLinesAdded         = "ai.lines.added"
	SignalLinesRemoved       = "ai.lines.removed"
	SignalEditsCount         = "ai.edits.count"
	SignalToolCallsCount     = "ai.tool_calls.count"
	SignalToolErrorsCount    = "ai.tool_errors.count"
	SignalRejectedCount      = "ai.rejected.count"
	SignalCompactionsCount   = "ai.compactions.count"
	SignalReworkLines        = "ai.rework.lines"
	SignalSkillTokens        = "ai.skill.tokens"
	SignalAgentTokens        = "ai.agent.tokens"
)
