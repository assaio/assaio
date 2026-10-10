package event

import (
	"errors"
	"fmt"
	"time"
)

var (
	reviewStates      = []string{"pending", "commented", "approved", "changes_requested", "dismissed"}
	checkRunStatuses  = []string{"queued", "in_progress", "completed", "waiting", "requested", "pending"}
	checkConclusions  = []string{"action_required", "timed_out", "cancelled", "failure", "success", "neutral", "skipped", "startup_failure", "stale"}
	checkRollupStates = []string{"error", "failure", "pending", "success", "expected"}
)

// Review records one forge review without the reviewer or review content.
type Review struct {
	Number      int64     `json:"number"`
	State       string    `json:"state"`
	Commit      string    `json:"commit,omitempty"`
	SubmittedAt time.Time `json:"submittedAt,omitzero"`
}

// Check records one head-check context. Kind separates GitHub check runs from commit statuses;
// neither carries its name, workflow, actor, URL or output.
type Check struct {
	Number      int64     `json:"number"`
	Kind        string    `json:"kind"`
	State       string    `json:"state"`
	Conclusion  string    `json:"conclusion,omitempty"`
	Commit      string    `json:"commit,omitempty"`
	Suite       string    `json:"suite,omitempty"`
	StartedAt   time.Time `json:"startedAt,omitzero"`
	CompletedAt time.Time `json:"completedAt,omitzero"`
}

func (Review) eventType() string { return TypeReview }
func (Check) eventType() string  { return TypeCheck }

func (p Review) validate() error {
	if p.Number <= 0 {
		return fmt.Errorf("pull request number %d is not positive", p.Number)
	}
	if !valid(reviewStates, p.State) {
		return fmt.Errorf("unknown review state %q", p.State)
	}
	if p.Commit != "" && !isObjectID(p.Commit) {
		return errors.New("reviewed commit is not a commit hash")
	}
	return nil
}

//nolint:gocritic // the Payload interface is satisfied by values; validated once per source observation.
func (p Check) validate() error {
	if p.Number <= 0 {
		return fmt.Errorf("pull request number %d is not positive", p.Number)
	}
	switch p.Kind {
	case "run":
		if !valid(checkRunStatuses, p.State) {
			return fmt.Errorf("unknown check-run state %q", p.State)
		}
		if p.Conclusion != "" && !valid(checkConclusions, p.Conclusion) {
			return fmt.Errorf("unknown check-run conclusion %q", p.Conclusion)
		}
		if p.State != "completed" && p.Conclusion != "" {
			return fmt.Errorf("unfinished check run carries conclusion %q", p.Conclusion)
		}
	case "status":
		if !valid(checkRollupStates, p.State) || p.Conclusion != "" {
			return fmt.Errorf("invalid commit status %q with conclusion %q", p.State, p.Conclusion)
		}
	default:
		return fmt.Errorf("unknown check kind %q", p.Kind)
	}
	if p.Commit != "" || p.Suite != "" {
		if p.Kind != "run" || !isObjectID(p.Commit) || !isForgeID(p.Suite) {
			return errors.New("historical check has no usable commit and suite")
		}
	}
	return nil
}
