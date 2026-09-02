package plugin

// The boundary check on what a plugin says it reads. A declaration is refused whole rather than
// repaired: a column assaio silently dropped from a projection is a column the plugin reads as
// absent, and absent is the one thing this protocol promises is never a zero.

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/assaio/assaio/internal/analyze"
)

// validateDeclaration returns every reason this declaration cannot be honoured.
func validateDeclaration(d Declaration) []string {
	var vs []string
	if len(d.Needs) == 0 {
		return []string{"needs is required and must name at least one capability (one of " +
			strings.Join(capabilityNames(), ", ") + ")"}
	}
	seen := map[analyze.Capability]bool{}
	for _, c := range d.Needs {
		switch {
		case !analyze.ValidCapability(c):
			vs = append(vs, fmt.Sprintf("needs %q is not one of %s", c, strings.Join(capabilityNames(), ", ")))
		case seen[c]:
			vs = append(vs, fmt.Sprintf("needs names %q twice", c))
		}
		seen[c] = true
	}
	for key, keep := range d.Fields {
		vs = validateFields(vs, key, keep, seen)
	}
	for key, allowed := range d.Where {
		vs = validatePredicate(vs, key, allowed, seen)
	}
	return vs
}

func validateFields(vs []string, key string, keep []string, declared map[analyze.Capability]bool) []string {
	s, ok := sections[key]
	if !ok {
		return append(vs, fmt.Sprintf("fields names section %q, which is not one of %s",
			key, strings.Join(sectionNames(), ", ")))
	}
	if !declared[s.capability] {
		return append(vs, fmt.Sprintf("fields projects section %q, whose capability %q is not in needs",
			key, s.capability))
	}
	if len(keep) == 0 {
		return append(vs, fmt.Sprintf("fields[%q] is empty; omit the section to receive it whole", key))
	}
	cols := columns(s.row)
	for _, column := range keep {
		if _, ok := cols[column]; !ok {
			vs = append(vs, fmt.Sprintf("fields[%q] names column %q, which is not one of %s",
				key, column, strings.Join(columnNames(s.row), ", ")))
		}
	}
	return vs
}

func validatePredicate(vs []string, key string, allowed []string, declared map[analyze.Capability]bool) []string {
	name, column, ok := splitPredicate(key)
	if !ok {
		return append(vs, fmt.Sprintf("where names %q; a predicate is addressed <section>.<column>", key))
	}
	s, known := sections[name]
	if !known {
		return append(vs, fmt.Sprintf("where names section %q, which is not one of %s",
			name, strings.Join(sectionNames(), ", ")))
	}
	// A predicate inside a nested section would drop rows from within one row -- steps out of a
	// sequence -- leaving that sequence's own ordinals and step count describing a set nobody
	// declared. The outer row is the smallest thing a predicate may remove.
	if strings.Contains(name, ".") {
		return append(vs, fmt.Sprintf("where names nested section %q; a predicate may only drop a top-level row", name))
	}
	if !declared[s.capability] {
		return append(vs, fmt.Sprintf("where filters section %q, whose capability %q is not in needs", name, s.capability))
	}
	index, exists := columns(s.row)[column]
	if !exists {
		return append(vs, fmt.Sprintf("where[%q] names column %q, which is not one of %s",
			key, column, strings.Join(columnNames(s.row), ", ")))
	}
	if s.row.Field(index).Type.Kind() != reflect.String {
		return append(vs, fmt.Sprintf("where[%q] filters on %q, which is not a string column", key, column))
	}
	if len(allowed) == 0 {
		return append(vs, fmt.Sprintf("where[%q] admits no value; omit the predicate to receive every row", key))
	}
	return vs
}
