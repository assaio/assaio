package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestRateLimiterBoundsOneSecret(t *testing.T) {
	l := newRateLimiter(3)
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	for i := range 3 {
		if !l.allow("k", now) {
			t.Fatalf("request %d was refused inside the budget", i+1)
		}
	}
	if l.allow("k", now) {
		t.Fatal("a fourth request inside the minute was allowed")
	}
	if !l.allow("k", now.Add(time.Minute)) {
		t.Fatal("the window did not reset after a minute")
	}
}

// TestRateLimiterKeysBySecretNotByCaller: several members legitimately share an address behind
// NAT, and one member's runaway loop must not lock out their colleagues.
func TestRateLimiterSeparatesSecrets(t *testing.T) {
	l := newRateLimiter(1)
	now := time.Now()
	if !l.allow("alice", now) || !l.allow("bob", now) {
		t.Fatal("two secrets shared one budget")
	}
	if l.allow("alice", now) {
		t.Fatal("alice's second request was allowed past her budget")
	}
}

// TestRateLimiterForgetsExpiredWindows keeps the map bounded by secrets seen in the last
// minute; an unauthenticated caller can otherwise grow it forever by varying the header.
func TestRateLimiterForgetsExpiredWindows(t *testing.T) {
	l := newRateLimiter(10)
	base := time.Now()
	for i := range 100 {
		l.allow(string(rune('a'+i%26))+string(rune('a'+i)), base)
	}
	l.allow("later", base.Add(2*time.Minute))
	if len(l.windows) != 1 {
		t.Fatalf("windows = %d, want only the live one kept", len(l.windows))
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	l := newRateLimiter(-1)
	now := time.Now()
	for range 1000 {
		if !l.allow("k", now) {
			t.Fatal("a negative limit did not disable the check")
		}
	}
}

// TestServerRefusesPastTheLimit proves the middleware is wired, not just written.
func TestServerRefusesPastTheLimit(t *testing.T) {
	s, _ := newTestServer(t)
	s = s.WithRateLimit(2)
	var last int
	for range 5 {
		last = doDashboardGet(s, testToken).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("status after five requests against a budget of two = %d, want 429", last)
	}
}

// TestAnUnknownSecretCannotBuyAFreshBudget is the bypass the first cut shipped: keying on the
// presented string handed a caller varying its header a new window per request, which is not a
// rate limit, and grew the window map by one entry per header under the same mutex.
func TestAnUnknownSecretCannotBuyAFreshBudget(t *testing.T) {
	s, _ := newTestServer(t)
	s = s.WithRateLimit(2)
	var last int
	for i := range 6 {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer guess-"+strconv.Itoa(i))
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		last = rec.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("six requests with six different unknown secrets ended in %d, want 429", last)
	}
}

// TestHealthzIsNeverRateLimited: it is what an orchestrator polls, and a liveness probe that
// fails because of unrelated traffic restarts a healthy server.
func TestHealthzIsNeverRateLimited(t *testing.T) {
	s, _ := newTestServer(t)
	s = s.WithRateLimit(1)
	for range 20 {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("healthz = %d, want 200 on every probe", rec.Code)
		}
	}
}
