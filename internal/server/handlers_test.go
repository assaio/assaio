package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

const testToken = "test-token-long-enough"

func newTestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st, testToken, BuildDashboard), st
}

func newV2TestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	s, st := newTestServer(t)
	return s.WithMembers(Members{testDigest: testToken, bobDigest: bobToken}), st
}

func pushBody(t *testing.T, member string, recs []usage.Record) []byte {
	t.Helper()
	digest := testDigest
	if member == "bob" {
		digest = bobDigest
	}
	wires := make([]SyncRecordV2, len(recs))
	for i := range recs {
		wires[i] = NewSyncRecordV2(&recs[i])
	}
	return v2Body(t, digest, wires...)
}

func doUsagePush(s *Server, token string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v2/usage", bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func TestHandleUsageValidTokenInsertsAndDedupes(t *testing.T) {
	s, st := newV2TestServer(t)
	rec := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m",
		InputTokens: 1, DedupeKey: "a1", Granularity: "turn",
	}
	body := pushBody(t, "alice", []usage.Record{rec})

	rr := doUsagePush(s, testToken, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var result usagePushResultV2
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
	if len(recs) != 1 || recs[0].Member != testDigest || recs[0].DedupeKey != testDigest+":a1" {
		t.Fatalf("stored records = %+v, want one record under the v2 digest", recs)
	}

	// Re-push the identical payload: the member-prefixed dedupe key must dedupe it,
	// not double-insert.
	rr2 := doUsagePush(s, testToken, body)
	var result2 usagePushResultV2
	if err := json.Unmarshal(rr2.Body.Bytes(), &result2); err != nil {
		t.Fatal(err)
	}
	if result2.Inserted != 0 || result2.Received != 1 {
		t.Fatalf("re-push result = %+v, want Inserted=0 Received=1 (idempotent)", result2)
	}
}

func TestHandleUsageMissingTokenReturns401(t *testing.T) {
	s, _ := newV2TestServer(t)
	body := pushBody(t, "alice", []usage.Record{
		{Tool: "codex", SessionID: "s1", Timestamp: time.Now(), Model: "m", DedupeKey: "a1"},
	})
	rr := doUsagePush(s, "", body)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestHandleUsageWrongTokenReturns401(t *testing.T) {
	s, _ := newV2TestServer(t)
	body := pushBody(t, "alice", []usage.Record{
		{Tool: "codex", SessionID: "s1", Timestamp: time.Now(), Model: "m", DedupeKey: "a1"},
	})
	rr := doUsagePush(s, "wrong-token", body)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestHandleUsageMalformedBodyReturns400(t *testing.T) {
	s, _ := newV2TestServer(t)
	rr := doUsagePush(s, testToken, []byte("{not json"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "malformed v2 request body" {
		t.Fatalf("body = %q, want the generic message with no decode-error detail leaked", got)
	}
}

func TestHandleUsageTwoMembersSameDedupeKeyBothPersist(t *testing.T) {
	s, st := newV2TestServer(t)
	rec := usage.Record{Tool: "codex", SessionID: "s1", Timestamp: time.Now(), Model: "m", DedupeKey: "shared", Granularity: "turn"}

	rrA := doUsagePush(s, testToken, pushBody(t, "alice", []usage.Record{rec}))
	rrB := doUsagePush(s, bobToken, pushBody(t, "bob", []usage.Record{rec}))
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
	if !members[testDigest] || !members[bobDigest] {
		t.Fatalf("members present = %+v, want two distinct digests", members)
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
	rec := doDashboardGet(s, testToken)
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
	s, _ := newV2TestServer(t)
	rrA := doUsagePush(s, testToken, pushBody(t, "alice", []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m", InputTokens: 1, DedupeKey: "a1", Granularity: "turn"},
	}))
	rrB := doUsagePush(s, bobToken, pushBody(t, "bob", []usage.Record{
		{Tool: "claude-code", SessionID: "s2", Timestamp: time.Now(), Model: "m", InputTokens: 1, DedupeKey: "b1", Granularity: "turn"},
	}))
	if rrA.Code != http.StatusOK || rrB.Code != http.StatusOK {
		t.Fatalf("status A=%d B=%d, want both 200", rrA.Code, rrB.Code)
	}

	rec := doDashboardGet(s, testToken)
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

// TestDashboardRequiresAToken is the v0.24 correction: the page carries a whole team's usage,
// and leaving the one route that renders it open while guarding the route that writes it
// protected the wrong direction.
func TestDashboardRequiresAToken(t *testing.T) {
	s, _ := newTestServer(t)
	if code := doDashboardGet(s, "").Code; code != http.StatusUnauthorized {
		t.Fatalf("GET / without a token = %d, want 401", code)
	}
	if code := doDashboardGet(s, "wrong-token").Code; code != http.StatusUnauthorized {
		t.Fatalf("GET / with the wrong token = %d, want 401", code)
	}
	if code := doDashboardGet(s, testToken).Code; code != http.StatusOK {
		t.Fatalf("GET / with the token = %d, want 200", code)
	}
}

// doDashboardGet reads the dashboard with the given bearer token; an empty token sends no
// Authorization header at all.
func doDashboardGet(s *Server, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

// The protocol read must not leak a database error to an authenticated client.
func TestHandleUsageProtocolReadFailureReturnsGenericError(t *testing.T) {
	s, st := newV2TestServer(t)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	body := pushBody(t, "alice", []usage.Record{newValidRecord()})

	rr := doUsagePush(s, testToken, body)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "failed to read sync protocol" {
		t.Fatalf("body = %q, want the generic message with no DB detail leaked", got)
	}
}

// TestDashboardHandlerBuildFailureReturnsGenericError: GET / must never echo internal (DB/schema)
// detail, even on failure -- a caller holding a token is not thereby trusted with a schema.
func TestDashboardHandlerBuildFailureReturnsGenericError(t *testing.T) {
	s, st := newTestServer(t)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	rec := doDashboardGet(s, testToken)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "failed to build dashboard" {
		t.Fatalf("body = %q, want the generic message with no DB detail leaked", got)
	}
}

// TestUsagePushRejectsBeforeReadingTheBody is the ordering, not the status code: both a
// pre-decode and a post-decode rejection answer 401, so only the *cost* separates them. An
// unauthenticated caller must not be able to make the server allocate a batch first.
func TestUsagePushRejectsBeforeReadingTheBody(t *testing.T) {
	s, _ := newTestServer(t)
	body := &countingReader{r: bytes.NewReader(v2Body(t, testDigest))}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v2/usage", body)
	req.Header.Set("Authorization", "Bearer wrong-token-long-enough")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if body.n != 0 {
		t.Fatalf("the server read %d body bytes before rejecting the token; it must read none", body.n)
	}
}

// countingReader records how much of the body a handler consumed.
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}
