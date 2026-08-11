package docs

import (
	"reflect"

	"github.com/assaio/assaio/internal/analyze"
)

// Field is one field of the in-tree metric contract, as a Go author writes it.
type Field struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// validatorInput is what analyze.Input hands every built-in validator. It is published beside
// the plugin wire because the two are the same promise made twice, and B155 is what their
// drifting cost: five shipped validators read six fields no plugin could see.
func validatorInput() []Field {
	t := reflect.TypeOf(analyze.Input{})
	out := make([]Field, 0, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		out = append(out, Field{Name: f.Name, Type: f.Type.String()})
	}
	return out
}
