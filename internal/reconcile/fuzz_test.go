package reconcile

import (
	"strings"
	"testing"
	"time"
)

// FuzzReadCSV drives the reader with arbitrary bytes: an export is a file from outside the
// project, so the reader must skip and count anything it cannot make sense of rather than
// panic or invent a row. The invariant checked here is the one a delta rests on -- every row
// that survives carries a real date and a parsed amount.
func FuzzReadCSV(f *testing.F) {
	f.Add("date,amount\n2026-08-01,4.72\n")
	f.Add("Usage Date,Model,Cost USD\n2026-08-01,sonnet,\"$1,234.50\"\n")
	f.Add("date,amount\nnot-a-date,4.72\n2026-08-01,n/a\n")
	f.Add("date,amount\n2026-08-01,(12.50)\n")
	f.Add("widget,quantity\nfoo,2\n")
	f.Add("")

	start, _ := time.Parse("2006-01-02", "2026-08-01")
	end := start.AddDate(0, 0, 30)

	f.Fuzz(func(t *testing.T, body string) {
		exp, err := readCSV(strings.NewReader(body), nil)
		if err != nil {
			return
		}
		if exp.Skipped > exp.Total {
			t.Fatalf("skipped %d of %d rows", exp.Skipped, exp.Total)
		}
		for _, r := range exp.Rows {
			if _, ok := parseDay(r.Day); !ok {
				t.Fatalf("row kept an unparseable day %q", r.Day)
			}
		}
		// A parsed export must always reconcile without panicking, including against no
		// local usage at all -- the case a first-time user hits.
		if _, err := Reconcile(exp, nil, start, end); err != nil && exp.Currency == "" {
			t.Fatalf("reconcile of a USD export failed: %v", err)
		}
	})
}
