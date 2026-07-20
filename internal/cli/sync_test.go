package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

// syncCapture is what a fake team server records about the last /v1/usage push it
// received, for assertions in the tests below.
type syncCapture struct {
	auth string
	body struct {
		Member  string         `json:"member"`
		Records []usage.Record `json:"records"`
	}
}

func newFakeSyncServer(t *testing.T, captured *syncCapture, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.auth = r.Header.Get("Authorization")
		if status != http.StatusOK {
			http.Error(w, "denied", status)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&captured.body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{
			"inserted": len(captured.body.Records),
			"received": len(captured.body.Records),
		})
	}))
}

func TestSyncPushesRecordsAndDerivesPseudonymMember(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t) // seeds one local usage record

	var captured syncCapture
	ts := newFakeSyncServer(t, &captured, http.StatusOK)
	defer ts.Close()

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--server", ts.URL, "--token", "sekret"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync err = %v, output=%s", err, out.String())
	}

	if captured.auth != "Bearer sekret" {
		t.Fatalf("Authorization = %q, want Bearer sekret", captured.auth)
	}
	if len(captured.body.Records) != 1 {
		t.Fatalf("pushed %d records, want 1", len(captured.body.Records))
	}
	if !regexp.MustCompile(`^member-[0-9a-f]{10}$`).MatchString(captured.body.Member) {
		t.Fatalf("member = %q, want an auto-derived 40-bit pseudonym member-xxxxxxxxxx", captured.body.Member)
	}
	if !strings.Contains(out.String(), "sent 1") || !strings.Contains(out.String(), "inserted 1") {
		t.Fatalf("stdout = %q, want mention of sent/inserted counts", out.String())
	}
}

func TestSyncMemberFlagOverridesPseudonym(t *testing.T) {
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
	root.SetArgs([]string{"sync", "--server", ts.URL, "--token", "sekret", "--member", "alice"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync err = %v, output=%s", err, out.String())
	}

	if captured.body.Member != "alice" {
		t.Fatalf("member = %q, want alice (explicit opt-in)", captured.body.Member)
	}
	if !strings.Contains(out.String(), "synced as alice") {
		t.Fatalf("stdout = %q, want to mention synced as alice", out.String())
	}
}

func TestSyncWrongTokenReturns401Error(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)

	var captured syncCapture
	ts := newFakeSyncServer(t, &captured, http.StatusUnauthorized)
	defer ts.Close()

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--server", ts.URL, "--token", "wrong"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error on 401 from server")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error = %v, want to mention 401", err)
	}
}

func TestSyncServerDownReturnsClearError(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)

	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ts.Close() // closed immediately: nothing is listening at ts.URL anymore

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--server", ts.URL, "--token", "t"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when the server is unreachable")
	}
}

func TestSyncRequiresServer(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--token", "t"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "--server is required") {
		t.Fatalf("err = %v, want --server required error", err)
	}
}

func TestSyncRequiresToken(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"sync", "--server", "http://example.invalid"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "--token is required") {
		t.Fatalf("err = %v, want --token required error", err)
	}
}
