package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/i18n"
)

func runExplainCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"explain"}, args...))
	err := root.Execute()
	return out.String(), err
}

// TestEveryValidatorHasAnExplainPage is the structural guard: the pages live in the i18n
// catalog rather than on the Validator interface, so nothing in the type system forces a
// new metric to ship one. This test does.
func TestEveryValidatorHasAnExplainPage(t *testing.T) {
	for _, v := range analyze.Validators() {
		page, ok := i18n.Explain(v.Name())
		if !ok || strings.TrimSpace(page) == "" {
			t.Errorf("validator %q has no explain page; add one in internal/i18n", v.Name())
		}
	}
}

// TestNoOrphanedExplainPages catches the other direction: a page for a metric that no
// longer exists is dead content nobody will ever see.
func TestNoOrphanedExplainPages(t *testing.T) {
	known := make(map[string]bool)
	for _, v := range analyze.Validators() {
		known[v.Name()] = true
	}
	for name := range i18n.For("").Explain {
		if !known[name] {
			t.Errorf("explain page %q has no matching validator", name)
		}
	}
}

// TestExplainPageOpensWithTheValidatorTitle keeps the page and the metric consistent: a
// page whose heading disagrees with `analyze --list` describes something else.
func TestExplainPageOpensWithTheValidatorTitle(t *testing.T) {
	for _, v := range analyze.Validators() {
		page, ok := i18n.Explain(v.Name())
		if !ok {
			continue
		}
		if first, _, _ := strings.Cut(page, "\n"); first != v.Title() {
			t.Errorf("explain %q opens with %q, want the validator title %q", v.Name(), first, v.Title())
		}
	}
}

func TestExplainPrintsThePage(t *testing.T) {
	out, err := runExplainCmd(t, "friction")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Friction", "What it measures", "Limits"} {
		if !strings.Contains(out, want) {
			t.Errorf("explain friction output missing %q", want)
		}
	}
}

func TestExplainWithNoArgumentListsEveryMetric(t *testing.T) {
	out, err := runExplainCmd(t)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range analyze.Validators() {
		if !strings.Contains(out, v.Name()) {
			t.Errorf("explain listing is missing %q", v.Name())
		}
	}
}

func TestExplainUnknownMetricNamesTheValidOnes(t *testing.T) {
	_, err := runExplainCmd(t, "no-such-metric")
	if err == nil {
		t.Fatal("an unknown metric must error")
	}
	if !strings.Contains(err.Error(), "adoption") {
		t.Errorf("error must list the valid names, got %q", err)
	}
}

// TestExplainNeedsNoStore keeps explain usable before any data exists: it is
// documentation, so opening the store would be both pointless and a side effect.
func TestExplainNeedsNoStore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := runExplainCmd(t, "adoption"); err != nil {
		t.Fatalf("explain must work with no store: %v", err)
	}
}
