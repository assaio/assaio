package docs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/docs"
)

// The plugin-wiring recipes are configuration, so the check that means anything is whether
// assaio's own loader accepts them and validates clean. A block that named a field the config
// does not have would read fine and do nothing when pasted.
func TestRecipePluginConfigLoads(t *testing.T) {
	tests := []struct {
		page, recipe, wantName string
	}{
		{"rule-plugins", "rule-config", "house-rules"},
		{"extensions", "metric-config", "weekday-split"},
	}
	for _, tt := range tests {
		t.Run(tt.recipe, func(t *testing.T) {
			recipes := docs.ExtractRecipes(read(t, repoRoot+"/docs/recipes/"+tt.page+".md"))
			block, err := recipes.Get(tt.recipe)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(block), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(path)
			if err != nil {
				t.Fatalf("the published recipe does not load: %v", err)
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("the published recipe loads but does not validate: %v", err)
			}
			entries := append(append([]config.PluginConfig{}, cfg.Rules...), cfg.Metrics...)
			if len(entries) != 1 {
				t.Fatalf("the recipe declares %d plugin entries, want exactly one", len(entries))
			}
			if entries[0].Name != tt.wantName {
				t.Errorf("name = %q, want %q", entries[0].Name, tt.wantName)
			}
			if entries[0].Command == "" || entries[0].Timeout == "" {
				t.Errorf("entry = %+v, want a command and a timeout", entries[0])
			}
		})
	}
}
