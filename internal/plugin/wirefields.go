package plugin

import (
	"reflect"
	"sort"
	"strings"
	"time"
)

// WireField is one field of a metric plugin's protocol document, addressed the way its author
// writes it in JSON. It describes the document's *shape*; which fields the boundary demands, and
// which it discards, is enforced by validateMetricResult and stated in docs/extending. Deriving
// "required" from a struct tag would publish a different contract from the one the code applies:
// `name` carries no omitempty and is overwritten on arrival, `describe` may be absent.
type WireField struct {
	Path string `json:"path"`
	Type string `json:"type"`
	// Nullable marks a field encoded as null rather than a zero when it has no value -- the
	// unpriced-cost distinction the whole product rests on. A decoder generated from this
	// document must accept null there.
	Nullable bool `json:"nullable,omitempty"`
	// OmitEmpty marks a field the encoder drops when empty, so a reader knows its absence is
	// normal. It is a fact about encoding, never about obligation.
	OmitEmpty bool `json:"omitEmpty,omitempty"`
}

// MetricWire is the metric-plugin protocol as a list rather than a paragraph: what a plugin
// reads on stdin, and what it may return on stdout (ADR 0004).
type MetricWire struct {
	Input  []WireField `json:"input"`
	Result []WireField `json:"result"`
}

// MetricWireFields enumerates both documents. It lives here because both are shaped by
// unexported types no other package can reflect over.
func MetricWireFields() MetricWire {
	return MetricWire{
		Input:  wireFields(reflect.TypeOf(metricInput{})),
		Result: wireFields(reflect.TypeOf(wireResult{})),
	}
}

// maxWireDepth bounds the walk into nested element types. Three levels covers every shape the
// protocol has; the bound is here so a future self-referential type cannot hang the export.
const maxWireDepth = 3

func wireFields(t reflect.Type) []WireField {
	var out []WireField
	var walk func(t reflect.Type, prefix string, depth int)
	walk = func(t reflect.Type, prefix string, depth int) {
		if depth > maxWireDepth {
			return
		}
		for i := range t.NumField() {
			f := t.Field(i)
			name, optional, ok := jsonName(&f)
			if !ok {
				continue
			}
			// An embedded struct with no tag of its own is flattened by encoding/json, so its
			// fields belong to the document at this level rather than under a name.
			if name == "" {
				walk(f.Type, prefix, depth)
				continue
			}
			path := prefix + name
			out = append(out, WireField{
				Path: path, Type: jsonType(f.Type),
				Nullable: f.Type.Kind() == reflect.Pointer, OmitEmpty: optional,
			})
			if elem, ok := nested(f.Type); ok {
				walk(elem, path+suffix(f.Type), depth+1)
			}
		}
	}
	walk(t, "", 0)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// jsonName reports the key a field is encoded under, whether it may be omitted, and whether it
// is encoded at all. An embedded struct without a tag returns an empty name, which the caller
// reads as "flatten this".
func jsonName(f *reflect.StructField) (name string, optional, encoded bool) {
	if f.PkgPath != "" {
		return "", false, false // unexported: never encoded
	}
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}
	parts := strings.Split(tag, ",")
	optional = strings.Contains(tag, ",omitempty")
	if parts[0] != "" {
		return parts[0], optional, true
	}
	if f.Anonymous {
		return "", optional, true
	}
	return f.Name, optional, true
}

func nested(t reflect.Type) (reflect.Type, bool) {
	t = deref(t)
	if t == reflect.TypeOf(time.Time{}) {
		return nil, false
	}
	switch t.Kind() {
	case reflect.Struct:
		return t, true
	case reflect.Slice, reflect.Array, reflect.Map:
		elem := deref(t.Elem())
		if elem.Kind() == reflect.Struct && elem != reflect.TypeOf(time.Time{}) {
			return elem, true
		}
	}
	return nil, false
}

// suffix is what a reader appends to address a nested field. A map value is reached through a
// key the document does not fix, so it cannot wear the same "[]" an array element does: a reader
// who iterated `prices` as a list would get model names where objects were expected.
func suffix(t reflect.Type) string {
	switch deref(t).Kind() {
	case reflect.Slice, reflect.Array:
		return "[]."
	case reflect.Map:
		return "{}."
	default:
		return "."
	}
}

func jsonType(t reflect.Type) string {
	t = deref(t)
	if t == reflect.TypeOf(time.Time{}) {
		return "timestamp"
	}
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array of " + jsonType(t.Elem())
	case reflect.Map:
		return "object keyed by " + jsonType(t.Key()) + ", values " + jsonType(t.Elem())
	case reflect.Struct:
		return "object"
	case reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number"
	default:
		return t.Kind().String()
	}
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}
