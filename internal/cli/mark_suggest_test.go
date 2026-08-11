package cli

import (
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/store"
)

func TestCollectSignalsKeepsValuesDistinctPerSource(t *testing.T) {
	rows := []store.SessionSignalRow{
		{SessionID: "a", Branch: "feat/x", Skill: "planning"},
		{SessionID: "a", Branch: "feat/x", Skill: "review"},
		{SessionID: "a", Branch: "fix/y"},
		{SessionID: "b", Entrypoint: "cli"},
		// Same session id, two members: a shared store must not merge their signals.
		{SessionID: "shared", Member: "ann", Branch: "fix/hers"},
		{SessionID: "shared", Member: "bob", Branch: "feat/his"},
	}
	got := collectSignals(rows)
	if len(got[signalKey{"a", ""}][label.SourceBranch]) != 2 {
		t.Errorf("branches = %v, want the two distinct ones", got[signalKey{"a", ""}][label.SourceBranch])
	}
	if len(got[signalKey{"a", ""}][label.SourceSkill]) != 2 {
		t.Errorf("skills = %v, want two", got[signalKey{"a", ""}][label.SourceSkill])
	}
	if len(got[signalKey{"b", ""}][label.SourceEntrypoint]) != 1 {
		t.Errorf("entrypoints = %v, want one", got[signalKey{"b", ""}][label.SourceEntrypoint])
	}
	if _, ok := got[signalKey{"b", ""}][label.SourceBranch]; ok {
		t.Error("an empty branch became a signal; silence must record nothing")
	}
	ann := got[signalKey{"shared", "ann"}][label.SourceBranch]
	bob := got[signalKey{"shared", "bob"}][label.SourceBranch]
	if len(ann) != 1 || ann[0] != "fix/hers" || len(bob) != 1 || bob[0] != "feat/his" {
		t.Errorf("two members sharing a session id merged: ann=%v bob=%v", ann, bob)
	}
}

func TestDerivedLabelsNeverTouchAnAxisSetByHand(t *testing.T) {
	engine, err := label.NewEngine(label.DefaultRules)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	sessions := []store.SessionRow{
		{SessionID: "hand", Task: "refactor"},
		{SessionID: "free"},
		{SessionID: "other-axis", Outcome: "done"},
	}
	signals := []store.SessionSignalRow{
		{SessionID: "hand", Branch: "feat/x"},
		{SessionID: "free", Branch: "feat/x"},
		{SessionID: "other-axis", Branch: "feat/x"},
	}
	got := deriveSuggestions(engine, sessions, signals)

	bySession := map[string][]label.Suggestion{}
	for _, s := range got {
		bySession[s.ref.SessionID] = s.axes
	}
	if _, proposed := bySession["hand"]; proposed {
		t.Error("a session whose task was set by hand received a task suggestion")
	}
	if len(bySession["free"]) != 1 || bySession["free"][0].Value != "feature" {
		t.Errorf("unlabeled session got %+v, want one feature", bySession["free"])
	}
	// A session someone already annotated is answered, even on the axes they left empty:
	// clearing one axis leaves the row behind, so a blank there is a decision, not a gap.
	if _, proposed := bySession["other-axis"]; proposed {
		t.Errorf("a session already labeled by hand received a suggestion: %+v", bySession["other-axis"])
	}
}

func TestSessionsWithoutASuggestionAreNotReported(t *testing.T) {
	engine, _ := label.NewEngine(label.DefaultRules)
	sessions := []store.SessionRow{{SessionID: "a"}, {SessionID: "b"}}
	signals := []store.SessionSignalRow{
		{SessionID: "a", Branch: "main"},
		{SessionID: "b", Branch: "develop"},
	}
	if got := deriveSuggestions(engine, sessions, signals); len(got) != 0 {
		t.Errorf("conventionless branches produced %+v, want nothing", got)
	}
}

func TestConfiguredRulesExtendOrReplaceTheDefaults(t *testing.T) {
	own := []config.LabelRule{{Source: "branch", Match: `^faza-`, Axis: "task", Value: "feature"}}

	extended, err := newLabelEngine(&config.Config{Labels: config.Labels{Rules: own}})
	if err != nil {
		t.Fatalf("newLabelEngine: %v", err)
	}
	if got := extended.Derive(label.Signals{label.SourceBranch: {"fix/x"}}); len(got) != 1 {
		t.Errorf("a built-in rule stopped applying when a custom one was added: %+v", got)
	}
	if got := extended.Derive(label.Signals{label.SourceBranch: {"faza-1"}}); len(got) != 1 {
		t.Errorf("the configured rule did not apply: %+v", got)
	}

	no := false
	only, err := newLabelEngine(&config.Config{Labels: config.Labels{Rules: own, Defaults: &no}})
	if err != nil {
		t.Fatalf("newLabelEngine: %v", err)
	}
	if got := only.Derive(label.Signals{label.SourceBranch: {"fix/x"}}); len(got) != 0 {
		t.Errorf("defaults: false still applied a built-in rule: %+v", got)
	}
	if got := only.Derive(label.Signals{label.SourceBranch: {"faza-1"}}); len(got) != 1 {
		t.Errorf("defaults: false dropped the configured rule too: %+v", got)
	}
}

// The rule number in the error has to be the one the person can find in their own file.
func TestABrokenConfiguredRuleIsNumberedAsWritten(t *testing.T) {
	_, err := newLabelEngine(&config.Config{Labels: config.Labels{
		Rules: []config.LabelRule{{Source: "branch", Match: `[`, Axis: "task", Value: "feature"}},
	}})
	if err == nil {
		t.Fatal("a rule that does not compile was accepted")
	}
	if want := "labels.rules: rule 1"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to name %q", err, want)
	}
}

// mark's every other mode acts on one session. Accepting suggestions writes across the whole
// window, so a session id or --last passed alongside it is a person expecting something the
// command will not do -- and labels are the one thing no re-import can rebuild.
func TestAcceptSuggestedRefusesTheFlagsItCannotHonour(t *testing.T) {
	seedMarkStore(t)
	tests := []struct {
		name string
		args []string
		flag string
		want string
	}{
		{"a named session", []string{"a1b2c3d4"}, "--accept-suggested", "cannot take a session"},
		{"--last", nil, "--accept-suggested --last", "--last"},
		{"an axis flag", nil, "--accept-suggested --task feature", "cannot be combined with setting one"},
		{"--unmark", nil, "--accept-suggested --unmark", "--unmark"},
		{"both modes", nil, "--suggest --accept-suggested", "pass one"},
		{"a named session with --suggest", []string{"a1b2c3d4"}, "--suggest", "cannot take a session"},
	}
	for _, tc := range tests {
		out, err := runCLI(t, append([]string{"mark"}, append(strings.Fields(tc.flag), tc.args...)...)...)
		if err == nil {
			t.Errorf("%s: accepted, want an error (output: %s)", tc.name, out)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error = %q, want it to mention %q", tc.name, err, tc.want)
		}
	}
}
