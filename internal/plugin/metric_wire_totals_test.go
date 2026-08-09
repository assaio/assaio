package plugin

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// TestMetricWireCarriesTheSameWindow is the metric-plugin half of the cross-surface rule
// (internal/calibration/surface.go): a plugin renders beside the built-in validators and may
// not disagree with them about the window it was handed. The wire is where that could break
// silently -- a field added to store.UsageRow and not to the wire row is a class the plugin
// never sees, and it would report a smaller total with no error anywhere.
func TestMetricWireCarriesTheSameWindow(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-03-02", Tool: "claude-code", Model: "claude-opus-5", Granularity: "turn",
			In: 10, Out: 20, CacheRead: 30, CacheWrite: 40, CacheWrite1h: 25, Reasoning: 5,
			LinesAdded: 7, LinesRemoved: 3, ReworkLines: 2, Edits: 1, ToolCalls: 4,
			Rejected: 1, Compactions: 1,
		},
		{
			Day: "2026-03-02", Tool: "codex", Model: "gpt-5.6-sol", Granularity: "turn",
			In: 100, Out: 200, CacheRead: 300, CacheWrite: 0, Reasoning: 50,
			LinesAdded: 70, LinesRemoved: 30, ReworkLines: 20, Edits: 10, ToolCalls: 40,
		},
	}
	in := analyze.BuildInput(rows, nil, pricing.Table{}, time.Now(), 7*24*time.Hour, analyze.Delegation{})
	wire := buildMetricInput(&in)

	var got struct{ in, out, cacheRead, cacheWrite, cacheWrite1h, reasoning, added, removed, rework, edits, calls int64 }
	for i := range wire.Usage {
		r := &wire.Usage[i]
		got.in += r.In
		got.out += r.Out
		got.cacheRead += r.CacheRead
		got.cacheWrite += r.CacheWrite
		got.cacheWrite1h += r.CacheWrite1h
		got.reasoning += r.Reasoning
		got.added += r.LinesAdded
		got.removed += r.LinesRemoved
		got.rework += r.ReworkLines
		got.edits += r.Edits
		got.calls += r.ToolCalls
	}
	want := got
	want.in, want.out, want.cacheRead, want.cacheWrite = 110, 220, 330, 40
	want.cacheWrite1h, want.reasoning = 25, 55
	want.added, want.removed, want.rework, want.edits, want.calls = 77, 33, 22, 11, 44
	if got != want {
		t.Errorf("the wire lost part of the window:\n got  %+v\n want %+v", got, want)
	}
	if in.Totals.Input != got.in || in.Totals.Output != got.out ||
		in.Totals.CacheRead != got.cacheRead || in.Totals.CacheWrite != got.cacheWrite {
		t.Errorf("the wire and analyze.Totals disagree: wire %+v, totals %+v", got, in.Totals)
	}
}
