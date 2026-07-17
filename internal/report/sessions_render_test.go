package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderSessionsBlockWithData(t *testing.T) {
	stats := SessionStats{
		Count: 770, MedianTurns: 13, P90Turns: 47,
		MedianOutputTokens: 3120, MedianPeakContextTokens: 85000,
		MedianActiveMinutes: 12, CodeSessionShare: 0.18,
		CompactionRate: 0.01, SessionsPerActiveDay: 20.8,
	}
	var buf bytes.Buffer
	if err := RenderSessionsBlock(&buf, stats); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Sessions", "770 sessions", "median 13 turns (p90 47)", "12min active work",
		"peak context ~85k tokens", "3,120 output tokens/session",
		"18% produced code, 82% conversational", "1% hit context compaction", "20.8/active day",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("sessions block missing %q: %s", want, out)
		}
	}
	// The dropped cumulative-context and wall-clock-duration metrics must not reappear.
	for _, banned := range []string{"median length", "median context", "context tokens/session"} {
		if strings.Contains(out, banned) {
			t.Fatalf("sessions block must not render the retired metric %q: %s", banned, out)
		}
	}
}

func TestRenderSessionsBlockCompactTokens(t *testing.T) {
	var buf bytes.Buffer
	stats := SessionStats{Count: 1, MedianPeakContextTokens: 1_250_000, MedianOutputTokens: 900}
	if err := RenderSessionsBlock(&buf, stats); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "~1.2M tokens") {
		t.Fatalf("million-token peak must render as ~1.2M: %s", buf.String())
	}
}

func TestRenderSessionsBlockEmptyIsHonest(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderSessionsBlock(&buf, SessionStats{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Sessions") || !strings.Contains(out, "No sessions in this window.") {
		t.Fatalf("empty session stats must render an honest empty state: %s", out)
	}
}
