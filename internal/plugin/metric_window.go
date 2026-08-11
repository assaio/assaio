package plugin

// The window aggregates a metric plugin reads beside the usage rows: attribution, the raw
// per-turn grain, and stated cache-miss reasons. Each is a question about the whole window
// that no per-row sum can answer, which is why they cross the boundary as their own shapes.

import (
	"github.com/assaio/assaio/internal/store"
)

// metricAttributionRow is one skill's or one sub-agent's totals. Sessions counts the
// distinct sessions the name appeared in, so a plugin can tell one heavy session from a
// habit.
type metricAttributionRow struct {
	Name     string `json:"name"`
	Tokens   int64  `json:"tokens"`
	Lines    int64  `json:"lines"`
	Records  int64  `json:"records"`
	Sessions int64  `json:"sessions"`
}

// metricModelTurns counts, per model, output-producing turns and how many of them were
// small. SmallTurns is a subset of Turns, never a separate population.
type metricModelTurns struct {
	Model      string `json:"model"`
	Turns      int64  `json:"turns"`
	SmallTurns int64  `json:"smallTurns"`
}

// metricCacheMissRow is one tool's count of turns that stated one cache-miss reason. The
// reason vocabulary is the tool's own, not assaio's.
type metricCacheMissRow struct {
	Tool   string `json:"tool"`
	Reason string `json:"reason"`
	Turns  int64  `json:"turns"`
}

func attributionWire(rows []store.AttributionRow) []metricAttributionRow {
	out := make([]metricAttributionRow, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, metricAttributionRow{
			Name: r.Name, Tokens: r.Tokens, Lines: r.Lines,
			Records: r.Records, Sessions: r.Sessions,
		})
	}
	return out
}

func turnSizingWire(rows []store.ModelTurns) []metricModelTurns {
	out := make([]metricModelTurns, 0, len(rows))
	for i := range rows {
		out = append(out, metricModelTurns{
			Model: rows[i].Model, Turns: rows[i].Turns, SmallTurns: rows[i].SmallTurns,
		})
	}
	return out
}

func cacheMissWire(rows []store.CacheMissRow) []metricCacheMissRow {
	out := make([]metricCacheMissRow, 0, len(rows))
	for i := range rows {
		out = append(out, metricCacheMissRow{
			Tool: rows[i].Tool, Reason: rows[i].Reason, Turns: rows[i].Turns,
		})
	}
	return out
}
