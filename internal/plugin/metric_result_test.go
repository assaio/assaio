package plugin

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
)

func validMetricResult() analyze.Result {
	return analyze.Result{
		Title:     "Demo Metric",
		Read:      analyze.Read{Key: "watch", Label: "WATCH"},
		Purity:    0.4,
		HowToRead: "Directional demo signal.",
		Figures:   []analyze.Figure{{Label: "x", Value: "1"}},
		Takeaway:  "Demo takeaway.",
	}
}

func mustDoc(t *testing.T, r *analyze.Result) []byte {
	t.Helper()
	doc, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// A plugin's verdict can describe part of the window just as a built-in's can -- a metric
// over one source's records in a five-source window is the ordinary case. Declaring it has
// to survive the boundary, or the plugin is judged as resting on data it never read.
func TestParseMetricResultCarriesDeclaredSignalCoverage(t *testing.T) {
	r := validMetricResult()
	r.Confidence = analyze.Confidence{Samples: 12, Unit: "sessions"}
	r.Confidence.Covering(0.05)

	got, violations, err := parseMetricResult(mustDoc(t, &r), "demo")
	if err != nil {
		t.Fatalf("parseMetricResult() err = %v (violations %v)", err, violations)
	}
	in := analyze.BuildInput(nil, nil, nil, time.Time{}, 0, analyze.Delegation{})
	analyze.Stamp(&got, &in)
	if got.Confidence.Label != analyze.ConfidenceLow {
		t.Fatalf("Label = %q, want %q: a plugin covering 5%% of the window must not read as covering all of it",
			got.Confidence.Label, analyze.ConfidenceLow)
	}
}

// A plugin that declares no signal coverage keeps describing the whole window, which is what
// every plugin released before the axis existed meant.
func TestParseMetricResultWithoutSignalCoverageStillCoversTheWindow(t *testing.T) {
	r := validMetricResult()
	r.Confidence = analyze.Confidence{Samples: 12, Unit: "sessions"}

	got, _, err := parseMetricResult(mustDoc(t, &r), "demo")
	if err != nil {
		t.Fatal(err)
	}
	in := analyze.BuildInput([]store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 100},
	}, nil, nil, time.Time{}, 0, analyze.Delegation{})
	analyze.Stamp(&got, &in)
	if got.Confidence.Label == analyze.ConfidenceInsufficient {
		t.Fatalf("Label = %q: an undeclared axis must not be read as covering nothing", got.Confidence.Label)
	}
}

func TestParseMetricResultValid(t *testing.T) {
	r := validMetricResult()
	r.Name = "adoption"
	r.Purity = 1.7
	r.Bars = []analyze.Bar{{Label: "web", Value: "40", Frac: 2.5}}
	r.BarsPseudonym = analyze.PseudonymProject

	got, violations, err := parseMetricResult(mustDoc(t, &r), "demo")
	if err != nil {
		t.Fatalf("parseMetricResult() err = %v (violations %v)", err, violations)
	}
	if got.Name != "plugin:demo" {
		t.Fatalf("Name = %q, want stamped plugin:demo (wire name must be ignored)", got.Name)
	}
	if got.Purity != 1.0 {
		t.Fatalf("Purity = %v, want clamped 1.0", got.Purity)
	}
	if got.Bars[0].Frac != 1.0 {
		t.Fatalf("Bars[0].Frac = %v, want clamped 1.0", got.Bars[0].Frac)
	}
	if got.BarsPseudonym != analyze.PseudonymProject {
		t.Fatal("BarsPseudonym must carry through: the dashboard's anonymization depends on it")
	}
	if got.Title != "Demo Metric" || got.Read.Label != "WATCH" || got.Takeaway != "Demo takeaway." {
		t.Fatalf("carried fields wrong: %+v", got)
	}
}

