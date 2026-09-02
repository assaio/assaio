package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The published conformance catalogue (docs/conformance) is what a plugin author's CI runs
// without this binary. It is only worth publishing while it still describes this boundary, so
// the same vectors drive assaio's own tests: a vector nobody runs is a vector that lies.

const catalogueDir = "../../docs/conformance"

type catalogue struct {
	Contract string `json:"contract"`
	Protocol int    `json:"protocol"`
	// Now is the clock the parser-record vectors' timestamps are judged against. Without it
	// every timestamp vector would drift out of range as the wall clock moved past it.
	Now     time.Time `json:"now"`
	Vectors []vector  `json:"vectors"`
}

type vector struct {
	ID     string `json:"id"`
	Doc    string `json:"doc"`
	Accept bool   `json:"accept"`
	// Expect is a substring the rejection reason must contain, so a vector pins the reason a
	// document is refused rather than only the fact of it.
	Expect string `json:"expect"`
	Why    string `json:"why"`
}

func TestConformanceVectors(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(catalogueDir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no conformance catalogue at %s: %v", catalogueDir, err)
	}
	for _, path := range files {
		c := readCatalogue(t, path)
		t.Run(c.Contract, func(t *testing.T) {
			for _, v := range c.Vectors {
				t.Run(v.ID, func(t *testing.T) { checkVector(t, c, v) })
			}
		})
	}
}

func checkVector(t *testing.T, c catalogue, v vector) {
	t.Helper()
	reasons, err := verdictFor(c, v.Doc)
	if v.Accept {
		if err != nil {
			t.Fatalf("rejected a vector the catalogue says is valid: %v (%s)", err, strings.Join(reasons, "; "))
		}
		return
	}
	if err == nil {
		t.Fatalf("accepted a vector the catalogue says is invalid: %s", v.Why)
	}
	if got := err.Error() + " " + strings.Join(reasons, "; "); !strings.Contains(got, v.Expect) {
		t.Fatalf("rejected for %q, want a reason mentioning %q", got, v.Expect)
	}
}

// verdictFor runs one document through the same boundary the runtime uses. Nothing here is a
// reimplementation of a check: a catalogue validated against a second copy of the rules would
// only prove the copies agree.
func verdictFor(c catalogue, doc string) ([]string, error) {
	switch c.Contract {
	case "metric-declaration":
		_, reasons, err := parseDeclaration([]byte(doc))
		return reasons, err
	case "metric-result":
		_, reasons, err := parseMetricResult([]byte(doc), "demo")
		return reasons, err
	case "rule-alerts":
		_, reasons, err := parseRuleAlerts([]byte(doc), "demo")
		return reasons, err
	case "parser-record":
		_, err := parseRecordLineAt([]byte(doc), "demo", c.Now)
		return nil, err
	}
	return nil, fmt.Errorf("no boundary claims contract %q", c.Contract)
}

// The versions a catalogue publishes are what a plugin author builds against, so a bump that
// left them behind would ship a catalogue teaching a handshake the runtime rejects.
func TestCatalogueProtocolVersionsMatchTheRuntime(t *testing.T) {
	want := map[string]int{
		"metric-declaration": metricInputVersion,
		"metric-result":      metricInputVersion,
		"rule-alerts":        ruleInputVersion,
		"parser-record":      protocolVersion,
	}
	files, _ := filepath.Glob(filepath.Join(catalogueDir, "*.json"))
	for _, path := range files {
		c := readCatalogue(t, path)
		if got, known := want[c.Contract]; !known {
			t.Errorf("%s declares contract %q, which no runtime boundary claims", path, c.Contract)
		} else if c.Protocol != got {
			t.Errorf("%s publishes protocol %d, runtime speaks %d", path, c.Protocol, got)
		}
	}
	if len(files) != len(want) {
		t.Errorf("the catalogue has %d files for %d contracts; every boundary a plugin author "+
			"can fail needs vectors", len(files), len(want))
	}
}

// A catalogue of nothing but valid documents teaches a decoder, not a boundary: the refusals
// are the half a plugin author cannot infer from the happy path.
func TestEveryCatalogueCarriesBothVerdicts(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join(catalogueDir, "*.json"))
	for _, path := range files {
		c := readCatalogue(t, path)
		var accepted, rejected int
		ids := map[string]bool{}
		for _, v := range c.Vectors {
			if v.Accept {
				accepted++
			} else {
				rejected++
			}
			if v.Why == "" {
				t.Errorf("%s: vector %q states no reason it exists", path, v.ID)
			}
			if !v.Accept && v.Expect == "" {
				t.Errorf("%s: vector %q is rejected but pins no reason", path, v.ID)
			}
			if ids[v.ID] {
				t.Errorf("%s: two vectors share the id %q", path, v.ID)
			}
			ids[v.ID] = true
		}
		if accepted == 0 || rejected == 0 {
			t.Errorf("%s carries %d accepted and %d rejected vectors; a catalogue needs both",
				path, accepted, rejected)
		}
	}
}

func readCatalogue(tb testing.TB, path string) catalogue {
	tb.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // a repository path this test walks
	if err != nil {
		tb.Fatal(err)
	}
	var c catalogue
	if err := json.Unmarshal(raw, &c); err != nil {
		tb.Fatalf("%s: %v", path, err)
	}
	return c
}

// catalogueSeeds hands one contract's published vectors to a fuzzer. The catalogue is the
// densest corpus of near-miss documents this project has -- every one of them is a shape a real
// plugin got wrong -- so seeding from it costs nothing and starts the mutator at the boundary
// rather than at the empty string.
func catalogueSeeds(tb testing.TB, contract string) []string {
	tb.Helper()
	c := readCatalogue(tb, filepath.Join(catalogueDir, contract+".json"))
	docs := make([]string, 0, len(c.Vectors))
	for _, v := range c.Vectors {
		docs = append(docs, v.Doc)
	}
	return docs
}
