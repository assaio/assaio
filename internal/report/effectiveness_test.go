package report

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

func TestBuildEffectivenessGroupsByProject(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web",
			In: 100, Out: 200, CacheWrite: 50, CacheRead: 800,
			LinesAdded: 120, LinesRemoved: 20, Edits: 4, ToolCalls: 6, Rejected: 1,
		},
		{
			Day: "2026-07-02", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web",
			In: 50, Out: 10, LinesAdded: 30, Edits: 1, ToolCalls: 1,
		},
		{Day: "2026-07-01", Tool: "codex", Model: "unknown", Project: "api", In: 10},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if len(eff) != 2 {
		t.Fatalf("len(eff) = %d want 2: %+v", len(eff), eff)
	}
	byGroup := map[string]EffRow{}
	for _, r := range eff {
		byGroup[r.Group] = r
	}

	web, ok := byGroup["web"]
	if !ok {
		t.Fatalf("missing web group: %+v", eff)
	}
	if web.LinesAdded != 150 || web.LinesRemoved != 20 || web.Edits != 5 || web.ToolCalls != 7 || web.Rejected != 1 {
		t.Fatalf("web activity = %+v", web)
	}
	if web.Cost == nil {
		t.Fatalf("web group must be priced: %+v", web)
	}
	if web.HasUnpriced {
		t.Fatalf("web group has no unpriced rows: %+v", web)
	}
	wantTokens := int64(100 + 200 + 800 + 50 + 50 + 10)
	if web.TokensTotal != wantTokens {
		t.Fatalf("web TokensTotal = %d want %d", web.TokensTotal, wantTokens)
	}

	api, ok := byGroup["api"]
	if !ok {
		t.Fatalf("missing api group: %+v", eff)
	}
	if !api.HasUnpriced {
		t.Fatalf("api group must carry HasUnpriced (only unpriced usage): %+v", api)
	}
	if api.Cost != nil {
		t.Fatalf("api group must have nil Cost (only unpriced usage): %+v", api)
	}
}

func TestBuildEffectivenessCostPer100Lines(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web", In: 1_000_000, LinesAdded: 200},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if eff[0].Cost == nil || eff[0].CostPer100Lines == nil {
		t.Fatalf("expected priced row with a computed ratio: %+v", eff[0])
	}
	wantRatio := *eff[0].Cost / (200.0 / 100)
	if diff := *eff[0].CostPer100Lines - wantRatio; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("CostPer100Lines = %v want %v", *eff[0].CostPer100Lines, wantRatio)
	}
}

func TestBuildEffectivenessNilRatioWhenNoLines(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Project: "planning", In: 1000, Out: 500},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if eff[0].LinesAdded != 0 {
		t.Fatalf("fixture must have zero lines: %+v", eff[0])
	}
	if eff[0].Cost == nil {
		t.Fatal("this group is still priced -- zero lines with cost is legitimate exploration/planning")
	}
	if eff[0].CostPer100Lines != nil {
		t.Fatalf("CostPer100Lines must be nil when LinesAdded == 0, got %v", *eff[0].CostPer100Lines)
	}
}

func TestBuildEffectivenessHasUnpriced(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "codex", Model: "unknown-model", Project: "api", In: 10, LinesAdded: 5},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if !eff[0].HasUnpriced {
		t.Fatal("group with only unpriced usage must carry HasUnpriced")
	}
	if eff[0].Cost != nil {
		t.Fatalf("group with only unpriced usage must have nil Cost, got %v", *eff[0].Cost)
	}
	if eff[0].CostPer100Lines != nil {
		t.Fatal("CostPer100Lines must be nil when cost is unknown, even with lines > 0")
	}
}

func TestBuildEffectivenessUnknownDim(t *testing.T) {
	_, err := BuildEffectiveness(nil, table(), "bogus")
	if err == nil {
		t.Fatal("expected error for unknown dimension")
	}
	for _, dim := range []string{"day", "project", "tool", "model", "entrypoint"} {
		if !strings.Contains(err.Error(), dim) {
			t.Fatalf("error %q must list valid dim %q", err.Error(), dim)
		}
	}
}

