package plugin

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
)

func ruleTestVerdicts() []analyze.Result {
	return []analyze.Result{{
		Name:      "burn-anomaly",
		Title:     "Burn Anomaly",
		Read:      analyze.Read{Key: "watch", Label: "WATCH"},
		Purity:    0.3,
		HowToRead: "Directional demo signal.",
		Takeaway:  "One day burned far outside the window's typical day.",
	}}
}

func runRuleScript(t *testing.T, name string, timeout time.Duration) ([]Alert, error) {
	t.Helper()
	cfg := Config{Name: "demo", Command: script(t, name), Timeout: timeout}
	return RunRule(context.Background(), cfg, ruleTestVerdicts())
}

func TestRunRuleHappyPath(t *testing.T) {
	got, err := runRuleScript(t, "rule_good.sh", 5*time.Second)
	if err != nil {
		t.Fatalf("RunRule() err = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(alerts) = %d, want 2", len(got))
	}
	if got[0].Plugin != "demo" {
		t.Fatalf("Plugin = %q, want the configured name stamped at the boundary", got[0].Plugin)
	}
	if got[0].Rule != "token-spike" || got[0].Severity != SeverityWarn || got[0].Validator != "burn-anomaly" {
		t.Fatalf("alert[0] = %+v, want the emitted warn alert", got[0])
	}
	if got[1].Severity != SeverityInfo || got[1].Validator != "" {
		t.Fatalf("alert[1] = %+v, want an info alert with no validator reference", got[1])
	}
}

func TestRunRuleCarriesErrorSeverity(t *testing.T) {
	got, err := runRuleScript(t, "rule_error.sh", 5*time.Second)
	if err != nil {
		t.Fatalf("RunRule() err = %v", err)
	}
	if len(got) != 1 || got[0].Severity != SeverityError {
		t.Fatalf("alerts = %+v, want one error-severity alert (the gating case)", got)
	}
}

func TestRunRuleEmptyVerdictsSafe(t *testing.T) {
	cfg := Config{Name: "demo", Command: script(t, "rule_good.sh"), Timeout: 5 * time.Second}
	if _, err := RunRule(context.Background(), cfg, nil); err != nil {
		t.Fatalf("RunRule(no verdicts) err = %v", err)
	}
}

func TestRunRuleFailures(t *testing.T) {
	cases := []struct {
		script  string
		timeout time.Duration
		wantSub string
	}{
		{"rule_bad_handshake.sh", 5 * time.Second, "handshake protocol 0"},
		{"rule_name_mismatch.sh", 5 * time.Second, "handshake name"},
		{"rule_invalid_severity.sh", 5 * time.Second, "severity"},
		{"rule_malformed.sh", 5 * time.Second, "decoding alerts"},
		{"rule_two_docs.sh", 5 * time.Second, "trailing data"},
		{"rule_oversize.sh", 5 * time.Second, "exceeded"},
		{"rule_timeout.sh", 200 * time.Millisecond, "timed out"},
		{"rule_nonzero.sh", 5 * time.Second, "exit status 3"},
		{"rule_silent.sh", 5 * time.Second, "no handshake"},
	}
	for _, tc := range cases {
		t.Run(tc.script, func(t *testing.T) {
			alerts, err := runRuleScript(t, tc.script, tc.timeout)
			if err == nil {
				t.Fatal("err = nil, want failure")
			}
			if len(alerts) != 0 {
				t.Fatalf("alerts = %+v, want none: a failing rule is dropped whole", alerts)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("err = %v, want substring %q", err, tc.wantSub)
			}
			if !strings.Contains(err.Error(), "rule plugin demo") {
				t.Fatalf("err = %v, want the failing plugin named", err)
			}
		})
	}
}

func TestBuildRuleInputCarriesVerdictsOnly(t *testing.T) {
	raw, err := marshalRuleInput(buildRuleInput(ruleTestVerdicts()))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"assaio_rule_input":1`, `"verdicts":[`, `"burn-anomaly"`} {
		if !strings.Contains(raw, want) {
			t.Fatalf("envelope = %s, want substring %q", raw, want)
		}
	}
	empty, err := marshalRuleInput(buildRuleInput(nil))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(empty, `"verdicts":[]`) {
		t.Fatalf("empty envelope = %s, want an empty JSON array, never null", empty)
	}
}

func marshalRuleInput(in ruleInput) (string, error) {
	raw, err := in.marshal()
	return string(raw), err
}
