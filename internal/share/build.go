package share

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/humanize"
)

// Build shapes one window into the card, the reel and the post beside them. results is
// the same analyze output the text report and the dashboard render, and every figure here
// is read from it or from the prepared aggregates those validators read -- nothing is
// computed for the card alone. sample marks a run on bundled demo usage, which the first
// frame then says out loud.
//
//nolint:gocritic // Input is the value bundle the validator framework threads everywhere; taking it the same way keeps this one more reader of it.
func Build(in analyze.Input, results []analyze.Result, window string, sample bool) Assay {
	v := index(results)
	f := gather(&in, v)
	f.perHundred = perHundred(&f)

	a := Assay{
		Window:    window,
		Days:      int(f.activeDays),
		Layer:     windowLayer,
		Hook:      pick(&f, sample),
		Scale:     scaleStats(&f, v),
		Models:    modelSlices(in.ByModel),
		Tools:     toolSlices(in.Usage),
		Archetype: classify(&f),
		Rays:      fingerprint(in.Sessions, in.ByModel),
		Profile:   buildProfile(&f),
		Ledger:    ledgerStats(&f),
		Fine:      fine(&f, v, sample),
		Limit:     limitLine(&f, sample),
		Hashtag:   Hashtag,
		Install:   InstallCommand,
		ShareCmd:  ShareCommand,
	}
	a.Post = post(&a, &f)
	nonNil(&a)
	return a
}

// nonNil replaces every nil slice with an empty one. A nil slice marshals to `null`, and the
// preview page walks all four without a guard -- one `null` threw inside the draw loop and
// left the canvas frozen on its background, which is the state "Save the reel" would then
// have recorded. A window with nothing in it must render as an empty card, not a dead one.
func nonNil(a *Assay) {
	if a.Models == nil {
		a.Models = []Slice{}
	}
	if a.Tools == nil {
		a.Tools = []Slice{}
	}
	if a.Rays.Rays == nil {
		a.Rays.Rays = []Ray{}
	}
	if a.Profile.Axes == nil {
		a.Profile.Axes = []Axis{}
	}
	if a.Ledger == nil {
		a.Ledger = []Stat{}
	}
	if a.Scale == nil {
		a.Scale = []Stat{}
	}
	if a.Profile.Caveats == nil {
		a.Profile.Caveats = []string{}
	}
	if a.Fine == nil {
		a.Fine = []string{}
	}
}

// limitLine is the short form of Fine, for the artifact rather than the page. It names the
// layer, marks an estimate as an estimate, and says sample data is sample data -- the three
// things a reader of a shared image cannot check for themselves.
func limitLine(f *facts, sample bool) string {
	parts := []string{windowLayer + " layer"}
	if f.cost != nil {
		if f.costIsComplete() {
			parts = append(parts, "$ estimated at public API prices, not a bill")
		} else {
			parts = append(parts, "$ is a floor: "+f.pricedCoverage+" of usage is priced")
		}
	}
	parts = append(parts, "no repository named")
	line := strings.Join(parts, " · ")
	if sample {
		return "SAMPLE DATA · " + line
	}
	return line
}

func scaleStats(f *facts, v verdicts) []Stat {
	out := []Stat{
		{Label: "tokens", Value: humanize.Count(f.tokens)},
		{Label: "sessions", Value: humanize.Int(f.sessions)},
		{Label: "active days", Value: strconv.FormatInt(f.activeDays, 10)},
		{Label: "models · tools", Value: fmt.Sprintf("%d · %d", f.models, f.tools)},
	}
	if s := v.value("adoption", "projects"); s != "" {
		// A count, never a name: the redaction rule allows how many repositories, never which.
		out = append(out, Stat{Label: "repositories", Value: s, Note: "never named"})
	}
	return out
}

func ledgerStats(f *facts) []Stat {
	var out []Stat
	if f.linesOK {
		out = append(out, Stat{Label: "lines the AI wrote", Value: humanize.Int(f.lines)})
	}
	if f.perHundred != "" {
		out = append(out, Stat{Label: "per 100 lines", Value: f.perHundred, Note: pricedNote(f)})
	}
	switch {
	case f.multiple != "" && f.planPaysOff:
		out = append(out, Stat{Label: "plan value", Value: f.multiple, Note: "$" + humanize.USD(f.planCost) + "/mo plan against " + f.apiEquiv + " projected at API prices"})
	case f.multiple != "":
		// Below 1x the multiple alone reads as a win at a glance. subscription-fit never
		// publishes it without the direction beside it, so neither does the card.
		out = append(out, Stat{Label: "plan vs API", Value: f.multiple, Note: f.vsAPI + " -- API pay-as-you-go could be cheaper at this volume"})
	case f.cost != nil:
		out = append(out, Stat{Label: "at API prices", Value: humanize.USDCompact(*f.cost), Note: "an estimate, not a bill"})
	}
	// Three, deliberately: both renderers draw the whole slice, so the list is bounded here
	// rather than sliced at the edge. Rework is not dropped -- it is the "first try — reworks"
	// axis on the profile frame, where it sits beside the reserve it belongs to.
	return out
}

// perHundred is the cost efficiency figure the effectiveness report already publishes per
// project, recomputed here over the window's own totals -- the one figure a card is
// incomplete without, because it is the answer to the criticism the whole category attracts.
// withTokens drops models that carried none. It exists for "<synthetic>", Claude Code's
// marker for its own locally-generated turns: it has no tokens by construction, so publishing
// it spends a palette slot on a labelled 0.0% band and counts a model nobody ran.
// pricedNote marks a cost figure the price table could not fully cover, the way report marks
// the same figure with an asterisk. Silence here would publish a floor as a total.
func pricedNote(f *facts) string {
	if f.costIsComplete() {
		return "an estimate at public API prices, not a bill"
	}
	return "a floor: only " + f.pricedCoverage + " of this window's usage is priced"
}

func withTokens(byModel []analyze.ModelStat) []analyze.ModelStat {
	out := make([]analyze.ModelStat, 0, len(byModel))
	for i := range byModel {
		if byModel[i].Tokens > 0 {
			out = append(out, byModel[i])
		}
	}
	return out
}

func perHundred(f *facts) string {
	if f.cost == nil || !f.linesOK || f.lines <= 0 {
		return ""
	}
	// USDCell keeps the cents this figure lives in, where USDCompact would round $4.19 to "$4";
	// its sub-cent bound is unreachable here because a window with lines and a cost cannot
	// divide below it without also being unpriced, which pricedNote states.
	return "$" + humanize.USDCell(*f.cost/float64(f.lines)*100)
}

func modelSlices(byModel []analyze.ModelStat) []Slice {
	byModel = withTokens(byModel)
	var total int64
	for i := range byModel {
		total += byModel[i].Tokens
	}
	if total == 0 {
		return nil
	}
	palette := hues(byModel)
	var out []Slice
	var rest int64
	for i := range byModel {
		m := &byModel[i]
		if i >= 4 {
			rest += m.Tokens
			continue
		}
		out = append(out, Slice{
			Label: m.Model,
			Value: humanize.PercentAt(float64(m.Tokens)/float64(total), 1),
			Frac:  float64(m.Tokens) / float64(total),
			Hue:   palette[m.Model],
		})
	}
	if rest > 0 {
		out = append(out, Slice{
			Label: fmt.Sprintf("+ %d more", len(byModel)-4),
			Value: humanize.PercentAt(float64(rest)/float64(total), 1),
			Frac:  float64(rest) / float64(total),
			Hue:   4,
		})
	}
	return out
}
