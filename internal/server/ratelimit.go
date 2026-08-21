package server

import (
	"net/http"
	"sync"
	"time"
)

// DefaultRateLimitPerMinute is what one secret may send without configuring anything. A sync
// pushes a whole window in one request, so a client doing its job needs a handful of requests
// an hour; this leaves three orders of magnitude of headroom and still turns an unbounded
// hammering into a bounded one.
const DefaultRateLimitPerMinute = 120

// rateLimiter is a fixed-window counter keyed by the presented secret, not by IP: several
// members legitimately share an address behind NAT, and one member's runaway loop should not
// lock out their colleagues. It is deliberately not a token bucket -- this bounds abuse, it
// does not shape traffic, and a simpler mechanism is one fewer thing to get wrong.
type rateLimiter struct {
	perMinute int
	mu        sync.Mutex
	windows   map[string]*window
}

type window struct {
	started time.Time
	count   int
}

func newRateLimiter(perMinute int) *rateLimiter {
	if perMinute == 0 {
		perMinute = DefaultRateLimitPerMinute
	}
	return &rateLimiter{perMinute: perMinute, windows: map[string]*window{}}
}

// allow reports whether this key may make another request now. A negative limit disables the
// check entirely, which is what a deployment behind its own gateway configures.
func (l *rateLimiter) allow(key string, now time.Time) bool {
	if l.perMinute < 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.evict(now)
	w, ok := l.windows[key]
	if !ok || now.Sub(w.started) >= time.Minute {
		l.windows[key] = &window{started: now, count: 1}
		return true
	}
	w.count++
	return w.count <= l.perMinute
}

// evict drops windows that have expired, so the map is bounded by the number of secrets seen
// in the last minute rather than by every secret ever presented -- an unauthenticated caller
// can otherwise grow it without limit by varying the header.
func (l *rateLimiter) evict(now time.Time) {
	for key, w := range l.windows {
		if now.Sub(w.started) >= time.Minute {
			delete(l.windows, key)
		}
	}
}

// limit wraps a handler, refusing a caller past its budget with 429 and a Retry-After. The key
// is the presented secret; a request with no Authorization header is keyed as one anonymous
// caller, which is the right shape -- every such request is going to be rejected anyway, and
// the limit is what stops it being rejected a million times a second.
func (s *Server) limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := bearer(r)
		if !ok {
			key = "anonymous"
		}
		if !s.rate.allow(key, time.Now()) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
