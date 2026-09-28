package report

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

type lineRow struct {
	tool          string
	lines, tokens int64
	cost          float64
	priced        bool
}

// TestLineRateDividesOnlyWhatCouldHaveFedItsLines: both sides of $/100 lines come from priced
// usage of sources that record lines, and the disclosure says what else the window holds --
// in tokens, because a line-blind source on an unpriced model has no cost share to state.
func TestLineRateDividesOnlyWhatCouldHaveFedItsLines(t *testing.T) {
	claude := lineRow{"claude-code", 40, 1000, 2, true}
	tests := []struct {
		name     string
		rows     []lineRow
		wantRate float64 // 0 means nil
		wantNote string  // "" means no disclosure
	}{
		{"line-recording usage alone", []lineRow{claude}, 5, ""},
		{
			"a priced line-blind source is left out of the numerator",
			[]lineRow{claude, {"gemini-cli", 0, 3000, 6, true}},
			5, "75% from sources that record no lines (75% of the priced cost)",
		},
		{
			"an unpriced line-blind source is stated in tokens",
			[]lineRow{claude, {"cline", 0, 3000, 0, false}},
			5, "75% from sources that record no lines.",
		},
		{
			"unpriced line-recording usage leaves its lines out too",
			[]lineRow{claude, {"claude-code", 60, 1000, 0, false}},
			5, "50% on line-recording models with no known price (60 of 100 AI lines)",
		},
		{
			"only unpriced line-recording usage withholds the rate",
			[]lineRow{{"claude-code", 40, 1000, 0, false}, {"gemini-cli", 0, 1000, 3, true}},
			0, "ran only models with no known price",
		},
		{
			"every line from unpriced usage beside priced usage that wrote none",
			[]lineRow{{"claude-code", 0, 1000, 2, true}, {"claude-code", 40, 1000, 0, false}},
			0, "Every AI line in this window came from usage with no known price",
		},
		{
			"priced and unpriced line-recording usage that wrote no line claims nothing",
			[]lineRow{{"claude-code", 0, 1000, 2, true}, {"claude-code", 0, 1000, 0, false}},
			0, "",
		},
		{"no line-recording source", []lineRow{{"gemini-cli", 0, 1000, 3, true}}, 0, "absent, not zero"},
		{"an activity-only source with no tokens still has no lines", []lineRow{{"agy", 0, 0, 0, false}}, 0, "absent, not zero"},
		{"priced lines with no tokens are no rate", []lineRow{{"claude-code", 40, 0, 0, true}}, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b LineRateBasis
			for _, r := range tt.rows {
				b.add(r.tool, r.lines, r.tokens, r.cost, r.priced)
			}
			got := b.Per100()
			switch {
			case tt.wantRate == 0 && got != nil:
				t.Fatalf("Per100 = %v, want none", *got)
			case tt.wantRate != 0 && (got == nil || *got != tt.wantRate):
				t.Fatalf("Per100 = %v, want %v", got, tt.wantRate)
			}
			note := lineRateNote(&b, "this window")
			if (tt.wantNote == "") != (note == "") || !strings.Contains(note, tt.wantNote) {
				t.Fatalf("disclosure = %q, want %q", note, tt.wantNote)
			}
			if strings.HasPrefix(note, "†") != b.Partial() {
				t.Fatalf("disclosure %q: † prefix must mark exactly a ratio that leaves usage out", note)
			}
		})
	}
}

// mixedProject is one project whose priced Claude Code usage wrote 40 lines beside priced
// Gemini CLI usage that records none.
func mixedProject(day string) []store.UsageRow {
	return []store.UsageRow{
		{Day: day, Tool: "claude-code", Model: "claude-opus-4-5", Project: "web", In: 1000, Out: 500, LinesAdded: 40},
		{Day: day, Tool: "gemini-cli", Model: "claude-opus-4-5", Project: "web", In: 4000, Out: 2000},
	}
}

