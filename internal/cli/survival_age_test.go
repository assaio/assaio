package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/survival"
)

// TestSurvivalAgeLineStatesTheAgeBehindTheRate is B179's regression: survival is monotonic in
// commit age -- this repository reads 99% over a week and 92% over a year -- so a rate printed
// without its age invites a comparison between two windows that measured different things.
func TestSurvivalAgeLineStatesTheAgeBehindTheRate(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	res := survival.Result{
		OldestCommit: now.Add(-34 * 24 * time.Hour),
		NewestCommit: now,
		MedianCommit: now.Add(-13 * 24 * time.Hour),
	}
	got := survivalAgeLine(&res, now)
	for _, want := range []string{"median commit is 13 day(s) old", "spanning 34 day(s)", "monotonic in commit age"} {
		if !strings.Contains(got, want) {
			t.Fatalf("age line = %q, want it to contain %q", got, want)
		}
	}
}

// TestSurvivalAgeLineSaysUnknownRatherThanZero holds the no-fabrication rule: commits with no
// timestamp have no age, and printing "0 days old" would be a number nobody measured.
func TestSurvivalAgeLineSaysUnknownRatherThanZero(t *testing.T) {
	got := survivalAgeLine(&survival.Result{}, time.Now())
	if !strings.Contains(got, "unknown") {
		t.Fatalf("age line = %q, want an explicit unknown for undated commits", got)
	}
	if strings.Contains(got, "0 day(s) old") {
		t.Fatalf("age line = %q, want no invented age", got)
	}
}
