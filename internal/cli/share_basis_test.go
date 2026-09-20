package cli

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

func TestShareBasisReadsTheRetainedRuns(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 9, d, 8, 0, 0, 0, time.UTC) }
	run := func(d, found int) store.SourceRun { return store.SourceRun{RanAt: day(d), Discovered: found} }
	usage := []store.UsageRow{{Tool: "claude-code"}, {Tool: "codex"}, {Tool: "agy"}, {Tool: "claude-code"}}
	cases := []struct {
		name    string
		runs    map[string][]store.SourceRun
		through time.Time
		stopped bool
	}{
		{"the earliest of the latest runs", map[string][]store.SourceRun{
			"claude-code": {run(1, 3), run(19, 4)}, "codex": {run(17, 2)}, "agy": {run(19, 1)},
		}, day(17), false},
		{"a counting source with no run on record", map[string][]store.SourceRun{
			"claude-code": {run(19, 4)},
		}, time.Time{}, false},
		{"no counting source at all is read through now", nil, day(20), false},
		{"a source that stopped being found", map[string][]store.SourceRun{
			"claude-code": {run(1, 3), run(19, 0)}, "codex": {run(19, 2)},
		}, day(19), true},
		{"a source that never found anything is not one that stopped", map[string][]store.SourceRun{
			"claude-code": {run(19, 4)}, "codex": {run(1, 0), run(19, 0)},
		}, day(19), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tools := countingTools(usage)
			if len(tools) != 2 {
				t.Fatalf("countingTools = %v, want the two token- or line-counting sources", tools)
			}
			if c.runs == nil {
				tools = countingTools([]store.UsageRow{{Tool: "agy"}})
			}
			if got := readThrough(c.runs, tools, day(20)); !got.Equal(c.through) {
				t.Errorf("readThrough = %v, want %v", got, c.through)
			}
			if got := stoppedReading(c.runs, tools); got != c.stopped {
				t.Errorf("stoppedReading = %v, want %v", got, c.stopped)
			}
		})
	}
}
