package report

import (
	"github.com/assaio/assaio/internal/label"
)

// Aggregate groups rows by the single dimension by, summing tokens and cost. Cost is
// summed only across priced rows in each group; a group containing any unpriced usage
// keeps HasUnpriced=true so callers can flag that its cost excludes that usage. by="day"
// returns rows unchanged, which is not the same as one row per day: the store keys a group
// by tool, model, project, entrypoint, member and granularity as well, so a day arrives as
// many rows. CollapseForTable is what folds those, and only for the display that has no
// column for the difference. An unknown dimension returns an error listing the valid ones.
func Aggregate(rows []Row, by string) ([]Row, error) {
	if by == "day" {
		return rows, nil
	}
	if err := DimError(by); err != nil {
		return nil, err
	}

	out := groupBy(
		len(rows),
		func(i int) string { return dimValue(&rows[i], by) },
		func(key string) Row { return *newGroup(by, key) },
		func(g *Row, i int) { accumulate(g, &rows[i]) },
	)
	for i := range out {
		out[i].CacheEff = cacheEff(out[i].In, out[i].CacheRead)
	}
	return out, nil
}

// newGroup starts an empty group row, stamping the dimension field that key identifies.
func newGroup(by, key string) *Row {
	g := &Row{}
	switch by {
	case "project":
		g.Project = key
	case "tool":
		g.Tool = key
	case "model":
		g.Model = key
	case "entrypoint":
		g.Entrypoint = key
	case label.Task:
		g.Task = key
	case label.Outcome:
		g.Outcome = key
	case label.Difficulty:
		g.Difficulty = key
	}
	return g
}

// GranularityMixed marks a group that merged per-turn and session-aggregate records. It
// exists only on an aggregated row: the store never produces one, because it groups by
// granularity instead of summing across it.
const GranularityMixed = "mixed"

// foldGranularity combines a group's granularity with the row being added. Grouping is the
// one place a session aggregate can vanish into a per-turn total, so the merge is recorded
// rather than resolved to whichever value arrived first.
func foldGranularity(group, row string) string {
	switch {
	case group == "":
		return row
	case row == "" || group == row:
		return group
	default:
		return GranularityMixed
	}
}

// accumulate folds r's tokens and cost into group g.
func accumulate(g, r *Row) {
	g.Granularity = foldGranularity(g.Granularity, r.Granularity)
	g.Tokened = g.Tokened || r.Tokened
	g.In += r.In
	g.Out += r.Out
	g.CacheRead += r.CacheRead
	g.CacheWrite += r.CacheWrite
	g.Reasoning += r.Reasoning
	if r.Priced {
		if g.Cost == nil {
			zero := 0.0
			g.Cost = &zero
		}
		*g.Cost += *r.Cost
		g.Priced = true
	}
	if r.HasUnpriced {
		g.HasUnpriced = true
	}
	g.UnpricedTokens += r.UnpricedTokens
}
