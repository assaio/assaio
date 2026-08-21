package runtime_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/runtime"
	"github.com/assaio/assaio/internal/runtime/dcgm"
	"github.com/assaio/assaio/internal/runtime/openmetrics"
	"github.com/assaio/assaio/internal/runtime/vllm"
)

var readAt = time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

func snapshotOf(t *testing.T, path, source string, catalog []runtime.Capability) runtime.Snapshot {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // a fixture path this test built itself
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	doc, err := openmetrics.Parse(f, openmetrics.Limits{})
	if err != nil {
		t.Fatalf("Parse %s: %v", path, err)
	}
	return runtime.Inspect(source, "file:"+path, doc, catalog, readAt)
}

func finding(t *testing.T, s *runtime.Snapshot, key string) runtime.Finding {
	t.Helper()
	for i := range s.Findings {
		if s.Findings[i].Key == key {
			return s.Findings[i]
		}
	}
	t.Fatalf("no finding for %q", key)
	return runtime.Finding{}
}

func TestVLLMFixtureReadsItsGauges(t *testing.T) {
	s := snapshotOf(t, "testdata/vllm.prom", vllm.Source, vllm.Catalog())
	if s.Availability != runtime.AvailabilityServing {
		t.Fatalf("Availability = %q, want %q", s.Availability, runtime.AvailabilityServing)
	}
	for _, tc := range []struct {
		key  string
		want float64
		kind string
	}{
		{"running-requests", 6, runtime.KindGauge},
		{"waiting-requests", 14, runtime.KindGauge},
		{"kv-cache-usage", 0.9412, runtime.KindGauge},
	} {
		f := finding(t, &s, tc.key)
		if !f.Present || len(f.Readings) != 1 || f.Readings[0].Value != tc.want || f.Kind != tc.kind {
			t.Errorf("%s = %+v, want one %s reading of %v", tc.key, f, tc.kind, tc.want)
		}
		if f.Note != "" {
			t.Errorf("%s carries a counter note; it is a gauge", tc.key)
		}
	}
}

// TestACounterIsNeverPresentedAsARate is the rule that keeps this feature honest. A cumulative
// total beside a per-second figure a reader assumed is the easiest way for a snapshot to lie.
func TestACounterIsNeverPresentedAsARate(t *testing.T) {
	s := snapshotOf(t, "testdata/vllm.prom", vllm.Source, vllm.Catalog())
	for _, key := range []string{"prompt-tokens", "generation-tokens", "prefix-cache-hits", "preemptions"} {
		f := finding(t, &s, key)
		if !f.Present {
			continue
		}
		if f.Kind != runtime.KindCounter {
			t.Errorf("%s kind = %q, want counter", key, f.Kind)
		}
		if !strings.Contains(f.Note, "not a rate") {
			t.Errorf("%s note = %q, want it to say the value is not a rate", key, f.Note)
		}
	}
}

// TestAHistogramReportsItsCountAndNotAPercentile: percentiles computed from one snapshot's
// buckets describe the whole life of the process, which is not what "latency" means to anyone
// reading it.
func TestAHistogramReportsItsCountAndNotAPercentile(t *testing.T) {
	s := snapshotOf(t, "testdata/vllm.prom", vllm.Source, vllm.Catalog())
	f := finding(t, &s, "time-to-first-token")
	if f.Kind != runtime.KindHistogram || len(f.Readings) != 1 || f.Readings[0].Value != 20842 {
		t.Fatalf("ttft = %+v, want the observation count alone", f)
	}
	if !strings.Contains(f.Note, "percentile") {
		t.Fatalf("ttft note = %q, want the percentile limit stated", f.Note)
	}
}

// TestAnAbsentMetricIsUnavailableNotZero is the difference between a reader going to look for a
// configuration flag and a reader concluding their GPUs are idle.
func TestAnAbsentMetricIsUnavailableNotZero(t *testing.T) {
	s := snapshotOf(t, "testdata/dcgm.prom", dcgm.Source, dcgm.Catalog())
	f := finding(t, &s, "xid-errors")
	if f.Present {
		t.Fatal("the fixture does not publish XID errors, but the snapshot says it is present")
	}
	if len(f.Readings) != 0 {
		t.Fatalf("an absent metric carries readings: %+v", f.Readings)
	}
	if len(s.Missing()) == 0 {
		t.Fatal("Missing() is empty for a fixture that omits two families")
	}
}

