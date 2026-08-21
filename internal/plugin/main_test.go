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

// tracingPlugin is the config a test uses when it asserts something about the trace section.
// It is here rather than in one test file because several assert on it, and the alternative --
// each spelling out the declaration -- is how they drift apart.
func tracingPlugin() Config {
	return Config{Name: "test", Needs: []analyze.Capability{analyze.CapTrace}}
}

// pluginNeeding builds a config entry declaring one need, for the boundary tests.
func pluginNeeding(need string) config.PluginConfig {
	return config.PluginConfig{Name: "p", Command: "/bin/echo", Needs: []string{need}}
}
