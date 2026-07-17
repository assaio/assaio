package report

import (
	"sort"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// hotProjects ranks projects by cost within the recent sub-window, highest spend
// first. Unpriced-only projects sort last (nil cost) rather than being hidden, the
// same "show it, mark it unpriced" convention as the effectiveness table.
func hotProjects(recentRows []store.UsageRow, t pricing.Table, topN int) []GroupStat {
	stats := projectStats(recentRows, t)
	sortByCostDesc(stats)
	return capTop(stats, topN)
}

// goingStaleProjects ranks, by prior cost desc, projects with usage in priorRows that
// have no usage at all in recentRows -- i.e. were active, then went quiet.
func goingStaleProjects(priorRows, recentRows []store.UsageRow, t pricing.Table, topN int) []GroupStat {
	stillActive := distinctValues(recentRows, projectKey)
	stale := excluding(priorRows, stillActive, projectKey)
	stats := projectStats(stale, t)
	sortByCostDesc(stats)
	return capTop(stats, topN)
}

// dormantTools reports tools with usage in priorRows but none in recentRows: set up or
// used before, untouched recently -- the local analog of a dormant seat, scoped to
// tools rather than named individuals.
func dormantTools(priorRows, recentRows []store.UsageRow, t pricing.Table) []GroupStat {
	stillActive := distinctValues(recentRows, toolKey)
	dormant := excluding(priorRows, stillActive, toolKey)
	return groupStats(dormant, toolKey, t)
}

func projectKey(u *store.UsageRow) string { return u.Project }

func toolKey(u *store.UsageRow) string { return u.Tool }

// projectStats groups rows by project into GroupStat.
func projectStats(rows []store.UsageRow, t pricing.Table) []GroupStat {
	return groupStats(rows, projectKey, t)
}

// groupStats groups rows by keyAt into GroupStat, in groupBy's key-sorted order, with
// CostPer100Lines filled in for each resulting group.
func groupStats(rows []store.UsageRow, keyAt func(*store.UsageRow) string, t pricing.Table) []GroupStat {
	out := groupBy(
		len(rows),
		func(i int) string { return keyAt(&rows[i]) },
		func(key string) GroupStat { return GroupStat{Name: key} },
		func(g *GroupStat, i int) { accumulateGroupStat(g, &rows[i], t) },
	)
	for i := range out {
		out[i].CostPer100Lines = costPer100Lines(out[i].Cost, out[i].LinesAdded)
	}
	return out
}

// accumulateGroupStat folds u's AI lines, last-active day, and (when priced) cost into g.
func accumulateGroupStat(g *GroupStat, u *store.UsageRow, t pricing.Table) {
	g.LinesAdded += u.LinesAdded
	if u.Day > g.LastActive {
		g.LastActive = u.Day
	}
	cost, ok := t.CostTokens(u.Model, u.In, u.Out, u.CacheWrite, u.CacheRead)
	if !ok {
		g.HasUnpriced = true
		return
	}
	if g.Cost == nil {
		zero := 0.0
		g.Cost = &zero
	}
	*g.Cost += cost
}

// distinctValues returns the set of distinct keyAt values across rows.
func distinctValues(rows []store.UsageRow, keyAt func(*store.UsageRow) string) map[string]struct{} {
	set := make(map[string]struct{}, len(rows))
	for i := range rows {
		set[keyAt(&rows[i])] = struct{}{}
	}
	return set
}

// excluding returns the rows whose keyAt value is not in exclude.
func excluding(rows []store.UsageRow, exclude map[string]struct{}, keyAt func(*store.UsageRow) string) []store.UsageRow {
	out := make([]store.UsageRow, 0, len(rows))
	for i := range rows {
		if _, skip := exclude[keyAt(&rows[i])]; !skip {
			out = append(out, rows[i])
		}
	}
	return out
}

// sortByCostDesc orders stats by cost descending. A stable sort preserves groupBy's
// alphabetical starting order as the deterministic tie-break for equal costs.
func sortByCostDesc(stats []GroupStat) {
	sort.SliceStable(stats, func(i, j int) bool {
		return costValue(stats[i].Cost) > costValue(stats[j].Cost)
	})
}

// costValue treats an unpriced group's nil cost as below any real cost (including
// zero), so it sorts last rather than tying with a legitimately zero-cost group.
func costValue(c *float64) float64 {
	if c == nil {
		return -1
	}
	return *c
}

// capTop returns the first topN stats; topN <= 0 means unlimited.
func capTop(stats []GroupStat, topN int) []GroupStat {
	if topN > 0 && len(stats) > topN {
		return stats[:topN]
	}
	return stats
}
