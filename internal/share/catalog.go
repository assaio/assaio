package share

import (
	"fmt"

	"github.com/assaio/assaio/internal/humanize"
)

// rules is the catalog, in the tie-breaking order a dead heat resolves by. Adding one is
// a single entry: the selection below reads this slice and nothing else.
var rules = []rule{
	{
		family: "scale", threshold: 1e9,
		at: func(f *facts) num { return num{v: float64(f.tokens), ok: f.tokens > 0} },
		line: func(f *facts) string {
			return fmt.Sprintf("%s tokens in %d days.", humanize.Count(f.tokens), f.activeDays)
		},
		sub: outputSub,
	},
	{
		family: "scale", threshold: 20,
		at: func(f *facts) num { return f.sessionsPerDay },
		line: func(f *facts) string {
			return humanize.Int(f.sessions) + " sessions. " + f.sessionsPerDay.text() + " a day."
		},
		sub: scaleSub,
	},
	{
		family: "scale", threshold: 50_000,
		at: func(f *facts) num { return num{v: float64(f.lines), ok: f.lines > 0} },
		line: func(f *facts) string {
			return humanize.Int(f.lines) + " lines the AI wrote. In one window."
		},
		sub: moneySub,
	},
	{
		family: "scale", threshold: 20_000,
		at: func(f *facts) num { return num{v: float64(f.toolCalls), ok: f.toolCalls > 0} },
		line: func(f *facts) string {
			return humanize.Int(f.toolCalls) + " tool calls. I approved almost all of them."
		},
		sub: func(f *facts) string {
			return "I declined " + humanize.Int(f.rejections) + ", among the calls whose source records a refusal."
		},
	},
	{
		family: "counterintuition", threshold: 70,
		at: func(f *facts) num { return f.conversational },
		line: func(f *facts) string {
			return f.conversational.text() + " of my AI sessions never touched a file."
		},
		sub: scaleSub,
	},
	{
		family: "counterintuition", threshold: 40,
		at: func(f *facts) num { return f.commands },
		line: func(f *facts) string {
			return f.commands.text() + " of my AI's classified tool calls ran a command rather than wrote code."
		},
		sub: scaleSub,
	},
	{
		family: "counterintuition", threshold: 10,
		at: func(f *facts) num { return f.delegation },
		line: func(f *facts) string {
			return f.delegation.text() + " of my work ran inside sub-agents I never watched."
		},
		sub: scaleSub,
	},
	{
		family: "counterintuition", threshold: 5,
		at:   func(f *facts) num { return num{v: float64(f.models), ok: f.models > 0} },
		line: func(f *facts) string { return fmt.Sprintf("%d models. %d tools. One window.", f.models, f.tools) },
		sub:  scaleSub,
	},
	{
		family: "money", threshold: 1000,
		at: costNum,
		line: func(f *facts) string {
			return humanize.USDCompact(costOf(f)) + " at public API prices. That was one window."
		},
		sub: moneySub,
	},
	{
		family: "money", threshold: 1,
		at: func(f *facts) num { return num{v: planMultiple(f), ok: f.planCost > 0 && f.multiple != ""} },
		line: func(f *facts) string {
			return fmt.Sprintf("A $%.0f/mo plan returned %s at API prices.", f.planCost, f.multiple)
		},
		sub: moneySub,
	},
	{
		family: "human", threshold: 45,
		at: func(f *facts) num { return f.offHours },
		line: func(f *facts) string {
			return f.offHours.text() + " of this happened outside working hours."
		},
		sub: func(f *facts) string { return f.weekend.text() + " of it was weekends." },
	},
	{
		family: "human", threshold: 20,
		at:   func(f *facts) num { return f.weekend },
		line: func(f *facts) string { return f.weekend.text() + " of it was weekends." },
		sub:  func(f *facts) string { return "Longest focused session: " + f.focusedP95 + "." },
	},
	{
		family: "self-critical", threshold: 3,
		at:   func(f *facts) num { return f.rework },
		line: func(f *facts) string { return "The AI deleted " + f.rework.text() + " of what it wrote." },
		sub:  moneySub,
	},
	{
		family: "self-critical", threshold: 80,
		at: func(f *facts) num { return f.premium },
		line: func(f *facts) string {
			return f.premium.text() + " of my tokens ran on premium models. I know."
		},
		sub: moneySub,
	},
	{
		family: "self-critical", threshold: 100,
		at:   func(f *facts) num { return num{v: float64(f.rejections), ok: f.rejections > 0} },
		line: func(f *facts) string { return "I said no " + humanize.Int(f.rejections) + " times." },
		sub:  scaleSub,
	},
	{
		family: "craft", threshold: 95,
		at:   func(f *facts) num { return f.cacheRead },
		line: func(f *facts) string { return f.cacheRead.text() + " of my context came from cache." },
		sub:  scaleSub,
	},
}
