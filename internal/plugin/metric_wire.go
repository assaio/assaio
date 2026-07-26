package plugin

import "github.com/assaio/assaio/internal/analyze"

// wireResult is what a metric plugin actually writes on the wire. It exists so the decoder
// can reject unknown fields -- a misspelled key must not silently disarm a setting -- while
// still accepting the one key that was renamed after plugins had already shipped against it.
type wireResult struct {
	analyze.Result
	// LegacyBarsAreProjects is the pre-rename spelling of Result.BarsPseudonym. Plugins
	// released against it keep working, and keep being pseudonymized: dropping the key on
	// the floor would have published the real project names those plugins rank by.
	LegacyBarsAreProjects bool `json:"barsAreProjects,omitempty"`
}

// result folds the legacy key into the current field. An explicit BarsPseudonym always
// wins, so a plugin that sets both is read at its word rather than downgraded.
func (w *wireResult) result() analyze.Result {
	r := w.Result
	if r.BarsPseudonym == "" && w.LegacyBarsAreProjects {
		r.BarsPseudonym = analyze.PseudonymProject
	}
	return r
}
