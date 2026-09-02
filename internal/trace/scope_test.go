package trace

import "testing"

// The one rule that decides which population a sequence belongs to. The rows that matter for
// honesty are the last three: an entrypoint this build does not know, and one that only looks
// like the cli value, must land in Unstated rather than be promoted to a person's session.
func TestScope(t *testing.T) {
	tests := []struct {
		name       string
		timeline   string
		entrypoint string
		want       string
	}{
		{"a terminal session's main loop", "", entrypointCLI, Interactive},
		{"a sub-agent launched from a terminal session", "agent-1", entrypointCLI, SubAgent},
		{"a sub-agent launched from an SDK run", "agent-1", entrypointSDKPy, SubAgent},
		{"a sub-agent whose session states no entrypoint", "agent-1", "", SubAgent},
		{"the python SDK", "", entrypointSDKPy, Programmatic},
		{"the rust SDK's cli", "", entrypointSDKCLIRs, Programmatic},
		{"a source that records no entrypoint", "", "", Unstated},
		{"an entrypoint this build has never seen", "", "vscode", Unstated},
		{"an entrypoint that only looks like the cli one", "", "CLI", Unstated},
		{"an entrypoint with the cli value inside it", "", "cli-wrapper", Unstated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := seq("s1", tt.timeline, tt.entrypoint, origin, 1)
			if got := Scope(&seq); got != tt.want {
				t.Errorf("Scope() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Scope's answer is always one a detector can declare. A fifth string would name a population no
// view can be asked for, and its sequences would then be excluded from every figure while
// appearing in none -- lost rather than reported.
func TestScopeAnswersOnlyWithTheDeclarableFour(t *testing.T) {
	declarable := map[string]bool{Interactive: true, SubAgent: true, Programmatic: true, Unstated: true}
	entrypoints := []string{entrypointCLI, entrypointSDKPy, entrypointSDKCLIRs, "", "vscode", "CLI", "  "}

	for _, e := range entrypoints {
		for _, timeline := range []string{"", "agent-1"} {
			seq := seq("s1", timeline, e, origin, 1)
			if got := Scope(&seq); !declarable[got] {
				t.Errorf("Scope(entrypoint=%q, timeline=%q) = %q, which no view can be asked for",
					e, timeline, got)
			}
		}
	}
}

// The four are distinct strings; two sharing a value would silently merge two populations the
// partition assumes are disjoint.
func TestTheFourScopesAreDistinct(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range []string{Interactive, SubAgent, Programmatic, Unstated} {
		if seen[s] {
			t.Errorf("%q is used by two scope constants", s)
		}
		seen[s] = true
	}
	if len(seen) != 4 {
		t.Errorf("the vocabulary holds %d distinct scopes, want 4", len(seen))
	}
}
