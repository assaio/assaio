package event

import "testing"

func TestDeliveryPayloadsRejectUnknownOrContradictoryStates(t *testing.T) {
	for _, tt := range []struct {
		name string
		body Payload
		want string
	}{
		{"requested changes", Review{Number: 1, State: "changes_requested"}, ""},
		{"reviewer name is not a state", Review{Number: 1, State: "alice"}, "unknown review state"},
		{"completed failure", Check{Number: 1, Kind: "run", State: "completed", Conclusion: "failure"}, ""},
		{"pending without conclusion", Check{Number: 1, Kind: "run", State: "pending"}, ""},
		{"pending with conclusion", Check{Number: 1, Kind: "run", State: "pending", Conclusion: "success"}, "unfinished"},
		{"status context", Check{Number: 1, Kind: "status", State: "expected"}, ""},
		{"status with conclusion", Check{Number: 1, Kind: "status", State: "success", Conclusion: "success"}, "invalid commit status"},
		{"free kind", Check{Number: 1, Kind: "workflow-name", State: "success"}, "unknown check kind"},
	} {
		t.Run(tt.name, func(t *testing.T) { assertErr(t, tt.body.validate(), tt.want) })
	}
}
