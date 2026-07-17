package dashboard

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

var fixtureNow = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

func fixturePrices() pricing.Table {
	return pricing.Table{
		"claude-sonnet-4-5": {Input: 3e-6, Output: 1.5e-5},
		"claude-opus-4-5":   {Input: 1.5e-5, Output: 7.5e-5},
	}
}

// fixtureInput seeds two projects: "web" (opus, higher cost, fewer lines) and "api"
// (sonnet, cheaper, the most AI lines) -- enough contrast to make Build's top-project
// selection and per-tier model-fit scoping directly assertable.
func fixtureInput() analyze.Input {
	usage := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web",
			In: 100000, Out: 200000, LinesAdded: 1916, Edits: 5, ToolCalls: 10, Rejected: 1,
		},
		{
			Day: "2026-07-11", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "api",
			In: 1000, Out: 2000, LinesAdded: 18496, Edits: 20, ToolCalls: 25,
		},
		{
			Day: "2026-07-01", Tool: "codex", Model: "claude-sonnet-4-5", Project: "web",
			In: 200, Out: 300, LinesAdded: 50,
		},
	}
	sessions := []store.SessionRow{
		{
			SessionID: "s1", Project: "web", Tool: "claude-code", Model: "claude-opus-4-5",
			FirstTs: fixtureNow.Add(-2 * time.Hour), LastTs: fixtureNow.Add(-time.Hour),
			Turns: 8, OutputTokens: 200000, PeakContextTokens: 50000, Edits: 5, ActiveMinutes: 22,
		},
		{
			SessionID: "s2", Project: "api", Tool: "claude-code", Model: "claude-sonnet-4-5",
			FirstTs: fixtureNow.Add(-3 * time.Hour), LastTs: fixtureNow.Add(-2 * time.Hour),
			Turns: 4, OutputTokens: 2000, PeakContextTokens: 20000, Edits: 3, ActiveMinutes: 14,
		},
	}
	return analyze.BuildInput(usage, sessions, fixturePrices(), fixtureNow, 7*24*time.Hour, analyze.Delegation{Sub: 25, Total: 1000})
}

// fixtureSubpaths is api's (the fixture's top project) subpath breakdown, summing to its
// 18496 AI lines.
func fixtureSubpaths() []store.SubpathRow {
	return []store.SubpathRow{
		{Subpath: "apps/mobile", Lines: 12000, Sessions: 1},
		{Subpath: "", Lines: 6496, Sessions: 1},
	}
}

func findVerdict(verdicts []analyze.Result, name string) analyze.Result {
	for i := range verdicts {
		if verdicts[i].Name == name {
			return verdicts[i]
		}
	}
	return analyze.Result{}
}

func containsSubstring(lines []string, sub string) bool {
	for _, l := range lines {
		if strings.Contains(l, sub) {
			return true
		}
	}
	return false
}

func TestBuildProducesVerdictsForEveryRegisteredValidator(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)
	want := len(analyze.Validators())
	if len(d.Verdicts) != want {
		t.Fatalf("len(Verdicts) = %d, want %d (one per registered validator)", len(d.Verdicts), want)
	}
	for _, name := range []string{"adoption", "model-fit", "context", "throughput", "rework"} {
		if findVerdict(d.Verdicts, name).Name != name {
			t.Fatalf("Verdicts missing built-in validator %q: %+v", name, d.Verdicts)
		}
	}
}

func TestBuildDrillsIntoTopProjectByLines(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", false, fixtureSubpaths(), nil)
	if d.Drill == nil {
		t.Fatal("Drill = nil, want the top project by AI lines")
	}
	if d.Drill.Name != "api" {
		t.Fatalf("Drill.Name = %q, want %q (18496 lines beats web's 1966)", d.Drill.Name, "api")
	}
	if d.Drill.Lines != 18496 {
		t.Fatalf("Drill.Lines = %d, want 18496", d.Drill.Lines)
	}
	if d.Drill.Sessions != 1 {
		t.Fatalf("Drill.Sessions = %d, want 1", d.Drill.Sessions)
	}
	if len(d.Drill.Verdicts) != len(analyze.Validators()) {
		t.Fatalf("len(Drill.Verdicts) = %d, want %d", len(d.Drill.Verdicts), len(analyze.Validators()))
	}
	if len(d.Drill.Subpaths) != 2 {
		t.Fatalf("Drill.Subpaths = %+v, want the 2 fixture rows passed through unchanged", d.Drill.Subpaths)
	}
}

