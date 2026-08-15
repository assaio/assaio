package signal

import (
	"testing"

	"github.com/assaio/assaio/internal/layer"
)

// TestEverySignalStatesItsLayer holds ROADMAP.md's "a metric states which one it is" on the
// catalog side. A struct field is a field an entry will forget -- writing this list, one of
// the twenty-five did -- so the register is checked rather than trusted.
func TestEverySignalStatesItsLayer(t *testing.T) {
	for _, s := range Catalog() {
		if !layer.Valid(s.Layer) {
			t.Errorf("signal %s declares layer %q, which is not one of activity|output|outcome|impact", s.ID, s.Layer)
		}
	}
	if len(Catalog()) == 0 {
		t.Fatal("the catalog is empty, so the assertion above never ran")
	}
}

// TestStepOutcomeIsAnActivitySignal is the mistake the layer vocabulary exists to make
// impossible to reach by accident: ai.step.outcome names how one call ended inside a session,
// which is what happened, not whether the code held. An ID is not a layer.
func TestStepOutcomeIsAnActivitySignal(t *testing.T) {
	for _, s := range Catalog() {
		if s.ID == "ai.step.outcome" && s.Layer != layer.Activity {
			t.Fatalf("ai.step.outcome declares %q; a step's ok/error is activity, and calling it outcome is the relabeling ROADMAP.md forbids", s.Layer)
		}
	}
}
