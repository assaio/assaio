// Package server implements assaio's team-server MVP: a self-hosted HTTP endpoint that
// a fleet of `assaio-agent sync` runs push local usage to, and that serves the
// aggregated Assay dashboard back. It is the "collect the team's usage in one place"
// backbone -- see ROADMAP.md for the fuller org-analytics server this is a foundation
// for.
//
// SECURITY (MVP boundary, read before deploying): there is no TLS, so run this behind a reverse
// proxy on a trusted network rather than exposing it directly. Every route requires a bearer
// token, including the dashboard. Member identity is server-derived when per-member tokens are
// configured and client-asserted otherwise. See Server's doc comment for the exact boundary.
package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// shutdownGrace bounds how long Run waits for in-flight requests to finish once ctx is
// canceled.
const shutdownGrace = 5 * time.Second

// readHeaderTimeout bounds how long a client may take to send request headers, so a
// stalled client can't hold a connection open indefinitely (gosec G112).
const readHeaderTimeout = 10 * time.Second

// readTimeout bounds how long a client may take to send the full request, headers and
// body, including a large usage push (see maxUsageBodyBytes).
const readTimeout = 30 * time.Second

// writeTimeout bounds one request end-to-end -- from the end of its headers to the end
// of its response -- so a slow client or a stuck handler can't hold a connection open
// indefinitely.
const writeTimeout = 60 * time.Second

// idleTimeout bounds how long a keep-alive connection may sit idle between requests.
const idleTimeout = 120 * time.Second

// Server serves the team-server MVP: usage ingestion and the aggregated dashboard, both
// backed by one central *store.Store.
//
// SECURITY (MVP boundary): there is no TLS -- run this behind a reverse proxy that terminates
// it, on a network you trust. Reads are authenticated as of v0.24, and identity has two modes:
// with per-member tokens (WithMembers) the member is derived from the presented secret and
// cannot be asserted by the client; with a single shared token, any holder can still push as
// any member, which is what the operator diagnostics say out loud.
type Server struct {
	store          *store.Store
	token          string
	members        Members
	rate           *rateLimiter
	buildDashboard DashboardBuilder
}

// New builds a Server in shared-token mode: one secret for everyone, and the member name is
// whatever the client asserts. New does not validate the token -- refusing to start without one
// is the caller's job (see internal/cli/serve.go), so this package stays reusable without baking
// in a CLI-shaped policy.
func New(st *store.Store, token string, build DashboardBuilder) *Server {
	return &Server{store: st, token: token, buildDashboard: build, rate: newRateLimiter(0)}
}

// WithRateLimit sets how many requests one secret may make per minute. A negative value
// disables the limit for a deployment that bounds traffic somewhere else.
func (s *Server) WithRateLimit(perMinute int) *Server {
	s.rate = newRateLimiter(perMinute)
	return s
}

// WithMembers puts the server in server-derived identity mode: the member is whoever holds the
// secret, decided from the presented token and never read from the request body. Validate the
// set first; an invalid one is a configuration error the caller reports, not a runtime surprise.
func (s *Server) WithMembers(m Members) *Server {
	s.members = m
	return s
}

// Identity reports how this server decides who a request is, for the operator diagnostics that
// have to be able to say which of the two it is running in.
func (s *Server) Identity() Identity { return s.members.Mode() }

// Run listens on addr until ctx is canceled, then shuts down gracefully, waiting up to
// shutdownGrace for in-flight requests before returning.
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := newHTTPServer(addr, s.Handler())

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// newHTTPServer builds the *http.Server Run listens with. Every timeout is set -- a
// stalled or slow client, or a connection left idle, can never hold a socket open
// indefinitely.
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
