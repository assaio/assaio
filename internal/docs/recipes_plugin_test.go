package docs_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/docs"
)

// The window a published rule plugin is judged against: one verdict that is fine, one flagged,
// one withheld for want of data. A recipe that cannot read this envelope cannot read a real one.
const ruleInput = `{
  "assaio_rule_input": 1,
  "verdicts": [
    {"name":"adoption","title":"Adoption & Usage Breadth","read":{"key":"good","label":"STRONG"},
     "takeaway":"Usage is broad.","confidence":{"label":"high"}},
    {"name":"rework","title":"Rework & Rejection","read":{"key":"watch","label":"WATCH"},
     "takeaway":"Churn is up week over week.","confidence":{"label":"medium"}},
    {"name":"model-fit","title":"Model Fit","read":{"key":"watch","label":"WATCH"},
     "takeaway":"Premium share is high for the work done.","confidence":{"label":"high"}},
    {"name":"context","title":"Context Health","read":{"key":"watch","label":"WATCH"},
     "takeaway":"Sessions are hitting compaction often.","confidence":{"label":"high"}},
    {"name":"cache-hygiene","title":"Cache Hygiene","read":{"key":"neutral","label":"—"},
     "takeaway":"Nothing in this window records a cache write.",
     "confidence":{"label":"insufficient"}}
  ]
}`

// A printed plugin nobody runs is a plugin that has quietly stopped working. Each recipe is
// executed exactly as `check` would execute it -- the protocol's handshake line, then one alerts
// document -- and asserted on. The alternative is a cookbook that reads authoritative and hands
// out code that fails on the first real window.
func TestRecipeRulePlugins(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not on PATH; the published recipes are written in it")
	}
	doc, err := os.ReadFile(repoRoot + "/docs/recipes/rule-plugins.md")
	if err != nil {
		t.Fatal(err)
	}
	recipes := docs.ExtractRecipes(string(doc))

	tests := []struct {
		recipe string
		want   []alert
	}{
		{
			recipe: "withheld",
			want:   []alert{{Rule: "withheld-verdict", Severity: "warn", Validator: "cache-hygiene"}},
		},
		{
			recipe: "named-read",
			// cache-hygiene is neutral in this window, not watch, so only the two reads
			// that actually turned bad raise anything -- which is the discrimination the
			// recipe is for.
			want: []alert{
				{Rule: "rework-watch", Severity: "error", Validator: "rework"},
				{Rule: "context-watch", Severity: "warn", Validator: "context"},
			},
		},
		{
			recipe: "cannot-decide",
			want:   []alert{{Rule: "premium-share", Severity: "error", Validator: "model-fit"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.recipe, func(t *testing.T) {
			block, err := recipes.Get(tt.recipe)
			if err != nil {
				t.Fatal(err)
			}
			got := runRule(t, block)
			if len(got) != len(tt.want) {
				t.Fatalf("raised %d alert(s), want %d: %+v", len(got), len(tt.want), got)
			}
			for i, want := range tt.want {
				switch {
				case got[i].Rule != want.Rule:
					t.Errorf("alert %d rule = %q, want %q", i, got[i].Rule, want.Rule)
				case got[i].Severity != want.Severity:
					t.Errorf("alert %d severity = %q, want %q", i, got[i].Severity, want.Severity)
				case got[i].Validator != want.Validator:
					t.Errorf("alert %d validator = %q, want %q", i, got[i].Validator, want.Validator)
				case strings.TrimSpace(got[i].Message) == "":
					t.Errorf("alert %d carries no message for a human to read", i)
				}
			}
		})
	}
}

type alert struct {
	Rule      string `json:"rule"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Validator string `json:"validator"`
}

// runScript runs one published recipe as a subprocess, exactly as the protocol invokes it.
func runScript(t *testing.T, script, verb, env, stdin string) *bytes.Buffer {
	t.Helper()
	path := filepath.Join(t.TempDir(), "recipe.py")
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil { //nolint:gosec // path is t.TempDir()
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", path, verb) //nolint:gosec // a recipe from this repository, run in a temp dir
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = append(os.Environ(), env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("the published recipe failed to run: %v\n%s", err, stderr.String())
	}
	return &stdout
}

// runRule executes one recipe the way the gate does and enforces the shape the boundary
// enforces: a handshake naming protocol 1, then exactly one alerts document.
func runRule(t *testing.T, script string) []alert {
	t.Helper()
	stdout := runScript(t, script, "evaluate", "ASSAIO_RULE_PROTOCOL=1", ruleInput)

	dec := json.NewDecoder(stdout)
	var handshake struct {
		Version int    `json:"assaio_rule"`
		Name    string `json:"name"`
	}
	if err := dec.Decode(&handshake); err != nil {
		t.Fatalf("no handshake line: %v", err)
	}
	if handshake.Version != 1 || handshake.Name == "" {
		t.Fatalf("handshake = %+v, want version 1 and a name", handshake)
	}
	var doc struct {
		Alerts []alert `json:"alerts"`
	}
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("no alerts document: %v", err)
	}
	return doc.Alerts
}
