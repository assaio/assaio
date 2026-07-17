package report

import (
	"bytes"
	"encoding/csv"
	"regexp"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

func table() pricing.Table {
	return pricing.Table{"claude-opus-4-5": {Input: 5e-6, Output: 2.5e-5, CacheWrite: 6.25e-6, CacheRead: 5e-7}}
}

func TestBuildComputesCost(t *testing.T) {
	rows := []store.UsageRow{{
		Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5",
		In: 100, Out: 200, CacheWrite: 50, CacheRead: 800,
	}}
	built := Build(rows, table())
	if len(built) != 1 || !built[0].Priced || built[0].Cost == nil {
		t.Fatalf("built = %+v", built)
	}
	want := 0.0062125
	if diff := *built[0].Cost - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Cost = %.10f want %.10f", *built[0].Cost, want)
	}
}

func TestBuildUnpricedModel(t *testing.T) {
	rows := []store.UsageRow{{Day: "d", Tool: "codex", Model: "unknown", In: 10}}
	built := Build(rows, table())
	if built[0].Priced {
		t.Fatal("unknown model must be marked unpriced with zero cost")
	}
	if built[0].Cost != nil {
		t.Fatalf("unknown model must have nil Cost, got %v", *built[0].Cost)
	}
}

func TestRenderCSVHasHeader(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderCSV(&buf, Build([]store.UsageRow{{Day: "d", Tool: "codex", Model: "m", In: 1}}, table())); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "day,tool,model,") {
		t.Fatalf("csv header missing: %q", buf.String())
	}
}

func TestRenderCSVUnpricedEmptyCost(t *testing.T) {
	rows := []store.UsageRow{{Day: "d", Tool: "codex", Model: "unknown", In: 10}}
	var buf bytes.Buffer
	if err := RenderCSV(&buf, Build(rows, table())); err != nil {
		t.Fatal(err)
	}
	cr := csv.NewReader(&buf)
	records, err := cr.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	header, row := records[0], records[1]
	costCol := indexOf(header, "cost")
	pricedCol := indexOf(header, "priced")
	if row[costCol] != "" {
		t.Fatalf("unpriced row must have an empty cost cell, got %q", row[costCol])
	}
	if row[pricedCol] != "false" {
		t.Fatalf("unpriced row must have priced=false, got %q", row[pricedCol])
	}
}

func indexOf(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

func TestAggregateByProject(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web",
			In: 100, Out: 200, CacheWrite: 50, CacheRead: 800,
		},
		{
			Day: "2026-07-02", Tool: "codex", Model: "claude-opus-4-5", Project: "web",
			In: 50, Out: 10,
		},
		{Day: "2026-07-01", Tool: "claude-code", Model: "unknown", Project: "api", In: 10},
	}
	built := Build(rows, table())
	agg, err := Aggregate(built, "project")
	if err != nil {
		t.Fatal(err)
	}
	if len(agg) != 2 {
		t.Fatalf("len(agg) = %d want 2: %+v", len(agg), agg)
	}
	byProject := map[string]Row{}
	for _, r := range agg {
		byProject[r.Project] = r
	}

	web, ok := byProject["web"]
	if !ok {
		t.Fatalf("missing web group: %+v", agg)
	}
	if web.In != 150 || web.Out != 210 || web.CacheRead != 800 || web.CacheWrite != 50 {
		t.Fatalf("web tokens = %+v", web)
	}
	if !web.Priced || web.Cost == nil {
		t.Fatalf("web group must be priced (both rows priced): %+v", web)
	}
	wantCost := *built[0].Cost + *built[1].Cost
	if diff := *web.Cost - wantCost; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("web Cost = %v want %v", *web.Cost, wantCost)
	}
	if web.HasUnpriced {
		t.Fatalf("web group has no unpriced rows: %+v", web)
	}
	if web.Day != "" || web.Tool != "" || web.Model != "" {
		t.Fatalf("non-grouped dims must be blank: %+v", web)
	}

	api, ok := byProject["api"]
	if !ok {
		t.Fatalf("missing api group: %+v", agg)
	}
	if api.Priced {
		t.Fatalf("api group must not be priced (only unpriced row): %+v", api)
	}
	if !api.HasUnpriced {
		t.Fatalf("api group must carry has-unpriced marker: %+v", api)
	}
}

func TestAggregateUnknownDim(t *testing.T) {
	_, err := Aggregate(nil, "bogus")
	if err == nil {
		t.Fatal("expected error for unknown dimension")
	}
	for _, dim := range []string{"day", "project", "tool", "model", "entrypoint", "member"} {
		if !strings.Contains(err.Error(), dim) {
			t.Fatalf("error %q must list valid dim %q", err.Error(), dim)
		}
	}
}

