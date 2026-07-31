package cli

import (
	"strings"
	"testing"
)

func TestCompactReportsTheSizeItReclaimed(t *testing.T) {
	driftHome(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, "compact")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "store:") || !strings.Contains(out, "B") {
		t.Fatalf("compact must report the store's size: %q", out)
	}
}

// TestCompactOnAnEmptyStoreIsHarmless keeps the maintenance path safe to put in a cron
// job: nothing to reclaim is a normal outcome, not an error.
func TestCompactOnAnEmptyStoreIsHarmless(t *testing.T) {
	driftHome(t)
	if _, err := runCommand(t, "compact"); err != nil {
		t.Fatalf("compact on an empty store must succeed: %v", err)
	}
}

func TestDoctorReportsStoreSize(t *testing.T) {
	driftHome(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, "doctor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "size:") {
		t.Fatalf("doctor must make store growth legible: %q", out)
	}
}

// TestClearPointsAtTheSpaceItDidNotReclaim is the honest half of deleting rows: SQLite
// keeps the pages, so a user who ran clear to free disk space has to be told the file did
// not actually shrink. It needs a corpus large enough to occupy more than the store's
// minimum page allocation -- below that there is genuinely nothing to reclaim, and saying
// otherwise would be the noise this nudge is meant to avoid.
func TestClearPointsAtTheSpaceItDidNotReclaim(t *testing.T) {
	bulkHome(t, 3000)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, "clear", "--all", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "compact") {
		t.Fatalf("clear must say how to actually reclaim the space: %q", out)
	}
}

// TestClearStaysQuietWhenThereIsNothingToReclaim is the same nudge's other half.
func TestClearStaysQuietWhenThereIsNothingToReclaim(t *testing.T) {
	driftHome(t)
	if _, err := runCommand(t, "backfill"); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, "clear", "--all", "--yes")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "compact") {
		t.Fatalf("nothing was reclaimable, so clear must not suggest compacting: %q", out)
	}
}
