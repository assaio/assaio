package analyze

import (
	"strings"
	"time"

	"github.com/assaio/assaio/internal/threshold"
)

// citationLines is the published-threshold register's answer for one metric, rendered beside
// the verdict on the same path provenance and confidence already take (Result.Caveats, which
// both the text report and the dashboard print). Two shapes come back and the difference
// between them is the point: one says a figure has an authority behind it, the other names the
// published figure a reader would otherwise reach for and says why it is not this one.
//
// Naming the refused citation is not decoration. assaio prints a rework percentage and GitClear
// prints a churn percentage; a reader who is told nothing will subtract them, and the silence
// assaio leaves is the thing that lets them.
//
// The candidates are passed in rather than looked up here so a validator's citation rendering
// is testable without registering a fixture into the process-wide register.
func citationLines(candidates []threshold.Candidate, now time.Time) []string {
	var out []string
	for i := range candidates {
		c := &candidates[i]
		// Expiry is asked first and the order is load-bearing. Every candidate registered today is
		// unfit, so a fit-first switch reaches the expired branch for none of them and quotes a
		// superseded figure forever: the source restates these series annually, and a reader shown
		// last edition's number with no date beside it cannot know that.
		switch {
		case !c.Citation.Current(now):
			out = append(out, expiredCitation(c))
		case c.Fits():
			out = append(out, appliedCitation(c))
		default:
			out = append(out, unfitCitation(c))
		}
	}
	return out
}

// CitationLines is citationLines for a surface that computes its figure outside this package.
// `survival` is one: no validator produces it, so the CLI renders the register's answer itself
// and has to render the same shape, or the same refusal reads as two different claims.
func CitationLines(candidates []threshold.Candidate, now time.Time) []string {
	return citationLines(candidates, now)
}

// appliedCitation is the line a graded figure rests on: the source, the population, and the
// date the grade stops being defensible.
func appliedCitation(c *threshold.Candidate) string {
	return "Line: " + c.Citation.Cite() + ", " + population(c) +
		"How that population differs from yours: " + c.Differs + ". This line is checked out to " +
		c.Citation.Expiry() + "; past that the verdict returns to withheld until the source is read again."
}

// unfitCitation names a published figure that measures something else. It states the source's
// own definition rather than a paraphrase bent toward assaio's, because the reader has to be
// able to see the two definitions side by side and reach the same conclusion.
func unfitCitation(c *threshold.Candidate) string {
	return "Not a line here: " + c.Citation.Cite() + ", " + population(c) +
		"It counts " + c.Citation.Definition + ", which is not what this figure counts -- " +
		propertyList(c.Differences()) + " differ. " + c.Differs +
		". Both are percentages and neither is the other, so assaio names the study rather than grading against it."
}

// expiredCitation is a citation past its declared validity, whether or not the pairing ever fit.
// Fit stops mattering at that date: the figure behind it has been restated at the source, nothing
// local can tell how far, and quoting either half of a superseded study -- as a line or as the
// study a reader is told not to subtract -- states a number nobody has re-read.
func expiredCitation(c *threshold.Candidate) string {
	return "Not a line here: " + c.Citation.Cite() +
		" stopped counting on " + c.Citation.Expiry() +
		", so nothing here is graded against it, or named beside it, until the source is read again and the register updated."
}

// population renders the citation's population as a finished clause. Population is prose entered
// by whoever registered the citation and may or may not carry its own full stop; the sentence
// boundary belongs to the formatter, since a caller that relied on the data to supply it renders
// correctly only for the entries that happen to.
func population(c *threshold.Candidate) string {
	return "over " + strings.TrimSuffix(strings.TrimSpace(c.Citation.Population), ".") + ". "
}

// propertyList renders the differing properties in the order a reader checks them.
func propertyList(ps []threshold.Property) string {
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = string(p)
	}
	return strings.Join(names, ", ")
}