// TestEffectivenessRefusesToRankNamedIndividuals is the Refusals line in BACKLOG.md held on
// the surface that broke it: `effectiveness --by member` printed MEMBER | AI LINES | EDITS |
// REJ | COST $ | $/100 LINES, a per-person output-and-spend ranking, under a caveat that
// called itself "a diagnostic per project". json and csv carried the same rows with no caveat
// at all. The dashboard removed the same figure in v0.14 (B141); this is that decision
// reaching the other two surfaces.
func TestEffectivenessRefusesToRankNamedIndividuals(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Member: "alice", In: 100, LinesAdded: 10},
	}
	_, err := BuildEffectiveness(rows, table(), "member")
	if err == nil {
		t.Fatal("effectiveness --by member must be refused, not rendered")
	}
	if !strings.Contains(err.Error(), "does not rank named individuals") {
		t.Fatalf("the refusal must say why, got %q", err.Error())
	}
}

// TestEffectivenessCaveatNamesTheDimensionItIsGroupedBy: the caveat scoped efficiency to
// "per project" whatever the table was grouped by, which is a claim about rows that carry no
// project at all.
func TestEffectivenessCaveatNamesTheDimensionItIsGroupedBy(t *testing.T) {
	rows := []store.UsageRow{{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", In: 100, LinesAdded: 10}}
	eff, err := BuildEffectiveness(rows, table(), "model")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "model"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "a diagnostic per model") {
		t.Fatalf("caveat must name the grouping dimension: %s", buf.String())
	}
}

func TestRenderEffectivenessTableShowsDimAndCaveat(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web", In: 100, Out: 200, LinesAdded: 100},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "project"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The line-coverage note points at `depth` rather than naming sources: a note that spells
	// them out is one a new parser makes wrong, which is what happened to this one.
	for _, want := range []string{"PROJECT", "web", "directional", "never a performance metric", "records changed lines"} {
		if !strings.Contains(out, want) {
			t.Fatalf("table missing %q: %s", want, out)
		}
	}
}

func TestRenderEffectivenessTableZeroLinesShowsDash(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Project: "planning", In: 1000, Out: 500},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "project"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "—") {
		t.Fatalf("zero-lines group must render as an em dash: %s", buf.String())
	}
}

func TestRenderEffectivenessTableUnpricedFootnote(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "codex", Model: "unknown-model", Project: "api", In: 10, LinesAdded: 5},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "project"); err != nil {
		t.Fatal(err)
	}
	// The share is the point: the whole of this window is on a model with no price, and the
	// footnote has to say so rather than repeat a marker that reads the same at 0.1%.
	if !strings.Contains(buf.String(), "cost excludes 100.0% of the tokens in this table") {
		t.Fatalf("table missing the quantified unpriced footnote: %s", buf.String())
	}
}

