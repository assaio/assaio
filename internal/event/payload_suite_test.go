package event

import "testing"

func TestSuiteAndHistoryCountsPreserveAbsenceAndRejectContradictions(t *testing.T) {
	for _, tt := range []struct {
		name string
		body Payload
		want string
	}{
		{"unavailable runs", CheckSuite{Number: 1, Commit: mergeCommit, State: "queued"}, ""},
		{"stated empty runs", CheckSuite{Number: 1, Commit: mergeCommit, State: "completed", Runs: &Population{}}, ""},
		{"bounded runs", CheckSuite{Number: 1, Commit: mergeCommit, State: "completed", Conclusion: "failure", Runs: &Population{Total: 7, Listed: 5}}, ""},
		{"negative total", CheckSuite{Number: 1, Commit: mergeCommit, State: "completed", Runs: &Population{Total: -1}}, "invalid connection"},
		{"more runs than total", CheckSuite{Number: 1, Commit: mergeCommit, State: "completed", Runs: &Population{Total: 1, Listed: 2}}, "invalid connection"},
		{"missing hash", CheckSuite{Number: 1, State: "completed"}, "not a commit hash"},
		{"unfinished conclusion", CheckSuite{Number: 1, Commit: mergeCommit, State: "queued", Conclusion: "success"}, "unfinished"},
		{"historical run", Check{Number: 1, Kind: "run", State: "completed", Commit: mergeCommit, Suite: "CS_1"}, ""},
		{"suite free text", Check{Number: 1, Kind: "run", State: "completed", Commit: mergeCommit, Suite: "build alice"}, "no usable commit"},
		{"suite without hash", Check{Number: 1, Kind: "run", State: "completed", Suite: "CS_1"}, "no usable commit"},
		{"bad review hash", Review{Number: 1, State: "changes_requested", Commit: "abc"}, "not a commit hash"},
	} {
		t.Run(tt.name, func(t *testing.T) { assertErr(t, tt.body.validate(), tt.want) })
	}
}
