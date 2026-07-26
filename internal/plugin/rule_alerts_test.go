package plugin

import (
	"encoding/json"
	"strings"
	"testing"
)

func alertsDoc(t *testing.T, alerts ...wireAlert) []byte {
	t.Helper()
	doc, err := json.Marshal(wireAlerts{Alerts: &alerts})
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func validWireAlert() wireAlert {
	return wireAlert{Rule: "token-spike", Severity: SeverityWarn, Message: "Tokens up sharply.", Validator: "burn-anomaly"}
}

func TestParseRuleAlertsValid(t *testing.T) {
	got, violations, err := parseRuleAlerts(alertsDoc(t, validWireAlert()), "demo")
	if err != nil {
		t.Fatalf("parseRuleAlerts() err = %v (violations %v)", err, violations)
	}
	if len(got) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(got))
	}
	want := Alert{Plugin: "demo", Rule: "token-spike", Severity: SeverityWarn, Message: "Tokens up sharply.", Validator: "burn-anomaly"}
	if got[0] != want {
		t.Fatalf("alert = %+v, want %+v", got[0], want)
	}
}

func TestParseRuleAlertsAcceptsEverySeverityAndNoAlerts(t *testing.T) {
	for _, severity := range []string{SeverityInfo, SeverityWarn, SeverityError} {
		a := validWireAlert()
		a.Severity = severity
		got, _, err := parseRuleAlerts(alertsDoc(t, a), "demo")
		if err != nil || len(got) != 1 || got[0].Severity != severity {
			t.Fatalf("severity %q: alerts = %+v, err = %v", severity, got, err)
		}
	}
	got, _, err := parseRuleAlerts([]byte(`{"alerts":[]}`), "demo")
	if err != nil || len(got) != 0 {
		t.Fatalf("empty alerts: got %+v, err = %v: a quiet rule is a pass, not a failure", got, err)
	}
}

func TestParseRuleAlertsViolations(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*wireAlert)
		wantSub string
	}{
		{"unknown severity", func(a *wireAlert) { a.Severity = "critical" }, "severity"},
		{"empty severity", func(a *wireAlert) { a.Severity = "" }, "severity"},
		{"upper-case severity", func(a *wireAlert) { a.Severity = "ERROR" }, "severity"},
		{"missing rule", func(a *wireAlert) { a.Rule = "" }, "alerts[0].rule is required"},
		{"oversized rule", func(a *wireAlert) { a.Rule = strings.Repeat("r", 65) }, "alerts[0].rule exceeds"},
		{"missing message", func(a *wireAlert) { a.Message = "" }, "alerts[0].message is required"},
		{"oversized message", func(a *wireAlert) { a.Message = strings.Repeat("m", 401) }, "alerts[0].message exceeds"},
		{"oversized validator", func(a *wireAlert) { a.Validator = strings.Repeat("v", 65) }, "alerts[0].validator exceeds"},
		{"control char in message", func(a *wireAlert) { a.Message = "bad\x1b[31m" }, "control character"},
		{"control char in rule", func(a *wireAlert) { a.Rule = "bad\nrule" }, "control character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := validWireAlert()
			tc.mutate(&a)
			got, violations, err := parseRuleAlerts(alertsDoc(t, a), "demo")
			if err == nil {
				t.Fatal("err = nil, want a contract violation error")
			}
			if len(got) != 0 {
				t.Fatalf("alerts = %+v, want none: the document is rejected whole", got)
			}
			joined := strings.Join(violations, "; ")
			if !strings.Contains(joined, tc.wantSub) {
				t.Fatalf("violations = %q, want substring %q", joined, tc.wantSub)
			}
		})
	}
}

func TestParseRuleAlertsRejectsWholeDocumentOnOneBadAlert(t *testing.T) {
	bad := validWireAlert()
	bad.Severity = "fatal"
	got, violations, err := parseRuleAlerts(alertsDoc(t, validWireAlert(), bad), "demo")
	if err == nil || len(got) != 0 {
		t.Fatalf("got %+v, err = %v: one invalid alert rejects the whole document", got, err)
	}
	if len(violations) == 0 || !strings.Contains(violations[0], "alerts[1]") {
		t.Fatalf("violations = %v, want the offending index named", violations)
	}
}

func TestParseRuleAlertsCap(t *testing.T) {
	alerts := make([]wireAlert, maxRuleAlerts+1)
	for i := range alerts {
		alerts[i] = validWireAlert()
	}
	got, violations, err := parseRuleAlerts(alertsDoc(t, alerts...), "demo")
	if err == nil || len(got) != 0 {
		t.Fatalf("got %d alerts, err = %v, want the cap enforced", len(got), err)
	}
	if !strings.Contains(strings.Join(violations, "; "), "alerts exceeds 50 entries") {
		t.Fatalf("violations = %v, want the alert cap reason", violations)
	}

	atCap := make([]wireAlert, maxRuleAlerts)
	for i := range atCap {
		atCap[i] = validWireAlert()
	}
	if _, _, err := parseRuleAlerts(alertsDoc(t, atCap...), "demo"); err != nil {
		t.Fatalf("exactly %d alerts must pass: %v", maxRuleAlerts, err)
	}
}

func TestParseRuleAlertsMalformedDocuments(t *testing.T) {
	cases := []struct {
		name    string
		doc     string
		wantSub string
	}{
		{"not json", "hello", "decoding alerts"},
		{"empty document", "", "decoding alerts"},
		{"array instead of object", `[{"rule":"r","severity":"info","message":"m"}]`, "decoding alerts"},
		{"misspelled alerts key", `{"alert":[{"rule":"r","severity":"info","message":"m"}]}`, "decoding alerts"},
		{"misspelled severity key", `{"alerts":[{"rule":"r","sevrity":"info","message":"m"}]}`, "decoding alerts"},
		{"two documents", `{"alerts":[]}` + "\n" + `{"alerts":[]}`, "trailing data"},
		{"trailing garbage", `{"alerts":[]} }`, "trailing data"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := parseRuleAlerts([]byte(tc.doc), "demo")
			if err == nil {
				t.Fatal("err = nil, want a document error")
			}
			if len(got) != 0 {
				t.Fatalf("alerts = %+v, want none", got)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("err = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

// TestParseRuleAlertsRejectsDocumentWithoutAlertsKey covers the fail-open hole a jq-based
// rule falls into when its filter stops matching: `null` and `{}` decode without error, and
// treating either as an empty alert set reports a clean gate for a rule that evaluated
// nothing. An explicit empty list stays valid -- that is a rule saying it found nothing.
func TestParseRuleAlertsRejectsDocumentWithoutAlertsKey(t *testing.T) {
	for _, doc := range []string{`null`, `{}`} {
		if _, _, err := parseRuleAlerts([]byte(doc), "demo"); err == nil {
			t.Fatalf("parseRuleAlerts(%s) = nil error; a document with no alerts key must fail the gate", doc)
		}
	}
	if _, _, err := parseRuleAlerts([]byte(`{"alerts":[]}`), "demo"); err != nil {
		t.Fatalf("an explicit empty alert set must stay valid, got %v", err)
	}
}
