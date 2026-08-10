// Package reconcile compares a vendor's own billing or usage export against the estimate
// assaio computed from local logs. It is the only external oracle the project can have for
// aggregate tokens and money, and it is deliberately offline: the export is a file the user
// downloaded, never a credentialed pull.
//
// The one rule the package exists to hold: no figure is ever adjusted to make the two sides
// agree. Both totals travel through unchanged, named causes are subtracted only where they
// are computable from evidence, and whatever is left is reported as unexplained rather than
// absorbed. A delta this package cannot explain is its output, not its failure.
package reconcile

import "time"

// Row is one line of an imported export, reduced to the fields a reconciliation can use.
// Day is the vendor's own bucket date, normalized to YYYY-MM-DD. Model is "" when the
// export does not break spend down by model, which costs the per-model causes their
// evidence -- Result says so rather than reporting them as zero.
type Row struct {
	Day   string  `json:"day"`
	Model string  `json:"model"`
	Cost  float64 `json:"cost"`
	// Tokens is the export's own total token count, and Stated reports whether the export
	// carried one at all. A vendor export that bills only in money leaves this false; a
	// zero would read as "the vendor says no tokens".
	Tokens       int64 `json:"tokens"`
	TokensStated bool  `json:"tokens_stated"`
}

// Export is a parsed vendor export together with everything a reader needs to judge it:
// where it came from, which column was read as what, and what was skipped getting here.
type Export struct {
	Rows []Row `json:"rows"`
	// Source is the file path the export was read from, and Binding records which of its
	// columns became which field. Both are printed: a reconciliation whose reader guessed
	// wrong is worse than no reconciliation, and only the operator can catch that.
	Source  string  `json:"source"`
	Binding Binding `json:"binding"`
	// Currency is the export's stated currency, "" when it states none. Anything other
	// than USD stops the run: assaio prices in USD and converting would invent a rate.
	Currency string `json:"currency"`
	// Skipped counts rows the reader could not parse, and Total counts data rows seen.
	// Skipping is the parser contract's policy; hiding how much was skipped is not.
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
}

// Cost sums the export's own money over the rows in days, ignoring rows outside it.
func (e *Export) Cost(days map[string]bool) float64 {
	var sum float64
	for i := range e.Rows {
		if days == nil || days[e.Rows[i].Day] {
			sum += e.Rows[i].Cost
		}
	}
	return sum
}

// Days returns the set of dates the export covers.
func (e *Export) Days() map[string]bool {
	out := make(map[string]bool, len(e.Rows))
	for i := range e.Rows {
		out[e.Rows[i].Day] = true
	}
	return out
}

// Span returns the export's first and last date, both "" when it has no rows.
func (e *Export) Span() (first, last string) {
	for i := range e.Rows {
		d := e.Rows[i].Day
		if first == "" || d < first {
			first = d
		}
		if last == "" || d > last {
			last = d
		}
	}
	return first, last
}

// day renders t as the YYYY-MM-DD bucket the store and every export share.
func day(t time.Time) string { return t.UTC().Format("2006-01-02") }
