package plugin

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/config"
)

func TestResolveAbsoluteCommand(t *testing.T) {
	abs := script(t, "good.sh")
	cfg, err := Resolve(config.PluginConfig{Name: "demo", Command: abs, Timeout: "5s"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Command != abs {
		t.Fatalf("Command = %q, want %q", cfg.Command, abs)
	}
	if cfg.Timeout != 5*time.Second {
		t.Fatalf("Timeout = %s, want 5s", cfg.Timeout)
	}
}

func TestResolveDefaultTimeout(t *testing.T) {
	cfg, err := Resolve(config.PluginConfig{Name: "demo", Command: script(t, "good.sh")})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 60*time.Second {
		t.Fatalf("Timeout = %s, want 60s default", cfg.Timeout)
	}
}

func TestResolveLooksUpNonAbsoluteCommand(t *testing.T) {
	_, err := Resolve(config.PluginConfig{Name: "demo", Command: "sh"})
	if err != nil {
		t.Fatalf("Resolve() err = %v, want sh to resolve via PATH", err)
	}
}

func TestResolveUnknownCommand(t *testing.T) {
	for name, command := range map[string]string{
		"not on PATH":                "assaio-plugin-does-not-exist",
		"absolute path missing":      filepath.Join(t.TempDir(), "missing"),
		"absolute path not runnable": notExecutable(t),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Resolve(config.PluginConfig{Name: "demo", Command: command}); err == nil {
				t.Fatalf("Resolve(%q) err = nil, want a lookup failure before anything is started", command)
			}
		})
	}
}

func notExecutable(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "plugin")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveRejectsInvalidName(t *testing.T) {
	_, err := Resolve(config.PluginConfig{Name: "Bad Name", Command: "sh"})
	if err == nil {
		t.Fatal("Resolve() err = nil, want name validation failure")
	}
}
