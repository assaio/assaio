package drift

import (
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// TestHealthyShapesStayQuiet is the heuristics' disqualifying test: every shape here is
// what normal operation looks like on a real machine, and a canary firing on any of them
// would make the whole mechanism noise. The figures come from a 4.5 GB Claude Code corpus
// -- 5742 transcripts, 119698 records, 197 of them carrying no tokens (0.165%).
func TestHealthyShapesStayQuiet(t *testing.T) {
	steady, plugin := run(5742, 40, 840, 0, 1), run(0, 0, 500, 0, 0)
	tests := []struct {
		name    string
		history []store.SourceRun
		current store.SourceRun
	}{
		{
			name:    "the first full backfill, with no history at all",
			history: nil,
			current: run(5742, 5742, 119698, 0, 197),
		},
		{
			name:    "incremental steady state",
			history: repeat(9, &steady),
			current: steady,
		},
		{
			name:    "an incremental run where nothing changed on disk",
			history: repeat(9, &steady),
			current: run(5742, 0, 0, 0, 0),
		},
		{
			name:    "a run that only picked up one freshly started session",
			history: repeat(9, &steady),
			current: run(5743, 1, 2, 0, 0),
		},
		{
			name:    "a full re-parse after a new build invalidated the ingest state",
			history: repeat(9, &steady),
			current: run(5742, 5742, 119698, 0, 197),
		},
		{
			name:    "the vendor's own retention pruning old transcripts run after run",
			history: []store.SourceRun{run(6200, 40, 840, 0, 1), run(6050, 40, 840, 0, 1), run(5900, 40, 840, 0, 1)},
			current: run(5742, 40, 840, 0, 1),
		},
		{
			name:    "a plugin source, which has no files to discover",
			history: repeat(9, &plugin),
			current: plugin,
		},
		{
			name:    "a source whose corpus genuinely grew",
			history: repeat(9, &steady),
			current: run(9000, 200, 4200, 0, 5),
		},
		{
			name:    "one noisy run in an otherwise healthy history",
			history: []store.SourceRun{steady, run(5742, 1, 1, 0, 0), steady, steady},
			current: steady,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Evaluate(append(tt.history, tt.current)); len(got) != 0 {
				t.Fatalf("healthy shape fired %+v", got)
			}
		})
	}
}
