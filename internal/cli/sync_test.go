package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/server"
)

const testSyncIdentityKey = "1111111111111111111111111111111111111111111111111111111111111111"

func setSyncIdentityKey(t *testing.T) { t.Setenv("ASSAIO_SYNC_IDENTITY_KEY", testSyncIdentityKey) }

// syncCapture is what a fake team server records about the last /v2/usage push it
// received, for assertions in the tests below.
type syncCapture struct {
	auth string
	path string
	body struct {
		Protocol     int                   `json:"protocol"`
		MemberDigest string                `json:"memberDigest"`
		Records      []server.SyncRecordV2 `json:"records"`
	}
}

func newFakeSyncServer(t *testing.T, captured *syncCapture, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.auth = r.Header.Get("Authorization")
		captured.path = r.URL.Path
		if status != http.StatusOK {
			http.Error(w, "denied", status)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&captured.body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{
			"protocol": 2,
			"inserted": len(captured.body.Records),
			"received": len(captured.body.Records),
		})
	}))
}

func TestSyncPushesRecordsAndDerivesPseudonymMember(t *testing.T) {
	setSyncIdentityKey(t)
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
	if captured.path != "/v2/usage" || captured.body.Protocol != 2 {
		t.Fatalf("path=%q protocol=%d, want v2", captured.path, captured.body.Protocol)
	}
	if !regexp.MustCompile(`^member-v2-[0-9a-f]{32}$`).MatchString(captured.body.MemberDigest) {
		t.Fatalf("member = %q, want keyed v2 digest", captured.body.MemberDigest)
	}
	if !strings.Contains(out.String(), "sent 1") || !strings.Contains(out.String(), "inserted 1") {
		t.Fatalf("stdout = %q, want mention of sent/inserted counts", out.String())
	}
}

func TestSyncMemberFlagOverridesPseudonym(t *testing.T) {
	setSyncIdentityKey(t)
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

	want, err := syncMemberDigest("alice")
	if err != nil {
		t.Fatal(err)
	}
	if captured.body.MemberDigest != want {
		t.Fatalf("member = %q, want keyed digest %q", captured.body.MemberDigest, want)
	}
	if !strings.Contains(out.String(), "synced as "+want) {
		t.Fatalf("stdout = %q, want to mention keyed digest", out.String())
	}
}

func TestSyncWrongTokenReturns401Error(t *testing.T) {
	setSyncIdentityKey(t)
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
	setSyncIdentityKey(t)
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

func TestSyncPrintMemberDigestWithoutARequest(t *testing.T) {
	setSyncIdentityKey(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"sync", "--member", "alice", "--print-member-digest"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	want, err := syncMemberDigest("alice")
	if err != nil || strings.TrimSpace(out.String()) != want {
		t.Fatalf("printed %q, want digest %q (err=%v)", out.String(), want, err)
	}
}

func TestSyncRefusesOldServerWithoutV1Fallback(t *testing.T) {
	setSyncIdentityKey(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	seedDashboardStore(t)
	var paths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer ts.Close()
	root := NewRootCmd()
	root.SetArgs([]string{"sync", "--server", ts.URL, "--token", "t"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "upgrade the server") {
		t.Fatalf("error = %v, want actionable old-server refusal", err)
	}
	if len(paths) != 1 || paths[0] != "/v2/usage" {
		t.Fatalf("requests = %+v, want only /v2/usage", paths)
	}
}
