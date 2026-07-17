package report

import (
	"fmt"
	"strings"
)

// Aggregate groups rows by the single dimension by, summing tokens and cost. Cost is
// summed only across priced rows in each group; a group containing any unpriced usage
// keeps HasUnpriced=true so callers can flag that its cost excludes that usage. by="day"
// returns rows unchanged (Build already groups by day/tool/model). An unknown dimension
// returns an error listing the valid ones.
func Aggregate(rows []Row, by string) ([]Row, error) {
	if by == "day" {
		return rows, nil
	}
	if !isValidDim(by) {
		return nil, fmt.Errorf("unknown dimension %q (want one of %s)", by, strings.Join(validDims, ", "))
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

func isValidDim(by string) bool {
	for _, d := range validDims {
		if d == by {
			return true
		}
	}
	return false
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
	case "member":
		g.Member = key
	}
	return g
}

// accumulate folds r's tokens and cost into group g.
func accumulate(g, r *Row) {
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
}
