package cli

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func growthStore(t *testing.T, oldest time.Time) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "g.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if _, err := st.Insert(context.Background(), []usage.Record{{
		Tool: "claude-code", SessionID: "s1", Timestamp: oldest, Model: "m",
		DedupeKey: "k1", Granularity: "turn", InputTokens: 10,
	}}); err != nil {
		t.Fatal(err)
	}
	return st
}

// TestGrowthLineProjectsFromTheStoreItRead is the operator figure B173 and B174 both wanted: a
// central store inherits every member's growth at once, and until now nobody could see it.
func TestGrowthLineProjectsFromTheStoreItRead(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	line := growthLine(context.Background(), growthStore(t, now.AddDate(0, 0, -60)), now)
	for _, want := range []string{"live over", "/day", "At most", "/year", "upper bound, not an estimate"} {
		if !strings.Contains(line, want) {
			t.Fatalf("growth line = %q, want it to contain %q", line, want)
		}
	}
}

// TestGrowthLineCountsLiveBytesNotTheFile is the defect that made deleting history raise the
// projected year: free pages are the normal state after every step prune, and a rate over the
// file size counts space the next insert reuses.
func TestGrowthLineCountsLiveBytesNotTheFile(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	st := growthStore(t, now.AddDate(0, 0, -60))
	size, err := st.Size(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	line := growthLine(context.Background(), st, now)
	if !strings.Contains(line, humanize.Bytes(size.Bytes-size.Reclaimable)+" live") {
		t.Fatalf("growth line = %q, want it to report %s of live bytes", line, humanize.Bytes(size.Bytes-size.Reclaimable))
	}
}

// TestGrowthLineRefusesToProjectFromTooShortASpan: a store two days old projects a year from
// two days, and the answer says more about the two days than about the year.
func TestGrowthLineRefusesToProjectFromTooShortASpan(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	line := growthLine(context.Background(), growthStore(t, now.AddDate(0, 0, -2)), now)
	if strings.Contains(line, "At most") {
		t.Fatalf("growth line = %q, want no yearly bound from a two-day store", line)
	}
	if !strings.Contains(line, "too few to project") {
		t.Fatalf("growth line = %q, want it to say why there is no projection", line)
	}
}

// TestDoctorReadsTheStoreItWasPointedAt is B174: an operator could diagnose a local store and
// no other, which is the one they cannot walk over to.
func TestDoctorReadsTheStoreItWasPointedAt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "team.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := runCLI(t, "doctor", "--db", path)
	if err != nil {
		t.Fatalf("doctor --db: %v\n%s", err, out)
	}
	if !strings.Contains(out, path) {
		t.Fatalf("doctor read some other store:\n%s", out)
	}
}

// TestGrowthLineIgnoresReclaimablePages is the defect that made deleting history raise the
// projected year. The earlier test could not see it: a fresh store has no free pages, so
// `Bytes` and `Bytes - Reclaimable` were the same number. This one deletes without vacuuming,
// which is exactly the state every step prune leaves behind.
func TestGrowthLineIgnoresReclaimablePages(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	st := growthStore(t, now.AddDate(0, 0, -60))

	recs := make([]usage.Record, 0, 400)
	for i := range 400 {
		recs = append(recs, usage.Record{
			Tool: "claude-code", SessionID: "s", Timestamp: now.AddDate(0, 0, -90), Model: "m",
			DedupeKey: "bulk" + strconv.Itoa(i), Granularity: "turn", InputTokens: 1000,
			Project: strings.Repeat("p", 200),
		})
	}
	if _, err := st.Insert(ctx, recs); err != nil {
		t.Fatal(err)
	}
	// Clear is the real deletion path, and like every one of them it frees pages without
	// shrinking the file -- which is the state this test needs.
	// Older than the bulk, newer than the seed: the span the projection divides by is unchanged
	// and only the freed pages differ, which is the one variable this test is about.
	if _, err := st.Clear(ctx, now.AddDate(0, 0, -70), ""); err != nil {
		t.Fatal(err)
	}
	size, err := st.Size(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if size.Reclaimable == 0 {
		t.Skip("this SQLite build reclaimed the pages immediately; the branch cannot be exercised here")
	}

	line := growthLine(ctx, st, now)
	if strings.Contains(line, humanize.Bytes(size.Bytes)+" live") {
		t.Fatalf("growth line = %q, want live bytes rather than the file size (%s reclaimable)",
			line, humanize.Bytes(size.Reclaimable))
	}
	if !strings.Contains(line, humanize.Bytes(size.Bytes-size.Reclaimable)+" live") {
		t.Fatalf("growth line = %q, want %s of live bytes", line, humanize.Bytes(size.Bytes-size.Reclaimable))
	}
}
