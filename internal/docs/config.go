package docs

import (
	"fmt"
	"reflect"
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
				walk(f.Type.Elem(), reflect.Value{}, key+"[].", false)
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
