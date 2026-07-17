package report

import (
	"bytes"
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
	for _, dim := range []string{"day", "project", "tool", "model", "entrypoint", "member"} {
		if !strings.Contains(err.Error(), dim) {
			t.Fatalf("error %q must list valid dim %q", err.Error(), dim)
		}
	}
}

// TestBuildEffectivenessByMemberRendersLocalPlaceholder guards the team-view dimension on
// the effectiveness view: grouping by member must separate real members from the local ""
// group, and the table must render "" as "(local)", not the generic "(unknown)".
func TestBuildEffectivenessByMemberRendersLocalPlaceholder(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Member: "alice", In: 100, LinesAdded: 10},
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", Member: "", In: 50, LinesAdded: 5},
	}
	eff, err := BuildEffectiveness(rows, table(), "member")
	if err != nil {
		t.Fatal(err)
	}
	if len(eff) != 2 {
		t.Fatalf("len(eff) = %d want 2 (alice, local): %+v", len(eff), eff)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "member"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "alice") {
		t.Fatalf("table missing member name %q: %s", "alice", out)
	}
	if !strings.Contains(out, "(local)") {
		t.Fatalf("table missing (local) placeholder for the empty member: %s", out)
	}
	if strings.Contains(out, "(unknown)") {
		t.Fatalf("member dimension must never fall back to the generic (unknown) placeholder: %s", out)
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
	for _, want := range []string{"PROJECT", "web", "directional", "never a performance metric", "Claude Code and Codex today"} {
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
	if !strings.Contains(buf.String(), "unpriced usage excluded from cost") {
		t.Fatalf("table missing unpriced footnote: %s", buf.String())
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
	var buf bytes.Buffer
	if err := RenderEffectivenessCSV(&buf, eff); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if !strings.Contains(lines[0], "cost_per_100_lines") {
		t.Fatalf("csv header missing cost_per_100_lines: %q", lines[0])
	}
	if !strings.HasSuffix(lines[1], ",") {
		t.Fatalf("zero-line row must end with an empty cost_per_100_lines cell: %q", lines[1])
	}
}
