package docs_test

import (
	"os"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"

	"github.com/assaio/assaio/internal/docs"
	"github.com/assaio/assaio/internal/label"
)

// A cookbook whose recipes have quietly stopped working is worse than no cookbook: it reads
// authoritative and hands out a rule set that derives nothing. Each case below runs the exact
// block the page prints -- extracted from the Markdown, not retyped here -- and asserts what it
// derives. Renaming an axis value or tightening a pattern fails this before it reaches a reader.
func TestRecipeLabelRules(t *testing.T) {
	doc, err := os.ReadFile(repoRoot + "/docs/recipes/label-rules.md")
	if err != nil {
		t.Fatal(err)
	}
	recipes := docs.ExtractRecipes(string(doc))

	tests := []struct {
		recipe string
		sig    label.Signals
		want   map[string]string
	}{
		{
			recipe: "conventional-branches",
			sig:    label.Signals{label.SourceBranch: {"feat/rules-engine"}},
			want:   map[string]string{label.Task: "feature"},
		},
		{
			recipe: "conventional-branches",
			sig:    label.Signals{label.SourceBranch: {"hotfix/unpriced-share"}},
			want:   map[string]string{label.Task: "bugfix"},
		},
		{
			recipe: "conventional-branches",
			sig:    label.Signals{label.SourceBranch: {"doc/rewrite"}},
			want:   map[string]string{label.Task: "docs"},
		},
		{
			recipe: "conventional-branches",
			sig:    label.Signals{label.SourceBranch: {"main"}},
			want:   map[string]string{},
		},
		{
			recipe: "ticket-keys",
			sig:    label.Signals{label.SourceBranch: {"BUG-1421-cache-miss"}},
			want:   map[string]string{label.Task: "bugfix"},
		},
		{
			recipe: "ticket-keys",
			sig:    label.Signals{label.SourceBranch: {"us-88-team-panel"}},
			want:   map[string]string{label.Task: "feature"},
		},
		{
			recipe: "spikes",
			sig:    label.Signals{label.SourceBranch: {"spike/evidence-graph"}},
			want:   map[string]string{label.Task: "research", label.Difficulty: "high"},
		},
		{
			recipe: "skills-and-agents",
			sig:    label.Signals{label.SourceSkill: {"code-review"}},
			want:   map[string]string{label.Task: "review"},
		},
		{
			recipe: "skills-and-agents",
			sig:    label.Signals{label.SourceAgent: {"reviewer"}},
			want:   map[string]string{label.Task: "review"},
		},
		{
			// A source the session never recorded derives nothing: silence is not evidence.
			recipe: "skills-and-agents",
			sig:    label.Signals{label.SourceBranch: {"feat/x"}},
			want:   map[string]string{},
		},
		{
			recipe: "entrypoints",
			sig:    label.Signals{label.SourceEntrypoint: {"cron"}},
			want:   map[string]string{label.Difficulty: "low"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.recipe+"/"+first(tt.sig), func(t *testing.T) {
			block, err := recipes.Get(tt.recipe)
			if err != nil {
				t.Fatal(err)
			}
			engine, err := label.NewEngine(rulesFromYAML(t, block))
			if err != nil {
				t.Fatalf("the published recipe does not load: %v", err)
			}
			got := map[string]string{}
			for _, s := range engine.Derive(tt.sig) {
				got[s.Axis] = s.Value
			}
			if len(got) != len(tt.want) {
				t.Fatalf("derived %v, want %v", got, tt.want)
			}
			for axis, want := range tt.want {
				if got[axis] != want {
					t.Errorf("%s = %q, want %q", axis, got[axis], want)
				}
			}
		})
	}
}

// rulesFromYAML loads a recipe the way a reader's config.yaml would, so a block that is valid
// here is a block that works when pasted.
func rulesFromYAML(t *testing.T, block string) []label.Rule {
	t.Helper()
	k := koanf.New(".")
	if err := k.Load(rawbytes.Provider([]byte(block)), yaml.Parser()); err != nil {
		t.Fatalf("the published recipe is not valid YAML: %v", err)
	}
	var cfg struct {
		Labels struct {
			Rules []label.Rule `koanf:"rules"`
		} `koanf:"labels"`
	}
	if err := k.Unmarshal("", &cfg); err != nil {
		t.Fatalf("the published recipe does not decode into the config shape: %v", err)
	}
	if len(cfg.Labels.Rules) == 0 {
		t.Fatal("the published recipe declares no rules")
	}
	return cfg.Labels.Rules
}

func first(sig label.Signals) string {
	for _, source := range label.Sources {
		if v := sig[source]; len(v) > 0 {
			return v[0]
		}
	}
	return "nothing"
}
