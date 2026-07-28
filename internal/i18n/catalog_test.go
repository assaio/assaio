package i18n

import (
	"reflect"
	"strings"
	"testing"
)

// TestEveryStringIsPopulated asserts no catalog field is empty -- an empty chrome string
// renders as a silent gap. Reflection keeps a newly added field covered automatically,
// with no list to maintain by hand.
func TestEveryStringIsPopulated(t *testing.T) {
	surfaces := map[string]any{
		"Dashboard": For("").Dashboard,
		"CLI":       For("").CLI,
	}
	for name, surface := range surfaces {
		v := reflect.ValueOf(surface)
		typ := v.Type()
		for i := range v.NumField() {
			if v.Field(i).String() == "" {
				t.Errorf("%s.%s is empty", name, typ.Field(i).Name)
			}
		}
	}
}

func TestForFallsBackToEnglish(t *testing.T) {
	for _, locale := range []string{"", "en", "pl", "nonsense"} {
		if got := For(locale); got.Dashboard != en.Dashboard {
			t.Errorf("For(%q) must fall back to English until another locale exists", locale)
		}
	}
}

// TestExplainPagesAreSubstantial guards against a page being reduced to a stub: the point
// of explain is the long form, and a one-liner already exists as the validator's Describe.
func TestExplainPagesAreSubstantial(t *testing.T) {
	for name, page := range For("").Explain {
		if len(strings.TrimSpace(page)) < 200 {
			t.Errorf("explain page %q is too short to be a long-form page", name)
		}
		for _, section := range []string{"What it measures", "How to read it", "What to do about it", "Limits"} {
			if !strings.Contains(page, section) {
				t.Errorf("explain page %q is missing the %q section", name, section)
			}
		}
	}
}

func TestExplainLookup(t *testing.T) {
	if _, ok := Explain("adoption"); !ok {
		t.Error(`Explain("adoption") must resolve`)
	}
	if _, ok := Explain("no-such-validator"); ok {
		t.Error("Explain must report a miss for an unknown name")
	}
}
