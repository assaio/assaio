package pricing

import (
	"encoding/json"
	"flag"
	"os"
	"slices"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "fold litellm.json's unprefixed keys into retained.json")

// unprefixed reports whether name is a key a log can carry verbatim. LiteLLM's routing
// prefixes (vertex_ai/, azure/eu/) appear in no tool's log and NormalizeModel never adds one.
func unprefixed(name string) bool { return !strings.Contains(name, "/") }

// This is the guard on a price-table refresh. A refresh that skipped the -update fails here,
// and the -update only adds or reprices, so a key LiteLLM drops keeps its last price.
func TestRetainedHoldsEveryUnprefixedKey(t *testing.T) {
	snapshot := readTable(t, "litellm.json")
	retained := readTable(t, "retained.json")
	if *update {
		for name, p := range snapshot {
			if unprefixed(name) {
				retained[name] = p
			}
		}
		writeRetained(t, "retained.json", retained)
		var dropped []string
		for name := range retained {
			if _, ok := snapshot[name]; !ok {
				dropped = append(dropped, name)
			}
		}
		slices.Sort(dropped)
		t.Logf("%d keys are priced only by retained.json: %v", len(dropped), dropped)
	}
	var stale []string
	for name, p := range snapshot {
		if kept, ok := retained[name]; unprefixed(name) && (!ok || kept != p) {
			stale = append(stale, name)
		}
	}
	if len(stale) > 0 {
		slices.Sort(stale)
		t.Fatalf("retained.json lacks or misprices %d unprefixed keys of litellm.json, first %q; "+
			"run go test ./internal/pricing/ -run TestRetainedHoldsEveryUnprefixedKey -update",
			len(stale), stale[0])
	}
}

func TestFillFrom(t *testing.T) {
	old, cur := Price{Input: 1, Output: 2}, Price{Input: 3, Output: 4}
	kept := old
	kept.Retained = true
	tests := []struct {
		name      string
		snapshot  Table
		listed    []string
		retained  Table
		want      Table
		wantAdded int
	}{
		{"a dropped key is filled and marked", Table{"a": cur}, []string{"a"}, Table{"b": old}, Table{"a": cur, "b": kept}, 1},
		{"the snapshot wins on overlap", Table{"a": cur}, []string{"a"}, Table{"a": old}, Table{"a": cur}, 0},
		{"a key listed without a token rate stays unpriced", Table{}, []string{"b"}, Table{"b": old}, Table{}, 0},
		{"nothing retained", Table{"a": cur}, []string{"a"}, Table{}, Table{"a": cur}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listed := map[string]bool{}
			for _, name := range tt.listed {
				listed[name] = true
			}
			added := tt.snapshot.fillFrom(tt.retained, listed)
			if added != tt.wantAdded {
				t.Errorf("added = %d, want %d", added, tt.wantAdded)
			}
			if len(tt.snapshot) != len(tt.want) {
				t.Fatalf("table = %v, want %v", tt.snapshot, tt.want)
			}
			for name, p := range tt.want {
				if tt.snapshot[name] != p {
					t.Errorf("%s = %v, want %v", name, tt.snapshot[name], p)
				}
			}
		})
	}
}

func TestRetainedReportsOnlyLedgerPrices(t *testing.T) {
	tbl := Table{"listed": {Input: 1, Output: 1}}
	tbl.fillFrom(Table{"dropped": {Input: 2, Output: 2}}, map[string]bool{"listed": true})
	tests := []struct {
		model string
		want  bool
	}{
		{"listed", false},
		{"dropped", true},
		{"dropped-20250805", true},
		{"unknown", false},
	}
	for _, tt := range tests {
		if got := tbl.Retained(tt.model); got != tt.want {
			t.Errorf("Retained(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}

func readTable(t *testing.T, path string) Table {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // one of this package's two vendored tables
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	tbl, err := LoadReader(f)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return tbl
}

// writeRetained writes one sorted key per line, every rate explicit, so LoadReader reads back
// the identical Price and a refresh reviews as a line diff.
func writeRetained(t *testing.T, path string, tbl Table) {
	t.Helper()
	type entry struct {
		Input        float64 `json:"input_cost_per_token"`
		Output       float64 `json:"output_cost_per_token"`
		CacheWrite   float64 `json:"cache_creation_input_token_cost"`
		CacheWrite1h float64 `json:"cache_creation_input_token_cost_above_1hr"`
		CacheRead    float64 `json:"cache_read_input_token_cost"`
	}
	names := make([]string, 0, len(tbl))
	for name := range tbl {
		names = append(names, name)
	}
	slices.Sort(names)
	var b strings.Builder
	b.WriteString("{\n")
	for i, name := range names {
		p := tbl[name]
		key, err := json.Marshal(name)
		if err != nil {
			t.Fatal(err)
		}
		val, err := json.Marshal(entry{p.Input, p.Output, p.CacheWrite, p.CacheWrite1h, p.CacheRead})
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString("  " + string(key) + ": " + string(val))
		if i < len(names)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}
