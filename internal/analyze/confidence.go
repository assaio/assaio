package analyze

import (
	"fmt"
	"time"
)

// The confidence labels, from a verdict that rests on complete data to one that rests on
// none. "insufficient" is deliberately distinct from a neutral read: one says the data
// gave no answer, the other says there was not enough data to ask.
const (
	ConfidenceHigh         = "high"
	ConfidenceMedium       = "medium"
	ConfidenceLow          = "low"
	ConfidenceInsufficient = "insufficient"
)

const (
	// confidenceStrongFloor is the share every axis must clear for a confident read -- the
	// same floor the coverage validator already calls "solid".
	confidenceStrongFloor = 0.8
	// confidenceWeakFloor is where a thin axis stops being a caveat and starts being the
	// headline.
	confidenceWeakFloor = 0.5
	// confidenceSampleFloor is the count below which no coverage figure rescues a verdict,
	// matching the floor `context` already applies before it will read a session mix.
	confidenceSampleFloor = 3
)

// Confidence is what a Result rests on. It is deliberately not one opaque score: Label may
// summarize for a surface that has room for one word, but the components stay inspectable,
// because "which axis is thin" is the part a reader can act on. A low-coverage verdict that
// travels without this can be quoted as a solid one.
type Confidence struct {
	// Label summarizes the components below: high, medium, low, or insufficient.
	Label string `json:"label"`
	// Activity is the share of the window's tokens from sources that extract line and edit
	// signals, Priced the share on models with a known price, and Turn the share from
	// per-turn rather than whole-session records. Each 0..1, and the weakest one sets Label.
	Activity float64 `json:"activityCoverage"`
	Priced   float64 `json:"pricedCoverage"`
	Turn     float64 `json:"turnCoverage"`
	// Samples is how many observations this verdict rests on and Unit names what they are
	// ("sessions", "active days"). Set by the validator, the only thing that knows what it
	// counted; a validator that counted nothing reports zero and reads as insufficient.
	Samples int    `json:"samples"`
	Unit    string `json:"samplesUnit,omitempty"`
	// Ingested is when the newest data behind this verdict was read, and ParsedBy the build
	// that read it -- a figure can be perfectly covered and still be a week stale, or come
	// from a parser that did not yet extract a field a metric needs. Neither moves Label;
	// both answer questions Label cannot.
	Ingested time.Time `json:"ingested,omitempty"`
	ParsedBy string    `json:"parsedBy,omitempty"`
}

// Evaluate runs v and stamps the window-level confidence onto its Result, so every verdict
// carries what it rests on without each validator assembling it. The split is deliberate:
// the framework fills what is true of the whole window, and the validator contributes only
// what it alone knows -- how many observations it counted.
func Evaluate(v Validator, in *Input) Result {
	r := v.Analyze(*in)
	Stamp(&r, in)
	return r
}

// Stamp fills the window-level components on an already-computed Result and derives its
// label. Exported for the one Result assaio does not compute itself: an exec metric
// plugin's. Its verdict rests on the same window as every built-in one, so it carries the
// same coverage -- and a plugin that declared no sample basis reads as insufficient, which
// is the honest reading of "did not say what it rests on".
func Stamp(r *Result, in *Input) {
	r.Confidence.Activity, r.Confidence.Priced, r.Confidence.Turn = coverageShares(in)
	r.Confidence.Ingested, r.Confidence.ParsedBy = in.Ingested, in.ParsedBy
	r.Confidence.Label = r.Confidence.derive()
}

// derive reduces the components to one word, weakest axis first.
func (c *Confidence) derive() string {
	if c.Samples <= 0 {
		return ConfidenceInsufficient
	}
	if c.Samples < confidenceSampleFloor {
		return ConfidenceLow
	}
	switch weakest := min(c.Activity, c.Priced, c.Turn); {
	case weakest >= confidenceStrongFloor:
		return ConfidenceHigh
	case weakest >= confidenceWeakFloor:
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

// coverageShares are the three window-level axes every verdict rests on. The coverage
// validator reports the same three as its own subject, reading them from here so the
// figures it prints and the envelope every other metric carries cannot disagree.
func coverageShares(in *Input) (activity, priced, turn float64) {
	if in.Totals.Tokens == 0 {
		return 0, 0, 0
	}
	activityTokens, _ := tokensByTool(in.Usage)
	turnShare, _ := turnGranularityShare(in.Usage)
	return fracOf(activityTokens, in.Totals.Tokens),
		fracOf(pricedTokenSum(in.ByModel), in.Totals.Tokens),
		turnShare
}

// restsOn records what this verdict was computed from: how many observations, and what
// they are. Every validator calls it once, at the point it knows its own subject -- the
// framework cannot, since "how much evidence" means sessions for one metric, active days
// for another, and tool calls for a third. A validator that returns before calling it has
// counted nothing, which is exactly what reads as insufficient.
func (r *Result) restsOn(n int, unit string) {
	r.Confidence.Samples, r.Confidence.Unit = n, unit
}

// activeDays is the observation count for every window-share metric: the number of days
// the window actually contains usage for, not the number of days it spans.
func activeDays(in *Input) int { return len(tokensPerDay(in.Usage)) }

// ConfidenceSummary renders the envelope in one line: the label, what it was counted from,
// and the weakest coverage axis when one of them is what held the label down. Naming the
// axis is the point -- "medium" alone tells a reader nothing they can act on. Exported so
// the text report and the dashboard render the same sentence rather than two that can drift.
func ConfidenceSummary(c *Confidence) string {
	if c.Label == ConfidenceInsufficient {
		return ConfidenceInsufficient + " — nothing to measure in this window"
	}
	line := fmt.Sprintf("%s · %d %s", c.Label, c.Samples, c.Unit)
	if axis, share, weak := weakestAxis(c); weak {
		line += fmt.Sprintf(" · %s coverage %s", axis, formatPercent(share, 0))
	}
	return line
}

// weakestAxis names the coverage component holding the label back, if any is below the
// strong floor.
func weakestAxis(c *Confidence) (name string, share float64, weak bool) {
	axes := []struct {
		name  string
		share float64
	}{{"activity", c.Activity}, {"priced", c.Priced}, {"turn-level", c.Turn}}
	name, share = axes[0].name, axes[0].share
	for _, a := range axes[1:] {
		if a.share < share {
			name, share = a.name, a.share
		}
	}
	return name, share, share < confidenceStrongFloor
}
