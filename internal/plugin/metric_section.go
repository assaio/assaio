package plugin

import (
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/assaio/assaio/internal/analyze"
)

// section is one addressable array on the stdin envelope: the capability that carries it, and
// the wire row type whose json tags name its columns. A projection is validated against this
// table rather than against a list written twice, so a misspelled column is refused at the
// boundary -- once the document is written, a column that was projected away and one that never
// existed read identically, and the second is a bug the plugin author cannot see.
type section struct {
	capability analyze.Capability
	row        reflect.Type
}

// sections is every projectable array on the envelope, addressed by its JSON key. The
// object-shaped parts of a capability -- totals, delegation, prices, planMonthlyCost -- ride
// with it whole: they are bounded by the number of models and projects rather than by the
// number of observations, so projecting them would buy bytes nobody spends at the price of a
// wider contract.
var sections = map[string]section{
	"usage":       {analyze.CapUsage, reflect.TypeOf(metricUsageRow{})},
	"byModel":     {analyze.CapUsage, reflect.TypeOf(metricModelStat{})},
	"byProject":   {analyze.CapUsage, reflect.TypeOf(metricProjectStat{})},
	"sessions":    {analyze.CapSessions, reflect.TypeOf(metricSessionRow{})},
	"trace":       {analyze.CapTrace, reflect.TypeOf(metricTimeline{})},
	"trace.steps": {analyze.CapTrace, reflect.TypeOf(metricStep{})},
	"skills":      {analyze.CapAttribution, reflect.TypeOf(metricAttributionRow{})},
	"agents":      {analyze.CapAttribution, reflect.TypeOf(metricAttributionRow{})},
	"turnSizing":  {analyze.CapTurnSizing, reflect.TypeOf(metricModelTurns{})},
	"cacheMisses": {analyze.CapCacheMisses, reflect.TypeOf(metricCacheMissRow{})},
}

// sectionNames lists every projectable section, sorted, for the error a bad declaration gets.
func sectionNames() []string {
	names := make([]string, 0, len(sections))
	for name := range sections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// columnIndex caches each wire row type's JSON key to field index, since a projection reads it
// once per row and the reflection is the same answer every time.
var columnIndex sync.Map

// columns maps a wire row type's JSON keys onto its field indexes. Every wire row type is a
// flat struct of tagged exported fields; an untagged or embedded field would be invisible to a
// projection, which is why one has never been added to these types.
func columns(t reflect.Type) map[string]int {
	if cached, ok := columnIndex.Load(t); ok {
		return cached.(map[string]int)
	}
	out := make(map[string]int, t.NumField())
	for i := range t.NumField() {
		key, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		if key != "" && key != "-" {
			out[key] = i
		}
	}
	columnIndex.Store(t, out)
	return out
}

// columnNames lists a section's columns, sorted, so the document a projection produces has a
// stable key order and an error message names the alternatives in a readable one.
func columnNames(t reflect.Type) []string {
	cols := columns(t)
	names := make([]string, 0, len(cols))
	for name := range cols {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// splitPredicate addresses one column as `<section>.<column>`. The split is on the last dot
// because a section name may itself contain one (trace.steps), and a reader who split on the
// first would address a section that does not exist.
func splitPredicate(key string) (sectionName, column string, ok bool) {
	i := strings.LastIndex(key, ".")
	if i <= 0 || i == len(key)-1 {
		return "", "", false
	}
	return key[:i], key[i+1:], true
}
