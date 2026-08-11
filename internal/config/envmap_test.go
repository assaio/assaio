package config

import "testing"

// EnvVar is what the generated reference publishes as "set this variable to change this
// setting", and envKeyResolver is what actually routes a variable to a key. A reference that
// named a variable koanf does not resolve would be a documented setting nobody can set.
func TestEveryKeyRoundTripsThroughItsEnvVar(t *testing.T) {
	resolve := envKeyResolver()
	for _, key := range envKeyMap() {
		if got := resolve(EnvVar(key)); got != key {
			t.Errorf("EnvVar(%q) = %q, which resolves back to %q", key, EnvVar(key), got)
		}
	}
}
