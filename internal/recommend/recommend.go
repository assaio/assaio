// Package recommend is assaio's contract for advice. A verdict says what is; a recommendation
// says what to do, and the two need different evidence. Every field here exists because prose
// could not carry it: a string takeaway can explain a result, but it cannot state what the
// action requires, what it risks, how to undo it, when to look again, or which figure would
// show whether it worked -- so nothing could ever be checked afterwards (ADR 0015).
//
// Rendered text projects a Record. It never adds a claim the Record does not hold, which is
// the property that makes the advice auditable rather than persuasive.
package recommend

import "slices"

// Status is where a recommendation stands. The vocabulary is closed and complete from the
// start even though only Proposed is reachable today: the lifecycle is what makes advice
// verifiable, and a status added later would leave every earlier record unable to say which
// of these it had been.
type Status string

const (
	// StatusProposed is a recommendation assaio derived and nobody has decided on.
	StatusProposed Status = "proposed"
	// StatusAccepted is one a person decided to run.
	StatusAccepted Status = "accepted"
	// StatusRejected is one a person decided against; the reason belongs with the decision.
	StatusRejected Status = "rejected"
	// StatusRunning is accepted and inside its review window.
	StatusRunning Status = "running"
	// StatusVerified is one whose follow-up measurement moved as expected.
	StatusVerified Status = "verified"
	// StatusInconclusive is one whose review window closed without a readable answer. It is a
	// real outcome, not a failure to record one: most interventions on a small corpus end here,
	// and a lifecycle that could not say so would only ever report successes.
	StatusInconclusive Status = "inconclusive"
)

var statuses = []Status{
	StatusProposed, StatusAccepted, StatusRejected,
	StatusRunning, StatusVerified, StatusInconclusive,
}

// ValidStatus reports whether s is one this build understands.
func ValidStatus(s Status) bool { return slices.Contains(statuses, s) }

// Record is one proposed experiment, complete enough to be judged before it is run and
// checked after.
type Record struct {
	// ID is stable for the same family and scope, so the same advice on a later window is
	// recognizably the same advice rather than a new one.
	ID string `json:"id"`
	// Family names the rule that produced this. Precision is measured per family, and a family
	// that keeps being rejected can be turned off as a unit.
	Family string `json:"family"`
	// Title is the one-line statement of what to do.
	Title string `json:"title"`
	// Evidence is what triggered this, in figures rather than adjectives. Empty is invalid:
	// advice with no evidence is an opinion.
	Evidence []string `json:"evidence"`
	// Confidence reuses analyze's vocabulary (high|medium|low|insufficient) so a reader is not
	// asked to hold two scales.
	Confidence string `json:"confidence"`
	// Scope is what the action applies to: the window, a project, a model, a tool.
	Scope string `json:"scope"`
	// Action is the single thing to do. One, not a list: an action nobody can point at is one
	// nobody can verify afterwards.
	Action string `json:"action"`
	// Prerequisites are what must be true before the action makes sense.
	Prerequisites []string `json:"prerequisites,omitempty"`
	// Effect is the expected direction, in words. Deliberately never a number: assaio has no
	// counterfactual, and a predicted percentage would be the fabricated figure this project
	// refuses.
	Effect string `json:"effect"`
	// Risks are what could go wrong, including the case where the action is simply wrong here.
	Risks []string `json:"risks,omitempty"`
	// Rollback is how to undo it. Required: an action nobody can undo is not an experiment.
	Rollback string `json:"rollback"`
	// ReviewWindow is when to look again, e.g. "14d".
	ReviewWindow string `json:"reviewWindow"`
	// FollowUp is the figure that would show whether it worked, named so the check is decided
	// before the result is known rather than after.
	FollowUp string `json:"followUp"`
	// Status is where this stands. Proposed until a lifecycle exists to move it.
	Status Status `json:"status"`
}

// Validate rejects a Record the contract cannot vouch for. It is called where records are
// produced, so a family cannot ship advice that is missing the parts that make it checkable.
func (r *Record) Validate() error {
	for _, missing := range []struct {
		empty bool
		field string
	}{
		{r.ID == "", "id"},
		{r.Family == "", "family"},
		{r.Title == "", "title"},
		{len(r.Evidence) == 0, "evidence"},
		{r.Action == "", "action"},
		{r.Effect == "", "effect"},
		{r.Rollback == "", "rollback"},
		{r.ReviewWindow == "", "reviewWindow"},
		{r.FollowUp == "", "followUp"},
	} {
		if missing.empty {
			return &IncompleteError{Family: r.Family, Field: missing.field}
		}
	}
	if !ValidStatus(r.Status) {
		return &IncompleteError{Family: r.Family, Field: "status"}
	}
	return nil
}

// IncompleteError names the part of the contract a record failed to carry.
type IncompleteError struct{ Family, Field string }

func (e *IncompleteError) Error() string {
	return "recommendation from " + e.Family + " is missing " + e.Field +
		", which is what would make it checkable afterwards"
}
