package reconcile

import "github.com/assaio/assaio/internal/report"

// Estimate is what the local logs say, restricted to the overlap. Priced and unpriced are
// kept apart all the way through: their sum is not a better number, it is a different
// claim, and only one of them is a measurement.
type Estimate struct {
	// Priced is cost assaio computed from a known model price -- the figure it stands
	// behind. It is a lower bound on real spend whenever UnpricedTokens is non-zero.
	Priced float64 `json:"priced"`
	// UnpricedTokens is usage assaio read but could not price, and Extrapolated is what
	// those tokens would cost at the window's own observed rate. Extrapolated is an
	// extrapolation, never added to Priced -- it is the width of the band instead.
	UnpricedTokens int64   `json:"unpriced_tokens"`
	Extrapolated   float64 `json:"extrapolated"`
	Tokens         int64   `json:"tokens"`
	// UnpricedRows counts rows with no known price, which is not the same question as
	// UnpricedTokens: a row can be unpriced and carry no tokens to extrapolate from, and
	// reporting a point band as "every row carried a price" would be wrong about it.
	UnpricedRows int `json:"unpriced_rows"`
	// ByModel is priced cost per model, and ByDay per date; both restricted to the overlap.
	ByModel map[string]float64 `json:"by_model"`
	ByDay   map[string]float64 `json:"by_day"`
}

// Upper is the top of the band the estimate could honestly claim.
func (e *Estimate) Upper() float64 { return e.Priced + e.Extrapolated }

// Banded reports whether the estimate is a range rather than a point.
func (e *Estimate) Banded() bool { return e.Extrapolated > 0 }

// estimateDays sums priced cost per date across every row, before any scope restriction.
// buildScope needs this to see which dates the local side covers at all.
func estimateDays(rows []report.Row) map[string]float64 {
	out := make(map[string]float64, len(rows))
	for i := range rows {
		r := &rows[i]
		if r.Cost != nil {
			out[r.Day] += *r.Cost
		} else if _, seen := out[r.Day]; !seen {
			out[r.Day] = 0
		}
	}
	return out
}

// buildEstimate sums the local side over the overlap only. The extrapolation rate is the
// window's own priced cost per token, which is crude by construction -- a cache read and a
// completion token do not cost the same -- and is why it sizes a band instead of becoming a
// figure of its own.
func buildEstimate(rows []report.Row, s *Scope) Estimate {
	e := Estimate{ByModel: map[string]float64{}, ByDay: map[string]float64{}}
	var pricedTokens, unpricedTokens int64
	for i := range rows {
		r := &rows[i]
		if !s.Overlap(r.Day) {
			continue
		}
		tokens := rowTokens(r)
		e.Tokens += tokens
		if r.Priced && r.Cost != nil {
			e.Priced += *r.Cost
			e.ByModel[r.Model] += *r.Cost
			e.ByDay[r.Day] += *r.Cost
			pricedTokens += tokens
			continue
		}
		unpricedTokens += tokens
		e.UnpricedRows++
	}
	e.UnpricedTokens = unpricedTokens
	if pricedTokens > 0 && unpricedTokens > 0 {
		e.Extrapolated = e.Priced / float64(pricedTokens) * float64(unpricedTokens)
	}
	return e
}

func rowTokens(r *report.Row) int64 {
	return r.In + r.Out + r.CacheRead + r.CacheWrite
}
