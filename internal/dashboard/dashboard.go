// Package dashboard builds and renders assaio's offline HTML "Assay" report: the
// analyze.Validator verdicts, a bounded top-project drill-down, and the header/footer
// stats around them. It sits between internal/cli and internal/analyze -- cli owns I/O
// (opening the store, running queries), Build shapes that data into Data, and render.go
// turns Data into self-contained HTML. This package exists separately from
// internal/report because internal/analyze already imports internal/report, so a
// dashboard consuming analyze.Result cannot live inside report without an import cycle.
package dashboard

import (
	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// Data is the render-ready shape for the Assay HTML report. Verdicts is the single
// source of truth: RenderHTML walks it generically off each Result's own Read/Purity/
// Figures/Bars/Takeaway/Caveats (see render.go's faceplateCell/ledgerEntry templates),
// so a newly registered analyze.Validator appears on the report with no template
// change -- the extensibility seam a future navigation/multi-page layer builds on.
type Data struct {
	Window     string
	Anonymized bool
	Verdicts   []analyze.Result
	Drill      *ProjectDrill
	// Team is the per-member adoption breakdown, present only when the queried usage
	// carries a non-empty Member (a central team-server store); nil for a purely local
	// store, so the report never renders an empty "Team" section.
	Team      *Team
	Inventory Inventory
	CostBasis string
	Caveats   []string
}

// Inventory is the masthead/panel-label header counts for the queried window.
type Inventory struct {
	Projects   int
	Sessions   int
	ActiveDays int
}

// Build shapes in into the Assay report's render-ready Data: it runs every registered
// analyze.Validator over in for Verdicts, re-runs them scoped to the top project by AI
// lines for Drill, and formats the header/footer stats. subpaths is the drill project's
// subpath breakdown, pre-fetched by the caller via TopProject + Store.Subpaths -- the one
// store query subpath breakdown needs, kept out of this otherwise pure function. extra
// is pre-computed exec-metric-plugin verdicts (internal/plugin's metric protocol),
// appended after the built-ins and before anonymization so BarsAreProjects
// pseudonymization applies to them identically; the caller runs those subprocesses --
// this function stays pure, and the team server deliberately passes nil (ADR 0004).
//
//nolint:gocritic // Input is a small value bundle threaded through the validator framework; Build receives it the same way every Validator.Analyze does.
func Build(in analyze.Input, window string, anonymize bool, subpaths []store.SubpathRow, extra []analyze.Result) Data {
	verdicts := append(runValidators(&in), extra...)
	drill := buildDrill(in, subpaths, anonymize)
	if anonymize {
		anonymizeVerdicts(verdicts)
	}
	team := buildTeam(in.Usage, in.Sessions, in.Prices, anonymize)

	inv := report.BuildInventory(in.Usage, in.Prices)
	return Data{
		Window:     window,
		Anonymized: anonymize,
		Verdicts:   verdicts,
		Drill:      drill,
		Team:       team,
		Inventory:  Inventory{Projects: inv.Projects, Sessions: len(in.Sessions), ActiveDays: inv.Days},
		CostBasis:  costBasis(inv, window),
		Caveats:    caveats(anonymize, inv.HasUnpriced),
	}
}

// runValidators runs every registered analyze.Validator over in, in analyze.Validators's
// stable name-sorted order.
func runValidators(in *analyze.Input) []analyze.Result {
	validators := analyze.Validators()
	out := make([]analyze.Result, len(validators))
	for i, v := range validators {
		out[i] = v.Analyze(*in)
	}
	return out
}

// anonymizeVerdicts pseudonymizes every Bars list a Validator has marked as
// project-labeled via Result.BarsAreProjects (the built-in throughput validator's
// top-projects ranking is the only one today, but this applies to any Validator, built-in
// or custom). Every other validator's Bars label models or something else, and must never
// be anonymized (mirrors the pre-existing ByModel/ByTool convention: only project-scoped
// names are pseudonymized).
func anonymizeVerdicts(verdicts []analyze.Result) {
	for i := range verdicts {
		if !verdicts[i].BarsAreProjects {
			continue
		}
		for j := range verdicts[i].Bars {
			verdicts[i].Bars[j].Label = report.Pseudonym("project", verdicts[i].Bars[j].Label)
		}
	}
}

// caveats returns the colophon's honesty notes: the locale's directional/coverage/quality
// lines plus the shared cost-estimate disclosure, adding the pseudonymization note only
// when anonymize is true.
func caveats(anonymize, hasUnpriced bool) []string {
	// report.CostEstimateDisclosure is sourced from internal/report so the cost-basis
	// wording is identical here and on the CLI cost tables -- one canonical string.
	out := []string{en.DirectionalCaveat, en.LineCoverageCaveat, en.QualityCaveat, report.CostEstimateDisclosure}
	if hasUnpriced {
		out = append(out, en.UnpricedCaveat)
	}
	if anonymize {
		out = append(out, en.AnonymizedCaveat)
	}
	return out
}
