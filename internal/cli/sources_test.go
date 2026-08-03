package cli

import (
	"testing"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/parser"
)

// doctor scans its own list of sources, which is a second place a new parser has to be wired
// into. Binding it to the matrix means a source can only ever be invisible to doctor by being
// invisible everywhere -- the failure this release exists to make impossible.
func TestScanSourcesCoversExactlyTheMatrix(t *testing.T) {
	scans := scanSources(t.TempDir(), &config.Sources{})
	got := map[string]bool{}
	for i := range scans {
		got[scans[i].tool] = true
	}
	for _, tool := range parser.Tools() {
		if !got[tool] {
			t.Errorf("%s publishes a depth row but doctor never scans for it", tool)
		}
		delete(got, tool)
	}
	for tool := range got {
		t.Errorf("doctor scans for %s, which publishes no depth row", tool)
	}
}
