package layer

import "testing"

// Valid is the gate analyze.Register panics on, so what it lets through is the whole
// vocabulary any surface can ever state. The rejected rows are the shapes a drifting
// caller actually produces: a layer left unset, a constant spelled for a heading rather
// than for a switch, and a plausible fifth word nobody defined.
func TestValid(t *testing.T) {
	tests := []struct {
		name string
		l    Layer
		want bool
	}{
		{"activity", Activity, true},
		{"output", Output, true},
		{"outcome", Outcome, true},
		{"impact", Impact, true},
		{"unset", Layer(""), false},
		{"title case", Layer("Activity"), false},
		{"upper case", Layer("OUTPUT"), false},
		{"padded", Layer(" outcome"), false},
		{"pluralized", Layer("outcomes"), false},
		{"a plausible fifth layer", Layer("velocity"), false},
		{"a layer named after the thing it must not claim", Layer("productivity"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Valid(tt.l); got != tt.want {
				t.Errorf("Valid(%q) = %v, want %v", tt.l, got, tt.want)
			}
		})
	}
}

// All is the second statement of the same closed set, and the order is part of its contract:
// a surface that ranks or compares layers reads strength off this slice's index. A constant
// added to one of the two and not the other is the drift ADR 0013 exists to prevent.
func TestAllIsTheVocabularyValidAccepts(t *testing.T) {
	all := All()
	want := []Layer{Activity, Output, Outcome, Impact}
	if len(all) != len(want) {
		t.Fatalf("All() holds %d layers, want %d: %v", len(all), len(want), all)
	}
	seen := make(map[Layer]bool, len(all))
	for i, l := range all {
		if l != want[i] {
			t.Errorf("All()[%d] = %q, want %q -- the order runs weakest claim first", i, l, want[i])
		}
		if !Valid(l) {
			t.Errorf("All() offers %q, which Valid rejects", l)
		}
		if seen[l] {
			t.Errorf("All() lists %q twice", l)
		}
		seen[l] = true
	}
}

// A caller holding the vocabulary must not be able to edit it for everyone else.
func TestAllHandsBackACopy(t *testing.T) {
	All()[0] = "velocity"
	if got := All()[0]; got != Activity {
		t.Fatalf("All()[0] = %q after a caller wrote to a previous result, want %q", got, Activity)
	}
}
