package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/humanize"
)

const (
	burnName     = "burn-anomaly"
	burnTitle    = "Burn Anomaly"
	burnDescribe = "Days whose token burn stands far outside this window's typical day."
	// burnHowToRead is Result.HowToRead for this validator -- see its doc comment.
	burnHowToRead = "A spike is a prompt to go look, not a fault: a migration or a long refactor legitimately burns more. What it rules out is a spike nobody noticed -- a runaway loop, a re-ingested backfill, or an agent left running."
	// burnMinDays is the day floor below which a baseline is meaningless: a handful of
	// days has no typical day to stand out from.
	burnMinDays = 7
	// burnZThreshold is the modified z-score above which a day counts as a spike. 3.5 is
	// the conventional cutoff for the median/MAD form, which -- unlike a mean and standard
	// deviation -- is not itself dragged upward by the very outlier being tested for.
	burnZThreshold = 3.5
	// burnMADToSigma scales the median absolute deviation onto the standard-deviation
	// scale for normally distributed data, so burnZThreshold keeps its conventional
	// meaning.
	burnMADToSigma = 1 / 0.6745
	// burnMeanADToSigma scales a mean absolute deviation the same way. It is the fallback
	// for a zero median absolute deviation, which happens whenever more than half the days
	// share one value -- precisely the flat baseline a single spike stands out from, where
	// the median form would otherwise report no spike at all.
	burnMeanADToSigma = 1.253314
	burnTopN          = 5
)

func init() { Register(burnValidator{}) }

// burnValidator reads daily token burn for outlier days -- the spend equivalent of a
// smoke alarm, robust to a window where most days differ wildly.
type burnValidator struct{}

func (burnValidator) Name() string     { return burnName }
func (burnValidator) Title() string    { return burnTitle }
func (burnValidator) Describe() string { return burnDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (burnValidator) Analyze(in Input) Result {
	r := Result{Name: burnName, Title: burnTitle, Describe: burnDescribe, HowToRead: burnHowToRead}
	days := tokensPerDay(in.Usage)
	if len(days) == 0 {
		r.noData("active days", "No usage in this window.")
		return r
	}
	r.restsOn(len(days), "active days")
	median := medianTokens(days)
	sufficient := len(days) >= burnMinDays
	spikes := burnSpikes(days, median)
	steady := sufficient && len(spikes) == 0

	r.Read = burnRead(sufficient, steady)
	r.Purity = burnPurity(sufficient, len(spikes), len(days))
	r.Figures = []Figure{
		{Label: "active days", Value: strconv.Itoa(len(days))},
		{Label: "typical day", Value: humanize.Count(median), Note: "median tokens"},
		burnPeakFigure(days, median),
		burnSpikeFigure(sufficient, len(spikes)),
	}
	r.Bars = burnBars(spikes, median, burnTopN)
	r.Takeaway = burnTakeaway(sufficient, steady)
	r.Caveats = append(r.Caveats, "Prov.: days are bucketed in UTC, so an evening that runs past midnight UTC is split across two of them -- inflating the active-day count and halving the typical day it is measured against.")
	return r
}

// burnRead reports the neutral no-verdict Read while the window is too short to hold a
// baseline, rather than a favorable one earned by having nothing to compare against.
func burnRead(sufficient, steady bool) Read {
	if !sufficient {
		return noDataRead
	}
	return readFor(steady, "Steady")
}

// burnPurity is full when no day stands out, falling as spike days take a larger share of
// the window; too short a window yields the neutral 0.5 used for "not enough evidence yet".
func burnPurity(sufficient bool, spikes, days int) float64 {
	if !sufficient || days == 0 {
		return 0.5
	}
	return clamp01(1 - float64(spikes)/float64(days))
}

// burnPeakFigure reports the window's heaviest day as a multiple of the typical day, the
// form that makes an outlier legible at a glance.
func burnPeakFigure(days []dayBurn, median int64) Figure {
	var peak int64
	for i := range days {
		if days[i].Tokens > peak {
			peak = days[i].Tokens
		}
	}
	f := Figure{Label: "heaviest day", Value: humanize.Count(peak), Note: "tokens"}
	if median > 0 {
		f.Note = strconv.FormatFloat(float64(peak)/float64(median), 'f', 1, 64) + "× the typical day"
	}
	return f
}

// burnSpikeFigure counts the spike days, "—" below the day floor where no day can be told
// apart from an ordinary one -- a zero there would read as "none found" rather than "not
// looked for".
func burnSpikeFigure(sufficient bool, spikes int) Figure {
	if !sufficient {
		return Figure{Label: "spike days", Value: "—"}
	}
	return Figure{Label: "spike days", Value: strconv.Itoa(spikes)}
}

// burnBars lists the spike days, largest first. The slice is always non-nil so a window
// with no spikes renders the honest "none in this window" line rather than no section at
// all. topN <= 0 is unlimited.
func burnBars(spikes []dayBurn, median int64, topN int) []Bar {
	kept := spikes
	if topN > 0 && len(kept) > topN {
		kept = kept[:topN]
	}
	var maxTokens int64
	if len(kept) > 0 {
		maxTokens = kept[0].Tokens
	}
	bars := make([]Bar, 0, len(kept))
	for i := range kept {
		value := humanize.Count(kept[i].Tokens) + " tokens"
		if median > 0 {
			value += " · " + strconv.FormatFloat(float64(kept[i].Tokens)/float64(median), 'f', 1, 64) + "×"
		}
		bars = append(bars, Bar{Label: kept[i].Day, Value: value, Frac: fracOf(kept[i].Tokens, maxTokens)})
	}
	return bars
}

func burnTakeaway(sufficient, steady bool) string {
	switch {
	case !sufficient:
		return "Too few active days this window to tell an unusual day from an ordinary one."
	case steady:
		return "No day burned far outside this window's normal range."
	default:
		return "Some days burned far above the typical day -- worth checking what ran on them."
	}
}
