package claude

import (
	"strings"
	"testing"
)

func TestParseAttributesSidechainSkillAndAgentPerLine(t *testing.T) {
	const log = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","isSidechain":true,"attributionSkill":"superpowers:brainstorming","attributionAgent":"Explore","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
{"type":"assistant","uuid":"a2","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":1,"output_tokens":1}}}
`
	recs, _, err := Parse(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(recs), recs)
	}
	if recs[0].Sidechain != 1 || recs[0].Skill != "superpowers:brainstorming" || recs[0].Agent != "Explore" {
		t.Fatalf("a1 Sidechain/Skill/Agent = %d/%q/%q, want 1/superpowers:brainstorming/Explore", recs[0].Sidechain, recs[0].Skill, recs[0].Agent)
	}
	if recs[1].Sidechain != 0 || recs[1].Skill != "" || recs[1].Agent != "" {
		t.Fatalf("a2 Sidechain/Skill/Agent = %d/%q/%q, want 0/\"\"/\"\" (these labels are per line and never carry forward)", recs[1].Sidechain, recs[1].Skill, recs[1].Agent)
	}
}
