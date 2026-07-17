package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

const testToken = "testtok"

func newTestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st, testToken, BuildDashboard), st
}

func pushBody(t *testing.T, member string, recs []usage.Record) []byte {
	t.Helper()
	body, err := json.Marshal(usagePush{Member: member, Records: recs})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func doUsagePush(s *Server, token string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/usage", bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestHandleUsageValidTokenInsertsAndDedupes(t *testing.T) {
	s, st := newTestServer(t)
	rec := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m",
		InputTokens: 1, DedupeKey: "a1", Granularity: "turn",
	}
	body := pushBody(t, "alice", []usage.Record{rec})

	rr := doUsagePush(s, testToken, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var result usagePushResult
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 || result.Received != 1 {
		t.Fatalf("result = %+v, want Inserted=1 Received=1", result)
	}

	recs, err := st.Export(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Member != "alice" || recs[0].DedupeKey != "alice:a1" {
		t.Fatalf("stored records = %+v, want one record member=alice dedupe_key=alice:a1", recs)
	}

	// Re-push the identical payload: the member-prefixed dedupe key must dedupe it,
	// not double-insert.
	rr2 := doUsagePush(s, testToken, body)
	var result2 usagePushResult
	if err := json.Unmarshal(rr2.Body.Bytes(), &result2); err != nil {
		t.Fatal(err)
	}
	if result2.Inserted != 0 || result2.Received != 1 {
		t.Fatalf("re-push result = %+v, want Inserted=0 Received=1 (idempotent)", result2)
	}
}

func TestHandleUsageMissingTokenReturns401(t *testing.T) {
	s, _ := newTestServer(t)
	body := pushBody(t, "alice", []usage.Record{
		{Tool: "codex", SessionID: "s1", Timestamp: time.Now(), Model: "m", DedupeKey: "a1"},
	})
	rr := doUsagePush(s, "", body)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestHandleUsageWrongTokenReturns401(t *testing.T) {
	s, _ := newTestServer(t)
	body := pushBody(t, "alice", []usage.Record{
		{Tool: "codex", SessionID: "s1", Timestamp: time.Now(), Model: "m", DedupeKey: "a1"},
	})
	rr := doUsagePush(s, "wrong-token", body)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestHandleUsageMalformedBodyReturns400(t *testing.T) {
	s, _ := newTestServer(t)
	rr := doUsagePush(s, testToken, []byte("{not json"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "malformed request body" {
		t.Fatalf("body = %q, want the generic message with no decode-error detail leaked", got)
	}
}

func TestHandleUsageTwoMembersSameDedupeKeyBothPersist(t *testing.T) {
	s, st := newTestServer(t)
	rec := usage.Record{Tool: "codex", SessionID: "s1", Timestamp: time.Now(), Model: "m", DedupeKey: "shared", Granularity: "turn"}

	rrA := doUsagePush(s, testToken, pushBody(t, "alice", []usage.Record{rec}))
	rrB := doUsagePush(s, testToken, pushBody(t, "bob", []usage.Record{rec}))
	if rrA.Code != http.StatusOK || rrB.Code != http.StatusOK {
		t.Fatalf("status A=%d B=%d, want both 200", rrA.Code, rrB.Code)
	}

	recs, err := st.Export(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("stored records = %+v, want 2 (one per member, same original dedupe key)", recs)
	}
	members := map[string]bool{}
	for _, r := range recs {
		members[r.Member] = true
	}
	if !members["alice"] || !members["bob"] {
		t.Fatalf("members present = %+v, want alice and bob both present", members)
	}
}

func TestHealthzReturnsOK(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("healthz = %d %q, want 200 ok", rec.Code, rec.Body.String())
	}
}

func TestDashboardHandlerReturnsHTML(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "assay") {
		t.Fatalf("dashboard body does not mention the Assay report: %s", rec.Body.String())
	}
}

// TestDashboardHandlerIncludesTeamSectionWithMembers is the end-to-end SERVER-B proof at
// the HTTP layer: once usage pushed under two different members lands in the central
// store, GET / must render the Team section (see also TestBuildDashboardIncludesTeam
// SectionWithMembers, the same assertion one layer down against BuildDashboard directly).
func TestDashboardHandlerIncludesTeamSectionWithMembers(t *testing.T) {
	s, _ := newTestServer(t)
	rrA := doUsagePush(s, testToken, pushBody(t, "alice", []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m", InputTokens: 1, DedupeKey: "a1", Granularity: "turn"},
	}))
	rrB := doUsagePush(s, testToken, pushBody(t, "bob", []usage.Record{
		{Tool: "claude-code", SessionID: "s2", Timestamp: time.Now(), Model: "m", InputTokens: 1, DedupeKey: "b1", Granularity: "turn"},
	}))
	if rrA.Code != http.StatusOK || rrB.Code != http.StatusOK {
		t.Fatalf("status A=%d B=%d, want both 200", rrA.Code, rrB.Code)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `class="team"`) {
		t.Fatalf("GET / must include the Team section once the store has usage from more than one member: %s", body)
	}
	if strings.Contains(body, "alice") || strings.Contains(body, "bob") {
		t.Fatalf("GET / must pseudonymize member names, never show them raw: %s", body)
	}
}

func TestDashboardHandlerNoAuthRequired(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / without a token = %d, want 200 (dashboard viewing is not gated in this MVP)", rec.Code)
	}
}

// TestHandleUsageInsertFailureReturnsGenericError forces Insert to fail (by closing the
// underlying DB) without depending on any particular driver error string, and proves the
// client sees a generic message while the detail goes to the server log instead.
func TestHandleUsageInsertFailureReturnsGenericError(t *testing.T) {
	s, st := newTestServer(t)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	body := pushBody(t, "alice", []usage.Record{newValidRecord()})

	rr := doUsagePush(s, testToken, body)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "failed to store usage" {
		t.Fatalf("body = %q, want the generic message with no DB detail leaked", got)
	}
}

// TestDashboardHandlerBuildFailureReturnsGenericError is the unauthenticated-caller case
// fix 5 is strictest about: GET / must never echo internal (DB/schema) detail, even on
// failure.
func TestDashboardHandlerBuildFailureReturnsGenericError(t *testing.T) {
	s, st := newTestServer(t)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "failed to build dashboard" {
		t.Fatalf("body = %q, want the generic message with no DB detail leaked", got)
	}
}