// TestAggregateByMemberGroupsAndRendersLocalPlaceholder guards the team-view dimension:
// --by member must group real members apart from the local "" group, and the table must
// render "" as "(local)" -- never a blank cell or the generic "(unknown)" other
// dimensions use, since an unsynced local row isn't a missing value.
func TestAggregateByMemberGroupsAndRendersLocalPlaceholder(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5", Member: "alice", In: 100, Out: 200},
		{Day: "2026-07-02", Tool: "claude-code", Model: "claude-opus-4-5", Member: "", In: 50, Out: 10},
	}
	built := Build(rows, table())
	agg, err := Aggregate(built, "member")
	if err != nil {
		t.Fatal(err)
	}
	if len(agg) != 2 {
		t.Fatalf("len(agg) = %d want 2 (alice, local): %+v", len(agg), agg)
	}
	byMember := map[string]Row{}
	for _, r := range agg {
		byMember[r.Member] = r
	}
	if alice, ok := byMember["alice"]; !ok || alice.In != 100 {
		t.Fatalf("alice group = %+v, ok=%v, want In=100", alice, ok)
	}
	if local, ok := byMember[""]; !ok || local.In != 50 {
		t.Fatalf("local group = %+v, ok=%v, want In=50", local, ok)
	}

	var buf bytes.Buffer
	if err := RenderTable(&buf, agg, "member"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "MEMBER") {
		t.Fatalf("table missing MEMBER header: %s", out)
	}
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

func TestCacheEff(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", In: 200, CacheRead: 800},
		{Day: "d", Tool: "codex", Model: "unknown", In: 0, CacheRead: 0},
	}
	built := Build(rows, table())
	if built[0].CacheEff == nil {
		t.Fatal("CacheEff must be set when In+CacheRead > 0")
	}
	if diff := *built[0].CacheEff - 0.8; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("CacheEff = %v want 0.8", *built[0].CacheEff)
	}
	if built[1].CacheEff != nil {
		t.Fatalf("CacheEff must be nil when In+CacheRead == 0, got %v", *built[1].CacheEff)
	}

	agg, err := Aggregate(built, "tool")
	if err != nil {
		t.Fatal(err)
	}
	var claudeCode Row
	for _, r := range agg {
		if r.Tool == "claude-code" {
			claudeCode = r
		}
	}
	if claudeCode.CacheEff == nil {
		t.Fatal("aggregated CacheEff must be recomputed from summed tokens")
	}
	if diff := *claudeCode.CacheEff - 0.8; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("aggregated CacheEff = %v want 0.8", *claudeCode.CacheEff)
	}
}

func TestRenderTableByDayShowsDayToolModel(t *testing.T) {
	rows := []store.UsageRow{{
		Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5",
		In: 100, Out: 200, CacheWrite: 50, CacheRead: 800,
	}}
	var buf bytes.Buffer
	if err := RenderTable(&buf, Build(rows, table()), "day"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"DAY", "TOOL", "MODEL", "2026-07-01", "claude-code", "claude-opus-4-5"} {
		if !strings.Contains(out, want) {
			t.Fatalf("table missing %q: %s", want, out)
		}
	}
}

func TestRenderTableByProjectShowsNames(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web",
			In: 100, Out: 200, CacheWrite: 50, CacheRead: 800,
		},
		{
			Day: "2026-07-02", Tool: "codex", Model: "claude-opus-4-5", Project: "",
			In: 50, Out: 10,
		},
	}
	built := Build(rows, table())
	agg, err := Aggregate(built, "project")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderTable(&buf, agg, "project"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "PROJECT") {
		t.Fatalf("table missing PROJECT header: %s", out)
	}
	if !strings.Contains(out, "web") {
		t.Fatalf("table missing project name %q: %s", "web", out)
	}
	if !strings.Contains(out, "(unknown)") {
		t.Fatalf("table missing (unknown) label for empty project: %s", out)
	}
	if strings.Contains(out, "MODEL") {
		t.Fatalf("table must not show MODEL column when by=project: %s", out)
	}
}

func TestRenderJSONUnpricedNull(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "d", Tool: "codex", Model: "unknown", In: 10},
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", In: 100, Out: 200, CacheWrite: 50, CacheRead: 800},
	}
	var buf bytes.Buffer
	if err := RenderJSON(&buf, Build(rows, table())); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"cost": null`) {
		t.Fatalf("unpriced row must serialize cost as null: %q", out)
	}
	if !regexp.MustCompile(`"cost": 0\.\d+`).MatchString(out) {
		t.Fatalf("priced row must serialize cost as a number: %q", out)
	}
}
