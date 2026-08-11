package label

import "testing"

func TestDefaultRulesDeriveTheCommonConventions(t *testing.T) {
	e, err := NewEngine(DefaultRules)
	if err != nil {
		t.Fatalf("NewEngine(DefaultRules): %v", err)
	}
	tests := []struct {
		branch string
		want   string // "" means: derive nothing
	}{
		{"feat/rules-engine", "feature"},
		{"feature/JIRA-123-login", "feature"},
		{"features-payment", "feature"},
		{"fix/nil-deref", "bugfix"},
		{"hotfix_prod", "bugfix"},
		{"BUG-cache", "bugfix"},
		{"test/golden-files", "test"},
		{"refactor/store", "refactor"},
		{"docs/readme", "docs"},
		{"spike/otel", "research"},
		{"audit/pricing", "review"},
		{"FEAT/upper-case", "feature"},

		// A repository with no convention yields nothing rather than a guess.
		{"main", ""},
		{"develop", ""},
		{"JIRA-4821", ""},
		{"faza-1", ""},
		{"", ""},
		// The separator is what makes the prefix a prefix.
		{"fixture/loader", ""},
		{"features", ""},
		{"documentation", ""},
		{"testing-library-upgrade", "test"},
	}
	for _, tc := range tests {
		got := e.Derive(Signals{SourceBranch: {tc.branch}})
		switch {
		case tc.want == "" && len(got) != 0:
			t.Errorf("%q derived %+v, want nothing", tc.branch, got)
		case tc.want != "" && len(got) != 1:
			t.Errorf("%q derived %+v, want one %s", tc.branch, got, tc.want)
		case tc.want != "" && len(got) == 1 && got[0].Value != tc.want:
			t.Errorf("%q derived %q, want %q", tc.branch, got[0].Value, tc.want)
		}
	}
}

func TestDisagreeingSourcesDeriveNothing(t *testing.T) {
	e, err := NewEngine([]Rule{
		{Source: SourceBranch, Match: `^fix/`, Axis: Task, Value: "bugfix"},
		{Source: SourceSkill, Match: `^tdd$`, Axis: Task, Value: "test"},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	sig := Signals{SourceBranch: {"fix/thing"}, SourceSkill: {"tdd"}}
	if got := e.Derive(sig); len(got) != 0 {
		t.Errorf("a branch saying bugfix and a skill saying test derived %+v, want nothing", got)
	}
}

func TestAgreeingSourcesNameThemBoth(t *testing.T) {
	e, err := NewEngine([]Rule{
		{Source: SourceBranch, Match: `^fix/`, Axis: Task, Value: "bugfix"},
		{Source: SourceSkill, Match: `^debugging$`, Axis: Task, Value: "bugfix"},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	got := e.Derive(Signals{SourceBranch: {"fix/thing"}, SourceSkill: {"debugging"}})
	if len(got) != 1 || got[0].Value != "bugfix" {
		t.Fatalf("derived %+v, want one bugfix", got)
	}
	if got[0].Source != "branch+skill" {
		t.Errorf("source = %q, want branch+skill", got[0].Source)
	}
}

// A session that ran on two branches deriving two different classes is ambiguous, and the
// engine must not pick the first one it happened to read.
func TestOneSessionOnTwoBranchesDerivesNothing(t *testing.T) {
	e, _ := NewEngine(DefaultRules)
	got := e.Derive(Signals{SourceBranch: {"fix/one", "feat/two"}})
	if len(got) != 0 {
		t.Errorf("two branches disagreeing derived %+v, want nothing", got)
	}
}

func TestSameBranchTwiceIsNotADisagreement(t *testing.T) {
	e, _ := NewEngine(DefaultRules)
	got := e.Derive(Signals{SourceBranch: {"fix/one", "fix/two"}})
	if len(got) != 1 || got[0].Value != "bugfix" {
		t.Errorf("derived %+v, want one bugfix", got)
	}
}

func TestRuleSetIsValidatedAtLoad(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
	}{
		{"unknown source", Rule{Source: "commit", Match: `x`, Axis: Task, Value: "feature"}},
		{"unknown axis", Rule{Source: SourceBranch, Match: `x`, Axis: "mood", Value: "feature"}},
		{"value outside the vocabulary", Rule{Source: SourceBranch, Match: `x`, Axis: Task, Value: "chore"}},
		{"empty value", Rule{Source: SourceBranch, Match: `x`, Axis: Task, Value: ""}},
		{"pattern that does not compile", Rule{Source: SourceBranch, Match: `[`, Axis: Task, Value: "feature"}},
	}
	for _, tc := range tests {
		if _, err := NewEngine([]Rule{tc.rule}); err == nil {
			t.Errorf("%s: NewEngine accepted %+v, want an error", tc.name, tc.rule)
		}
	}
}

func TestSilenceDerivesNothing(t *testing.T) {
	e, _ := NewEngine(DefaultRules)
	if got := e.Derive(Signals{}); len(got) != 0 {
		t.Errorf("no signals derived %+v, want nothing", got)
	}
	if got := e.Derive(Signals{SourceBranch: {""}}); len(got) != 0 {
		t.Errorf("an empty branch derived %+v, want nothing", got)
	}
}
