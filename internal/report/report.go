// Package report aggregates stored usage rows into priced report rows and renders
// them as a table, JSON, or CSV.
package report

import (
	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// Row is one day/tool/model/project/entrypoint usage group with its computed cost.
type Row struct {
	// Day, Tool, Model identify the usage group.
	Day   string `json:"day"`
	Tool  string `json:"tool"`
	Model string `json:"model"`
	// Project, Entrypoint further narrow the usage group.
	Project    string `json:"project"`
	Entrypoint string `json:"entrypoint"`
	// Member is "" for purely local usage; non-empty only on a central store synced
	// from a team member.
	Member string `json:"member"`
	// Granularity is "turn" for per-request records, "session" for session-aggregate
	// sources, or GranularityMixed once a group has merged both. It travels with the row
	// because a session aggregate summed into a per-turn total reads as per-turn, and
	// nothing downstream could tell the difference otherwise.
	Granularity string `json:"granularity"`
	// In, Out, CacheRead, CacheWrite, Reasoning are summed token counts.
	In         int64 `json:"in"`
	Out        int64 `json:"out"`
	CacheRead  int64 `json:"cache_read"`
	CacheWrite int64 `json:"cache_write"`
	Reasoning  int64 `json:"reasoning"`
	// Cost is the USD cost computed from the pricing table; nil when the model is unpriced.
	Cost *float64 `json:"cost"`
	// Priced reports whether Cost was computed from a known model price.
	Priced bool `json:"priced"`
	// CacheEff is CacheRead / (In + CacheRead); nil when both are zero.
	CacheEff *float64 `json:"cache_eff"`
	// HasUnpriced reports whether this row aggregates any usage with an unknown model
	// price, meaning Cost excludes that usage.
	HasUnpriced bool `json:"has_unpriced"`
	// UnpricedTokens is how much of In+Out+CacheRead+CacheWrite carried no known price.
	// HasUnpriced says a cost is incomplete; this says by how much, which is the difference
	// between a rounding error and a headline that is half the real figure. It rides on the
	// row so an aggregate can total it without a second pricing pass.
	UnpricedTokens int64 `json:"unpriced_tokens"`
	// Task, Outcome and Difficulty are the session annotations this group belongs to, and
	// are populated only when the caller grouped by one of them (store.UsageByLabel).
	Task       string `json:"task,omitempty"`
	Outcome    string `json:"outcome,omitempty"`
	Difficulty string `json:"difficulty,omitempty"`
}

// cacheEff returns CacheRead / (in + CacheRead), or nil when both are zero.
func cacheEff(in, cacheRead int64) *float64 {
	if in+cacheRead == 0 {
		return nil
	}
	eff := float64(cacheRead) / float64(in+cacheRead)
	return &eff
}

// Build prices each usage row against t, matching the model exactly first and falling
// back to pricing.NormalizeModel. Unmatched models are returned with Priced=false and
// Cost=nil.
func Build(rows []store.UsageRow, t pricing.Table) []Row {
	out := make([]Row, 0, len(rows))
	for i := range rows {
		u := &rows[i]
		r := Row{
			Day: u.Day, Tool: u.Tool, Model: u.Model, Project: u.Project, Entrypoint: u.Entrypoint, Member: u.Member,
			Granularity: u.Granularity,
			In:          u.In, Out: u.Out, CacheRead: u.CacheRead, CacheWrite: u.CacheWrite, Reasoning: u.Reasoning,
			Task: u.Task, Outcome: u.Outcome, Difficulty: u.Difficulty,
		}
		if cost, ok := t.CostTokens(u.Model, pricing.Tokens{In: u.In, Out: u.Out, CacheWrite: u.CacheWrite, CacheRead: u.CacheRead, CacheWrite1h: u.CacheWrite1h}); ok {
			r.Cost = &cost
			r.Priced = true
		} else {
			r.HasUnpriced = true
			r.UnpricedTokens = r.In + r.Out + r.CacheRead + r.CacheWrite
		}
		r.CacheEff = cacheEff(r.In, r.CacheRead)
		out = append(out, r)
	}
	return out
}

// validDims are the dimensions Aggregate accepts for --by, in canonical order. The three
// annotation axes only carry a value when the caller read usage through
// store.UsageByLabel; see usageForDim in internal/cli.
var validDims = []string{
	"day", "project", "tool", "model", "entrypoint", "member",
	label.Task, label.Outcome, label.Difficulty,
}

// dimValue returns r's value for one of validDims.
func dimValue(r *Row, by string) string {
	switch by {
	case "day":
		return r.Day
	case "project":
		return r.Project
	case "tool":
		return r.Tool
	case "model":
		return r.Model
	case "entrypoint":
		return r.Entrypoint
	case "member":
		return r.Member
	case label.Task:
		return orUnlabeled(r.Task)
	case label.Outcome:
		return orUnlabeled(r.Outcome)
	case label.Difficulty:
		return orUnlabeled(r.Difficulty)
	default:
		return ""
	}
}

// orUnlabeled names the group usage from a session nobody marked belongs to. It is a real
// group with a real name, never an omitted row: most usage is unannotated, and hiding it
// would present a slice of the window as the whole of it.
func orUnlabeled(v string) string {
	if v == "" {
		return label.Unlabeled
	}
	return v
}

// IsLabelDim reports whether by groups on a session annotation, which is what tells a
// caller to read usage through store.UsageByLabel rather than store.Usage.
func IsLabelDim(by string) bool {
	return by == label.Task || by == label.Outcome || by == label.Difficulty
}

// emptyDimLabel is the table placeholder for a grouped row whose --by dimension value is
// "": "(local)" for member, since an empty member is the expected default for
// never-synced local usage, not a missing value; "(unknown)" for every other dimension.
func emptyDimLabel(by string) string {
	if by == "member" {
		return "(local)"
	}
	return "(unknown)"
}
