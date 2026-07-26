package parser

import "testing"

func TestToolCountsAddClassifiesByAllowlist(t *testing.T) {
	cases := []struct {
		name  string
		names []string
		want  ToolCounts
	}{
		{"reads", []string{"Read", "NotebookRead", "WebFetch"}, ToolCounts{Reads: 3}},
		{"searches", []string{"Grep", "Glob", "WebSearch", "ToolSearch"}, ToolCounts{Searches: 4}},
		{"commands", []string{"Bash", "BashOutput", "KillShell"}, ToolCounts{Commands: 3}},
		{"writes", []string{"Edit", "Write", "MultiEdit", "NotebookEdit"}, ToolCounts{Writes: 4}},
		// StructuredOutput returns a payload to the caller and writes no file: counting it as
		// production inflated the produce share of any turn that answered structurally.
		{"structured output is not production", []string{"StructuredOutput"}, ToolCounts{Other: 1}},
		{"unlisted names are other", []string{"Task", "Agent", "TodoWrite", "mcp__srv__tool", "NotATool", ""}, ToolCounts{Other: 6}},
		{"classification is case-sensitive", []string{"READ", "BASH", "Exec"}, ToolCounts{Other: 3}},
		{"codex names", []string{"read_file", "web_search", "exec", "apply_patch", "wait"}, ToolCounts{Reads: 1, Searches: 1, Commands: 1, Writes: 1, Other: 1}},
		{"mixed", []string{"Read", "Grep", "Bash", "Edit", "mcp__srv__tool"}, ToolCounts{Reads: 1, Searches: 1, Commands: 1, Writes: 1, Other: 1}},
		{"nothing", nil, ToolCounts{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got ToolCounts
			for _, n := range c.names {
				got.Add(n)
			}
			if got != c.want {
				t.Fatalf("counts = %+v, want %+v", got, c.want)
			}
			if got.Total() != int64(len(c.names)) {
				t.Fatalf("Total() = %d, want %d (every call lands in exactly one bucket)", got.Total(), len(c.names))
			}
		})
	}
}
