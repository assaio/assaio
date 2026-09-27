package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/ingest"
	"github.com/assaio/assaio/internal/store"
)

// TestBackfillExplainsWhatAReReadChanged: each count appears on its source's line and is
// explained once, and a run that changed nothing prints none of the explanations.
func TestBackfillExplainsWhatAReReadChanged(t *testing.T) {
	notes := []string{"restated-down:", "identity-changed:", "identity-kept:", "steps-changed:"}
	tests := []struct {
		name   string
		result ingest.Result
		want   []string
	}{
		{"nothing changed", ingest.Result{Tool: "claude-code"}, nil},
		{"a lowered figure", ingest.Result{Tool: "claude-code", Lowered: 2}, []string{"restated-down=2", "restated-down:"}},
		{
			"a replaced model",
			ingest.Result{Tool: "cline", Identity: store.IdentityChanges{Rows: 3, Model: 3}},
			[]string{"identity-changed=3", "identity-changed:", "model 3"},
		},
		{"a kept name", ingest.Result{Tool: "codex", Identity: store.IdentityChanges{Kept: 1}}, []string{"identity-kept=1", "identity-kept:"}},
		{"a changed step", ingest.Result{Tool: "codex", StepsChanged: 4}, []string{"steps-changed=4", "steps-changed:"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&out)
			printBackfillResults(cmd, []ingest.Result{tt.result})
			got := out.String()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Fatalf("output missing %q:\n%s", w, got)
				}
			}
			for _, n := range notes {
				wanted := false
				for _, w := range tt.want {
					wanted = wanted || w == n
				}
				if !wanted && strings.Contains(got, n) {
					t.Fatalf("output explains %q although nothing asked for it:\n%s", n, got)
				}
			}
		})
	}
}
