package plugin

import (
	"testing"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
)

// The prepared views are aggregates of the whole granted window. A predicate narrows the rows
// beside them and not the views, so sending both would put a denominator on the wire that the
// delivered rows do not describe -- sum(usage.in)/totals.tokens reading as a share of a
// population the plugin never received. There is no scope note that fixes that for a program:
// only the plugin's own predicate, echoed in projection.where, says which population is which,
// and a plugin that read it correctly would already be computing its own totals.
func TestPreparedViewsAreWithheldFromANarrowedWindow(t *testing.T) {
	in := tracedInput()
	in.Usage = append(in.Usage, store.UsageRow{Day: "2026-07-16", Tool: "codex", Model: "m", In: 5})

	for _, tc := range []struct {
		name  string
		where map[string][]string
		sent  bool
	}{
		{name: "no predicate", where: nil, sent: true},
		{
			name: "a predicate on a section this capability does not carry",
			// sessions is granted by its own capability, so filtering it leaves usage, totals and
			// delegation describing the same set.
			where: map[string][]string{"sessions.tool": {"claude-code"}}, sent: true,
		},
		{name: "a predicate on usage", where: map[string][]string{"usage.tool": {"claude-code"}}, sent: false},
		{name: "a predicate on byModel", where: map[string][]string{"byModel.model": {"m"}}, sent: false},
		{
			// The declaration decides, never the outcome: this predicate admits every row in this
			// window, and a rule that keyed on rows dropped would make the keys appear on one
			// window and vanish on the next.
			name:  "a usage predicate that happens to drop nothing",
			where: map[string][]string{"usage.tool": {"claude-code", "codex"}}, sent: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := granting(analyze.CapUsage, analyze.CapSessions)
			p.Where = tc.where
			doc := documentOf(&in, p)
			for _, key := range []string{"totals", "delegation"} {
				if _, present := doc[key]; present != tc.sent {
					t.Errorf("%q present = %v, want %v", key, present, tc.sent)
				}
			}
			// The rows themselves are never withheld, and the window's own denominator stays
			// reachable: that is what makes withholding an aggregate defensible rather than a
			// capability quietly removed.
			if _, present := doc["usage"]; !present {
				t.Error("usage rows were withheld, which no predicate asks for")
			}
			if got := doc["projection"].(metricProjection).Rows["usage"].Available; got != 2 {
				t.Errorf("projection.rows[usage].available = %d, want the window's own count of 2", got)
			}
		})
	}
}
