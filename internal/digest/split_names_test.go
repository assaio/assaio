package digest

import (
	"strings"
	"testing"
)

// TestANameThatSplitIsDeclaredWithoutNamingIt: a second repository with a stored name splits it
// between two runs of the same build, so the movement is partly a renaming. The caveat counts the
// names instead of printing them, because caveats bypass the project pseudonym.
func TestANameThatSplitIsDeclaredWithoutNamingIt(t *testing.T) {
	tests := []struct {
		name      string
		now, prev map[string]string
		want      string
	}{
		{"no name split in either run", nil, nil, ""},
		{"the same repositories behind the same name", map[string]string{"api": "1,2"}, map[string]string{"api": "1,2"}, ""},
		{"an older snapshot recorded none and nothing is split", map[string]string{}, nil, ""},
		{"a name split since the last run", map[string]string{"api": "1,2"}, nil, "1 project name(s)"},
		{"one name split and another rejoined", map[string]string{"api": "1,2"}, map[string]string{"web": "3,4"}, "2 project name(s)"},
		{"still split, but its first repository changed", map[string]string{"api": "2,3"}, map[string]string{"api": "1,2,3"}, "1 project name(s)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := Snapshot{Window: "7d", TakenAt: at(11), ParsedBy: "same", Priced: true, Splits: tt.now}
			prev := Snapshot{Window: "7d", TakenAt: at(4), ParsedBy: "same", Priced: true, Splits: tt.prev}
			d := Compare(&now, &prev, nil)
			if tt.want == "" {
				if len(d.Caveats) != 0 {
					t.Fatalf("caveats = %v, want none", d.Caveats)
				}
				return
			}
			if !hasCaveat(&d, tt.want) {
				t.Fatalf("caveats = %v, want one counting %q", d.Caveats, tt.want)
			}
			for _, c := range d.Caveats {
				if strings.Contains(c, "api") || strings.Contains(c, "web") {
					t.Fatalf("caveat names a project: %q", c)
				}
			}
		})
	}
}
