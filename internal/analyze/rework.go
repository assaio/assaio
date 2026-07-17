package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/report"
)

const (
	reworkName     = "rework"
	reworkTitle    = "Rework & Rejection"
	reworkDescribe = "Within-session code churn and human tool-call rejections -- a directional friction proxy."
	// reworkHowToRead is Result.HowToRead for this validator -- see its doc comment.
	reworkHowToRead = "Elevated rework or rejection flags friction worth a closer look at specific sessions, but the link between AI churn and real bugs is still contested, so treat it as a lead, not a verdict."
	// reworkWatchCeiling is the rework-rate/rejection-rate threshold above which
	// friction is flagged for a closer look.
	reworkWatchCeiling = 0.15
)

func init() { Register(reworkValidator{}) }

// reworkValidator reads within-session churn and rejection friction: AI-added code that
// got removed again in the same transcript, and tool proposals the human declined.
type reworkValidator struct{}

func (reworkValidator) Name() string     { return reworkName }
func (reworkValidator) Title() string    { return reworkTitle }
func (reworkValidator) Describe() string { return reworkDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (reworkValidator) Analyze(in Input) Result {
	r := Result{Name: reworkName, Title: reworkTitle, Describe: reworkDescribe, HowToRead: reworkHowToRead}
	if len(in.Usage) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	churn := report.BuildChurn(in.Usage)
	var rejected, toolCalls int64
	for i := range in.Usage {
		rejected += in.Usage[i].Rejected
		toolCalls += in.Usage[i].ToolCalls
	}
	rejectionKnown := toolCalls > 0
	var rejectionRate float64
	if rejectionKnown {
		rejectionRate = float64(rejected) / float64(toolCalls)
	}
	low := churn.ReworkRate <= reworkWatchCeiling && rejectionLow(rejectionRate, rejectionKnown)

	r.Read = readFor(low, "Low")
	r.Purity = reworkPurity(churn.ReworkRate, rejectionRate, rejectionKnown)
	r.Figures = []Figure{
		{
			Label: "rework", Value: shareOrDash(churn.ReworkLines, churn.LinesAdded, 0),
			Note: strconv.FormatInt(churn.ReworkLines, 10) + " lines, within-session thrash proxy",
		},
		{
			Label: "rejection rate", Value: shareOrDash(rejected, toolCalls, 0),
			Note: strconv.FormatInt(rejected, 10) + " of " + strconv.FormatInt(toolCalls, 10) + " tool calls declined",
		},
	}
	r.Caveats = reworkCaveats(rejectionKnown)
	r.Takeaway = reworkTakeaway(low)
	return r
}

// rejectionLow reports whether the rejection rate is confirmed within the watch ceiling.
// Unknown (no tool calls recorded this window) never reads as low: there is no rejection
// signal to confirm, so the pair can't be certified low on a fabricated zero.
func rejectionLow(rejectionRate float64, known bool) bool {
	return known && rejectionRate <= reworkWatchCeiling
}

// reworkPurity averages known rates only: an unknown rejection rate (no tool calls) is
// excluded rather than folded in as a fabricated zero, which would inflate purity above
// what the actually-observed rework signal supports.
func reworkPurity(reworkRate, rejectionRate float64, rejectionKnown bool) float64 {
	if !rejectionKnown {
		return clamp01(1 - reworkRate)
	}
	return clamp01(1 - (reworkRate+rejectionRate)/2)
}

func reworkCaveats(rejectionKnown bool) []string {
	caveats := []string{"Evidence on AI churn's real-world impact is contested; bug/survival impact needs the server stage."}
	if !rejectionKnown {
		caveats = append(caveats, "No tool calls recorded this window -- rejection rate is unconfirmed, not zero.")
	}
	return caveats
}

func reworkTakeaway(low bool) string {
	if low {
		return "Rework and rejection are both low."
	}
	return "Rework or rejection is elevated or unconfirmed -- worth a closer look."
}
