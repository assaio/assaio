package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
)

const (
	frictionName     = "friction"
	frictionTitle    = "Friction"
	frictionDescribe = "How often tool calls fail outright, beside how often a human declines one."
	// frictionHowToRead is Result.HowToRead for this validator -- see its doc comment.
	frictionHowToRead = "Some failure is normal -- an agent probes, a command exits non-zero, it adapts. A rising error rate is the useful signal: it usually means a broken command, a missing dependency, or a permission the agent keeps hitting, and every retry is paid for twice."
	// frictionWatchCeiling is the tool-error rate above which failures stop looking like
	// ordinary probing.
	frictionWatchCeiling = 0.15
	// frictionMinCalls is the tool-call floor below which a rate is too thin to call.
	frictionMinCalls = 30
)

func init() { Register(frictionValidator{}) }

// frictionValidator reads how much of a window's tool work failed or was declined -- the
// tokens spent on calls that produced nothing.
type frictionValidator struct{}

func (frictionValidator) Name() string     { return frictionName }
func (frictionValidator) Title() string    { return frictionTitle }
func (frictionValidator) Describe() string { return frictionDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (frictionValidator) Analyze(in Input) Result {
	r := Result{Name: frictionName, Title: frictionTitle, Describe: frictionDescribe, HowToRead: frictionHowToRead}
	if len(in.Usage) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	f := buildFriction(in.Usage)
	r.restsOn(int(f.Observed), "tool calls")
	r.covering(f.Coverage())
	sufficient := f.Observed >= frictionMinCalls && f.Coherent()
	lowErrors := f.ErrorRate() <= frictionWatchCeiling
	smooth := sufficient && lowErrors

	r.Read = frictionRead(sufficient, smooth)
	r.Purity = frictionPurity(f, sufficient)
	r.Figures = []Figure{
		{Label: "calls with failure capture", Value: strconv.FormatInt(f.Observed, 10), Note: "of " + strconv.FormatInt(f.Calls, 10) + " tool calls"},
		frictionRateFigure("error rate", f.Errors, f),
		// Rejections are counted on every tool call, including history ingested before
		// failure capture existed, so their denominator is every call -- not the
		// failure-observed subset the error rate divides by.
		{Label: "rejection rate", Value: shareOrDash(f.Rejected, f.Calls, 1), Note: strconv.FormatInt(f.Rejected, 10) + " declined by a human, of all calls"},
	}
	r.Takeaway = frictionTakeaway(sufficient, smooth, f)
	r.Caveats = append(r.Caveats, frictionCoverageCaveat(f))
	return r
}

// friction is the window's tool-call outcome counts. Observed is the subset of Calls whose
// failure state was actually recorded; a rate over Calls would count rows that cannot fail
// by construction and report the resulting zero as a finding.
type friction struct {
	Calls, Observed, Errors, Rejected int64
}

// buildFriction sums the window's tool-call outcomes. A row joins Observed -- and
// contributes its errors -- only when its tool records the outcome of every call it makes
// and the row was actually parsed by a build that captured it. Anything else stays in Calls
// alone: counting a call that cannot report a failure would dilute the rate toward zero and
// let the caveat claim coverage the window does not have.
func buildFriction(rows []store.UsageRow) friction {
	var f friction
	for i := range rows {
		f.Calls += rows[i].ToolCalls
		f.Rejected += rows[i].Rejected
		if !failureCapableTool(rows[i].Tool) || classifiedCalls(&rows[i]) == 0 {
			continue
		}
		f.Observed += rows[i].ToolCalls
		f.Errors += rows[i].ToolErrors
	}
	return f
}

// failureCapableTool reports whether a tool marks the outcome of every tool call, not just
// some -- asked of the depth matrix, so widening it is a change to that one row rather than
// to a name repeated here. A source that marks only some outcomes answers no error signal:
// its calls would sit in a denominator they can never appear in the numerator of.
func failureCapableTool(tool string) bool { return parser.Answers(tool, "ai.tool_errors.count") }

// Coherent reports whether the error count can be read as a share of the calls it was
// counted over. A tool may record a failure for work that never registered as a counted
// call, and once the errors outnumber those calls no percentage over them means anything.
func (f friction) Coherent() bool { return f.Errors <= f.Observed }

// Coverage is the share of the window's tool calls whose failure state was recorded.
func (f friction) Coverage() float64 { return shareOf(f.Observed, f.Calls) }

// ErrorRate is the share of failure-observed tool calls that came back an error, 0 when
// there were none.
func (f friction) ErrorRate() float64 { return shareOf(f.Errors, f.Observed) }

// frictionRead reports the neutral no-verdict Read while too few calls exist to trust a
// rate, rather than a favorable one earned from a handful of calls.
func frictionRead(sufficient, smooth bool) Read {
	if !sufficient {
		return noDataRead
	}
	return readFor(smooth, "Smooth")
}

// frictionPurity falls as the error rate rises, reaching zero at the point where a third of
// calls fail; too thin a sample yields the neutral 0.5 used for "not enough evidence yet".
func frictionPurity(f friction, sufficient bool) float64 {
	if !sufficient {
		return 0.5
	}
	return clamp01(1 - f.ErrorRate()/0.33)
}

// frictionRateFigure renders the error share against the failure-observed calls, "—" when
// those counts cannot be read as a share -- an impossible percentage is worse than no
// number.
func frictionRateFigure(label string, n int64, f friction) Figure {
	value := "—"
	if f.Coherent() {
		value = shareOrDash(n, f.Observed, 1)
	}
	return Figure{Label: label, Value: value, Note: strconv.FormatInt(n, 10) + " failed"}
}

// frictionCoverageCaveat states how much of the window could report a failure at all, so a
// zero error count is never mistaken for "nothing failed" when it means "nothing recorded".
func frictionCoverageCaveat(f friction) string {
	return "Prov.: " + formatPercent(f.Coverage(), 0) + " of tool calls record whether they failed, and only for sessions ingested by a build that captures it -- run `backfill` after upgrading. A source that marks only some outcomes is excluded from these rates rather than counted as successes; `assaio-agent signals coverage` says which does what."
}

func frictionTakeaway(sufficient, smooth bool, f friction) string {
	switch {
	case f.Observed == 0:
		return "No tool call in this window records whether it failed, so no failure rate can be read from it."
	case !f.Coherent():
		return "Recorded failures outnumber the calls counted for this window, so no failure rate can be read from it."
	case !sufficient:
		return "Too few tool calls with failure capture this window to read a failure rate."
	case smooth:
		return "Tool calls mostly succeed -- no systematic friction in this window."
	default:
		return "A large share of tool calls fails -- worth finding the command or permission the agent keeps hitting."
	}
}
