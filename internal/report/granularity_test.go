package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

func TestBuildCarriesGranularityFromTheStore(t *testing.T) {
	rows := Build([]store.UsageRow{
		{Day: "2026-07-01", Tool: "plugin:acme", Model: "m", Granularity: "session", In: 10},
	}, pricing.Table{})
	if len(rows) != 1 || rows[0].Granularity != "session" {
		t.Fatalf("Build = %+v, want the row's granularity preserved", rows)
	}
}

// TestAggregateMarksATotalThatBlendsGranularities is the honesty rule at the exact point
// it can be broken: grouping is what merges a session-aggregate row into a per-turn total,
// and a merged total that still claims "turn" is the silent misread B69 exists to stop.
func TestAggregateMarksATotalThatBlendsGranularities(t *testing.T) {
	got, err := Aggregate([]Row{
		{Tool: "claude-code", Granularity: "turn", In: 100},
		{Tool: "plugin:acme", Granularity: "session", In: 50},
	}, "member")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want one group, got %d", len(got))
	}
	if got[0].Granularity != GranularityMixed {
		t.Fatalf("Granularity = %q, want %q", got[0].Granularity, GranularityMixed)
	}
}

func TestAggregateKeepsAUniformGranularity(t *testing.T) {
	got, err := Aggregate([]Row{
		{Tool: "claude-code", Granularity: "turn", In: 100},
		{Tool: "codex", Granularity: "turn", In: 50},
	}, "member")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Granularity != "turn" {
		t.Fatalf("Granularity = %q, want turn", got[0].Granularity)
	}
}

func TestRenderTableFootnotesSessionGranularity(t *testing.T) {
	var buf bytes.Buffer
	err := RenderTable(&buf, []Row{
		{Day: "2026-07-01", Tool: "plugin:acme", Model: "m", Granularity: "session", In: 10},
	}, "day")
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "session") {
		t.Fatalf("table must disclose session-granularity rows: %q", out)
	}
}

func TestRenderTableStaysQuietOnUniformTurnRows(t *testing.T) {
	var buf bytes.Buffer
	err := RenderTable(&buf, []Row{
		{Day: "2026-07-01", Tool: "claude-code", Model: "m", Granularity: "turn", In: 10},
	}, "day")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "session-granularity") {
		t.Fatalf("uniform per-turn rows need no footnote: %q", buf.String())
	}
}

func TestRenderCSVCarriesGranularity(t *testing.T) {
	var buf bytes.Buffer
	err := RenderCSV(&buf, []Row{
		{Day: "2026-07-01", Tool: "plugin:acme", Model: "m", Granularity: "session"},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "granularity") || !strings.Contains(out, "session") {
		t.Fatalf("CSV must carry granularity: %q", out)
	}
}