// TestBuildDrillVerdictsAreScopedToProjectRows asserts Drill.Verdicts are re-analyzed on
// api's rows alone, not sliced from the top-level verdicts: web's usage is opus-heavy
// (premium), api's is entirely the cheaper sonnet model, so model-fit's Read must differ
// between the two scopes.
func TestBuildDrillVerdictsAreScopedToProjectRows(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", false, nil, nil)
	top := findVerdict(d.Verdicts, "model-fit")
	drill := findVerdict(d.Drill.Verdicts, "model-fit")
	if top.Read.Key != "watch" {
		t.Fatalf("top-level model-fit Read = %+v, want watch (web's opus usage dominates tokens)", top.Read)
	}
	if drill.Read.Key != "good" {
		t.Fatalf("drill model-fit Read = %+v, want good (api's rows are entirely the cheaper model)", drill.Read)
	}
}

func TestBuildAnonymizesThroughputBarsAndDrillName(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", true, nil, nil)
	if !d.Anonymized {
		t.Fatal("Anonymized = false, want true")
	}
	throughput := findVerdict(d.Verdicts, "throughput")
	for _, b := range throughput.Bars {
		if b.Label == "web" || b.Label == "api" {
			t.Fatalf("throughput Bars must be pseudonymized when anonymize=true: %+v", throughput.Bars)
		}
		if !strings.HasPrefix(b.Label, "project-") {
			t.Fatalf("throughput Bar label %q must look like a pseudonym", b.Label)
		}
	}
	if d.Drill.Name == "api" || !strings.HasPrefix(d.Drill.Name, "project-") {
		t.Fatalf("Drill.Name = %q, want a pseudonym", d.Drill.Name)
	}
}

func TestBuildNeverAnonymizesModelNames(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", true, nil, nil)
	modelFit := findVerdict(d.Verdicts, "model-fit")
	for _, b := range modelFit.Bars {
		if strings.HasPrefix(b.Label, "project-") {
			t.Fatalf("model-fit Bars label models, never projects, and must never be pseudonymized: %+v", modelFit.Bars)
		}
	}
}

// TestAnonymizeVerdictsAppliesToAnyValidatorsProjectBars asserts anonymizeVerdicts
// pseudonymizes any Result whose Bars are marked project-labeled (Result.BarsAreProjects),
// not just the built-in throughput validator by name -- the same rule a custom,
// out-of-tree Validator's project-ranked Bars need. Exercised directly against
// anonymizeVerdicts (rather than via Build + a registered fake Validator) so the test
// never mutates analyze's shared, process-wide validator registry.
func TestAnonymizeVerdictsAppliesToAnyValidatorsProjectBars(t *testing.T) {
	verdicts := []analyze.Result{
		{Name: "custom-metric", BarsAreProjects: true, Bars: []analyze.Bar{{Label: "web", Value: "1", Frac: 1}}},
		{Name: "model-fit", Bars: []analyze.Bar{{Label: "claude-sonnet-4-5", Value: "1", Frac: 1}}},
	}
	anonymizeVerdicts(verdicts)
	if got := verdicts[0].Bars[0].Label; got == "web" || !strings.HasPrefix(got, "project-") {
		t.Fatalf("a custom validator's BarsAreProjects Bars must be pseudonymized, got %q", got)
	}
	if got := verdicts[1].Bars[0].Label; got != "claude-sonnet-4-5" {
		t.Fatalf("a validator without BarsAreProjects must never be pseudonymized, got %q", got)
	}
}

func TestBuildRealNamesWhenNotAnonymized(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", false, nil, nil)
	if d.Anonymized {
		t.Fatal("Anonymized = true, want false")
	}
	throughput := findVerdict(d.Verdicts, "throughput")
	names := map[string]bool{}
	for _, b := range throughput.Bars {
		names[b.Label] = true
	}
	if !names["web"] || !names["api"] {
		t.Fatalf("throughput Bars must show real project names when anonymize=false: %+v", throughput.Bars)
	}
	if d.Drill.Name != "api" {
		t.Fatalf("Drill.Name = %q, want the real name %q", d.Drill.Name, "api")
	}
}

