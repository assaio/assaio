package calibration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/calibration"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/server"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// TestSurfacesAgreeOnOneWindow: report, status, the dashboard and every metric plugin read
// one window and may not print different totals for it. The roadmap has promised this since
// v0.5 and nothing enforced it -- each surface sums the rows itself, so a grouping or
// pricing step that drops a class is invisible until somebody compares two screens.
func TestSurfacesAgreeOnOneWindow(t *testing.T) {
	recs := everyTraceRecords(t)
	st := storeWith(t, recs)
	rows, err := st.Usage(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}

	base := calibration.FromRows(rows)
	if base.Tokens() == 0 {
		t.Fatal("the traces produced no tokens, so this test would pass on nothing")
	}
	prices := pricing.Table{}
	in := analyze.BuildInput(rows, nil, prices, time.Now(), 7*24*time.Hour, analyze.Delegation{})

	eff, err := report.BuildEffectiveness(rows, prices, "tool")
	if err != nil {
		t.Fatal(err)
	}
	surfaces := []struct {
		name    string
		window  calibration.Window
		figures []string
	}{
		{"report", calibration.FromReport(report.Build(rows, prices)), calibration.TokenFigures},
		{"effectiveness", calibration.FromEffectiveness(eff), calibration.LineFigures},
		{
			"analyze / dashboard / plugin", calibration.FromAnalyze(&in),
			append(append([]string{}, calibration.TokenFigures...), calibration.FigLinesAdded),
		},
	}
	for _, s := range surfaces {
		if d := calibration.Disagreements(&base, &s.window, s.figures...); len(d) > 0 {
			t.Errorf("%s disagrees with the store on %v\n    store:   %+v\n    surface: %+v",
				s.name, d, base, s.window)
		}
	}
}

// TestSyncPreservesTheWindow: a team member's push must land on the server as the same
// window it left as. sync is the one surface whose arithmetic happens on another machine,
// so a class dropped at the boundary shows up only here.
func TestSyncPreservesTheWindow(t *testing.T) {
	recs := everyTraceRecords(t)
	local := calibration.FromRows(rowsOf(t, storeWith(t, recs)))

	central := storeWith(t, nil)
	srv := server.New(central, "tok", nil)
	body, err := json.Marshal(map[string]any{"member": "m1", "records": recs})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/usage", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("push returned %d: %s", rec.Code, rec.Body.String())
	}

	synced := calibration.FromRows(rowsOf(t, central))
	if d := calibration.Disagreements(&local, &synced, allFigures...); len(d) > 0 {
		t.Errorf("sync changed the window on %v\n    local:  %+v\n    synced: %+v", d, local, synced)
	}
}

// TestRestatingAWindowDoesNotDoubleIt: every source's log is append-only and every import
// re-reads it, so the same records arriving twice must restate the window rather than add to
// it. This is the property the store's dedupe key exists for, asserted end to end.
func TestRestatingAWindowDoesNotDoubleIt(t *testing.T) {
	recs := everyTraceRecords(t)
	st := storeWith(t, recs)
	once := calibration.FromRows(rowsOf(t, st))
	if _, err := st.InsertLocal(context.Background(), recs); err != nil {
		t.Fatal(err)
	}
	twice := calibration.FromRows(rowsOf(t, st))
	if d := calibration.Disagreements(&once, &twice, allFigures...); len(d) > 0 {
		t.Errorf("a second import moved %v\n    once:  %+v\n    twice: %+v", d, once, twice)
	}
}

// allFigures is every figure a stored window carries, for the surfaces that carry all of
// them: a sync round-trip and a re-import must move nothing at all.
var allFigures = append(append([]string{}, calibration.TokenFigures...), calibration.LineFigures...)

func everyTraceRecords(t *testing.T) []usage.Record {
	t.Helper()
	answers, err := calibration.LoadAdjudicated("testdata")
	if err != nil {
		t.Fatal(err)
	}
	var out []usage.Record
	for i := range answers {
		recs, _, _ := parseTrace(t, answers[i].Source, answers[i].Trace)
		out = append(out, recs...)
	}
	if len(out) == 0 {
		t.Fatal("no records parsed from the traces")
	}
	return out
}

func storeWith(t *testing.T, recs []usage.Record) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "assaio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if len(recs) > 0 {
		if _, err := st.InsertLocal(context.Background(), recs); err != nil {
			t.Fatal(err)
		}
	}
	return st
}

func rowsOf(t *testing.T, st *store.Store) []store.UsageRow {
	t.Helper()
	rows, err := st.Usage(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
