package plugin

import (
	"encoding/json"
	"reflect"
	"slices"

	"github.com/assaio/assaio/internal/analyze"
)

// The document a plugin actually reads. metricInput is the protocol's full vocabulary -- it is
// what the published reference is generated from -- and this is the projection of it: only the
// sections this run grants, carrying only the columns and rows the plugin declared.
// TestEveryEnvelopeFieldReachesTheDocument is what keeps the two from drifting apart.

func (mi *metricInput) marshal() ([]byte, error) { return json.Marshal(mi.document()) }

func (mi *metricInput) document() map[string]any {
	// The header is unconditional: it is bounded by the number of tools rather than the number
	// of observations, and a plugin cannot judge what it did receive without knowing the window
	// it was cut from.
	doc := map[string]any{
		"assaio_metric_input": mi.Version,
		"now":                 mi.Now,
		"recentDays":          mi.RecentDays,
		"answers":             mi.Answers,
		"windowStart":         mi.WindowStart,
		"historyStart":        mi.HistoryStart,
	}
	if len(mi.Withheld) > 0 {
		doc["withheld"] = mi.Withheld
	}

	p := mi.Projection
	rows := map[string]rowCount{}
	put := func(key string, section any) {
		kept, count := filterRows(key, reflect.ValueOf(section), p.Where)
		rows[key] = count
		doc[key] = projectRows(key, kept, p.Fields)
	}
	if slices.Contains(p.Needs, analyze.CapUsage) {
		put("usage", mi.Usage)
		put("byModel", mi.ByModel)
		put("byProject", mi.ByProject)
		// The prepared views ride with the rows they aggregate: they are bounded by model and
		// project count, so a separate grant would be a wider contract bought with no bytes.
		//
		// A predicate on any of those sections voids that ground. Both views still span the whole
		// granted window, so sum(usage.in)/totals.tokens would be a share of a population the
		// delivered rows do not describe -- and a plugin author has no way to see that from the
		// document. They are withheld rather than sent unlabelled: absence is already a state
		// every plugin handles (one that does not declare CapUsage receives neither), and the
		// window's own denominator remains available as projection.rows[section].available, which
		// is what that count exists for. The condition is the declaration, never the outcome: a
		// predicate that happens to admit every row on one window and drop rows on the next must
		// not make these keys appear and disappear underneath a plugin.
		if !narrowsAnySection(p.Where, analyze.CapUsage) {
			doc["totals"], doc["delegation"] = mi.Totals, mi.Delegation
		}
	}
	if slices.Contains(p.Needs, analyze.CapSessions) {
		put("sessions", mi.Sessions)
	}
	if slices.Contains(p.Needs, analyze.CapPrices) {
		doc["prices"], doc["planMonthlyCost"] = mi.Prices, mi.PlanMonthlyCost
	}
	if slices.Contains(p.Needs, analyze.CapAttribution) {
		put("skills", mi.Skills)
		put("agents", mi.Agents)
	}
	if slices.Contains(p.Needs, analyze.CapTurnSizing) {
		put("turnSizing", mi.TurnSizing)
	}
	if slices.Contains(p.Needs, analyze.CapCacheMisses) {
		put("cacheMisses", mi.CacheMisses)
	}
	if slices.Contains(p.Needs, analyze.CapTrace) {
		put("trace", mi.Trace)
	}
	p.Rows = rows
	doc["projection"] = p
	return doc
}

// narrowsAnySection reports whether any declared predicate addresses a section this capability
// carries. A predicate naming a section it does not carry narrows nothing here: a filter on
// sessions leaves the usage rows, and the views aggregating them, in agreement.
func narrowsAnySection(where map[string][]string, capability analyze.Capability) bool {
	for key := range where {
		name, _, ok := splitPredicate(key)
		if ok && sections[name].capability == capability {
			return true
		}
	}
	return false
}
