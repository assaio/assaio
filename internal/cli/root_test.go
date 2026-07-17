package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/version"
)

func TestVersionCommand(t *testing.T) {
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		t.Fatal("version command produced no output")
	}
}

func TestVersionFlag(t *testing.T) {
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), version.Version) {
		t.Fatalf("--version output = %q, want to contain %q", out.String(), version.Version)
	}
}
