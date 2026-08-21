package openmetrics

import (
	"bytes"
	"testing"
)

// FuzzParse holds the boundary: an exposition is text from a network endpoint, so the parser
// must never panic, and must never report a sample it did not read. Seeds are the shapes real
// exporters emit plus the ones that have historically broken hand-written scanners.
func FuzzParse(f *testing.F) {
	f.Add("")
	f.Add("# EOF\n")
	f.Add("# TYPE a gauge\na 1\n")
	f.Add("# HELP a\n# TYPE a counter\na_total{x=\"1\"} 4 1699999999000\n")
	f.Add("a{b=\"\\\"quoted\\\",c\"} NaN\n")
	f.Add("# TYPE h histogram\nh_bucket{le=\"+Inf\"} 3\nh_sum 1\nh_count 3\n")
	f.Add("{} 1\n")
	f.Add("a{ 1\n")
	f.Add("a} 1\n")
	f.Add("a{b=} 1\n")
	f.Add("# UNIT\n# TYPE\n#\n")
	f.Add("a 1 2 3\n")

	f.Fuzz(func(t *testing.T, body string) {
		doc, err := Parse(bytes.NewReader([]byte(body)), Limits{MaxBytes: 1 << 16, MaxLineBytes: 4096})
		if err != nil {
			return
		}
		for i := range doc.Families {
			fam := &doc.Families[i]
			if fam.Name == "" {
				t.Fatalf("family with an empty name from %q", body)
			}
			if !validName(fam.Name) {
				t.Fatalf("family name %q is not a valid metric name, from %q", fam.Name, body)
			}
			for _, s := range fam.Samples {
				if !validName(s.Name) {
					t.Fatalf("sample name %q is not valid, from %q", s.Name, body)
				}
				for k := range s.Labels {
					if !validName(k) {
						t.Fatalf("label name %q is not valid, from %q", k, body)
					}
				}
			}
		}
	})
}
