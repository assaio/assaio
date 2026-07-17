package dashboard

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// TestLocaleStringsNonEmpty asserts every en field is populated -- an empty chrome
// string would render as a silent gap in the report. Uses reflection so a newly added
// localeStrings field is covered automatically, with no list to keep in sync by hand.
func TestLocaleStringsNonEmpty(t *testing.T) {
	v := reflect.ValueOf(en)
	typ := v.Type()
	for i := range v.NumField() {
		if v.Field(i).String() == "" {
			t.Fatalf("localeStrings.%s is empty", typ.Field(i).Name)
		}
	}
}

// TestLocaleTemplateFuncReturnsEN asserts the template's "locale" func -- the seam a
// future language switcher hooks into -- resolves to the en locale today.
func TestLocaleTemplateFuncReturnsEN(t *testing.T) {
	fn, ok := templateFuncs["locale"].(func() localeStrings)
	if !ok {
		t.Fatal(`templateFuncs["locale"] missing or has the wrong signature`)
	}
	if got := fn(); got != en {
		t.Fatalf("locale() = %+v, want en", got)
	}
}

// TestRenderHTMLUsesLocaleForToggleAndProvStrings is the finding-5 regression: the
// theme-toggle's initial aria-label and the "Prov." caveat stamp must render from
// localeStrings, not a hardcoded literal in dashboard.html.tmpl. Changing en's value and
// re-rendering is what proves the template actually reads it, rather than merely
// happening to contain the same text.
func TestRenderHTMLUsesLocaleForToggleAndProvStrings(t *testing.T) {
	original := en
	t.Cleanup(func() { en = original })

	en.ToggleDarkLabel = "zz-toggle-marker"
	en.ProvLabel = "zz-prov-marker"

	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if !strings.Contains(html, `aria-label="zz-toggle-marker"`) {
		t.Fatalf("theme toggle button must render ToggleDarkLabel from locale: %s", html)
	}
	if !strings.Contains(html, "zz-prov-marker") {
		t.Fatalf(`the "Prov." stamp must render ProvLabel from locale: %s`, html)
	}
}
