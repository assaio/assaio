// Package threshold is the register of published figures a verdict could rest on, and the
// check that each one still answers the question assaio's own metric asks. A verdict needs a
// line and a line needs an authority (internal/analyze/unsourced.go); this is where an
// authority is admitted, and where one is refused.
//
// Nothing here is a default, a target or a tuned constant. Every entry names a publication,
// its URL, the population it measured, the date its measurement ends, and the date it stops
// counting -- the same discipline internal/pricing applies to a vendored price table, for the
// same reason: a citation with no expiry stops being a reference and becomes a fact nobody
// re-read.
//
// A register that held only the figures assaio does use would be the less useful half. The
// refusals are the record: two rates are both percentages, and a reader who is not told they
// count different things will compare them anyway.
package threshold

import (
	"strings"
	"time"
)

// dateLayout is how every date in a rendered citation is written: unambiguous between readers
// who order a date differently.
const dateLayout = "2006-01-02"

// Property is one of the five things two rates must share before either may judge the other. A
// rate is defined by what it counts, what it counts that out of, over what span, from what
// evidence, and about what kind of claim. Two rates agreeing on some of those and not the rest
// are different quantities that happen to share a unit.
type Property string

// The five defining properties, in the order a reader checks them.
const (
	Numerator   Property = "numerator"
	Denominator Property = "denominator"
	Window      Property = "window"
	DataSource  Property = "data source"
	Layer       Property = "layer"
)

// Properties is the closed set, in the order a reader checks them. An adjudication addresses
// every one of them: a property nobody compared is a difference nobody looked for, which is
// how an unfit citation gets attached.
func Properties() []Property {
	return []Property{Numerator, Denominator, Window, DataSource, Layer}
}

// Comparison is one property stated twice -- as the citation's population defines it, and as
// assaio computes it -- with a declared verdict on whether the two are the same quantity. Same
// is declared rather than derived: no code can read two prose definitions and decide they
// agree. Its zero value is the safe one, so an entry someone half-filled in withholds rather
// than grades.
type Comparison struct {
	Property Property
	Cited    string
	Assaio   string
	Same     bool
}

// Citation is one published figure, carried with everything a reader needs to check it at the
// source rather than take assaio's word for it.
type Citation struct {
	Value string
	// Definition is what the source counts, in the source's own terms -- not a paraphrase that
	// has already been bent toward what assaio counts.
	Definition string
	Source     string
	URL        string
	// Population is what was measured, how much of it, and over what period.
	Population string
	// Measured is the last date the population covers, which is not the publication date. A
	// figure the report quotes for a later year is its forecast, not its measurement, and the
	// distinction is invisible once the number is quoted on its own.
	Measured time.Time
	// Checked is when assaio last read the source and confirmed the fields above.
	Checked time.Time
	// ValidUntil is when this figure stops carrying a verdict. It is mandatory and enforced
	// (see validate): the populations behind figures like these are restated every year, and an
	// unexpiring citation quietly grades this year's work against a superseded number. Past it
	// the verdict returns to withheld.
	ValidUntil time.Time
}

// Current reports whether c may still carry a verdict at now. A zero now is never current: a
// caller that cannot say when it is asking must not be handed a live citation, because the one
// thing this type promises is that an out-of-date figure stops grading.
func (c *Citation) Current(now time.Time) bool {
	return !now.IsZero() && now.Before(c.ValidUntil)
}

// Cite is the attribution rendered beside any figure that quotes c, in one line. It lives here
// rather than in each surface so a citation printed by the text report and one printed by the
// dashboard cannot become two different claims about the same source.
//
// The address is rendered without its scheme, which looks like a typo and is not. One of the
// surfaces this line reaches is the Assay dashboard, a file people email and post, and that
// file is required to carry no external address at all -- internal/dashboard's
// TestRenderHTMLSelfContained fails the build on a literal "https://" anywhere in it. A
// citation with no address is the worse answer, so the scheme goes and the address stays: it
// is still what a reader types, and it still loads nothing when the page is opened.
func (c *Citation) Cite() string {
	return c.Source + " (" + strings.TrimPrefix(c.URL, "https://") + "), measured through " +
		c.Measured.Format(dateLayout) + ": " + c.Value
}

// Expiry is when c stops counting, for a surface that has to say so.
func (c *Citation) Expiry() string { return c.ValidUntil.Format(dateLayout) }

// Candidate is one published figure weighed against one assaio metric. The pairing is what the
// register keeps, not the figure alone: the same churn study asks a different question of
// within-session rework than it does of git survival, and each pairing has its own answer.
type Candidate struct {
	// Metric is the assaio measurement this figure was weighed against. It is a measurement's
	// name, not necessarily a registered validator's: a candidate is adjudicated wherever the
	// comparison would be tempting, including where no surface reads the register yet.
	Metric   string
	Citation Citation
	// Differs is how assaio's population differs from the source's, in a sentence a reader can
	// act on. Required even when every property matches, because two studies can count the same
	// quantity over populations that are still not comparable.
	Differs string
	Fit     []Comparison
}

// Fits reports whether every defining property is the same quantity. An empty Fit is not a
// match: nothing was compared.
func (c *Candidate) Fits() bool {
	for i := range c.Fit {
		if !c.Fit[i].Same {
			return false
		}
	}
	return len(c.Fit) > 0
}

// Applies reports whether c may set a line for its metric at now: the same quantity, from a
// citation that has not expired. Both halves are required, and either one failing returns the
// verdict to withheld.
func (c *Candidate) Applies(now time.Time) bool {
	return c.Fits() && c.Citation.Current(now)
}

// Differences are the properties that are not the same quantity, in Properties order.
func (c *Candidate) Differences() []Property {
	var out []Property
	for _, p := range Properties() {
		for i := range c.Fit {
			if c.Fit[i].Property == p && !c.Fit[i].Same {
				out = append(out, p)
			}
		}
	}
	return out
}