// TestPartialEvidenceStaysPartial: a deployment exposing three of twelve families reports three
// present and nine unavailable, not "serving" and nothing else.
func TestPartialEvidenceStaysPartial(t *testing.T) {
	doc, err := openmetrics.Parse(strings.NewReader(
		"# TYPE vllm:num_requests_running gauge\nvllm:num_requests_running 2\n",
	), openmetrics.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	s := runtime.Inspect(vllm.Source, "file:partial", doc, vllm.Catalog(), readAt)
	if s.Availability != runtime.AvailabilityServing {
		t.Fatalf("Availability = %q, want serving: one family is still a reading", s.Availability)
	}
	if len(s.Present()) != 1 || len(s.Missing()) != len(vllm.Catalog())-1 {
		t.Fatalf("present = %d, missing = %d; want 1 and %d", len(s.Present()), len(s.Missing()), len(vllm.Catalog())-1)
	}
}

// TestAnExporterPublishingNothingKnownIsEmptyNotServing separates "assaio read it and it had
// nothing assaio knows" from "assaio could not read it".
func TestAnExporterPublishingNothingKnownIsEmptyNotServing(t *testing.T) {
	doc, err := openmetrics.Parse(strings.NewReader("# TYPE go_goroutines gauge\ngo_goroutines 12\n"), openmetrics.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	s := runtime.Inspect(vllm.Source, "file:other", doc, vllm.Catalog(), readAt)
	if s.Availability != runtime.AvailabilityEmpty {
		t.Fatalf("Availability = %q, want %q", s.Availability, runtime.AvailabilityEmpty)
	}
}

// TestUnreachableEstablishesNothing: a failed read must not be rendered as a deployment that
// exposes nothing, which is a claim about the deployment rather than about the read.
func TestUnreachableEstablishesNothing(t *testing.T) {
	s := runtime.Unreachable(dcgm.Source, "http://127.0.0.1:1/metrics", "connection refused", dcgm.Catalog(), readAt)
	if s.Availability != runtime.AvailabilityUnreachable || s.Error == "" {
		t.Fatalf("snapshot = %+v, want unreachable with a reason", s)
	}
	for _, f := range s.Findings {
		if f.Present {
			t.Fatalf("%s reads as present on a source that could not be read", f.Key)
		}
		if !strings.Contains(f.Note, "not established") {
			t.Fatalf("%s note = %q, want it to say nothing was established", f.Key, f.Note)
		}
	}
}

// TestRenderIsStableAcrossRuns keeps the output diffable: label order is a map's, and a map's
// order is not one.
func TestRenderIsStableAcrossRuns(t *testing.T) {
	s := snapshotOf(t, "testdata/dcgm.prom", dcgm.Source, dcgm.Catalog())
	var first, second strings.Builder
	if err := runtime.RenderText(&first, &s); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		second.Reset()
		if err := runtime.RenderText(&second, &s); err != nil {
			t.Fatal(err)
		}
		if first.String() != second.String() {
			t.Fatal("two renders of the same snapshot differ")
		}
	}
}

func TestFetchReadsALocalEndpoint(t *testing.T) {
	body := "# TYPE vllm:num_requests_running gauge\nvllm:num_requests_running 4\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	got, err := runtime.Fetch(context.Background(), srv.URL, runtime.DefaultFetchLimits())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(got) != body {
		t.Fatalf("Fetch returned %q", got)
	}
}

func TestFetchRefusesAnOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 4096)))
	}))
	defer srv.Close()

	_, err := runtime.Fetch(context.Background(), srv.URL, runtime.FetchLimits{Timeout: time.Second, MaxBytes: 512})
	if err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("Fetch error = %v, want the size limit refused it", err)
	}
}

func TestFetchTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		_ = w
	}))
	defer srv.Close()

	start := time.Now()
	_, err := runtime.Fetch(context.Background(), srv.URL, runtime.FetchLimits{Timeout: 150 * time.Millisecond, MaxBytes: 1 << 20})
	if err == nil {
		t.Fatal("a hanging endpoint returned no error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("the timeout did not bound the read: %s", elapsed)
	}
}

func TestFetchReportsAnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := runtime.Fetch(context.Background(), srv.URL, runtime.DefaultFetchLimits())
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("Fetch error = %v, want the status reported", err)
	}
}

// TestFetchBoundsRedirects: a metrics endpoint that redirects repeatedly is not the one the
// operator meant, and following it is how a read leaves the network they intended.
func TestFetchBoundsRedirects(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/again", http.StatusFound)
	}))
	defer srv.Close()

	_, err := runtime.Fetch(context.Background(), srv.URL, runtime.FetchLimits{Timeout: time.Second, MaxBytes: 1 << 20, MaxRedirects: 1})
	if err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("Fetch error = %v, want the redirect chain refused", err)
	}
}

// TestFetchForbidsRedirectsAtZero holds the strictest setting.
func TestFetchForbidsRedirectsAtZero(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("a 1\n"))
	}))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer srv.Close()

	_, err := runtime.Fetch(context.Background(), srv.URL, runtime.FetchLimits{Timeout: time.Second, MaxBytes: 1 << 20, MaxRedirects: 0})
	if err == nil {
		t.Fatal("a redirect was followed with MaxRedirects=0")
	}
}
