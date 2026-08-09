package report

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

// TestCSVCarriesTheAnnotationColumns covers `report --by task|outcome|difficulty --format csv`.
// Aggregating on a label dimension stamps the group key into Task/Outcome/Difficulty and
// leaves every other identity column empty, so a header without those three emitted rows that
// were byte-for-byte indistinguishable from one another -- a machine format with no key in it.
func TestCSVCarriesTheAnnotationColumns(t *testing.T) {
	var buf bytes.Buffer
	rows := []Row{
		{Task: "feature", In: 10, Out: 1},
		{Task: "bugfix", In: 20, Out: 2},
	}
	if err := RenderCSV(&buf, rows); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(buf.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d CSV lines, want header + 2 rows", len(records))
	}
	taskCol := columnIndex(t, records[0], "task")
	for _, want := range []string{"outcome", "difficulty"} {
		columnIndex(t, records[0], want)
	}
	if records[1][taskCol] != "feature" || records[2][taskCol] != "bugfix" {
		t.Fatalf("task column = %q/%q, want feature/bugfix", records[1][taskCol], records[2][taskCol])
	}
	for i, rec := range records[1:] {
		if len(rec) != len(records[0]) {
			t.Fatalf("row %d has %d fields, header has %d", i, len(rec), len(records[0]))
		}
	}
}

func columnIndex(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, h := range header {
		if h == name {
			return i
		}
	}
	t.Fatalf("CSV header %v has no %q column", header, name)
	return -1
}
