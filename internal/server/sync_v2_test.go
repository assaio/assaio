package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

const (
	testDigest = "member-v2-11111111111111111111111111111111"
	bobDigest  = "member-v2-22222222222222222222222222222222"
)

func TestSyncRecordV2NumericRoundTrip(t *testing.T) {
	at, err := time.Parse(time.RFC3339, "2026-10-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	local := usage.Record{
		Tool: "claude-code", SessionID: "s", Timestamp: at, Model: "m",
		InputTokens: 101, OutputTokens: 102, CacheReadTokens: 103,
		CacheWriteTokens: 104, CacheWrite1hTokens: 105, ReasoningTokens: 106,
		DedupeKey: "k", Project: "project", Subpath: "app", Entrypoint: "cli",
		Granularity: "turn", LinesAdded: 107, LinesRemoved: 108, Edits: 109,
		ToolCalls: 110, Rejected: 111, Compactions: 112, ToolReads: 113,
		ToolSearches: 114, ToolCommands: 115, ToolWrites: 116, ToolOther: 117,
		ToolErrors: 118, Sidechain: 119, ReworkLines: 120,
		CacheMissReason: "model_changed", Skill: "review", Agent: "worker",
		Member: "alice", GitBranch: "secret/branch",
	}
	wire, err := json.Marshal(NewSyncRecordV2(&local))
	if err != nil {
		t.Fatal(err)
	}
	var decoded SyncRecordV2
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	local.Member, local.GitBranch = "", ""
	if got := decoded.usageRecord(); !reflect.DeepEqual(got, local) {
		t.Fatalf("v2 roundtrip changed a field: got %+v, want %+v", got, local)
	}
}

func doV2Push(s *Server, token string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v2/usage", bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func v2Body(t *testing.T, digest string, records ...SyncRecordV2) []byte {
	t.Helper()
	body, err := json.Marshal(usagePushV2{Protocol: 2, MemberDigest: digest, Records: records})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestV2PushStoresBranchFreeRowsAndRestates(t *testing.T) {
	s, st := newTestServer(t)
	s.WithMembers(Members{testDigest: aliceToken})
	input := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "m",
		DedupeKey: "k1", Granularity: "turn", InputTokens: 7, OutputTokens: 3,
		CacheWriteTokens: 2, CacheWrite1hTokens: 1, ToolCalls: 2, ToolReads: 1,
		ToolWrites: 1, GitBranch: "private/branch", Member: "alice",
	}
	good := NewSyncRecordV2(&input)
	body := v2Body(t, testDigest, good)
	if bytes.Contains(body, []byte("GitBranch")) || bytes.Contains(body, []byte("Member\"")) || bytes.Contains(body, []byte("private/branch")) {
		t.Fatalf("v2 wire carries forbidden identity: %s", body)
	}
	for attempt, wantInserted := range []int{1, 0} {
		rr := doV2Push(s, aliceToken, body)
		if rr.Code != http.StatusOK {
			t.Fatalf("attempt %d status=%d body=%s", attempt, rr.Code, rr.Body.String())
		}
		var result usagePushResultV2
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Protocol != 2 || result.Inserted != wantInserted || result.Received != 1 {
			t.Fatalf("attempt %d result=%+v", attempt, result)
		}
	}
	recs, err := st.Export(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Member != testDigest || recs[0].DedupeKey != testDigest+":k1" || recs[0].GitBranch != "" ||
		recs[0].InputTokens != 7 || recs[0].CacheWrite1hTokens != 1 || recs[0].ToolWrites != 1 {
		t.Fatalf("stored rows = %+v", recs)
	}
}

func TestV2RefusesLegacyAndMisboundIdentity(t *testing.T) {
	s, _ := newTestServer(t)
	input := v2Body(t, testDigest)
	if got := doV2Push(s, testToken, input).Code; got != http.StatusConflict {
		t.Fatalf("shared-token v2 status=%d, want 409", got)
	}
	s.WithMembers(Members{testDigest: aliceToken})
	if got := doV2Push(s, bobToken, input).Code; got != http.StatusUnauthorized {
		t.Fatalf("unknown-token v2 status=%d, want 401", got)
	}
	if got := doV2Push(s, aliceToken, v2Body(t, "member-v2-22222222222222222222222222222222")).Code; got != http.StatusForbidden {
		t.Fatalf("mismatched digest status=%d, want 403", got)
	}
	s.WithMembers(Members{"alice": aliceToken})
	if got := doV2Push(s, aliceToken, input).Code; got != http.StatusConflict {
		t.Fatalf("unmigrated member config status=%d, want 409", got)
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/usage", strings.NewReader(`{"member":"alice","records":[]}`))
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUpgradeRequired {
		t.Fatalf("v1 status=%d, want 426", rr.Code)
	}
}

func TestV2RejectsUnknownAndIdentityFields(t *testing.T) {
	s, st := newTestServer(t)
	s.WithMembers(Members{testDigest: aliceToken})
	base := `{"protocol":2,"memberDigest":"` + testDigest + `","records":[{"Tool":"claude-code","SessionID":"s","Timestamp":"2026-10-01T00:00:00Z","Model":"m","DedupeKey":"k","Granularity":"turn",%s}]}`
	for _, field := range []string{`"Member":"alice"`, `"GitBranch":"feature/alice"`, `"Extra":1`} {
		body := []byte(strings.Replace(base, "%s", field, 1))
		if rr := doV2Push(s, aliceToken, body); rr.Code != http.StatusBadRequest {
			t.Fatalf("field %s status=%d, want 400", field, rr.Code)
		}
	}
	recs, err := st.Export(context.Background(), time.Time{})
	if err != nil || len(recs) != 0 {
		t.Fatalf("rejected v2 body wrote rows=%+v err=%v", recs, err)
	}
}
