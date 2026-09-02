package report

import (
	"sort"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// Unpriced is the part of a window's usage its cost figure cannot see: tokens on a model the
// vendored price table does not carry. Every surface that discloses the condition renders
// this one quantity, because the bare marker read identically at 0.1% and at the 45.5% that
// once left a window's estimate $15,452.42 short of what it became once the model had a price.
type Unpriced struct {
	// Tokens is billable tokens carrying no known price, of Total in the window. The share of
	// those two is the error bar on the cost; nothing else here is.
	Tokens int64
	Total  int64
	// Rows counts the rows with no known price, a different question: a row can be unpriced
	// and carry no token at all, and a window missing only those has a complete cost.
	Rows int
	// Untokened counts the unpriced rows whose source publishes no token counter at all. They
	// are unpriced for a different reason from the rest -- not a model the table has yet to
	// carry, which a refresh fixes, but a format with nothing to price, which no refresh will --
	// and telling a reader the first when it is the second sends them to look for a fix that
	// does not exist.
	Untokened int
}

// Share is the unpriced share of the window's tokens, 0 when the window holds none.
func (u *Unpriced) Share() float64 {
	if u.Total <= 0 {
		return 0
	}
	return float64(u.Tokens) / float64(u.Total)
}

// Missing reports whether the cost is understated -- unpriced usage that carries tokens.
// Unpriced rows carrying none leave the total whole and are deliberately not this.
func (u *Unpriced) Missing() bool { return u.Tokens > 0 }

// BuildUnpriced measures how much of a priced window the price table could not cost. It
// reads the pricing pass Build already made rather than repeating it, so an aggregate and
// the rows behind it can never disagree about the same share.
func BuildUnpriced(rows []Row) Unpriced {
	var u Unpriced
	for i := range rows {
		r := &rows[i]
		u.Total += r.In + r.Out + r.CacheRead + r.CacheWrite
		u.Tokens += r.UnpricedTokens
		if r.HasUnpriced {
			u.Rows++
			if !r.Tokened {
				u.Untokened++
			}
		}
	}
	return u
}

// UnpricedModels names the models carrying unpriced tokens, heaviest first -- what a price
// table refresh has to cover. It answers a question the share cannot, so it is separate: the
// share says how wrong the cost is, this says what would fix it.
func UnpricedModels(rows []store.UsageRow, t pricing.Table) []string {
	byModel := make(map[string]int64)
	for i := range rows {
		r := &rows[i]
		tokens := r.In + r.Out + r.CacheRead + r.CacheWrite
		if tokens == 0 {
			continue
		}
		if _, ok := t.CostTokens(r.Model, pricing.Tokens{
			In: r.In, Out: r.Out, CacheWrite: r.CacheWrite, CacheRead: r.CacheRead, CacheWrite1h: r.CacheWrite1h,
		}); ok {
			continue
		}
		byModel[r.Model] += tokens
	}
	out := make([]string, 0, len(byModel))
	for model := range byModel {
		out = append(out, model)
	}
	sort.Slice(out, func(i, j int) bool {
		if byModel[out[i]] != byModel[out[j]] {
			return byModel[out[i]] > byModel[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}
