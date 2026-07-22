package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfigShow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"config"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "since") || !strings.Contains(out.String(), "format") {
		t.Fatalf("config output = %q", out.String())
	}
}

func TestConfigShowWarnsOnInvalidFormat(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ASSAIO_FORMAT", "bogus")
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"config"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "warning:") {
		t.Fatalf("config output = %q, want a warning line", out.String())
	}
}

// TestReportEnforcesConfigValidation is the counterpart to the warn-only `config` behavior
// above: every other command refuses to run on an invalid config, so an honesty-relevant
// typo (here an invalid format, standing in for a misspelled pricing.mode) can't silently
// apply with reports carrying on as if the setting were valid.
func TestReportEnforcesConfigValidation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("ASSAIO_FORMAT", "bogus")
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"report"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected report to reject an invalid config value")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Fatalf("error = %q, want an invalid-format validation error", err)
	}
}
