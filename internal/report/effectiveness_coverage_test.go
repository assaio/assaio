package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// TestEffectivenessCoverageCountsRowsNotGroups: grouped by day, one group holds every source, so
// capability decided per group would claim that every source records lines and edits.
func TestEffectivenessCoverageCountsRowsNotGroups(t *testing.T) {
	claude := store.UsageRow{Day: "d", Tool: "claude-code", Model: "claude-opus-4-5", In: 1000, Out: 500, LinesAdded: 40, Edits: 2}
	tests := []struct {
		name  string
		other store.UsageRow
		want  string
	}{
		{"a priced source recording no lines", store.UsageRow{Day: "d", Tool: "gemini-cli", Model: "claude-opus-4-5", In: 4000, Out: 2000}, "20% of this table's tokens"},
		{"an activity-only source with no tokens", store.UsageRow{Day: "d", Tool: "agy", Edits: 3}, "AI lines and edits come only from the sources that record them"},
		{"a source recording lines but no edits", store.UsageRow{Day: "d", Tool: "copilot-cli", Model: "claude-opus-4-5", In: 1000, LinesAdded: 5}, "AI lines and edits come only from the sources that record them"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eff, err := BuildEffectiveness([]store.UsageRow{claude, tt.other}, table(), "day")
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			if err := RenderEffectivenessTable(&buf, eff, "day"); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			if strings.Contains(out, "Every source in this table records") || !strings.Contains(out, tt.want) {
				t.Fatalf("want %q and no claim of full coverage:\n%s", tt.want, out)
			}
		})
	}
}
