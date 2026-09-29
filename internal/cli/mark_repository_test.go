package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// TestMarkTargetsThisCheckoutNotAnotherWithItsName: mark with no session id acts on the newest
// session in the repository holding the working directory -- not on a newer one from another
// checkout that shares the directory name, and not on anything when only that other checkout
// is stored.
func TestMarkTargetsThisCheckoutNotAnotherWithItsName(t *testing.T) {
	for _, tt := range []struct {
		name      string
		seedHere  bool
		want      string
		wantError string
	}{
		{"both checkouts stored", true, "here-ses · api (2)", ""},
		{"only the other checkout stored", false, "", "assaio-agent repos"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			here := filepath.Join(t.TempDir(), "api")
			other := filepath.Join(t.TempDir(), "api")
			for _, dir := range []string{here, other} {
				if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o750); err != nil {
					t.Fatal(err)
				}
			}
			dbPath := evidenceDBPath(t)
			rec := func(key, id string, ago time.Duration) usage.Record {
				return usage.Record{
					Tool: "claude-code", SessionID: id, DedupeKey: id, Model: "m",
					Timestamp: time.Now().UTC().Add(-ago), Project: "api", RepoKey: key, Granularity: "turn",
				}
			}
			records := []usage.Record{rec(repoKeyIn(t, dbPath, other), "other-session", time.Minute)}
			if tt.seedHere {
				records = append(records, rec(repoKeyIn(t, dbPath, here), "here-session", time.Hour))
			}
			seedStoreAt(t, dbPath, records)
			t.Chdir(here)

			out, err := runCLI(t, "mark", "--task", "bugfix")
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("mark = %q, %v; want an error naming %q", out, err, tt.wantError)
				}
				return
			}
			if err != nil || !strings.Contains(out, tt.want) {
				t.Fatalf("mark = %q, %v; want it to act on %s", out, err, tt.want)
			}
		})
	}
}
