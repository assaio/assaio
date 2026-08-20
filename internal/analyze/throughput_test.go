package analyze

import (
	"strings"
	"testing"
)

// TestThroughputTrendDirectionIsAShareNotAPercentage is a unit regression: changePct is the
// fraction (recent-prior)/prior despite its name, so dividing it again renders a collapse to
// zero as "down 1%".
func TestThroughputTrendDirectionIsAShareNotAPercentage(t *testing.T) {
	for _, tc := range []struct {
		change float64
		want   string
	}{
		{-1, "down 100%"},
		{0.35, "up 35%"},
		{0, "unchanged"},
	} {
		if got := trendDirection(tc.change); got != tc.want {
			t.Errorf("trendDirection(%v) = %q, want %q", tc.change, got, tc.want)
		}
	}
}

// TestThroughputReadsACollapseAsADirection guards the floor's meaning: it exists to stop a
// 1-to-2-line swing reading as +100%, not to hide output falling from thousands of lines to
// none, which is exactly the direction worth reading.
func TestThroughputReadsACollapseAsADirection(t *testing.T) {
	got := throughputTakeaway(1000, -1, max(int64(0), int64(201347)) >= throughputMinLinesForTrend)
	if !strings.Contains(got, "down 100%") {
		t.Fatalf("Takeaway = %q, want the collapse stated as a direction", got)
	}
}
