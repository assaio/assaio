package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestServeAddrDefaultsToLoopback locks in the fix: binding all interfaces by default
// made an unauthenticated dashboard (GET /) reachable from any network the host is on
// unless the operator remembered to override --addr. The default must now be loopback,
// so exposing beyond localhost is an explicit, deliberate choice.
func TestServeAddrDefaultsToLoopback(t *testing.T) {
	f := newServeCmd().Flags().Lookup("addr")
	if f == nil {
		t.Fatal("--addr flag not found")
		return // unreachable; staticcheck cannot see t.Fatal terminate through the interface
	}
	if f.DefValue != "127.0.0.1:8787" {
		t.Fatalf("--addr default = %q, want loopback 127.0.0.1:8787", f.DefValue)
	}
}

// TestServeHelpDisclosesTheIdentityBoundary proves the boundary is surfaced where an operator
// actually looks -- `serve --help` -- and not only in the package doc. The dashboard is
// authenticated as of v0.24; what remains weaker than it looks is a shared token, under which
// any holder can push as anybody.
func TestServeHelpDisclosesTheIdentityBoundary(t *testing.T) {
	long := strings.ToLower(newServeCmd().Long)
	for _, want := range []string{"no tls", "every route requires the bearer token", "push usage under any member name"} {
		if !strings.Contains(long, want) {
			t.Fatalf("serve --help does not mention %q: %q", want, long)
		}
	}
}

func TestServeRequiresToken(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"serve", "--addr", "127.0.0.1:0"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --token is missing")
	}
	if !strings.Contains(err.Error(), "--token is required") {
		t.Fatalf("error = %q", err)
	}
}

// TestServeListensAndShutsDownGracefully proves the full wiring from the CLI's context
// down to server.Run's graceful shutdown: canceling the command context (standing in
// for Ctrl-C/SIGTERM) must make `serve` return cleanly instead of hanging.
func TestServeListensAndShutsDownGracefully(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dbPath := filepath.Join(t.TempDir(), "srv.db")

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"serve", "--addr", "127.0.0.1:0", "--token", "a-long-enough-test-token", "--db", dbPath})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := root.ExecuteContext(ctx); err != nil {
		t.Fatalf("serve returned err = %v, want nil (graceful shutdown on ctx cancel); output=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "listening on") {
		t.Fatalf("stdout = %q, want a listening confirmation", out.String())
	}
	if !strings.Contains(strings.ToLower(out.String()), "client-asserted") {
		t.Fatalf("stdout = %q, want the startup note to name the identity mode it is running in", out.String())
	}
}
