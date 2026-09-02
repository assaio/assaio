package plugin

import (
	"reflect"
	"slices"
)

// The two row-level halves of a projection: a predicate drops rows the plugin declared it does
// not read, and a column list drops the fields inside the ones that survive. Both are the
// plugin's own declaration, so the denominator they produce is the plugin's to defend -- which
// is why rowCount below travels with the result and states what the predicate excluded.

// rowCount is what a plugin needs to defend a share computed over a filtered section: how many
// rows it received, and how many the window held before its own predicate ran. Without the
// second number a pushed-down predicate is a new way to publish "100% of rows are X" over a set
// the plugin chose, which is the fabricated denominator this envelope exists to refuse.
type rowCount struct {
	Sent      int `json:"sent"`
	Available int `json:"available"`
}

// filterRows keeps the rows every predicate on this section admits, and reports how many rows
// existed before any of them ran. A section with no predicate is returned untouched, so the
// common case allocates nothing.
func filterRows(name string, rows reflect.Value, where map[string][]string) (reflect.Value, rowCount) {
	available := rows.Len()
	type predicate struct {
		field   int
		allowed []string
	}
	var preds []predicate
	cols := columns(sections[name].row)
	for key, allowed := range where {
		owner, column, ok := splitPredicate(key)
		if !ok || owner != name {
			continue
		}
		preds = append(preds, predicate{field: cols[column], allowed: allowed})
	}
	if len(preds) == 0 {
		return rows, rowCount{Sent: available, Available: available}
	}
	kept := reflect.MakeSlice(rows.Type(), 0, available)
	for i := range available {
		row := rows.Index(i)
		admit := true
		for _, p := range preds {
			if !slices.Contains(p.allowed, row.Field(p.field).String()) {
				admit = false
				break
			}
		}
		if admit {
			kept = reflect.Append(kept, row)
		}
	}
	return kept, rowCount{Sent: kept.Len(), Available: available}
}

// projectRows renders a section's rows as documents carrying only its projected columns,
// recursing into a nested section (trace.steps) that has a projection of its own. A section
// nobody projected is returned as the typed slice it already is: the encoder writes it whole,
// which is both the fast path and the one that cannot drop a column by accident.
func projectRows(name string, rows reflect.Value, fields map[string][]string) any {
	if !projects(name, fields) {
		return rows.Interface()
	}
	keep := fields[name]
	if keep == nil {
		// Only a nested column list applies here, so this level keeps every column it has.
		keep = columnNames(sections[name].row)
	}
	cols := columns(sections[name].row)
	out := make([]map[string]any, rows.Len())
	for i := range out {
		row := rows.Index(i)
		doc := make(map[string]any, len(keep))
		for _, column := range keep {
			value := row.Field(cols[column])
			if nested := name + "." + column; projects(nested, fields) {
				doc[column] = projectRows(nested, value, fields)
				continue
			}
			doc[column] = value.Interface()
		}
		out[i] = doc
	}
	return out
}

// projects reports whether anything narrows this section: its own column list, or one on a
// section nested inside it.
func projects(name string, fields map[string][]string) bool {
	if _, ok := fields[name]; ok {
		return true
	}
	for key := range fields {
		if owner, _, ok := splitPredicate(key); ok && owner == name {
			return true
		}
	}
	return false
}
