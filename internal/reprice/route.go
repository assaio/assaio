package reprice

import (
	"sort"

	"github.com/assaio/assaio/internal/analyze"
)

// Route is the same observed premium turns priced against one other model's table entry. It
// is arithmetic over stored tokens and nothing else: the target model never saw this work,
// and no field here is a claim about what it would have produced.
type Route struct {
	Target string `json:"target"`
	// Premium is the premium slice's observed tokens at Target's rates.
	Premium float64 `json:"premium"`
	// Window is the whole priced window with only the premium slice moved -- everything else
	// stays exactly as observed, which is what makes this a mix rather than a switch.
	Window float64 `json:"window"`
	// Delta is Basis.Cost - Window, positive when the re-priced window is the smaller one.
	Delta float64 `json:"delta"`
	// Share is Delta over the observed priced cost: how much of the window the move touches.
	Share float64 `json:"share"`
}

// routes prices the observed premium bundle against every candidate and returns them cheapest
// first, alongside the candidates the price table could not cost. A model with no entry is never
// guessed at -- but skipping it silently is only honest while nothing else fills the table: name
// two targets, price one, and the single row that comes back reads as the whole answer. The
// second return is what keeps the absence visible, and only a caller-named model can appear in
// it, since targets admits a window's own model only when it is already priced.
func routes(b *Basis, in *analyze.Input, against []string) (found []Route, unpriceable []string) {
	// Empty lists, never nil: the published document must not make a consumer tell "no route
	// this window supports" apart from "assaio failed" by the shape of the field.
	found, unpriceable = []Route{}, []string{}
	if !b.Priced || b.Premium.Cost <= 0 {
		return found, unpriceable
	}
	for _, target := range targets(b, in, against) {
		cost, ok := in.Prices.CostTokens(target, b.premium)
		if !ok {
			unpriceable = append(unpriceable, target)
			continue
		}
		r := Route{Target: target, Premium: cost, Window: b.Cost - b.Premium.Cost + cost}
		r.Delta = b.Cost - r.Window
		r.Share = share(r.Delta, b.Cost)
		found = append(found, r)
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].Window != found[j].Window {
			return found[i].Window < found[j].Window
		}
		return found[i].Target < found[j].Target
	})
	return found, unpriceable
}

// targets is the model list a route may be priced onto, in a deterministic order: the
// window's own priced models outside the premium slice, heaviest first as ByModel already
// sorts them, then whatever the caller named. A model already in the slice is not proposed on
// its own -- re-pricing the slice onto one of its members answers nothing -- but the caller
// may still name one explicitly, which asks a different question: what if all of this ran on
// that one.
func targets(b *Basis, in *analyze.Input, against []string) []string {
	inSlice := make(map[string]bool, len(b.Premium.Models))
	for _, m := range b.Premium.Models {
		inSlice[m] = true
	}
	seen := make(map[string]bool, len(in.ByModel))
	out := make([]string, 0, len(in.ByModel)+len(against))
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for i := range in.ByModel {
		if m := &in.ByModel[i]; m.Priced && !inSlice[m.Model] {
			add(m.Model)
		}
	}
	for _, name := range against {
		add(name)
	}
	return out
}
