package analyze

import (
	"strings"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"
)

const (
	reasoningName      = "reasoning-share"
	reasoningTitle     = "Reasoning Share"
	reasoningDescribe  = "How much generated output is extended-thinking (reasoning) tokens, among tools that report it -- flagging deep reasoning spent on shallow tasks."
	reasoningHowToRead = "Reasoning tokens are billed as output. A high share means much of the model's work is internal deliberation, which can be overkill on routine tasks. Only some sources report it at all, so the coverage figure says how much of your output this even covers."
	// reasoningWatchShare is the reasoning-of-output share above which the read flags heavy thinking.
	reasoningWatchShare = 0.3
)

func init() { Register(reasoningValidator{}) }

// reasoningValidator reports the reasoning share of output among tools that surface it, and
// how much of the window's output that covers, so a Claude-heavy window reads honestly.
type reasoningValidator struct{}

func (reasoningValidator) Name() string     { return reasoningName }
func (reasoningValidator) Title() string    { return reasoningTitle }
func (reasoningValidator) Describe() string { return reasoningDescribe }

//nolint:gocritic // Input is required by the Validator interface; analyzed once per run, not a hot path.
func (reasoningValidator) Analyze(in Input) Result {
	r := Result{Name: reasoningName, Title: reasoningTitle, Describe: reasoningDescribe, HowToRead: reasoningHowToRead}
	if in.Totals.Output == 0 {
		r.noData("active days", "No output tokens in this window.")
		return r
	}
	r.restsOn(activeDays(&in), "active days")
	var reasoning, reportingOutput int64
	for i := range in.Usage {
		u := &in.Usage[i]
		if !reportsReasoning(u.Tool) {
			continue
		}
		reasoning += u.Reasoning
		reportingOutput += u.Out
	}
	// Share is of output from tools that actually report reasoning, so a Claude-heavy
	// window (Claude doesn't surface it) isn't diluted to a meaningless near-zero -- which is
	// exactly why the verdict has to carry how much of the window that reporting output is.
	coverage := fracOf(reportingOutput, in.Totals.Output)
	r.covering(coverage)
	if reportingOutput == 0 {
		r.Read = noDataRead
		r.Figures = []Figure{{Label: "reporting coverage", Value: "0%", Note: "no tool here reports reasoning"}}
		r.Takeaway = "No source in this window reported reasoning tokens, so no share can be read from it."
		r.Caveats = []string{reasoningCoverageCaveat()}
		return r
	}

	share := fracOf(reasoning, reportingOutput)

	r.Read = readFor(share < reasoningWatchShare, "Lean")
	r.Purity = clamp01(1 - share)
	r.Figures = []Figure{
		{Label: "reasoning share", Value: honestPercent(share), Note: "of reporting output"},
		{Label: "reasoning tokens", Value: humanize.Count(reasoning)},
		{Label: "reporting coverage", Value: honestPercent(coverage), Note: "output from tools that report it"},
	}
	r.Takeaway = reasoningTakeaway(share)
	r.Caveats = []string{
		reasoningCoverageCaveat(),
		"Reasoning is a subset of output (already billed there); this is a composition signal, not extra cost.",
	}
	return r
}

// reportsReasoning asks the depth matrix rather than keeping a list of tool names.
func reportsReasoning(tool string) bool { return parser.Answers(tool, parser.SignalTokensReasoning) }

// reasoningCoverageCaveat names the sources that report reasoning from the depth matrix, so
// the sentence cannot outlive the list it used to spell out.
func reasoningCoverageCaveat() string {
	return "Reasoning is reported by " + strings.Join(sourcesAnswering(parser.SignalTokensReasoning), ", ") +
		"; the rest of your output comes from sources that never surface it."
}

func reasoningTakeaway(share float64) string {
	if share >= reasoningWatchShare {
		return "A large share of reported output is reasoning -- worth checking whether deep thinking is going to shallow tasks."
	}
	return "Reasoning is a modest share of reported output -- the thinking budget looks proportionate."
}
