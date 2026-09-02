package usage

import "testing"

// The kind vocabulary is closed at the store boundary, which rejects anything else rather than
// storing a value no validator can interpret. The rejected rows are the shapes a drifting caller
// produces: the raw tool name the classification was derived from, and a bucket nobody defined.
func TestValidStepKind(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want bool
	}{
		{"assistant", StepAssistant, true},
		{"read", StepRead, true},
		{"search", StepSearch, true},
		{"command", StepCommand, true},
		{"edit", StepEdit, true},
		{"other", StepOther, true},
		{"compaction", StepCompaction, true},
		{"unset", "", false},
		{"a tool name rather than its classification", "Bash", false},
		{"title case", "Read", false},
		{"a kind nobody defined", "thinking", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidStepKind(tt.kind); got != tt.want {
				t.Errorf("ValidStepKind(%q) = %v, want %v", tt.kind, got, tt.want)
			}
		})
	}
}

// "" is accepted and is not a sixth value: it means the source said nothing about how the step
// ended, which is a different fact from OutcomeOK and must never be read as one. A reader keeping
// the two apart depends on both crossing the boundary.
func TestValidStepOutcome(t *testing.T) {
	tests := []struct {
		name    string
		outcome string
		want    bool
	}{
		{"the source said nothing", "", true},
		{"ok", OutcomeOK, true},
		{"error", OutcomeError, true},
		{"denied", OutcomeDenied, true},
		{"truncated", OutcomeTruncated, true},
		{"a vocabulary member no log fills", "aborted", false},
		{"title case", "OK", false},
		{"a near miss", "errored", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidStepOutcome(tt.outcome); got != tt.want {
				t.Errorf("ValidStepOutcome(%q) = %v, want %v", tt.outcome, got, tt.want)
			}
		})
	}
}

// The two vocabularies are disjoint. They are stored in different columns and read by different
// code, so a value valid in both would let a miswired writer put an outcome where a kind belongs
// and pass every gate on the way in.
func TestTheKindAndOutcomeVocabulariesDoNotOverlap(t *testing.T) {
	kinds := []string{StepAssistant, StepRead, StepSearch, StepCommand, StepEdit, StepOther, StepCompaction}
	for _, k := range kinds {
		if ValidStepOutcome(k) {
			t.Errorf("%q is both a step kind and a step outcome", k)
		}
	}
	for _, o := range []string{OutcomeOK, OutcomeError, OutcomeDenied, OutcomeTruncated} {
		if ValidStepKind(o) {
			t.Errorf("%q is both a step outcome and a step kind", o)
		}
	}
}