// TestAGroupsRateDividesOnlyItsLineRecordingUsage: a project mixing sources divides its Claude
// Code cost alone by its lines, in the machine outputs and in `status`'s Hot list alike.
func TestAGroupsRateDividesOnlyItsLineRecordingUsage(t *testing.T) {
	claudeCost, _ := table().CostTokens("claude-opus-4-5", pricing.Tokens{In: 1000, Out: 500})
	want := claudeCost / 0.4
	rows := mixedProject("2026-08-15")

	eff, err := BuildEffectiveness(rows, table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if len(eff) != 1 || eff[0].CostPer100Lines == nil || *eff[0].CostPer100Lines != want {
		t.Fatalf("effectiveness cost_per_100_lines = %v, want %v", eff[0].CostPer100Lines, want)
	}
	if eff[0].LineCapableCost == nil || *eff[0].LineCapableCost != claudeCost {
		t.Fatalf("line_capable_cost = %v, want %v", eff[0].LineCapableCost, claudeCost)
	}

	in := BuildInsights(rows, table(), time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC), 7*24*time.Hour, 5)
	if len(in.Hot) != 1 || in.Hot[0].CostPer100Lines == nil || *in.Hot[0].CostPer100Lines != want {
		t.Fatalf("Hot cost_per_100_lines = %v, want %v", in.Hot, want)
	}
	var buf bytes.Buffer
	if err := RenderStatusSummary(&buf, &in); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "web — ") || !strings.Contains(buf.String(), "†/100 lines") {
		t.Fatalf("want the Hot row's partial ratio marked:\n%s", buf.String())
	}
}

// TestEffectivenessFooterDividesTheRowsPopulation: grouped by tool, a line-blind row shows no
// ratio of its own, so its cost must not reach the footer's ratio either -- while the COST
// total still counts it.
func TestEffectivenessFooterDividesTheRowsPopulation(t *testing.T) {
	eff, err := BuildEffectiveness(mixedProject("d"), table(), "tool")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "tool"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"0.088", "0.044†", "† $/100 lines divides only priced usage", "Of the tokens in this table"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

// TestTheFooterStatesUnpricedLinesInLines: grouped by model, the unpriced model's lines reach
// the footer only through the merged basis, and its row exports the tokens it leaves out.
func TestTheFooterStatesUnpricedLinesInLines(t *testing.T) {
	eff, err := BuildEffectiveness([]store.UsageRow{
		{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", In: 1000, Out: 500, LinesAdded: 40},
		{Day: "d", Tool: "claude-code", Model: "unpriced-local", In: 700, Out: 300, LinesAdded: 60},
	}, table(), "model")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "model"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "(60 of 100 AI lines)") {
		t.Fatalf("want the unpriced lines stated against the total:\n%s", buf.String())
	}
	records := effCSVRecords(t, eff)
	col := columnIndex(t, records[0], "line_capable_unpriced_tokens")
	for _, rec := range records[1:] {
		want := "0"
		if rec[columnIndex(t, records[0], "group")] == "unpriced-local" {
			want = "1000"
		}
		if rec[col] != want {
			t.Fatalf("%v: line_capable_unpriced_tokens = %q, want %q", rec, rec[col], want)
		}
	}
}

// TestEffectivenessCSVAppendsTheLineRateColumns: a consumer reading CSV by position keeps working,
// because the columns this basis added come after every column it already read.
func TestEffectivenessCSVAppendsTheLineRateColumns(t *testing.T) {
	eff, err := BuildEffectiveness(mixedProject("d"), table(), "project")
	if err != nil {
		t.Fatal(err)
	}
	header := effCSVRecords(t, eff)[0]
	want := []string{"line_capable_cost", "line_capable_unpriced_tokens", "line_capable_shared_tokens"}
	if got := header[len(header)-len(want):]; !slices.Equal(got, want) {
		t.Fatalf("header ends %q, want the line-rate columns last: %q", got, want)
	}
}
