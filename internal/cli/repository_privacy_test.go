package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// TestNoSurfacePrintsARepositoryKey: the key is a keyed hash of a directory path, and every
// machine-readable output, shared file and table stays free of it -- including when a name is
// split and the split is what the output shows.
func TestNoSurfacePrintsARepositoryKey(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	keys := []string{"v1:0123456789abcdef0123456789abcdef", "v1:fedcba9876543210fedcba9876543210"}
	at := time.Now().UTC().Add(-time.Hour)
	var records []usage.Record
	for i, key := range keys {
		records = append(records, usage.Record{
			Tool: "claude-code", SessionID: key[3:11], DedupeKey: key[3:11], Timestamp: at,
			Model: "claude-opus-4-5", InputTokens: 100, OutputTokens: 20, LinesAdded: int64(10 * (i + 1)),
			Project: "api", RepoKey: key, Granularity: "turn",
		})
	}
	seedStoreAt(t, evidenceDBPath(t), records)
	html := filepath.Join(t.TempDir(), "assay.html")

	for _, args := range [][]string{
		{"report", "--by", "project", "--format", "json"},
		{"report", "--by", "project", "--format", "csv"},
		{"report", "--by", "project"},
		{"effectiveness", "--by", "project", "--format", "json"},
		{"analyze", "--format", "json"},
		{"dashboard", "--no-anonymize", "--output", html},
	} {
		out, err := runCLI(t, args...)
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		if args[0] == "dashboard" {
			data, err := os.ReadFile(html) //nolint:gosec // test output under t.TempDir
			if err != nil {
				t.Fatal(err)
			}
			out = string(data)
		}
		if args[0] == "report" && !strings.Contains(out, "api (2)") {
			t.Fatalf("%v does not show the split it is meant to: %s", args, out)
		}
		for _, key := range keys {
			if strings.Contains(out, key) || strings.Contains(out, key[3:]) {
				t.Fatalf("%v printed a repository key", args)
			}
		}
	}
}