// TestBuildCaveatsIncludePseudonymizationNoteOnlyWhenAnonymized checks the
// --no-anonymize pointer (unique to the per-render AnonymizedCaveat) rather than the
// bare word "pseudonymized": the always-present DirectionalCaveat now legitimately
// mentions pseudonymization as assaio's default posture, so that word alone no longer
// signals whether this particular render was anonymized.
func TestBuildCaveatsIncludePseudonymizationNoteOnlyWhenAnonymized(t *testing.T) {
	anon := Build(fixtureInput(), "last 30 days", true, nil, nil)
	if !containsSubstring(anon.Caveats, "--no-anonymize") {
		t.Fatalf("anonymized Data must carry the pseudonymization caveat: %+v", anon.Caveats)
	}
	real := Build(fixtureInput(), "last 30 days", false, nil, nil)
	if containsSubstring(real.Caveats, "--no-anonymize") {
		t.Fatalf("non-anonymized Data must not carry the pseudonymization caveat: %+v", real.Caveats)
	}
	if !containsSubstring(real.Caveats, "aggregate and pseudonymized by default") {
		t.Fatalf("Data must always carry the directional, aggregate-by-default caveat: %+v", real.Caveats)
	}
	if containsSubstring(real.Caveats, "never per person") {
		t.Fatalf("Data must not claim an absolute never-per-person stance: %+v", real.Caveats)
	}
}

func TestBuildInventoryAndCostBasis(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", true, nil, nil)
	if d.Inventory.Projects != 2 {
		t.Fatalf("Inventory.Projects = %d, want 2", d.Inventory.Projects)
	}
	if d.Inventory.Sessions != 2 {
		t.Fatalf("Inventory.Sessions = %d, want 2", d.Inventory.Sessions)
	}
	if !strings.Contains(d.CostBasis, "last 30 days") {
		t.Fatalf("CostBasis = %q, want it to mention the window", d.CostBasis)
	}
	if !strings.HasPrefix(d.CostBasis, "$") {
		t.Fatalf("CostBasis = %q, want a priced total (fixture usage is fully priced)", d.CostBasis)
	}
}

// TestBuildEmptyInputIsHonestNotACrash covers the "no data yet" dashboard: every
// registered validator still renders its own honest no-data Result, Drill is nil (no
// project to drill into), and Inventory/CostBasis are honestly zero/dashed rather than
// panicking on empty input.
func TestBuildEmptyInputIsHonestNotACrash(t *testing.T) {
	in := analyze.BuildInput(nil, nil, fixturePrices(), fixtureNow, 7*24*time.Hour, analyze.Delegation{})
	d := Build(in, "last 30 days", true, nil, nil)

	if len(d.Verdicts) != len(analyze.Validators()) {
		t.Fatalf("empty input must still produce every validator's honest no-data Result, got %d verdicts", len(d.Verdicts))
	}
	for _, v := range d.Verdicts {
		if v.Read.Key != "neutral" {
			t.Fatalf("%s: Read = %+v on empty input, want the neutral no-data read", v.Name, v.Read)
		}
	}
	if d.Drill != nil {
		t.Fatalf("Drill = %+v, want nil when there is no usage to drill into", d.Drill)
	}
	if d.Inventory.Projects != 0 || d.Inventory.Sessions != 0 || d.Inventory.ActiveDays != 0 {
		t.Fatalf("Inventory = %+v, want all-zero on empty input", d.Inventory)
	}
	if !strings.Contains(d.CostBasis, "—") {
		t.Fatalf("CostBasis = %q, want an honest dash when there is no cost to report", d.CostBasis)
	}
}

func TestBuildAppendsExtraVerdicts(t *testing.T) {
	extra := []analyze.Result{{
		Name:            "plugin:demo",
		Title:           "Demo Metric",
		Read:            analyze.Read{Key: "watch", Label: "WATCH"},
		HowToRead:       "H",
		Takeaway:        "K",
		Bars:            []analyze.Bar{{Label: "web", Value: "40", Frac: 1}},
		BarsAreProjects: true,
	}}
	d := Build(fixtureInput(), "last 30 days", true, nil, extra)

	if d.Verdicts[0].Name == "plugin:demo" {
		t.Fatal("extra verdicts must render after the built-ins, not before")
	}
	last := d.Verdicts[len(d.Verdicts)-1]
	if last.Name != "plugin:demo" || last.Title != "Demo Metric" {
		t.Fatalf("last verdict = %+v, want the appended plugin metric", last)
	}
	if last.Bars[0].Label == "web" {
		t.Fatal("a plugin verdict's project-labeled bars must be pseudonymized under anonymize, like any built-in's")
	}
}

func TestBuildNilExtrasKeepsBuiltinsOnly(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", true, nil, nil)
	for _, v := range d.Verdicts {
		if strings.HasPrefix(v.Name, "plugin:") {
			t.Fatalf("nil extras must add no plugin verdicts, got %q", v.Name)
		}
	}
}
