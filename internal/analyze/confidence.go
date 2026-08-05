package analyze

import (
	"fmt"
	"time"

	"github.com/assaio/assaio/internal/parser"
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
	// Signal is the share of the window this verdict's own subject actually covers, 0..1.
	// The three axes above describe the window; this one describes the question, and only
	// the metric can answer it -- reasoning-share computed off under 1% of the output is a
	// real figure about a sliver, and nothing at window level can see that. It is a pointer
	// because "covers none of the window" and "did not say" are different answers and only
	// the first one is insufficient; absent means the whole window, the common case.
	Signal *float64 `json:"signalCoverage,omitempty"`
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

// derive reduces the components to one word, weakest axis first. Counting nothing and
// covering none of the window are the same answer: there was not enough data to ask.
func (c *Confidence) derive() string {
	if c.Samples <= 0 || (c.Signal != nil && *c.Signal <= 0) {
		return ConfidenceInsufficient
	}
	if c.Samples < confidenceSampleFloor {
		return ConfidenceLow
	}
	switch weakest := min(c.Activity, c.Priced, c.Turn, c.signalShare()); {
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
	activityTokens := capableTokens(tokensByTool(in.Usage), parser.HasFullActivity)
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

// covering is Confidence.Covering for a validator holding a Result, matching restsOn above.
func (r *Result) covering(share float64) { r.Confidence.Covering(share) }

// Covering records how much of the window this verdict's subject describes, 0..1. A metric
// whose figure is computed from part of the window declares that part; leaving it undeclared
// claims the whole window, which is what most metrics honestly do. Exported for the same
// reason Stamp is: a metric plugin's verdict answers for its reach like a built-in's does.
func (c *Confidence) Covering(share float64) {
	s := clamp01(share)
	c.Signal = &s
}

// signalShare is the declared coverage, or the whole window when none was declared.
func (c *Confidence) signalShare() float64 {
	if c.Signal == nil {
		return 1
	}
	return *c.Signal
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
		return ConfidenceInsufficient + " — " + c.insufficientReason()
	}
	line := fmt.Sprintf("%s · %d %s", c.Label, c.Samples, c.Unit)
	if axis, share, weak := weakestAxis(c); weak {
		line += fmt.Sprintf(" · %s coverage %s", axis, honestPercent(share))
	}
	return line
}

// insufficientReason says which of the three ways a verdict can rest on nothing applies.
// They are different facts about different things: the window may be full of usage none of
// which can answer this question, the metric may have counted none of its own observations,
// or it may never have said what it rests on -- which is what an exec metric plugin that
// omits its sample basis means. One sentence for all three read a full store as an empty one.
func (c *Confidence) insufficientReason() string {
	switch {
	case c.Signal != nil && *c.Signal <= 0:
		return "nothing in this window can answer it"
	case c.Unit != "":
		return "0 " + c.Unit
	default:
		return "no stated basis"
	}
}

// weakestAxis names the coverage component holding the label back, if any is below the
// strong floor.
func weakestAxis(c *Confidence) (name string, share float64, weak bool) {
	axes := []struct {
		name  string
		share float64
	}{{"activity", c.Activity}, {"priced", c.Priced}, {"turn-level", c.Turn}, {"signal", c.signalShare()}}
	name, share = axes[0].name, axes[0].share
	for _, a := range axes[1:] {
		if a.share < share {
			name, share = a.name, a.share
		}
	}
	return name, share, share < confidenceStrongFloor
}
