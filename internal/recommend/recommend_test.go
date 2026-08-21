package recommend

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/analyze"
)

func completeRecord() Record {
	return Record{
		ID: "f/window", Family: "f", Title: "do the thing",
		Evidence: []string{"a figure"}, Action: "do it", Effect: "directionally better",
		Rollback: "undo it", ReviewWindow: "14d", FollowUp: "the figure", Status: StatusProposed,
	}
}

// TestValidateRejectsAdviceThatCannotBeChecked is the contract's whole point: every field it
// insists on is one prose could not carry, and without which nothing could be verified later.
func TestValidateRejectsAdviceThatCannotBeChecked(t *testing.T) {
	for _, tc := range []struct {
		field  string
		breaks func(*Record)
	}{
		{field: "id", breaks: func(r *Record) { r.ID = "" }},
		{field: "title", breaks: func(r *Record) { r.Title = "" }},
		{field: "evidence", breaks: func(r *Record) { r.Evidence = nil }},
		{field: "action", breaks: func(r *Record) { r.Action = "" }},
		{field: "effect", breaks: func(r *Record) { r.Effect = "" }},
		{field: "rollback", breaks: func(r *Record) { r.Rollback = "" }},
		{field: "reviewWindow", breaks: func(r *Record) { r.ReviewWindow = "" }},
		{field: "followUp", breaks: func(r *Record) { r.FollowUp = "" }},
		{field: "status", breaks: func(r *Record) { r.Status = "sounds-good" }},
	} {
		t.Run(tc.field, func(t *testing.T) {
			r := completeRecord()
			tc.breaks(&r)
			err := r.Validate()
			if err == nil {
				t.Fatalf("a record missing %s validated", tc.field)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("error %q does not name the missing %s", err, tc.field)
			}
		})
	}
	complete := completeRecord()
	if err := complete.Validate(); err != nil {
		t.Fatalf("a complete record failed to validate: %v", err)
	}
}

// TestFromDropsAnIncompleteRecord holds the boundary: a family that gets the contract wrong
// ships nothing, rather than shipping advice with a warning attached.
func TestFromDropsAnIncompleteRecord(t *testing.T) {
	Register(brokenFamily{})
	t.Cleanup(func() { registry = registry[:len(registry)-1] })

	for _, r := range From(&Evidence{}) {
		if r.Family == "broken" {
			t.Fatalf("an incomplete record reached the output: %+v", r)
		}
	}
}

type brokenFamily struct{}

func (brokenFamily) Name() string { return "broken" }
func (brokenFamily) Propose(*Evidence) []Record {
	return []Record{{ID: "broken/x", Title: "no evidence, no rollback"}}
}

// TestAbstentionSaysItIsAnAbstention is the honesty rule for the empty case: "no
// recommendations" reads as a clean bill of health and is not one.
func TestAbstentionSaysItIsAnAbstention(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderText(&buf, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "abstention, not a clean bill") {
		t.Fatalf("empty render = %q, want it to say what an empty list means", out)
	}
}

// TestEnoughAbstainsOnWeakEvidence keeps the default: a recommendation magnifies whatever
// error is under it, so one built on a verdict assaio itself calls low-confidence is a precise
// action resting on evidence its own envelope calls thin.
func TestEnoughAbstainsOnWeakEvidence(t *testing.T) {
	for label, want := range map[string]bool{
		analyze.ConfidenceHigh:         true,
		analyze.ConfidenceMedium:       true,
		analyze.ConfidenceLow:          false,
		analyze.ConfidenceInsufficient: false,
	} {
		r := &analyze.Result{}
		r.Confidence.Label = label
		if got := enough(r); got != want {
			t.Errorf("enough(%s) = %v, want %v", label, got, want)
		}
	}
	withheld := &analyze.Result{Withheld: []analyze.Capability{analyze.CapTrace}}
	withheld.Confidence.Label = analyze.ConfidenceHigh
	if enough(withheld) {
		t.Fatal("a verdict missing a declared input was treated as enough to advise on")
	}
	if enough(nil) {
		t.Fatal("a missing verdict was treated as enough to advise on")
	}
}

// TestRenderProjectsOnlyTheRecord is the property that makes the advice auditable: every line
// of prose comes from a field, so a renderer cannot grow a claim the contract never carried.
func TestRenderProjectsOnlyTheRecord(t *testing.T) {
	r := completeRecord()
	var buf bytes.Buffer
	if err := RenderText(&buf, []Record{r}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{r.Title, r.Evidence[0], r.Action, r.Effect, r.Rollback, r.ReviewWindow, r.FollowUp} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("render dropped %q:\n%s", want, buf.String())
		}
	}
}
