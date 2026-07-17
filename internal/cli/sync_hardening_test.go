package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSyncMemberWithColonRejected(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)

	var captured syncCapture
	ts := newFakeSyncServer(t, &captured, http.StatusOK)
	defer ts.Close()

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--server", ts.URL, "--token", "sekret", "--member", "team:alice"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --member contains ':'")
	}
	if captured.auth != "" {
		t.Fatalf("server received a request for an invalid member, want sync to reject it locally before sending anything: auth=%q", captured.auth)
	}
}

func TestIsCleartextRemote(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"http localhost by name", "http://localhost:8787", false},
		{"http loopback 127.0.0.1", "http://127.0.0.1:8787", false},
		{"http loopback range 127.0.0.1/8", "http://127.5.5.5:8787", false},
		{"http loopback IPv6", "http://[::1]:8787", false},
		{"https remote host", "https://assaio.internal:8787", false},
		{"http remote hostname", "http://assaio.internal:8787", true},
		{"http remote IP", "http://10.0.0.5:8787", true},
		{"malformed url", "://bad", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCleartextRemote(tt.url); got != tt.want {
				t.Fatalf("isCleartextRemote(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestSyncHTTPClientHasTimeout(t *testing.T) {
	if syncHTTPClient.Timeout <= 0 {
		t.Fatal("syncHTTPClient must have a positive Timeout; http.DefaultClient's zero timeout can hang sync forever against an unresponsive server")
	}
}

// TestSyncCapsOversizedErrorBody proves a misbehaving server can't make pushUsage
// buffer an unbounded error response: the server writes far more than
// maxSyncErrorBodyBytes, and the resulting error message must stay bounded near that
// cap rather than growing with the response.
func TestSyncCapsOversizedErrorBody(t *testing.T) {
	huge := strings.Repeat("x", maxSyncErrorBodyBytes*2)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, huge, http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := pushUsage(context.Background(), ts.URL, "t", "alice", nil)
	if err == nil {
		t.Fatal("expected an error from a non-200 response")
	}
	if len(err.Error()) > maxSyncErrorBodyBytes+256 {
		t.Fatalf("error message is %d bytes, want it capped near maxSyncErrorBodyBytes (%d)",
			len(err.Error()), maxSyncErrorBodyBytes)
	}
}

// TestSyncAbortsPromptlyWhenContextCanceled proves the signal.NotifyContext wiring in
// runSync: canceling the command context (standing in for Ctrl-C/SIGTERM, exactly as
// TestServeListensAndShutsDownGracefully does for `serve`) while a push is in flight
// must abort it promptly instead of leaving `sync` hanging until the server responds or
// syncHTTPTimeout elapses.
func TestSyncAbortsPromptlyWhenContextCanceled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)

	block := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-block // released by the deferred cleanup below so ts.Close() doesn't hang
	}))
	defer func() {
		close(block)
		ts.Close()
	}()

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--server", ts.URL, "--token", "t"})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := root.ExecuteContext(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error when the context is canceled mid-push")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("sync took %s to abort after context cancellation, want well under syncHTTPTimeout (120s)", elapsed)
	}
}
