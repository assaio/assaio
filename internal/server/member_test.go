package server

import (
	"strings"
	"testing"
)

func TestValidateMember(t *testing.T) {
	tests := []struct {
		name    string
		member  string
		wantErr bool
	}{
		{"simple name", "alice", false},
		{"pseudonym label", "member-a1b2", false},
		{"dots underscores hyphens", "alice.smith_2-b", false},
		{"max length accepted", strings.Repeat("a", maxMemberLen), false},
		{"empty rejected", "", true},
		{"colon rejected", "team:alice", true},
		{"colon suffix rejected", "alice:", true},
		{"slash rejected", "team/alice", true},
		{"space rejected", "team alice", true},
		{"at sign rejected", "alice@example.com", true},
		{"newline rejected", "alice\nbob", true},
		{"too long rejected", strings.Repeat("a", maxMemberLen+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMember(tt.member)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateMember(%q) err = %v, wantErr %v", tt.member, err, tt.wantErr)
			}
		})
	}
}

// TestValidateMemberPreventsReportedCollision is the exact pair from the bug report:
// member "team:alice" key "x" and member "team" key "alice:x" both compose to the
// store's key "team:alice:x" under naive "<member>:<dedupe_key>" concatenation. The fix
// must reject the first so that collision can never happen.
func TestValidateMemberPreventsReportedCollision(t *testing.T) {
	if err := ValidateMember("team:alice"); err == nil {
		t.Fatal(`ValidateMember("team:alice") = nil, want an error (would collide with member "team" key "alice:x")`)
	}
	if err := ValidateMember("team"); err != nil {
		t.Fatalf(`ValidateMember("team") = %v, want nil`, err)
	}
}
