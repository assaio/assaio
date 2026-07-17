// Package server implements assaio's team-server MVP: a self-hosted HTTP endpoint that
// a fleet of `assaio-agent sync` runs push local usage to, and that serves the
// aggregated Assay dashboard back. It is the "collect the team's usage in one place"
// backbone -- see ROADMAP.md for the fuller org-analytics server this is a foundation
// for.
//
// SECURITY (MVP boundary, read before deploying): auth is a single shared bearer token
// compared against every write request; there is no TLS and no per-member access
// control. This is intentionally not production-hardened -- it is meant to run behind a
// reverse proxy on a trusted network, not be exposed directly to the open internet. See
// Server's doc comment for the exact boundary.
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
// SECURITY (MVP boundary): the token is one shared secret compared to every write
// request (constant-time, see auth.go); there is no TLS and no per-member access
// control -- anyone holding the token can push usage as any member and anyone can read
// the aggregated dashboard. Run this behind a reverse proxy that terminates TLS, on a
// network you trust; do not expose it directly to the open internet.
type Server struct {
	store          *store.Store
	token          string
	buildDashboard DashboardBuilder
}

// New builds a Server. token is the single shared secret every write client must
// present; New does not validate it -- refusing to start with an empty token is the
// caller's job (see internal/cli/serve.go), so this package stays reusable without
// baking in a CLI-shaped policy.
func New(st *store.Store, token string, build DashboardBuilder) *Server {
	return &Server{store: st, token: token, buildDashboard: build}
}

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
