package docs_test

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/assaio/assaio/internal/docs"
)

// The window a published metric plugin is judged against. Two tools: one that records edits and
// one that does not, so a recipe that ignores `answers` reports a silence as a zero and this
// notices. Saturday is in there because the weekday recipe turns on it.
const metricInput = `{
  "assaio_metric_input": 1,
  "answers": {
    "claude-code": ["ai.tokens.total","ai.lines.added","ai.edits.count"],
    "gemini-cli":  ["ai.tokens.total"]
  },
  "usage": [
    {"day":"2026-08-06","tool":"claude-code","model":"m","in":100,"out":50,"cacheRead":0,
     "cacheWrite":0,"linesAdded":300,"edits":10},
    {"day":"2026-08-08","tool":"gemini-cli","model":"m","in":100,"out":50,"cacheRead":0,
     "cacheWrite":0,"linesAdded":0,"edits":0}
  ]
}`

// Executed, not merely printed: each plugin is run the way `analyze` runs one and its result is
// held to the contract. A recipe that stopped decoding the envelope would fail here rather than
// on a reader's first window.
func TestRecipeMetricPlugins(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not on PATH; the published recipes are written in it")
	}
	recipes := docs.ExtractRecipes(read(t, repoRoot+"/docs/recipes/extensions.md"))

	tests := []struct {
		recipe   string
		wantRead string
		wantFig  string
	}{
		// 2026-08-06 is a Thursday and 2026-08-08 a Saturday, so half the tokens fall outside
		// the working week -- far enough below the recipe's own threshold to flag.
		{recipe: "plugin-weekday", wantRead: "watch", wantFig: "50.0%"},
		// Only claude-code answers both signals: 300 lines over 10 edits. Counting the other
		// tool's structural zeros would give 15.0 and call it a measurement.
		{recipe: "plugin-answers", wantRead: "good", wantFig: "30.0"},
	}
	for _, tt := range tests {
		t.Run(tt.recipe, func(t *testing.T) {
			block, err := recipes.Get(tt.recipe)
			if err != nil {
				t.Fatal(err)
			}
			got := runMetric(t, block)
			if got.Read.Key != tt.wantRead {
				t.Errorf("read.key = %q, want %q", got.Read.Key, tt.wantRead)
			}
			if len(got.Figures) != 1 || got.Figures[0].Value != tt.wantFig {
				t.Errorf("figures = %+v, want one reading %q", got.Figures, tt.wantFig)
			}
			for field, v := range map[string]string{
				"title": got.Title, "howToRead": got.HowToRead, "takeaway": got.Takeaway,
			} {
				if v == "" {
					t.Errorf("the boundary requires %s, and the recipe returns none", field)
				}
			}
			if len(got.Caveats) == 0 {
				t.Error("a directional metric published with no caveat is the one thing the " +
					"honesty rules do not allow")
			}
		})
	}
}

type metricResult struct {
	Title     string `json:"title"`
	HowToRead string `json:"howToRead"`
	Takeaway  string `json:"takeaway"`
	Read      struct {
		Key   string `json:"key"`
		Label string `json:"label"`
	} `json:"read"`
	Figures []struct {
		Label string `json:"label"`
		Value string `json:"value"`
	} `json:"figures"`
	Caveats []string `json:"caveats"`
}

func runMetric(t *testing.T, script string) metricResult {
	t.Helper()
	stdout := runScript(t, script, "analyze", "ASSAIO_METRIC_PROTOCOL=1", metricInput)

	dec := json.NewDecoder(stdout)
	var handshake struct {
		Version int    `json:"assaio_metric"`
		Name    string `json:"name"`
	}
	if err := dec.Decode(&handshake); err != nil {
		t.Fatalf("no handshake line: %v", err)
	}
	if handshake.Version != 1 || handshake.Name == "" {
		t.Fatalf("handshake = %+v, want version 1 and a name", handshake)
	}
	var out metricResult
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("no result document: %v", err)
	}
	return out
}
