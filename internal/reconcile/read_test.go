package reconcile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestReadCSV(t *testing.T) {
	path := writeTemp(t, "usage.csv", `Usage Date,Model,Total Tokens,Cost USD,Currency
2026-08-01,claude-sonnet-4-5,1200,"$1,234.50",USD
2026-08-02,claude-sonnet-4-5,900,12.25,USD
not-a-date,claude-sonnet-4-5,900,12.25,USD
2026-08-03,claude-sonnet-4-5,900,n/a,USD
`)
	exp, err := ReadFile(path, nil)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(exp.Rows) != 2 {
		t.Fatalf("read %d rows, want 2 (two unreadable ones skipped)", len(exp.Rows))
	}
	if exp.Skipped != 2 || exp.Total != 4 {
		t.Fatalf("skipped %d of %d, want 2 of 4", exp.Skipped, exp.Total)
	}
	if exp.Rows[0].Cost != 1234.50 || exp.Rows[0].Day != "2026-08-01" {
		t.Fatalf("first row = %+v", exp.Rows[0])
	}
	if !exp.Rows[0].TokensStated || exp.Rows[0].Tokens != 1200 {
		t.Fatalf("tokens = %d stated=%v", exp.Rows[0].Tokens, exp.Rows[0].TokensStated)
	}
	if exp.Currency != "USD" {
		t.Fatalf("currency = %q", exp.Currency)
	}
	if exp.Binding.Day != "Usage Date" || exp.Binding.Cost != "Cost USD" {
		t.Fatalf("binding = %+v", exp.Binding)
	}
}

func TestReadCSVWithoutATokenColumnLeavesTokensUnstated(t *testing.T) {
	path := writeTemp(t, "usage.csv", "date,amount\n2026-08-01,4.72\n")
	exp, err := ReadFile(path, nil)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if exp.Rows[0].TokensStated {
		t.Fatal("an absent token column must stay unstated, never a zero the vendor did not say")
	}
	if exp.Binding.Tokens != "" {
		t.Fatalf("tokens bound to %q with no such column", exp.Binding.Tokens)
	}
}

func TestReadJSON(t *testing.T) {
	path := writeTemp(t, "usage.json", `[
	  {"date":"2026-08-01","model":"sonnet","amount":10.5},
	  {"date":"2026-08-02","model":"sonnet","amount":2.25,"total_tokens":900}
	]`)
	exp, err := ReadFile(path, nil)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(exp.Rows) != 2 {
		t.Fatalf("read %d rows, want 2", len(exp.Rows))
	}
	// The token key appears only on the second object; the union of keys must still bind it.
	if exp.Binding.Tokens != "total_tokens" {
		t.Fatalf("tokens bound to %q, want total_tokens", exp.Binding.Tokens)
	}
	if exp.Rows[0].TokensStated {
		t.Fatal("the first object states no tokens")
	}
	if !exp.Rows[1].TokensStated || exp.Rows[1].Tokens != 900 {
		t.Fatalf("second row tokens = %d stated=%v", exp.Rows[1].Tokens, exp.Rows[1].TokensStated)
	}
}

func TestReadFileRejectsAnExportItCannotBind(t *testing.T) {
	path := writeTemp(t, "usage.csv", "widget,quantity\nfoo,2\n")
	_, err := ReadFile(path, nil)
	if err == nil {
		t.Fatal("an export with no date or money column must be an error naming what it saw")
	}
}

func TestOverridesWinOverAliases(t *testing.T) {
	path := writeTemp(t, "usage.csv", "date,amount,charge\n2026-08-01,1.00,9.99\n")
	exp, err := ReadFile(path, map[string]string{FieldCost: "charge"})
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if exp.Rows[0].Cost != 9.99 {
		t.Fatalf("cost = %v, want the overridden column's 9.99", exp.Rows[0].Cost)
	}
}

// TestReadFileRefusesATruncatedExport is the truncation defect: io.LimitReader's spare byte was there and nothing
// compared against it, so csv.Reader saw the cut as a clean end of file. Skipped stayed 0, no
// Limits entry was written, and every row past the cut was reported as Unexplained -- the one
// number this command exists to produce, inflated by a file it silently stopped reading.
func TestReadFileRefusesATruncatedExport(t *testing.T) {
	defer func(v int64) { maxExportBytes = v }(maxExportBytes)
	maxExportBytes = 64

	var b strings.Builder
	b.WriteString("date,cost\n")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&b, "2026-08-%02d,1.00\n", i)
	}
	path := writeTemp(t, "usage.csv", b.String())
	if _, err := ReadFile(path, nil); err == nil {
		t.Fatal("an export past the size bound must be an error, never a partial reconciliation")
	} else if !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("the error must name the bound it hit, got %q", err)
	}
}
