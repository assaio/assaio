package report

// CollapseForTable folds rows the ungrouped table layout has no column to tell apart into
// one row per day+tool+model. The store keys a usage group by project, entrypoint, member
// and granularity as well (and by the three annotations under UsageByLabel), so a single
// day arrives as several rows whose Day/Tool/Model cells are identical and whose order is
// decided by dimensions the reader cannot see -- and the first of them reads as the day's
// total. Rows already grouped on one dimension carry one row per group and are returned
// unchanged. Only the table collapses: JSON and CSV carry every dimension in a column, and
// are where the detail dropped here stays available.
func CollapseForTable(rows []Row, by string) []Row {
	if groupedDim(by) {
		return rows
	}
	out := groupBy(
		len(rows),
		func(i int) string { return tableKey(&rows[i]) },
		func(string) Row { return Row{} },
		func(g *Row, i int) {
			r := &rows[i]
			// Safe to assign rather than fold: tableKey makes these three equal across a group.
			g.Day, g.Tool, g.Model = r.Day, r.Tool, r.Model
			accumulate(g, r)
		},
	)
	for i := range out {
		out[i].CacheEff = cacheEff(out[i].In, out[i].CacheRead)
	}
	return out
}

// groupedDim reports whether by renders as a single named dimension column rather than the
// Day/Tool/Model layout. "" is the ungrouped layout too: it is what a caller that never
// aggregated passes.
func groupedDim(by string) bool { return by != "" && by != "day" }

// tableKey identifies the group one ungrouped table row stands for. The separator is NUL
// because no dimension value can contain one, so the key sorts field by field in order and
// two different splits of the same characters can never collide.
func tableKey(r *Row) string { return r.Day + "\x00" + r.Tool + "\x00" + r.Model }