func TestParseMetricResultViolations(t *testing.T) {
	overlong := strings.Repeat("x", 121)
	cases := []struct {
		name    string
		mutate  func(*analyze.Result)
		wantSub string
	}{
		{"invalid read key", func(r *analyze.Result) { r.Read.Key = "great" }, "read.key"},
		{"empty read label", func(r *analyze.Result) { r.Read.Label = "" }, "read.label is required"},
		{"overlong read label", func(r *analyze.Result) { r.Read.Label = "VERY LONG LABEL XX" }, "read.label exceeds"},
		{"empty title", func(r *analyze.Result) { r.Title = "" }, "title is required"},
		{"empty howToRead", func(r *analyze.Result) { r.HowToRead = "" }, "howToRead is required"},
		{"empty takeaway", func(r *analyze.Result) { r.Takeaway = "" }, "takeaway is required"},
		{"control char in figure label", func(r *analyze.Result) {
			r.Figures = []analyze.Figure{{Label: "bad\x1b[31m", Value: "1"}}
		}, "control character"},
		{"empty figure label", func(r *analyze.Result) {
			r.Figures = []analyze.Figure{{Label: "", Value: "1"}}
		}, "figures[0].label is required"},
		{"too many figures", func(r *analyze.Result) {
			r.Figures = make([]analyze.Figure, 13)
			for i := range r.Figures {
				r.Figures[i] = analyze.Figure{Label: "l", Value: "v"}
			}
		}, "figures exceeds"},
		{"too many bars", func(r *analyze.Result) {
			r.Bars = make([]analyze.Bar, 31)
			for i := range r.Bars {
				r.Bars[i] = analyze.Bar{Label: "l", Value: "v"}
			}
		}, "bars exceeds"},
		{"too many caveats", func(r *analyze.Result) {
			r.Caveats = make([]string, 9)
			for i := range r.Caveats {
				r.Caveats[i] = "c"
			}
		}, "caveats exceeds"},
		{"overlong bar label", func(r *analyze.Result) {
			r.Bars = []analyze.Bar{{Label: overlong, Value: "1"}}
		}, "bars[0].label exceeds"},
		{"overlong caveat", func(r *analyze.Result) {
			r.Caveats = []string{strings.Repeat("y", 401)}
		}, "caveats[0] exceeds"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := validMetricResult()
			tc.mutate(&r)
			_, violations, err := parseMetricResult(mustDoc(t, &r), "demo")
			if err == nil {
				t.Fatal("err = nil, want a contract violation error")
			}
			if len(violations) == 0 {
				t.Fatal("violations empty, want at least one reason")
			}
			joined := strings.Join(violations, "; ")
			if !strings.Contains(joined, tc.wantSub) {
				t.Fatalf("violations = %q, want substring %q", joined, tc.wantSub)
			}
		})
	}
}

func TestParseMetricResultMalformedDocuments(t *testing.T) {
	cases := []struct {
		name    string
		doc     string
		wantSub string
	}{
		{"not json", "hello", "decoding result"},
		{"two documents", `{"title":"T","read":{"key":"good","label":"OK"},"howToRead":"H","takeaway":"K"}` + "\n" + `{"title":"U"}`, "trailing data"},
		{"trailing garbage", `{"title":"T","read":{"key":"good","label":"OK"},"howToRead":"H","takeaway":"K"} }`, "trailing data"},
		{"empty document", "", "decoding result"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, violations, err := parseMetricResult([]byte(tc.doc), "demo")
			if err == nil {
				t.Fatal("err = nil, want a document error")
			}
			joined := strings.Join(violations, "; ") + "; " + err.Error()
			if !strings.Contains(joined, tc.wantSub) {
				t.Fatalf("got %q, want substring %q", joined, tc.wantSub)
			}
		})
	}
}

// TestParseMetricResultLegacyBarsKeyStillPseudonymizes is the regression guard for the
// BarsAreProjects -> BarsPseudonym rename: a plugin released against the old key kept
// emitting it, the decoder dropped it, and its project-named bars then escaped
// --anonymize entirely -- publishing the real repository names the flag exists to hide.
func TestParseMetricResultLegacyBarsKeyStillPseudonymizes(t *testing.T) {
	r := validMetricResult()
	r.Bars = []analyze.Bar{{Label: "acme-secret-repo", Value: "40", Frac: 1}}
	doc := mustDoc(t, &r)
	legacy := strings.Replace(string(doc), `"bars":`, `"barsAreProjects":true,"bars":`, 1)

	got, violations, err := parseMetricResult([]byte(legacy), "demo")
	if err != nil {
		t.Fatalf("legacy key must stay accepted: %v (violations %v)", err, violations)
	}
	if got.BarsPseudonym != analyze.PseudonymProject {
		t.Fatalf("BarsPseudonym = %q, want the legacy key mapped to %q so the bars are pseudonymized",
			got.BarsPseudonym, analyze.PseudonymProject)
	}
}

// TestParseMetricResultRejectsUnknownField asserts a misspelled key is a loud violation
// rather than a silently ignored setting -- the failure mode that let the rename leak.
func TestParseMetricResultRejectsUnknownField(t *testing.T) {
	r := validMetricResult()
	doc := strings.Replace(string(mustDoc(t, &r)), `"name":`, `"barsPseudonim":"project","name":`, 1)

	if _, _, err := parseMetricResult([]byte(doc), "demo"); err == nil {
		t.Fatal("an unknown field must be rejected, not ignored")
	}
}
