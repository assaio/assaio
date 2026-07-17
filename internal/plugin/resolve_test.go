package plugin

import (
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
	_, err := Resolve(config.PluginConfig{Name: "demo", Command: "assaio-plugin-does-not-exist"})
	if err == nil {
		t.Fatal("Resolve() err = nil, want lookup failure")
	}
}

func TestResolveRejectsInvalidName(t *testing.T) {
	_, err := Resolve(config.PluginConfig{Name: "Bad Name", Command: "sh"})
	if err == nil {
		t.Fatal("Resolve() err = nil, want name validation failure")
	}
}
