package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderSessionsBlockWithData(t *testing.T) {
	stats := SessionStats{
		Count: 770, Turned: 770, Paced: 770, Edited: 770, Compacting: 770,
		Tokened: 770, Contexted: 770,
		MedianTurns: 13, P90Turns: 47,
		MedianOutputTokens: 3120, MedianPeakContextTokens: 85000,
		MedianActiveMinutes: 12, CodeSessionShare: 0.18,
		CompactionRate: 0.01, SessionsPerActiveDay: 20.8,
	}
	var buf bytes.Buffer
	if err := RenderSessionsBlock(&buf, &stats); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"Sessions", "770 sessions", "median 13 turns (p90 47)", "12min active work",
		"peak context ~85.0K tokens", "3,120 output tokens/session",
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
	stats := SessionStats{
		Count: 1, Turned: 1, Paced: 1, Edited: 1, Compacting: 1, Tokened: 1, Contexted: 1,
		MedianPeakContextTokens: 1_250_000, MedianOutputTokens: 900,
	}
	if err := RenderSessionsBlock(&buf, &stats); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "~1.2M tokens") {
		t.Fatalf("million-token peak must render as ~1.2M: %s", buf.String())
	}
}

// A source that records every turn and no token -- Antigravity CLI is the first -- must not
// have its silence rendered as a peak context of zero or a session that produced no output.
// The turn count beside them stays readable, which is the whole point of separate bases.
func TestRenderSessionsBlockSeparatesTurnsFromTokens(t *testing.T) {
	var buf bytes.Buffer
	stats := SessionStats{
		Count: 248, Turned: 248, Paced: 248, Edited: 248, Compacting: 0, Tokened: 0, Contexted: 0,
		MedianTurns: 3, P90Turns: 3, SessionsPerActiveDay: 82.7,
	}
	if err := RenderSessionsBlock(&buf, &stats); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"median 3 turns (p90 3)", "peak context not recorded", "output tokens not recorded",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("sessions block missing %q: %s", want, out)
		}
	}
	for _, banned := range []string{"~0 tokens", "0 output tokens/session"} {
		if strings.Contains(out, banned) {
			t.Fatalf("a token figure no source records must not print a zero (%q): %s", banned, out)
		}
	}
}

func TestRenderSessionsBlockEmptyIsHonest(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderSessionsBlock(&buf, &SessionStats{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Sessions") || !strings.Contains(out, "No sessions in this window.") {
		t.Fatalf("empty session stats must render an honest empty state: %s", out)
	}
}

// A window whose figures rest on fewer sessions than it holds has to say so, and a figure no
// source records must read as unrecorded rather than as a confident zero.
func TestRenderSessionsBlockStatesANarrowerBasis(t *testing.T) {
	var buf bytes.Buffer
	stats := SessionStats{
		Count: 10, Turned: 10, Paced: 10, Edited: 6, Compacting: 0, Tokened: 10, Contexted: 10,
		MedianTurns: 5, P90Turns: 9, MedianOutputTokens: 800,
		MedianPeakContextTokens: 40000, MedianActiveMinutes: 9,
		CodeSessionShare: 0.5, SessionsPerActiveDay: 2,
	}
	if err := RenderSessionsBlock(&buf, &stats); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "compaction not recorded") {
		t.Fatalf("a figure no source records must say so, not print 0%%: %s", out)
	}
	// The name is the point: six figures share this line, and "the narrowest reads 6 of 10"
	// with none of them named leaves the reader unable to tell whether the thin one is the
	// edit share, the compaction rate or the peak context.
	if !strings.Contains(out, "The narrowest figure above, the code-session split, reads 6 of 10 sessions") {
		t.Fatalf("the narrower basis must name the figure it is about: %s", out)
	}
}

// TestRenderSessionsBlockOmitsTheBasisLineWhenNothingNarrowerWasMeasured covers a store whose
// only source is Antigravity CLI: it records turns, focused minutes and edits for every
// session and no tokens, peak context or compaction at all. The three it cannot answer print
// "not recorded" above, so counting their zero as the narrowest basis produced "reads 0 of 248
// sessions" -- a coverage warning about figures that never printed a number.
func TestRenderSessionsBlockOmitsTheBasisLineWhenNothingNarrowerWasMeasured(t *testing.T) {
	var buf bytes.Buffer
	stats := SessionStats{
		Count: 248, Turned: 248, Paced: 248, Edited: 248, Compacting: 0, Tokened: 0, Contexted: 0,
		MedianTurns: 3, P90Turns: 3, CodeSessionShare: 0.01, SessionsPerActiveDay: 82.7,
	}
	if err := RenderSessionsBlock(&buf, &stats); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); strings.Contains(out, "The narrowest figure above") {
		t.Fatalf("no figure here rests on fewer sessions than the window holds: %s", out)
	}
}
