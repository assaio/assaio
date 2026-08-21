package server

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

const (
	aliceToken = "alice-token-long-enough"
	bobToken   = "bob-token-long-enough-1"
)

func memberServer(t *testing.T) *Server {
	t.Helper()
	s, _ := newTestServer(t)
	members := Members{"alice": aliceToken, "bob": bobToken}
	if err := members.Validate(); err != nil {
		t.Fatal(err)
	}
	return s.WithMembers(members)
}

// TestServerDerivedIdentityIgnoresTheAssertedMember is the spoofing fix: the dedupe-key prefix
// has always assumed each row has exactly one possible writer, and until now nothing enforced
// it -- any token holder could push as anybody.
func TestServerDerivedIdentityIgnoresTheAssertedMember(t *testing.T) {
	s := memberServer(t)
	rec := doUsagePush(s, bobToken, pushBody(t, "alice", []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m", InputTokens: 1, DedupeKey: "k1", Granularity: "turn"},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: bob's token is valid", rec.Code)
	}

	rows, err := s.store.Usage(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Member != "bob" {
		t.Fatalf("stored member = %+v, want bob -- the token holder, not the name in the body", rows)
	}
}

func TestServerDerivedIdentityRejectsAnUnknownToken(t *testing.T) {
	s := memberServer(t)
	rec := doUsagePush(s, "not-a-configured-token", pushBody(t, "alice", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// TestSharedTokenModeStillWorksAndSaysSo keeps the existing deployment running while making the
// weaker guarantee legible rather than implicit.
func TestSharedTokenModeStillWorksAndSaysSo(t *testing.T) {
	s, _ := newTestServer(t)
	if s.Identity() != ClientAsserted {
		t.Fatalf("Identity = %v, want client-asserted without configured members", s.Identity())
	}
	if !strings.Contains(s.Identity().String(), "shared token") {
		t.Fatalf("Identity string = %q, want it to name the shared-token mode", s.Identity())
	}
	rec := doUsagePush(s, testToken, pushBody(t, "alice", []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m", InputTokens: 1, DedupeKey: "k1", Granularity: "turn"},
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want the shared-token path to keep working", rec.Code)
	}
}

func TestMembersValidateRejectsASharedSecret(t *testing.T) {
	err := Members{"alice": aliceToken, "bob": aliceToken}.Validate()
	if err == nil || !strings.Contains(err.Error(), "share a token") {
		t.Fatalf("Validate error = %v, want two members sharing a secret refused", err)
	}
}

func TestMembersValidateRejectsAShortToken(t *testing.T) {
	err := Members{"alice": "short"}.Validate()
	if err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("Validate error = %v, want a short token refused", err)
	}
}

func TestMembersValidateRejectsAnUnusableName(t *testing.T) {
	err := Members{"alice smith": aliceToken}.Validate()
	if err == nil {
		t.Fatal("a member name the store cannot key on was accepted")
	}
}

// TestMemberTokenReadsTheDashboard: a configured member holds a secret, so a member may read.
func TestMemberTokenReadsTheDashboard(t *testing.T) {
	s := memberServer(t)
	if code := doDashboardGet(s, aliceToken).Code; code != http.StatusOK {
		t.Fatalf("GET / with a member token = %d, want 200", code)
	}
	if code := doDashboardGet(s, "nope").Code; code != http.StatusUnauthorized {
		t.Fatalf("GET / with an unknown token = %d, want 401", code)
	}
}
