package pricing

import (
	"reflect"
	"testing"
)

func TestLoadEmbedded(t *testing.T) {
	tbl, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl) == 0 {
		t.Fatal("expected non-empty table")
	}
	// claude-opus-4-1 is one of the keys LiteLLM dropped past its deprecation date.
	for _, model := range []string{"claude-opus-4-5", "claude-opus-4-1"} {
		if _, ok := tbl[model]; !ok {
			t.Fatalf("expected %s to be priced", model)
		}
	}
}

// TestLoadCachesTable proves Load parses the embedded file at most once per process: two
// calls must return the very same underlying map (identity, not just equal content),
// which is only possible if the second call skipped the 1.5MB parse.
func TestLoadCachesTable(t *testing.T) {
	first, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if reflect.ValueOf(first).Pointer() != reflect.ValueOf(second).Pointer() {
		t.Fatal("Load() returned a different Table on the second call: the parse was not cached")
	}
}
