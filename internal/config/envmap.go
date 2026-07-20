package config

import (
	"reflect"
	"strings"
)

// envKeyMap maps an ASSAIO_-stripped, lowercased env var name (e.g. "pricing_effective_per_token")
// to its dotted koanf key ("pricing.effective_per_token"). It is built from the Config struct's
// own koanf tags so a leaf key that itself contains underscores stays addressable from the
// environment -- a blind "_"->"." replacement would mis-split it into a phantom nested path.
func envKeyMap() map[string]string {
	m := map[string]string{}
	var walk func(t reflect.Type, prefix string)
	walk = func(t reflect.Type, prefix string) {
		for i := range t.NumField() {
			f := t.Field(i)
			tag := f.Tag.Get("koanf")
			if tag == "" || tag == "-" {
				continue
			}
			key := prefix + tag
			if f.Type.Kind() == reflect.Struct {
				walk(f.Type, key+".")
				continue
			}
			m[strings.ReplaceAll(key, ".", "_")] = key
		}
	}
	walk(reflect.TypeOf(Config{}), "")
	return m
}

// envKeyResolver returns the env-var-name transform koanf's env provider uses: an exact
// match against a known scalar key wins (so underscore-containing leaves resolve), and any
// unknown name falls back to the historical "_"->"." mapping unchanged.
func envKeyResolver() func(string) string {
	keys := envKeyMap()
	return func(s string) string {
		trimmed := strings.ToLower(strings.TrimPrefix(s, "ASSAIO_"))
		if key, ok := keys[trimmed]; ok {
			return key
		}
		return strings.ReplaceAll(trimmed, "_", ".")
	}
}