func TestRenderEffectivenessJSONNullRatio(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Project: "planning", In: 1000, Out: 500},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessJSON(&buf, eff); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"cost_per_100_lines": null`) {
		t.Fatalf("zero-line group must serialize cost_per_100_lines as null: %s", buf.String())
	}
}

func TestRenderEffectivenessCSVEmptyRatioCell(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Project: "planning", In: 1000, Out: 500},
	}
	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	records := effCSVRecords(t, eff)
	if got := records[1][columnIndex(t, records[0], "cost_per_100_lines")]; got != "" {
		t.Fatalf("zero-line row must have an empty cost_per_100_lines cell, got %q", got)
	}
}

// TestEffectivenessCSVCarriesTheCapabilityBits: the table prints a dash where no source behind
// a column records what it counts, and CSV has no dash -- so without these four columns an
// `agy` group exported "0 lines, 0 tokens, no cost", byte-identical to a token- and
// line-recording source that did the work and produced nothing. The bits are the only thing in
// the machine format that tells the two apart.
func TestEffectivenessCSVCarriesTheCapabilityBits(t *testing.T) {
	eff, err := BuildEffectiveness([]store.UsageRow{
		{Day: "2026-09-01", Tool: "agy", Project: "unmeasured", Edits: 3, ToolCalls: 4},
		{Day: "2026-09-01", Tool: "claude-code", Model: "claude-opus-4-5", Project: "measured", In: 1000},
	}, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	records := effCSVRecords(t, eff)
	byGroup := map[string][]string{}
	for _, rec := range records[1:] {
		byGroup[rec[columnIndex(t, records[0], "group")]] = rec
	}
	for _, tt := range []struct {
		group  string
		column string
		want   string
	}{
		{"unmeasured", "line_capable", "false"},
		{"unmeasured", "edit_capable", "true"},
		{"unmeasured", "refusable", "false"},
		{"unmeasured", "tokened", "false"},
		{"measured", "line_capable", "true"},
		{"measured", "edit_capable", "true"},
		{"measured", "refusable", "true"},
		{"measured", "tokened", "true"},
	} {
		if got := byGroup[tt.group][columnIndex(t, records[0], tt.column)]; got != tt.want {
			t.Errorf("%s %s = %q, want %q", tt.group, tt.column, got, tt.want)
		}
	}
}

func effCSVRecords(t *testing.T, eff []EffRow) [][]string {
	t.Helper()
	var buf bytes.Buffer
	if err := RenderEffectivenessCSV(&buf, eff); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) < 2 {
		t.Fatalf("csv holds %d lines, want a header and at least one row", len(records))
	}
	return records
}

// TestEffectivenessWithholdsBothHalvesForATokenLessSource: the effectiveness table's whole
// point is lines against cost, and Antigravity CLI answers neither. Both columns and the total
// have to withhold -- a group reading "0 lines for $0.00" is the sentence that gets a tool
// dropped for producing nothing when nothing was ever measured.
func TestEffectivenessWithholdsBothHalvesForATokenLessSource(t *testing.T) {
	eff, err := BuildEffectiveness([]store.UsageRow{
		{Day: "2026-09-01", Tool: "agy", Project: "web", Edits: 3, ToolCalls: 4},
	}, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if eff[0].Tokened || eff[0].LineCapable {
		t.Fatalf("agy answers neither tokens nor changed lines: %+v", eff[0])
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "project"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "0.00") {
		t.Errorf("nothing priced must total —, not $0.00:\n%s", out)
	}
	if strings.Contains(out, "Every source in this table records changed lines.") {
		t.Errorf("the coverage note claims a capability no source in the table has:\n%s", out)
	}
	if !strings.Contains(out, "No source in this table records a changed line") {
		t.Errorf("the coverage note must say what is missing:\n%s", out)
	}
}

// TestEffectivenessRendersTheEditsItMeasuresAndDashesTheRefusalsItCannot is the other
// direction of the same rule. Antigravity CLI answers ai.edits.count and no refusal counter,
// so the EDITS cell and the EDITS total have to carry the 3 edits it recorded while the REJ
// cell and total withhold -- the table previously blanked the edits on the line capability and
// printed "0" under REJ, withholding a measurement and fabricating one in the same row.
func TestEffectivenessRendersTheEditsItMeasuresAndDashesTheRefusalsItCannot(t *testing.T) {
	eff, err := BuildEffectiveness([]store.UsageRow{
		{Day: "2026-09-01", Tool: "agy", Project: "web", Edits: 3, ToolCalls: 4},
	}, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if !eff[0].EditCapable || eff[0].Refusable {
		t.Fatalf("agy records edits and no refusal: %+v", eff[0])
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "project"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Both the group row and the TOTAL footer, each read whole: the cells are positional, and
	// an assertion on "3" alone would pass on a table that printed it in the wrong column.
	for _, want := range []string{
		"| web     |        — |     3 |   — |     —* |           — |",
		"| TOTAL   |        — |     3 |   — |      — |           — |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing row %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "edit and $/100-lines columns are withheld") {
		t.Errorf("the coverage note calls the edit column absent when it holds a measurement:\n%s", out)
	}
	if !strings.Contains(out, "the edit column beside them reads the sources that do record one") {
		t.Errorf("the coverage note must scope itself to the columns actually withheld:\n%s", out)
	}
}
