package event

import "fmt"

var (
	reviewStates      = []string{"pending", "commented", "approved", "changes_requested", "dismissed"}
	checkRunStatuses  = []string{"queued", "in_progress", "completed", "waiting", "requested", "pending"}
	checkConclusions  = []string{"action_required", "timed_out", "cancelled", "failure", "success", "neutral", "skipped", "startup_failure", "stale"}
	checkRollupStates = []string{"error", "failure", "pending", "success", "expected"}
)

// Review records one forge review without the reviewer or review content.
type Review struct {
	Number int64  `json:"number"`
	State  string `json:"state"`
}

// Check records one head-check context. Kind separates GitHub check runs from commit statuses;
// neither carries its name, workflow, actor, URL or output.
type Check struct {
	Number     int64  `json:"number"`
	Kind       string `json:"kind"`
	State      string `json:"state"`
	Conclusion string `json:"conclusion,omitempty"`
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
	return nil
}

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
	return nil
}
