package report

import (
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// GroupStat is one named group's cost, AI-line output, and recency signal. It is the
// shared unit both the terminal dashboard and a future HTML/team view render from.
type GroupStat struct {
	// Name is the group label, e.g. a project or tool name.
	Name string `json:"name"`
	// Cost is USD cost summed from priced usage only; nil when nothing in the group priced.
	Cost *float64 `json:"cost"`
	// HasUnpriced reports whether this group excludes some usage's cost because its
	// model has no known price.
	HasUnpriced bool `json:"has_unpriced"`
	// LinesAdded is AI-added code lines.
	LinesAdded int64 `json:"lines_added"`
	// CostPer100Lines is Cost per 100 AI lines; nil when LinesAdded is zero or Cost is unknown.
	CostPer100Lines *float64 `json:"cost_per_100_lines"`
	// LastActive is the most recent Day this group had any usage.
	LastActive string `json:"last_active"`
}

// Inventory summarizes distinct counts and cost/line totals across all queried usage.
type Inventory struct {
	Projects, Models, Tools, Entrypoints, Days int
	// TotalCost is USD cost summed from priced usage only; nil when nothing priced.
	TotalCost *float64
	// HasUnpriced reports whether some usage was excluded from TotalCost because its
	// model has no known price.
	HasUnpriced bool
	// TotalLinesAdded is AI-added code lines across all queried usage.
	TotalLinesAdded int64
}

// Insights is a pure, dependency-free snapshot of usage patterns computed from stored
// usage rows: where spend concentrates now (Hot), projects that were active but have
// gone quiet (GoingStale), tools set up before but untouched recently (DormantTools),
// and overall Inventory counts. It carries project- and tool-level signals only --
// never a named-individual ranking.
type Insights struct {
	Hot          []GroupStat
	GoingStale   []GroupStat
	DormantTools []GroupStat
	Inventory    Inventory
}

// BuildInsights splits rows into a recent sub-window (Day >= now-recent) and the
// remaining prior rows still within the queried range, then derives Hot, GoingStale,
// DormantTools, and Inventory from that split. topN caps Hot and GoingStale; topN <= 0
// means unlimited. BuildInsights never fails: on empty input it returns a zero-value
// Insights.
func BuildInsights(rows []store.UsageRow, t pricing.Table, now time.Time, recent time.Duration, topN int) Insights {
	recentRows, priorRows := splitRecent(rows, now, recent)
	return Insights{
		Hot:          hotProjects(recentRows, t, topN),
		GoingStale:   goingStaleProjects(priorRows, recentRows, t, topN),
		DormantTools: dormantTools(priorRows, recentRows, t),
		Inventory:    BuildInventory(rows, t),
	}
}

// splitRecent partitions rows into those on or after now-recent (recent) and the rest
// (prior), comparing Day as a "YYYY-MM-DD" string so no per-row time parsing is needed.
func splitRecent(rows []store.UsageRow, now time.Time, recent time.Duration) (recentRows, priorRows []store.UsageRow) {
	cutoff := now.UTC().Add(-recent).Format("2006-01-02")
	for i := range rows {
		if rows[i].Day >= cutoff {
			recentRows = append(recentRows, rows[i])
		} else {
			priorRows = append(priorRows, rows[i])
		}
	}
	return recentRows, priorRows
}

// BuildInventory counts distinct projects/models/tools/entrypoints/days and totals cost
// and AI lines across rows, independent of any recent/prior split. Exported so callers
// that only need inventory counts (e.g. doctor) can skip the Hot/GoingStale/DormantTools
// computation.
func BuildInventory(rows []store.UsageRow, t pricing.Table) Inventory {
	projects := make(map[string]struct{})
	models := make(map[string]struct{})
	tools := make(map[string]struct{})
	entrypoints := make(map[string]struct{})
	days := make(map[string]struct{})

	var inv Inventory
	var cost float64
	var hasCost bool
	for i := range rows {
		r := &rows[i]
		projects[r.Project] = struct{}{}
		models[r.Model] = struct{}{}
		tools[r.Tool] = struct{}{}
		entrypoints[r.Entrypoint] = struct{}{}
		days[r.Day] = struct{}{}
		inv.TotalLinesAdded += r.LinesAdded

		if c, ok := t.CostTokens(r.Model, r.In, r.Out, r.CacheWrite, r.CacheRead); ok {
			cost += c
			hasCost = true
		} else {
			inv.HasUnpriced = true
		}
	}
	inv.Projects, inv.Models, inv.Tools, inv.Entrypoints, inv.Days = len(projects), len(models), len(tools), len(entrypoints), len(days)
	if hasCost {
		inv.TotalCost = &cost
	}
	return inv
}
