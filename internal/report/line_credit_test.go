package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

func creditTable() pricing.Table {
	return pricing.Table{
		"premium": {Input: 1e-5},
		"cheap":   {Input: 1e-6},
	}
}

// sessionRow is one model's record of a Copilot CLI session: every model of a session shares
// its day, project and grain, and only the busiest carries the session's lines.
func sessionRow(day, model string, tokens, lines int64) store.UsageRow {
	return store.UsageRow{
		Day: day, Tool: "copilot-cli", Model: model, Project: "p", Granularity: "session",
		In: tokens, LinesAdded: lines,
	}
}

// TestASessionsLinesArePairedWithTheWholeSessionsCost covers the two ways a row-level price gate
// mis-pairs a session-credited source: its lines with part of its cost, and its cost with none
// of its lines.
func TestASessionsLinesArePairedWithTheWholeSessionsCost(t *testing.T) {
	deep := store.UsageRow{Day: "2026-09-01", Tool: "claude-code", Model: "premium", Project: "p", Granularity: "turn", In: 1000, LinesAdded: 50}
	tests := []struct {
		name                    string
		rows                    []store.UsageRow
		wantLines, wantTokens   int64
		wantUnpriced            int64
		wantShared, sharedLines int64
		wantRatioNote           bool
	}{
		{
			"lines on the priced model, an unpriced one beside it",
			[]store.UsageRow{sessionRow("2026-09-01", "premium", 1000, 100), sessionRow("2026-09-01", "mystery", 1000, 0)},
			0, 0, 0, 2000, 100, false,
		},
		{
			"lines on the unpriced model, the priced one's cost has none of them",
			[]store.UsageRow{deep, sessionRow("2026-09-01", "mystery", 1000, 100), sessionRow("2026-09-01", "premium", 1000, 0)},
			50, 1000, 0, 2000, 100, true,
		},
		{
			"every model priced: the session's lines against its whole cost",
			[]store.UsageRow{sessionRow("2026-09-01", "premium", 1000, 100), sessionRow("2026-09-01", "cheap", 1000, 0)},
			100, 2000, 0, 0, 0, false,
		},
		{
			"an unpriced model on another day belongs to another session",
			[]store.UsageRow{sessionRow("2026-09-01", "premium", 1000, 100), sessionRow("2026-09-02", "mystery", 1000, 0)},
			100, 1000, 1000, 0, 0, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := BuildInventory(tt.rows, creditTable()).LineRate
			if b.Lines != tt.wantLines || b.Tokens != tt.wantTokens {
				t.Errorf("ratio covers %d lines over %d tokens, want %d over %d", b.Lines, b.Tokens, tt.wantLines, tt.wantTokens)
			}
			if b.UnpricedLineTokens != tt.wantUnpriced || b.SharedTokens != tt.wantShared || b.SharedLines != tt.sharedLines {
				t.Errorf("left out %d unpriced tokens and %d shared tokens with %d lines, want %d, %d and %d",
					b.UnpricedLineTokens, b.SharedTokens, b.SharedLines, tt.wantUnpriced, tt.wantShared, tt.sharedLines)
			}
			if got := b.Partial(); got != tt.wantRatioNote {
				t.Errorf("Partial = %v, want %v", got, tt.wantRatioNote)
			}
		})
	}
}

// TestAPerModelSplitCreditsNoModelWithASharedSession: under --by model, the busiest model of a
// multi-model session would otherwise carry every line the session wrote against its own
// tokens alone. A single-model session, and a source whose lines are its own model's, keep
// their ratio.
func TestAPerModelSplitCreditsNoModelWithASharedSession(t *testing.T) {
	rows := []store.UsageRow{
		sessionRow("2026-09-01", "premium", 1000, 100),
		sessionRow("2026-09-01", "cheap", 3000, 0),
		sessionRow("2026-09-02", "premium", 1000, 40),
		{Day: "2026-09-01", Tool: "claude-code", Model: "cheap", Project: "p", Granularity: "session", In: 2000, LinesAdded: 20},
		{Day: "2026-09-01", Tool: "claude-code", Model: "premium", Project: "p", Granularity: "session", In: 2000, LinesAdded: 10},
	}
	eff, err := BuildEffectiveness(rows, creditTable(), "model")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct{ lines, tokens, shared, sharedLines int64 }{
		"premium": {50, 3000, 1000, 100},
		"cheap":   {20, 2000, 3000, 0},
	}
	if len(eff) != len(want) {
		t.Fatalf("got %d model rows, want %d", len(eff), len(want))
	}
	for i := range eff {
		r := &eff[i]
		w, ok := want[r.Group]
		if !ok {
			t.Fatalf("unexpected group %q", r.Group)
		}
		if r.lineRate.Lines != w.lines || r.lineRate.Tokens != w.tokens || r.lineRate.SharedTokens != w.shared || r.lineRate.SharedLines != w.sharedLines {
			t.Errorf("%s: lines %d over %d tokens, shared %d tokens and %d lines; want %+v",
				r.Group, r.lineRate.Lines, r.lineRate.Tokens, r.lineRate.SharedTokens, r.lineRate.SharedLines, w)
		}
		if r.LineCapableSharedTokens != w.shared {
			t.Errorf("%s: line_capable_shared_tokens = %d, want %d", r.Group, r.LineCapableSharedTokens, w.shared)
		}
	}

	byProject, err := BuildEffectiveness(rows, creditTable(), "project")
	if err != nil {
		t.Fatal(err)
	}
	if model, project := footerOf(t, eff, "model"), footerOf(t, byProject, "project"); model != project {
		t.Errorf("TOTAL moves with the grouping:\n by model   %s\n by project %s", model, project)
	}

	if b := byProject[0].lineRate; b.SharedTokens != 0 || b.Lines != 170 {
		t.Errorf("by project: %d lines, %d shared tokens; want every line paired and nothing shared", b.Lines, b.SharedTokens)
	}
}

