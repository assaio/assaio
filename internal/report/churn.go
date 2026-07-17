package report

import "github.com/assaio/assaio/internal/store"

// ChurnStat is the aggregate rework/thrash signal across a set of usage rows: how much
// AI-added code got removed again within the same transcript it was added in -- the
// honest local proxy for "AI wrote code that didn't stick" (see usage.Record.ReworkLines).
type ChurnStat struct {
	// LinesAdded is total AI-added code lines.
	LinesAdded int64
	// ReworkLines is the subset of LinesAdded later removed within the same transcript+file.
	ReworkLines int64
	// ReworkRate is ReworkLines / LinesAdded; 0 when LinesAdded is zero, never a
	// divide-by-zero panic. That 0 is a placeholder for an undefined ratio, not a
	// measured rate -- a renderer must check LinesAdded itself (e.g. via shareOrDash on
	// the raw counts) before formatting this as a confident percentage; see
	// internal/analyze/rework.go's "rework" Figure.
	ReworkRate float64
}

// BuildChurn aggregates rework signals across rows. Pure and empty-safe: no rows yields
// a zero-value ChurnStat.
func BuildChurn(rows []store.UsageRow) ChurnStat {
	var s ChurnStat
	for i := range rows {
		s.LinesAdded += rows[i].LinesAdded
		s.ReworkLines += rows[i].ReworkLines
	}
	s.ReworkRate = reworkRate(s.ReworkLines, s.LinesAdded)
	return s
}

// reworkRate is rework/added, 0 when added is zero -- never a divide-by-zero.
func reworkRate(rework, added int64) float64 {
	if added == 0 {
		return 0
	}
	return float64(rework) / float64(added)
}
