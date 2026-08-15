package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// TestEveryAnalysisInputCarriesThePlanPrice asserts the shared builder carries the plan price.
// Four of the five commands building an analyze.Input set it at their own call site and
// `metrics verify` did not, so a plugin gating on planMonthlyCost verified VALID against a zero.
func TestEveryAnalysisInputCarriesThePlanPrice(t *testing.T) {
	seedLocalStore(t, []usage.Record{{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(),
		Model: "claude-opus-4-5", InputTokens: 1000, OutputTokens: 500, DedupeKey: "1",
	}})
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	if err := os.MkdirAll(filepath.Join(cfgDir, "assaio"), 0o750); err != nil {
		t.Fatal(err)
	}
	cfg := "pricing:\n  mode: subscription\n  monthly_subscription_cost: 200\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "assaio", "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd, _, err := NewRootCmd().Find([]string{"analyze"})
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(context.Background())
	st, err := openReportStore(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	in, err := buildAnalyzeInput(cmd, st, time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatal(err)
	}
	if in.PlanMonthlyCost != 200 {
		t.Fatalf("PlanMonthlyCost = %v, want 200: the shared builder must carry it so no caller can omit it", in.PlanMonthlyCost)
	}
}
