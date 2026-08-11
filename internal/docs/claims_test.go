package docs

import (
	"strings"
	"testing"
)

// fixture is a reference small enough to reason about, so these tests are about the checking
// rule rather than about what assaio happens to ship today.
func fixture() *Reference {
	return &Reference{
		Version:    Version,
		Signals:    []Signal{{ID: "ai.tokens.total"}, {ID: "ai.lines.added"}},
		Sources:    []Source{{Tool: "claude-code"}, {Tool: "cline"}},
		Validators: []Validator{{Name: "model-fit"}},
		Commands: []Command{
			{Path: "digest"}, {Path: "signals"}, {Path: "signals coverage"}, {Path: "version"},
		},
		Config: []ConfigKey{{Key: "pricing.mode"}},
		MetricWire: MetricWire{Input: []WireField{
			{Path: "windowStart"}, {Path: "usage"}, {Path: "usage[].day"},
		}},
	}
}

func TestCheckClaims(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "a claim the binary can answer passes",
			content: `<code data-claim="command.digest">digest</code>`,
		},
		{
			name:    "a subcommand is addressable even though no page must list it",
			content: `<code data-claim="command.signals coverage">signals coverage</code>`,
		},
		{
			name:    "a claim about something removed fails",
			content: `<code data-claim="command.prune">prune</code>`,
			want:    `has no command for`,
		},
		{
			name:    "a covered set that never names a member fails",
			content: `<dl data-covers="commands"><dt data-claim="command.digest">digest</dt></dl>`,
			want:    `covers "commands" but never claims "command.signals"`,
		},
		{
			name: "a covered set naming every member passes",
			content: `<dl data-covers="commands">
				<dt data-claim="command.digest">digest</dt>
				<dt data-claim="command.signals">signals</dt></dl>`,
		},
		{
			name:    "an unknown set is a mistake in the page, not a silent pass",
			content: `<dl data-covers="widgets"></dl>`,
			want:    `which is not a set`,
		},
		{
			name:    "a spelled-out count that matches passes",
			content: `<span data-claim="signals.count">Two</span> signals`,
		},
		{
			name:    "a numeric count that matches passes",
			content: `<span data-claim="sources.count">2</span> sources`,
		},
		{
			name:    "a count that has fallen behind fails",
			content: `<span data-claim="validators.count">Nineteen</span> reads`,
			want:    `the registry has 1 and the element reads "Nineteen"`,
		},
		{
			name: "a second statement of the same count is checked too",
			content: `<h2><span data-claim="signals.count">Two</span> signals</h2>` +
				`<p>all <span data-claim="signals.count">three</span> of them</p>`,
			want: `the element reads "three"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problems := CheckClaims(fixture(), "page.html", tt.content)
			switch {
			case tt.want == "" && len(problems) > 0:
				t.Fatalf("expected no problem, got %v", problems)
			case tt.want == "":
				return
			case len(problems) == 0:
				t.Fatalf("expected a problem containing %q, got none", tt.want)
			}
			if !strings.Contains(problems[0].Text, tt.want) {
				t.Errorf("problem = %q, want it to contain %q", problems[0].Text, tt.want)
			}
		})
	}
}

// The commands set is what a published surface must enumerate, and it is deliberately narrower
// than the claim space. A change that folded subcommands into it would quietly demand that
// every page list `signals coverage` too.
func TestOnlyTopLevelCommandsAreCovered(t *testing.T) {
	ix := newIndex(fixture())
	for _, m := range ix.sets["commands"] {
		if strings.Contains(m.name, " ") {
			t.Errorf("%q is a subcommand and must not be in the covered set", m.name)
		}
	}
	if !ix.ids["command.signals coverage"] {
		t.Error("a subcommand must still be addressable by a page that chooses to name it")
	}
}
