package cli

import (
	"bytes"
	"encoding/csv"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/usage"
)

// TestReportDayTableCollapsesInvisibleDimensions is the wiring half of the fold: the store
// keys a group by project, entrypoint, member and granularity, none of which the default
// Day/Tool/Model table has a column for, so the same day printed as several identical lines
// and the first of them read as the day's total. The machine formats keep every group.
func TestReportDayTableCollapsesInvisibleDimensions(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	db := filepath.Join(t.TempDir(), "usage.db")
	ts := time.Now().UTC()
	day := ts.Format("2006-01-02")
	seedStoreAt(t, db, []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "claude-opus-4-5",
			InputTokens: 100, OutputTokens: 200, Project: "web", DedupeKey: "1",
		},
		{
			Tool: "claude-code", SessionID: "s2", Timestamp: ts, Model: "claude-opus-4-5",
			InputTokens: 50, OutputTokens: 10, Project: "api", DedupeKey: "2",
		},
	})

	table := runReportFormat(t, db, "table")
	if n := strings.Count(table, day); n != 1 {
		t.Fatalf("table shows %s on %d rows, want exactly 1:\n%s", day, n, table)
	}

	records, err := csv.NewReader(strings.NewReader(runReportFormat(t, db, "csv"))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("csv = %d lines, want header + 2 project rows: %v", len(records), records)
	}

	want := humanize.USDCell(csvCostTotal(t, records))
	if !strings.Contains(totalFooter(t, table), want) {
		t.Fatalf("table TOTAL %q, want the csv total %s", totalFooter(t, table), want)
	}
}

func runReportFormat(t *testing.T, db, format string) string {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report", "--db", db, "--format", format})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func csvCostTotal(t *testing.T, records [][]string) float64 {
	t.Helper()
	col := -1
	for i, h := range records[0] {
		if h == "cost" {
			col = i
		}
	}
	if col < 0 {
		t.Fatalf("csv header %v has no cost column", records[0])
	}
	var total float64
	for _, rec := range records[1:] {
		if rec[col] == "" {
			continue
		}
		v, err := strconv.ParseFloat(rec[col], 64)
		if err != nil {
			t.Fatal(err)
		}
		total += v
	}
	return total
}

func totalFooter(t *testing.T, rendered string) string {
	t.Helper()
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "TOTAL") {
			return line
		}
	}
	t.Fatalf("no TOTAL footer in %q", rendered)
	return ""
}
