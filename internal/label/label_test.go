package label

import "testing"

func TestValid(t *testing.T) {
	for _, tc := range []struct {
		name, axis, value string
		want              bool
	}{
		{"known task", Task, "refactor", true},
		{"known outcome", Outcome, "abandoned", true},
		{"known difficulty", Difficulty, "high", true},
		// "" is how an unset axis is stored, so it is valid everywhere.
		{"unset task", Task, "", true},
		{"unset outcome", Outcome, "", true},
		{"value from another axis", Task, "done", false},
		{"unknown value", Task, "chore", false},
		{"case is significant", Task, "Refactor", false},
		{"unknown axis", "mood", "great", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Valid(tc.axis, tc.value); got != tc.want {
				t.Fatalf("Valid(%q, %q) = %v want %v", tc.axis, tc.value, got, tc.want)
			}
		})
	}
}

func TestNamesCoverEveryAxis(t *testing.T) {
	if len(Names) != len(Axes) {
		t.Fatalf("Names has %d axes, Axes has %d -- a new axis must appear in both", len(Names), len(Axes))
	}
	for _, axis := range Names {
		if len(Axes[axis]) == 0 {
			t.Fatalf("axis %q in Names has no vocabulary", axis)
		}
	}
}

func TestValuesRendersVocabulary(t *testing.T) {
	if got, want := Values(Outcome), "done|partial|abandoned"; got != want {
		t.Fatalf("Values(Outcome) = %q want %q", got, want)
	}
	if got := Values("mood"); got != "" {
		t.Fatalf("Values(unknown axis) = %q want empty", got)
	}
}

// Unlabeled must never collide with a real value, or a report grouped by an axis would
// merge annotated usage into the unannotated bucket.
func TestUnlabeledIsNotAVocabularyValue(t *testing.T) {
	for axis, values := range Axes {
		for _, v := range values {
			if v == Unlabeled {
				t.Fatalf("axis %q contains %q, which is the unannotated group name", axis, Unlabeled)
			}
		}
	}
}
