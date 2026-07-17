package cli

import (
	"fmt"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// demoWindowDays is the span the bundled sample data covers, and the window demo queries --
// wide enough for the week-over-week trend and staleness signals to be real.
const demoWindowDays = 30

// demoProjects are the sample repositories usage is spread across, so report/effectiveness
// show a real per-project breakdown.
var demoProjects = []string{"web-app", "api-service", "data-pipeline"}

// demoModels weights the sample mix toward premium opus, with cheaper sonnet and a small
// slice of an unpriced model, so the litmus shows a confident premium/cheaper split, a real
// est.-savings line (opus repriced on sonnet), and the honest "unpriced (unknown model)"
// figure -- "acme-llm-v2" is deliberately absent from the price table.
var demoModels = []string{
	"claude-opus-4-5", "claude-opus-4-5", "claude-opus-4-5",
	"claude-sonnet-4-5", "claude-sonnet-4-5", "acme-llm-v2",
}

// demoRecords builds the bundled sample usage: a cycle of sessions per active day across
// demoWindowDays, deterministic (no randomness) so the demo reads identically every run.
// It skips one day each week so active-days stays below the window, keeping the pace honest.
func demoRecords(now time.Time) []usage.Record {
	var recs []usage.Record
	seq := 0
	for d := demoWindowDays - 1; d >= 1; d-- {
		if d%7 == 0 {
			continue
		}
		day := now.AddDate(0, 0, -d)
		for s := 0; s <= d%2; s++ {
			recs = append(recs, demoSession(day, d, s, &seq)...)
		}
	}
	return recs
}

// demoSession builds one session: several turn records sharing a session_id, minutes apart
// (so ActiveMinutes accrues), on one project and model, with realistic tokens/lines/edits
// and an occasional compaction on longer sessions.
func demoSession(day time.Time, dayIdx, sIdx int, seq *int) []usage.Record {
	model := demoModels[(dayIdx+sIdx)%len(demoModels)]
	project := demoProjects[(dayIdx+sIdx)%len(demoProjects)]
	sessionID := fmt.Sprintf("demo-%d-%d", dayIdx, sIdx)
	turns := 3 + (dayIdx+sIdx)%5
	recs := make([]usage.Record, 0, turns)
	for tIdx := range turns {
		*seq++
		recs = append(recs, usage.Record{
			Tool:             "claude-code",
			SessionID:        sessionID,
			Timestamp:        day.Add(time.Duration(tIdx*4) * time.Minute),
			Model:            model,
			InputTokens:      1200 + int64(tIdx)*300,
			OutputTokens:     600 + int64(tIdx)*150,
			CacheReadTokens:  8000 + int64(tIdx)*1500,
			CacheWriteTokens: 400,
			DedupeKey:        fmt.Sprintf("demo-%d", *seq),
			Project:          project,
			Entrypoint:       "cli",
			Granularity:      "turn",
			LinesAdded:       12 + int64((dayIdx+tIdx)%20),
			LinesRemoved:     int64((dayIdx + tIdx) % 5),
			Edits:            1 + int64(tIdx%3),
			ToolCalls:        2 + int64(tIdx%4),
			Rejected:         int64((dayIdx + tIdx) % 3 / 2),
			ReworkLines:      int64((dayIdx + tIdx) % 4),
			Compactions:      demoCompactions(tIdx, turns),
		})
	}
	return recs
}

// demoCompactions marks the last turn of a long session as compacted, so context health
// carries a little signal without reading as unhealthy.
func demoCompactions(tIdx, turns int) int64 {
	if turns >= 7 && tIdx == turns-1 {
		return 1
	}
	return 0
}
