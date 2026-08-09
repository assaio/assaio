package calibration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Adjudicated is one trace's independently counted answer. Origin says how the trace was
// captured and Method how the totals were arrived at; Derivations carries one line per
// figure, so a reviewer can check a number against the trace without running anything.
type Adjudicated struct {
	Source string `json:"source"`
	Trace  string `json:"trace"`
	// Capture is "real" for a redacted capture and "constructed" for a sample written in the
	// source's shape. A constructed one proves the reading; only a real one also proves the
	// shape is still what the vendor writes, and conflating the two would report a stronger
	// guarantee than the evidence carries.
	Capture     string            `json:"capture"`
	Origin      string            `json:"origin"`
	Method      string            `json:"method"`
	Calibrates  []string          `json:"calibrates"`
	Totals      Totals            `json:"totals"`
	Derivations map[string]string `json:"derivations"`
}

// LoadAdjudicated reads every adjudicated answer under dir, resolving each Trace to a path.
// It returns them as a list rather than a map by source: a source with no calibrated trace
// has to be a visible absence, which is what the coverage check counts.
func LoadAdjudicated(dir string) ([]Adjudicated, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*", "*.adjudicated.json"))
	if err != nil {
		return nil, err
	}
	out := make([]Adjudicated, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p) //nolint:gosec // a fixed testdata path from this package's own glob
		if err != nil {
			return nil, err
		}
		var a Adjudicated
		if err := json.Unmarshal(b, &a); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		a.Trace = filepath.Join(filepath.Dir(p), a.Trace)
		out = append(out, a)
	}
	return out, nil
}
