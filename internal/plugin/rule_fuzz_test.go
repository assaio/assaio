package plugin

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// FuzzRuleAlerts asserts parseRuleAlerts' invariants over arbitrary bytes: it never
// panics, and every accepted alert is fully validated -- stamped plugin, whitelisted
// severity, required and capped strings, no control characters.
func FuzzRuleAlerts(f *testing.F) {
	f.Add([]byte(`{"alerts":[{"rule":"r","severity":"warn","message":"m","validator":"adoption"}]}`))
	f.Add([]byte(`{"alerts":[]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))
	f.Add([]byte(`{"alerts":[{"rule":"r","severity":"ERROR","message":"m"}]}`))
	f.Add([]byte("{\"alerts\":[{\"rule\":\"r\x1b[31m\",\"severity\":\"info\",\"message\":\"m\"}]}"))
	f.Add([]byte(`{"alerts":[{"rule":"r","severity":"error","message":"m"}]} {"alerts":[]}`))
	f.Add([]byte("\xff\xfe"))

	for _, doc := range catalogueSeeds(f, "rule-alerts") {
		f.Add([]byte(doc))
	}

	f.Fuzz(func(t *testing.T, doc []byte) {
		alerts, violations, err := parseRuleAlerts(doc, "demo")
		if err != nil {
			return
		}
		if len(violations) != 0 {
			t.Fatalf("nil error with %d violations", len(violations))
		}
		if len(alerts) > maxRuleAlerts {
			t.Fatalf("accepted %d alerts, over the %d cap", len(alerts), maxRuleAlerts)
		}
		for _, a := range alerts {
			if a.Plugin != "demo" {
				t.Fatalf("accepted alert has Plugin %q, want the stamped configured name", a.Plugin)
			}
			if a.Severity != SeverityInfo && a.Severity != SeverityWarn && a.Severity != SeverityError {
				t.Fatalf("accepted alert has severity %q", a.Severity)
			}
			if a.Rule == "" || a.Message == "" {
				t.Fatalf("accepted alert misses a required field: %+v", a)
			}
			if utf8.RuneCountInString(a.Rule) > maxRuleIDLen || utf8.RuneCountInString(a.Message) > maxRuleMessageLen ||
				utf8.RuneCountInString(a.Validator) > maxRuleValidatorLen {
				t.Fatalf("accepted alert exceeds a length cap: %+v", a)
			}
			for _, s := range []string{a.Rule, a.Severity, a.Message, a.Validator} {
				if strings.ContainsFunc(s, unicode.IsControl) {
					t.Fatalf("accepted alert carries a control character: %q", s)
				}
			}
		}
	})
}
