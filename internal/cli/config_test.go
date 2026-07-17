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
