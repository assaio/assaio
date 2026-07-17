package plugin

import (
	"strings"
	"testing"
	"unicode"
)

// FuzzMetricResult asserts parseMetricResult's invariants over arbitrary bytes: it never
// panics, and any accepted Result is fully sanitized -- stamped name, whitelisted read
// key, clamped purity/frac, capped counts, control-char-free strings.
func FuzzMetricResult(f *testing.F) {
	f.Add([]byte(`{"title":"T","read":{"key":"watch","label":"WATCH"},"purity":0.4,"howToRead":"H","takeaway":"K"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))
	f.Add([]byte(`{"title":"T","read":{"key":"good","label":"OK"},"howToRead":"H","takeaway":"K","purity":9e99,"bars":[{"label":"a","value":"1","frac":-3}]}`))
	f.Add([]byte("{\"title\":\"T\x1b[31m\""))
	f.Add([]byte("\xff\xfe"))
	f.Add([]byte(`{"name":"adoption","title":"T","read":{"key":"good","label":"OK"},"howToRead":"H","takeaway":"K"}`))

	f.Fuzz(func(t *testing.T, doc []byte) {
		r, violations, err := parseMetricResult(doc, "demo")
		if err != nil {
			return
		}
		if len(violations) != 0 {
			t.Fatalf("nil error with %d violations", len(violations))
		}
		if r.Name != "plugin:demo" {
			t.Fatalf("accepted Result has Name %q, want stamped plugin:demo", r.Name)
		}
		if r.Read.Key != "good" && r.Read.Key != "watch" && r.Read.Key != "neutral" {
			t.Fatalf("accepted Result has read key %q", r.Read.Key)
		}
		if r.Purity < 0 || r.Purity > 1 {
			t.Fatalf("accepted Result has unclamped purity %v", r.Purity)
		}
		if len(r.Figures) > maxMetricFigures || len(r.Bars) > maxMetricBars || len(r.Caveats) > maxMetricCaveats {
			t.Fatalf("accepted Result exceeds caps: %d figures, %d bars, %d caveats", len(r.Figures), len(r.Bars), len(r.Caveats))
		}
		for _, b := range r.Bars {
			if b.Frac < 0 || b.Frac > 1 {
				t.Fatalf("accepted Result has unclamped bar frac %v", b.Frac)
			}
		}
		for _, s := range []string{r.Title, r.Describe, r.HowToRead, r.Takeaway, r.Read.Label} {
			if strings.ContainsFunc(s, unicode.IsControl) {
				t.Fatalf("accepted Result carries a control character: %q", s)
			}
		}
	})
}
