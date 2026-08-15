package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

func cost(v float64) *float64 { return &v }

func TestRenderStatusSummaryWithData(t *testing.T) {
	in := Insights{
		Hot: []GroupStat{
			{Name: "web", Cost: cost(0.0175), LinesAdded: 100, CostPer100Lines: cost(0.0175)},
		},
		GoingStale: []GroupStat{
			{Name: "infra", Cost: cost(0.005), LastActive: "2026-07-02"},
		},
		DormantTools: []GroupStat{
			{Name: "codex"},
		},
		Inventory: Inventory{
			Projects: 3, Models: 1, Tools: 2, Entrypoints: 1, Days: 5,
			TotalCost: cost(0.0225), TotalLinesAdded: 100,
		},
	}
	var buf bytes.Buffer
	if err := RenderStatusSummary(&buf, &in); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"3 projects", "1 models", "2 tools", "5 days of history",
		"Hot — where spend concentrates", "web",
		"Going stale", "infra", "2026-07-02",
		"Dormant tools", "codex", "set up, no recent activity",
		"directional", "never a per-person metric",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("status summary missing %q: %s", want, out)
		}
	}
}

func TestRenderStatusSummaryEmptySectionsAreHonest(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderStatusSummary(&buf, &Insights{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"0 projects", "0 models", "0 tools", "0 days of history",
		"Nothing has gone quiet.",
		"All detected tools active.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("empty-section status summary missing %q: %s", want, out)
		}
	}
	if strings.Contains(out, "Dormant tools") {
		t.Fatalf("no dormant tools must not print a Dormant tools header: %s", out)
	}
}

func TestRenderStatusSummaryHeadlineDashWhenNoLines(t *testing.T) {
	var buf bytes.Buffer
	in := Insights{Inventory: Inventory{TotalCost: cost(1.5)}}
	if err := RenderStatusSummary(&buf, &in); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "—/100 lines") {
		t.Fatalf("zero-line inventory must render the ratio as a dash: %s", buf.String())
	}
}

func TestRenderStatusSummaryUnpricedTotalMarksAsterisk(t *testing.T) {
	var buf bytes.Buffer
	in := Insights{Inventory: Inventory{TotalCost: cost(1.5), HasUnpriced: true, TotalLinesAdded: 10}}
	if err := RenderStatusSummary(&buf, &in); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "$1.50*") {
		t.Fatalf("unpriced total must carry an asterisk: %s", buf.String())
	}
}

func TestRenderChurnLineShowsCountAndRate(t *testing.T) {
	var buf bytes.Buffer
	c := ChurnStat{LinesAdded: 200, ReworkLines: 40, ReworkRate: 0.2, Rows: 2}
	if err := RenderChurnLine(&buf, c); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"rework:", "40", "20%", "thrash proxy", "within-session"} {
		if !strings.Contains(out, want) {
			t.Fatalf("churn line missing %q: %s", want, out)
		}
	}
}

func TestRenderChurnLineZeroIsHonest(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderChurnLine(&buf, ChurnStat{Rows: 3}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"rework: 0 lines reworked", "—"} {
		if !strings.Contains(out, want) {
			t.Fatalf("zero churn line missing %q: %s", want, out)
		}
	}
}

// A measured zero and a window nothing could measure must not print the same line.
func TestRenderChurnLineSaysWhenNoSourceRecordsIt(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderChurnLine(&buf, ChurnStat{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "not recorded by any source") {
		t.Fatalf("churn line must name the missing capture, got: %s", out)
	}
	if strings.Contains(out, "0%") {
		t.Fatalf("churn line must not print a rate nothing measured, got: %s", out)
	}
}

func TestRenderEmptyStatusHint(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderEmptyStatusHint(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"No usage yet", "backfill", "status"} {
		if !strings.Contains(out, want) {
			t.Fatalf("empty status hint missing %q: %s", want, out)
		}
	}
}

// TestStatusWithholdsLinesFromASourceThatNeverRecordsThem is the same ADR 0011 rule on the
// terminal dashboard: `%d AI lines` over a Gemini-only window printed "0 AI lines", and the
// $/100 ratio built on it divided the whole window's cost by that fabricated zero.
func TestStatusWithholdsLinesFromASourceThatNeverRecordsThem(t *testing.T) {
	var buf bytes.Buffer
	in := BuildInsights([]store.UsageRow{
		{Day: "2026-08-15", Tool: "gemini-cli", Model: "gemini-2.5-pro", In: 1000, Out: 500},
	}, pricing.Table{}, time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC), 7*24*time.Hour, 5)
	if err := RenderStatusSummary(&buf, &in); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "0 AI lines") {
		t.Fatalf("a structural silence rendered as a measured zero: %s", out)
	}
	if !strings.Contains(out, "— AI lines") {
		t.Fatalf("want the AI-line figure withheld: %s", out)
	}
	if !strings.Contains(out, "absent, not zero") {
		t.Fatalf("want the withheld figure to say why: %s", out)
	}
}

// TestStatusDisclosesAPartialLineDenominator is B5: the cost in $/100 lines is the whole
// window's while the lines come only from the sources that record them. `effectiveness` and
// the dashboard colophon both said so; `status` printed the ratio with no disclosure at all.
func TestStatusDisclosesAPartialLineDenominator(t *testing.T) {
	var buf bytes.Buffer
	in := BuildInsights([]store.UsageRow{
		{Day: "2026-08-15", Tool: "claude-code", Model: "claude-opus-4-5", In: 1000, Out: 500, LinesAdded: 40},
		{Day: "2026-08-15", Tool: "gemini-cli", Model: "gemini-2.5-pro", In: 4000, Out: 2000},
	}, pricing.Table{}, time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC), 7*24*time.Hour, 5)
	if err := RenderStatusSummary(&buf, &in); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "only from the sources that record them") {
		t.Fatalf("want the ratio's two populations disclosed: %s", buf.String())
	}
}
