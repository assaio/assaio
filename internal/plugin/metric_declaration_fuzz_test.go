package plugin

import (
	"slices"
	"testing"

	"github.com/assaio/assaio/internal/analyze"
)

// FuzzMetricDeclaration asserts parseDeclaration's invariants over arbitrary bytes: it never
// panics, and any declaration it accepts can be honoured -- every capability is one this build
// knows, and every projection and predicate addresses a section that capability actually
// carries. An accepted declaration that could not be honoured would either drop a section the
// plugin asked for or panic building the envelope, and both reach the plugin as an absence.
func FuzzMetricDeclaration(f *testing.F) {
	for _, doc := range catalogueSeeds(f, "metric-declaration") {
		f.Add([]byte(doc))
	}
	f.Add([]byte(``))
	f.Add([]byte(`{"needs":["usage"],"fields":{"usage":["day","day"]}}`))
	f.Add([]byte("{\"needs\":[\"usage\"],\"where\":{\"usage.tool\":[\"\x1b\"]}}"))
	f.Add([]byte("\xff\xfe"))

	f.Fuzz(func(t *testing.T, doc []byte) {
		declared, violations, err := parseDeclaration(doc)
		if err != nil {
			return
		}
		if len(violations) != 0 {
			t.Fatalf("nil error with %d violations", len(violations))
		}
		if len(declared.Needs) == 0 {
			t.Fatal("accepted a declaration that reads nothing")
		}
		for _, c := range declared.Needs {
			if !analyze.ValidCapability(c) {
				t.Fatalf("accepted unknown capability %q", c)
			}
		}
		for key, keep := range declared.Fields {
			s, known := sections[key]
			if !known || !slices.Contains(declared.Needs, s.capability) {
				t.Fatalf("accepted a projection of %q, which this declaration cannot carry", key)
			}
			for _, column := range keep {
				if _, ok := columns(s.row)[column]; !ok {
					t.Fatalf("accepted a projection of column %q, which %q has no such field", column, key)
				}
			}
		}
		for key := range declared.Where {
			name, column, ok := splitPredicate(key)
			if !ok {
				t.Fatalf("accepted an unaddressable predicate %q", key)
			}
			s, known := sections[name]
			if !known || !slices.Contains(declared.Needs, s.capability) {
				t.Fatalf("accepted a predicate on %q, which this declaration cannot carry", key)
			}
			if _, ok := columns(s.row)[column]; !ok {
				t.Fatalf("accepted a predicate on column %q, which %q has no such field", column, name)
			}
		}
		// The envelope builder is the thing an accepted declaration is a promise about, so the
		// fuzzer runs it: a projection nothing rejected must produce a document rather than a
		// panic or an encoder error.
		in := tracedInput()
		if _, err := envelopeOf(&in, negotiate(declared, nil)); err != nil {
			t.Fatalf("accepted declaration produced an unencodable envelope: %v", err)
		}
	})
}
