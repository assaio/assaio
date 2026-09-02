package plugin

import (
	"os"
	"testing"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/config"
)

// TestMain gives every test in this package a hermetic data directory: member
// pseudonymization persists a per-install secret (internal/pseudonym), and without this a
// `go test` run would read and write the real user's data directory.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "assaio-plugin-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// everything is the projection a test uses when its subject is not the projection: every
// capability granted, no column or row narrowed. It is here rather than in one test file
// because most of them need it, and the alternative -- each spelling the grant out -- is how
// they drift apart.
func everything() Projection { return Projection{Needs: analyze.Capabilities()} }

// granting is the projection of a plugin that declared exactly these capabilities and nothing
// was denied it.
func granting(caps ...analyze.Capability) Projection { return Projection{Needs: caps} }

// documentOf is the document a plugin with this projection actually reads, before encoding.
func documentOf(in *analyze.Input, p Projection) map[string]any {
	envelope := buildMetricInput(in, p)
	return envelope.document()
}

// envelopeOf is what crosses the pipe under this projection: the bytes every projection claim
// is measured on, and the encoder run an accepted declaration promises will succeed.
func envelopeOf(in *analyze.Input, p Projection) ([]byte, error) {
	envelope := buildMetricInput(in, p)
	return envelope.marshal()
}

func envelopeBytes(t *testing.T, in *analyze.Input, p Projection) []byte {
	t.Helper()
	out, err := envelopeOf(in, p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}

// pluginNeeding builds a config entry declaring one need, for the boundary tests. The command
// is this test binary itself: it exists on every platform the suite runs on, which a hardcoded
// /bin/echo does not -- that spelling passed on macOS and Linux and failed the Windows job.
func pluginNeeding(need string) config.PluginConfig {
	self, err := os.Executable()
	if err != nil {
		self = os.Args[0]
	}
	return config.PluginConfig{Name: "p", Command: self, Needs: []string{need}}
}
