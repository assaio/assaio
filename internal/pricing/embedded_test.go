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
	if _, ok := tbl["claude-opus-4-5"]; !ok {
		t.Fatal("expected claude-opus-4-5 to be priced")
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
