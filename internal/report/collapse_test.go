package report

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/humanize"
)

// dayRows are two store groups a reader cannot tell apart in the Day/Tool/Model layout:
// same day, same tool, same model, different project.
func dayRows() []Row {
	return []Row{
		{
			Day: "2026-08-03", Tool: "claude-code", Model: "claude-opus-4-7", Project: "web",
			Granularity: "turn", In: 100, Out: 10, CacheRead: 300, CacheWrite: 20,
			Cost: cost(1.25), Priced: true,
		},
		{
			Day: "2026-08-03", Tool: "claude-code", Model: "claude-opus-4-7", Project: "api",
			Granularity: "turn", In: 200, Out: 20, CacheRead: 100, CacheWrite: 5,
			Cost: cost(2.50), Priced: true,
		},
	}
}

func TestCollapseForTable(t *testing.T) {
	other := Row{
		Day: "2026-08-04", Tool: "codex", Model: "gpt-5", Entrypoint: "cli",
		Granularity: "turn", In: 7,
	}
	cases := []struct {
		name string
		rows []Row
		by   string
		want int
	}{
		{"two projects on one day+tool+model fold", dayRows(), "day", 1},
		{"an empty dimension is the ungrouped layout too", dayRows(), "", 1},
		{"a different day stays its own row", append(dayRows(), other), "day", 2},
		{"a grouped dimension is left alone", dayRows(), "project", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CollapseForTable(c.rows, c.by)
			if len(got) != c.want {
				t.Fatalf("CollapseForTable(--by %q) = %d rows, want %d", c.by, len(got), c.want)
			}
		})
	}
}

func TestCollapseForTableSumsTheFoldedRows(t *testing.T) {
	got := CollapseForTable(dayRows(), "day")[0]
	if got.In != 300 || got.Out != 30 || got.CacheRead != 400 || got.CacheWrite != 25 {
		t.Fatalf("tokens = %+v, want the two rows summed", got)
	}
	if !got.Priced || *got.Cost != 3.75 {
		t.Fatalf("cost = %v (priced %v), want 3.75", got.Cost, got.Priced)
	}
	// 400 cache reads of the 700 tokens that could have been read from cache, recomputed on
	// the group rather than carried from either member.
	if got.CacheEff == nil || *got.CacheEff != 400.0/700.0 {
		t.Fatalf("CacheEff = %v, want 400/700", got.CacheEff)
	}
}

// TestCollapseForTableKeepsTheCSVTotal is the honesty rule of the collapse: the table drops
// rows a reader cannot tell apart, never money. Both formats read the same window, so the
// figure they footer has to be the same figure.
func TestCollapseForTableKeepsTheCSVTotal(t *testing.T) {
	rows := dayRows()

	var tableOut bytes.Buffer
	if err := RenderTable(&tableOut, CollapseForTable(rows, "day"), "day"); err != nil {
		t.Fatal(err)
	}
	var csvOut bytes.Buffer
	if err := RenderCSV(&csvOut, rows); err != nil {
		t.Fatal(err)
	}

	records, err := csv.NewReader(strings.NewReader(csvOut.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d CSV lines, want header + 2 detail rows", len(records))
	}
	costCol := columnIndex(t, records[0], "cost")
	var csvTotal float64
	for _, rec := range records[1:] {
		v, err := strconv.ParseFloat(rec[costCol], 64)
		if err != nil {
			t.Fatal(err)
		}
		csvTotal += v
	}

	if n := strings.Count(tableOut.String(), "2026-08-03"); n != 1 {
		t.Fatalf("table shows the day on %d rows, want exactly 1", n)
	}
	want := humanize.USDCell(csvTotal)
	footer := totalLine(t, tableOut.String())
	if !strings.Contains(footer, want) {
		t.Fatalf("table TOTAL %q, want the CSV total %s", footer, want)
	}
}

// TestCollapseForTableMarksMixedGranularity covers the one thing folding can hide: the
// store keeps a session aggregate in its own row, and merging it into a per-turn total is
// exactly the merge the "‡" legend exists to name.
func TestCollapseForTableMarksMixedGranularity(t *testing.T) {
	rows := []Row{
		{Day: "2026-08-03", Tool: "claude-code", Model: "m", Granularity: "turn", In: 10},
		{Day: "2026-08-03", Tool: "claude-code", Model: "m", Granularity: "session", In: 5},
	}
	got := CollapseForTable(rows, "day")
	if len(got) != 1 || got[0].Granularity != GranularityMixed {
		t.Fatalf("CollapseForTable = %+v, want one %s row", got, GranularityMixed)
	}
	var buf bytes.Buffer
	if err := RenderTable(&buf, got, "day"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "‡") || !strings.Contains(buf.String(), mixedGranularityFootnote) {
		t.Fatalf("table must footnote the mixed total: %q", buf.String())
	}
}

func TestCollapseForTableKeepsUnpricedEvidence(t *testing.T) {
	rows := []Row{
		{Day: "2026-08-03", Tool: "claude-code", Model: "m", Project: "web", In: 10, Cost: cost(1), Priced: true},
		{Day: "2026-08-03", Tool: "claude-code", Model: "m", Project: "api", In: 40, HasUnpriced: true, UnpricedTokens: 40},
	}
	got := CollapseForTable(rows, "day")
	if len(got) != 1 {
		t.Fatalf("want one row, got %d", len(got))
	}
	if !got[0].HasUnpriced || got[0].UnpricedTokens != 40 {
		t.Fatalf("unpriced evidence = (%v, %d), want (true, 40)", got[0].HasUnpriced, got[0].UnpricedTokens)
	}
	if !got[0].Priced || *got[0].Cost != 1 {
		t.Fatalf("cost = %v, want the priced half kept", got[0].Cost)
	}
}

func totalLine(t *testing.T, rendered string) string {
	t.Helper()
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "TOTAL") {
			return line
		}
	}
	t.Fatalf("no TOTAL footer in %q", rendered)
	return ""
}
