package dashboard

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/i18n"
)

// TestLocaleTemplateFuncReturnsCatalog asserts the template's "locale" func -- the seam a
// future language switcher hooks into -- resolves to the shared catalog today.
func TestLocaleTemplateFuncReturnsCatalog(t *testing.T) {
	fn, ok := templateFuncs["locale"].(func() i18n.Dashboard)
	if !ok {
		t.Fatal(`templateFuncs["locale"] missing or has the wrong signature`)
	}
	if got := fn(); got != i18n.For("").Dashboard {
		t.Fatalf("locale() = %+v, want the en catalog", got)
	}
}

// TestTemplateLooksUpLocaleForToggleAndProv is half of the finding-5 regression: the
// theme-toggle's aria-label and the "Prov." stamp must be looked up from the catalog. A
// hardcoded literal that happened to match the catalog's current text would pass a
// rendered-output check, so the template source itself is asserted here.
func TestTemplateLooksUpLocaleForToggleAndProv(t *testing.T) {
	for _, want := range []string{"(locale).ToggleDarkLabel", "(locale).ProvLabel"} {
		if !strings.Contains(templateSource, want) {
			t.Errorf("dashboard.html.tmpl must render %s from the locale, not a literal", want)
		}
	}
}

// TestRenderHTMLRendersLocaleValues is the other half: the looked-up strings must actually
// reach the page, so neither is dropped by a template edit.
func TestRenderHTMLRendersLocaleValues(t *testing.T) {
	l := i18n.For("").Dashboard
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if !strings.Contains(html, `aria-label="`+l.ToggleDarkLabel+`"`) {
		t.Errorf("theme toggle button must render ToggleDarkLabel (%q)", l.ToggleDarkLabel)
	}
	if !strings.Contains(html, l.ProvLabel) {
		t.Errorf("the Prov. stamp must render ProvLabel (%q)", l.ProvLabel)
	}
}