// TestASharedSessionIsDisclosedBesideTheRatio: the tokens a per-model ratio leaves out are named
// with the lines they carried, and a model whose every line came from shared sessions gets no
// ratio rather than one set against usage that wrote none.
func TestASharedSessionIsDisclosedBesideTheRatio(t *testing.T) {
	var partial, none LineRateBasis
	partial.add("copilot-cli", 40, 1000, 1, true)
	partial.SharedTokens, partial.SharedLines = 1000, 60
	none.add("copilot-cli", 0, 1000, 1, true)
	none.SharedTokens, none.SharedLines = 1000, 60

	if note := lineRateNote(&partial, "this row"); !strings.Contains(note, "50% in "+SharedUsage+" (60 of 100 AI lines)") {
		t.Errorf("partial note = %q", note)
	}
	if none.Per100() != nil {
		t.Fatalf("Per100 = %v, want none when every line is shared", *none.Per100())
	}
	if note := lineRateNote(&none, "this row"); !strings.Contains(note, "Every AI line in this row came from "+SharedUsage) {
		t.Errorf("withheld note = %q", note)
	}
}

// TestHotProjectsPairASessionsLinesWithItsWholeCost: the status block's per-project ratio goes
// through the same credit as the window's, so a session with an unpriced model is withheld there
// too rather than set against its priced model's cost alone.
func TestHotProjectsPairASessionsLinesWithItsWholeCost(t *testing.T) {
	rows := []store.UsageRow{sessionRow("2026-09-01", "premium", 1000, 100), sessionRow("2026-09-01", "mystery", 1000, 0)}
	stats := groupStats(rows, projectKey, creditTable())
	if len(stats) != 1 {
		t.Fatalf("got %d groups, want 1", len(stats))
	}
	if stats[0].CostPer100Lines != nil {
		t.Fatalf("CostPer100Lines = %v, want none: the session's cost is not fully known", *stats[0].CostPer100Lines)
	}
}

// footerOf is the TOTAL row of the rendered table, with the group column's width squeezed out.
func footerOf(t *testing.T, rows []EffRow, by string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, rows, by); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, "TOTAL") {
			return strings.Join(strings.Fields(line), " ")
		}
	}
	t.Fatalf("no TOTAL row:\n%s", buf.String())
	return ""
}

// TestASharedSessionStillCountsAsLineRecording: a session left out of the ratio for its unpriced
// model still ran on a source that records lines, so the coverage note does not report the table
// as partly line-blind.
func TestASharedSessionStillCountsAsLineRecording(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-09-01", Tool: "claude-code", Model: "premium", Project: "p", Granularity: "turn", In: 1000, LinesAdded: 50},
		sessionRow("2026-09-01", "premium", 1000, 100),
		sessionRow("2026-09-01", "mystery", 1000, 0),
	}
	eff, err := BuildEffectiveness(rows, creditTable(), "project")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderEffectivenessTable(&buf, eff, "project"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "of this table's tokens") {
		t.Errorf("coverage note reads the shared session as line-blind:\n%s", buf.String())
	}
}

// TestAWithheldRatioNamesWhatItCouldNotPair: with no ratio left, the sentence names every kind of
// usage the ratio could not pair, and only those.
func TestAWithheldRatioNamesWhatItCouldNotPair(t *testing.T) {
	tests := []struct {
		name                    string
		pairedTokens            int64
		shared, sharedLines     int64
		unpriced, unpricedLines int64
		want                    string
	}{
		{"only shared usage", 0, 1000, 60, 0, 0, "All line-recording usage in this row is " + SharedUsage + ", so"},
		{"shared and unpriced usage", 0, 1000, 60, 1000, 10, "is either " + SharedUsage + " or on models with no known price"},
		{"lines only from shared and unpriced usage", 1000, 1000, 60, 1000, 10, "came from " + SharedUsage + " or from usage with no known price"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b LineRateBasis
			if tt.pairedTokens > 0 {
				b.add("copilot-cli", 0, tt.pairedTokens, 1, true)
			}
			if tt.unpriced > 0 {
				b.add("copilot-cli", tt.unpricedLines, tt.unpriced, 0, false)
			}
			b.LineRows++
			b.SharedTokens, b.SharedLines = tt.shared, tt.sharedLines
			if note := lineRateNote(&b, "this row"); !strings.Contains(note, tt.want) {
				t.Errorf("note = %q, want it to contain %q", note, tt.want)
			}
		})
	}
}
