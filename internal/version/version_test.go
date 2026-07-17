package version

import "testing"

func TestVersionHasDefault(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must never be empty")
	}
}
