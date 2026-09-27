package report

import (
	"encoding/csv"
	"io"
	"strconv"
)

// RenderCSV writes rows to w as CSV with a header row. The three annotation columns are
// part of the fixed shape rather than added per dimension: `--by task|outcome|difficulty`
// stamps the group key into one of them and leaves every other identity column empty, so a
// header without them emitted rows nothing could tell apart. cache_write_1h and
// cache_write_tiered are appended last so a consumer reading columns by position keeps working;
// cache_write_1h is empty when no source behind the row states the tier, as cost is when
// unpriced.
func RenderCSV(w io.Writer, rows []Row) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"day", "tool", "model", "project", "entrypoint", "member", "granularity",
		"task", "outcome", "difficulty", "in", "out",
		"cache_read", "cache_write", "reasoning", "cache_eff", "cost", "priced", "has_unpriced", "tokened",
		"cache_write_1h", "cache_write_tiered",
	})
	for i := range rows {
		r := &rows[i]
		cost := ""
		if r.Priced {
			cost = strconv.FormatFloat(*r.Cost, 'f', 6, 64)
		}
		cacheEffStr := ""
		if r.CacheEff != nil {
			cacheEffStr = strconv.FormatFloat(*r.CacheEff, 'f', 6, 64)
		}
		long := ""
		if r.CacheWrite1h != nil {
			long = strconv.FormatInt(*r.CacheWrite1h, 10)
		}
		_ = cw.Write([]string{
			r.Day, r.Tool, r.Model, r.Project, r.Entrypoint, r.Member, r.Granularity,
			r.Task, r.Outcome, r.Difficulty,
			strconv.FormatInt(r.In, 10), strconv.FormatInt(r.Out, 10),
			strconv.FormatInt(r.CacheRead, 10), strconv.FormatInt(r.CacheWrite, 10),
			strconv.FormatInt(r.Reasoning, 10), cacheEffStr,
			cost, strconv.FormatBool(r.Priced), strconv.FormatBool(r.HasUnpriced), strconv.FormatBool(r.Tokened),
			long, strconv.FormatInt(r.CacheWriteTiered, 10),
		})
	}
	cw.Flush()
	return cw.Error()
}
