package docs

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/assaio/assaio/internal/config"
)

// ConfigKey is one setting as a reader has to address it: the dotted YAML key, the environment
// variable that overrides it, and what applies when neither is set.
type ConfigKey struct {
	Key string `json:"key"`
	// Env is empty for a key with no working environment override. Publishing one anyway is
	// worse than publishing none: `ASSAIO_PLUGINS=demo` makes koanf's decode fail and every
	// command exits, and a bracketed name like ASSAIO_PLUGINS[]_NAME is not a variable name a
	// shell can export. Both were published before this field could be empty.
	Env  string `json:"env,omitempty"`
	Type string `json:"type"`
	// Default is the built-in value, rendered as it would be written. Empty means the zero
	// value applies -- unless Optional, where no value was set at all and what that means is
	// the key's own business rather than its Go type's.
	Default  string `json:"default"`
	Optional bool   `json:"optional,omitempty"`
	// List marks a key whose value is a sequence of entries; its own fields follow as
	// "<key>[].<field>" so a reader can address them.
	List bool `json:"list,omitempty"`
}

// configKeys walks the Config struct's koanf tags -- the same tags koanf itself binds -- so a
// setting that exists is documented by existing, and a documented one that was removed cannot
// survive its field. Defaults come from config.Defaults rather than a loaded config: reading a
// live one would fold whatever ASSAIO_ variables happen to be set into a committed file.
func configKeys() []ConfigKey {
	def := reflect.ValueOf(config.Defaults())
	var out []ConfigKey

	var walk func(t reflect.Type, v reflect.Value, prefix string, settable bool)
	walk = func(t reflect.Type, v reflect.Value, prefix string, settable bool) {
		for i := range t.NumField() {
			f := t.Field(i)
			tag := f.Tag.Get("koanf")
			if tag == "" || tag == "-" {
				continue
			}
			key := prefix + tag
			fv := fieldValue(v, i)

			switch {
			case f.Type.Kind() == reflect.Struct:
				walk(f.Type, fv, key+".", settable)
			case f.Type.Kind() == reflect.Slice && f.Type.Elem().Kind() == reflect.Struct:
				// A list of entries has no environment form -- one variable holds one string,
				// and the decode into a struct fails -- and neither does anything under it.
				out = append(out, ConfigKey{Key: key, Type: typeName(f.Type), List: true})
				before := len(out)
				walk(f.Type.Elem(), reflect.Value{}, key+"[].", false)
				out = out[:before+keptUnder(key, out[before:])]
			default:
				out = append(out, ConfigKey{
					Key: key, Env: envVar(key, settable),
					Type: typeName(f.Type), Default: render(fv),
					Optional: f.Type.Kind() == reflect.Pointer,
				})
			}
		}
	}
	walk(reflect.TypeOf(config.Config{}), def, "", true)

	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func envVar(key string, settable bool) string {
	if !settable {
		return ""
	}
	return config.EnvVar(key)
}

// fieldValue returns the default-config value behind a field, or an invalid Value when the
// walk has descended into a list element type, which no default can have an instance of.
func fieldValue(v reflect.Value, i int) reflect.Value {
	if !v.IsValid() {
		return reflect.Value{}
	}
	return v.Field(i)
}

func render(v reflect.Value) string {
	if !v.IsValid() || v.IsZero() {
		return ""
	}
	if v.Kind() == reflect.Pointer {
		return fmt.Sprint(v.Elem().Interface())
	}
	return fmt.Sprint(v.Interface())
}

// typeName renders a Go type the way a config file's author reads it, not the way Go prints
// it: a reader writing YAML needs "list of strings", not "[]string".
func typeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return typeName(t.Elem())
	case reflect.Slice:
		return plural(typeName(t.Elem()))
	case reflect.Struct:
		return "entry"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Float32, reflect.Float64:
		return "number"
	default:
		return t.Kind().String()
	}
}

func plural(s string) string {
	if strings.HasSuffix(s, "y") {
		return strings.TrimSuffix(s, "y") + "ies"
	}
	return s + "s"
}

// rejectedUnder names the keys one list shares with another by Go type but rejects at its own
// boundary. `plugins:`, `metrics:` and `rules:` decode into the same struct, so reflection sees
// `needs` under all three while PluginConfig.Validate refuses it for two -- and a reference that
// publishes a key the binary rejects is worse than one that omits it, because the reader has no
// way to tell which half is wrong.
var rejectedUnder = map[string][]string{
	"plugins": {"needs"},
	"rules":   {"needs"},
}

// keptUnder returns how many of a list's fields survive, having moved the rejected ones to the
// end. Order inside the list is not meaningful -- configKeys sorts the whole set afterwards.
func keptUnder(list string, fields []ConfigKey) int {
	rejected := rejectedUnder[list]
	if len(rejected) == 0 {
		return len(fields)
	}
	kept := 0
	for i := range fields {
		if slices.Contains(rejected, strings.TrimPrefix(fields[i].Key, list+"[].")) {
			continue
		}
		fields[kept], fields[i] = fields[i], fields[kept]
		kept++
	}
	return kept
}
