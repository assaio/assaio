package reconcile

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/report"
)

func cost(v float64) *float64 { return &v }

// localRow is a priced row of the local estimate; tokens matter only for the extrapolation.
func localRow(d, model string, in, out int64, c *float64) report.Row {
	r := report.Row{Day: d, Model: model, In: in, Out: out}
	if c != nil {
		r.Cost, r.Priced = c, true
	}
	return r
}

// window is the queried range every case in this file compares over: it opens on the first
// of August and the case chooses how far it runs.
func window(last string) (start, end time.Time) {
	s, _ := time.Parse("2006-01-02", "2026-08-01")
	e, _ := time.Parse("2006-01-02", last)
	return s, e
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestReconcileKeepsBothTotalsUnchanged(t *testing.T) {
	exp := &Export{
		Binding: Binding{Day: "date", Cost: "amount", Model: "model"},
		Rows: []Row{
			{Day: "2026-08-01", Model: "sonnet", Cost: 10},
			{Day: "2026-08-02", Model: "sonnet", Cost: 5},
		},
	}
	rows := []report.Row{
		localRow("2026-08-01", "sonnet", 100, 100, cost(9)),
		localRow("2026-08-02", "sonnet", 100, 100, cost(3)),
	}
	start, end := window("2026-08-02")
	res, err := Reconcile(exp, rows, start, end)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if !near(res.ExportCost, 15) {
		t.Fatalf("vendor total was altered: got %v, want 15", res.ExportCost)
	}
	if !near(res.Estimate.Priced, 12) {
		t.Fatalf("estimate was altered: got %v, want 12", res.Estimate.Priced)
	}
	if !near(res.Delta, 3) {
		t.Fatalf("delta = %v, want 3", res.Delta)
	}
	if !near(res.Unexplained, 3) {
		t.Fatalf("with no named cause the whole delta stays unexplained: got %v", res.Unexplained)
	}
}

func TestReconcileRestrictsToTheOverlapBeforeAnyDelta(t *testing.T) {
	// The export reaches a day before the window opens; that money must not enter the delta.
	exp := &Export{
		Binding: Binding{Day: "date", Cost: "amount"},
		Rows: []Row{
			{Day: "2026-07-20", Cost: 100},
			{Day: "2026-08-01", Cost: 10},
		},
	}
	rows := []report.Row{localRow("2026-08-01", "sonnet", 100, 100, cost(10))}
	start, end := window("2026-08-02")
	res, err := Reconcile(exp, rows, start, end)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if !near(res.ExportCost, 10) {
		t.Fatalf("compared over the overlap only: got %v, want 10", res.ExportCost)
	}
	if !near(res.Scope.ExportCostOutside, 100) {
		t.Fatalf("excluded money must be reported, got %v", res.Scope.ExportCostOutside)
	}
	if res.Scope.ExportOnlyDays != 1 {
		t.Fatalf("export-only days = %d, want 1", res.Scope.ExportOnlyDays)
	}
	if !near(res.Delta, 0) || !near(res.Unexplained, 0) {
		t.Fatalf("aligned days agree: delta %v, unexplained %v", res.Delta, res.Unexplained)
	}
}

func TestUnpricedUsageWidensTheBandAndNamesItself(t *testing.T) {
	exp := &Export{
		Binding: Binding{Day: "date", Cost: "amount"},
		Rows:    []Row{{Day: "2026-08-01", Cost: 20}},
	}
	rows := []report.Row{
		localRow("2026-08-01", "sonnet", 500, 500, cost(10)), // 1000 tokens -> $0.01/token
		localRow("2026-08-01", "unknown-model", 500, 500, nil),
	}
	start, end := window("2026-08-01")
	res, err := Reconcile(exp, rows, start, end)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if !near(res.Estimate.Priced, 10) {
		t.Fatalf("priced estimate must exclude unpriced usage: got %v", res.Estimate.Priced)
	}
	if !near(res.Estimate.Extrapolated, 10) {
		t.Fatalf("extrapolation = %v, want 10", res.Estimate.Extrapolated)
	}
	if !near(res.Estimate.Upper(), 20) {
		t.Fatalf("band top = %v, want 20", res.Estimate.Upper())
	}
	if !res.InBand() {
		t.Fatal("the vendor total sits at the band top and must read as inside it")
	}
	if len(res.Causes) != 1 || !near(res.Causes[0].Amount, 10) {
		t.Fatalf("want one named cause of 10, got %+v", res.Causes)
	}
	if !near(res.Unexplained, 0) {
		t.Fatalf("the cause accounts for the whole delta: unexplained %v", res.Unexplained)
	}
}

func TestUnexplainedIsWhateverNoCauseAccountsFor(t *testing.T) {
	exp := &Export{
		Binding: Binding{Day: "date", Cost: "amount", Model: "model"},
		Rows: []Row{
			{Day: "2026-08-01", Model: "sonnet", Cost: 10},
			{Day: "2026-08-01", Model: "opus", Cost: 4}, // never seen locally
		},
	}
	rows := []report.Row{localRow("2026-08-01", "sonnet", 100, 100, cost(7))}
	start, end := window("2026-08-01")
	res, err := Reconcile(exp, rows, start, end)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if !near(res.Delta, 7) {
		t.Fatalf("delta = %v, want 7", res.Delta)
	}
	var named float64
	for _, c := range res.Causes {
		named += c.Amount
	}
	if !near(named, 4) {
		t.Fatalf("named causes = %v, want 4 (the export-only model)", named)
	}
	if !near(res.Unexplained, 3) {
		t.Fatalf("unexplained = %v, want 3 -- the residual is never absorbed", res.Unexplained)
	}
}

func TestDisjointModelNamesProduceALimitNotAnInventedCause(t *testing.T) {
	exp := &Export{
		Binding: Binding{Day: "date", Cost: "amount", Model: "model"},
		Rows:    []Row{{Day: "2026-08-01", Model: "claude-sonnet-4.5", Cost: 10}},
	}
	rows := []report.Row{localRow("2026-08-01", "claude-sonnet-4-5-20250929", 100, 100, cost(9))}
	start, end := window("2026-08-01")
	res, err := Reconcile(exp, rows, start, end)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(res.Causes) != 0 {
		t.Fatalf("no vocabulary in common must yield no cause, got %+v", res.Causes)
	}
	if !near(res.Unexplained, 1) {
		t.Fatalf("the whole delta stays unexplained: got %v", res.Unexplained)
	}
	if !hasSubstring(res.Limits, "no value in common") {
		t.Fatalf("want a limit explaining why per-model causes were skipped, got %v", res.Limits)
	}
}

func TestMissingModelColumnIsStatedAsALimit(t *testing.T) {
	exp := &Export{Binding: Binding{Day: "date", Cost: "amount"}, Rows: []Row{{Day: "2026-08-01", Cost: 10}}}
	rows := []report.Row{localRow("2026-08-01", "sonnet", 100, 100, cost(10))}
	start, end := window("2026-08-01")
	res, err := Reconcile(exp, rows, start, end)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if !hasSubstring(res.Limits, "no model column") {
		t.Fatalf("want the missing-column limit, got %v", res.Limits)
	}
	if !hasSubstring(res.Limits, "no token column") {
		t.Fatalf("want the missing-token limit, got %v", res.Limits)
	}
}

func TestNonUSDExportIsRefusedRatherThanConverted(t *testing.T) {
	exp := &Export{Currency: "EUR", Binding: Binding{Day: "date", Cost: "amount"}, Rows: []Row{{Day: "2026-08-01", Cost: 10}}}
	start, end := window("2026-08-01")
	if _, err := Reconcile(exp, nil, start, end); err == nil {
		t.Fatal("a non-USD export must be refused, not converted at an invented rate")
	}
}

func TestEveryRunCarriesItsRefusals(t *testing.T) {
	exp := &Export{Binding: Binding{Day: "date", Cost: "amount"}, Rows: []Row{{Day: "2026-08-01", Cost: 1}}}
	start, end := window("2026-08-01")
	res, err := Reconcile(exp, nil, start, end)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(res.Refusals) != len(refusals) {
		t.Fatalf("refusals must always be present, got %d", len(res.Refusals))
	}
	if !hasSubstring(res.Refusals, "flat-rate plan") {
		t.Fatalf("the flat-plan refusal must be stated every run, got %v", res.Refusals)
	}
}

func hasSubstring(lines []string, want string) bool {
	for _, l := range lines {
		if strings.Contains(l, want) {
			return true
		}
	}
	return false
}
