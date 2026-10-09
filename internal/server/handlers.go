package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/assaio/assaio/internal/dashboard"
	"github.com/assaio/assaio/internal/usage"
)

// maxUsageBodyBytes caps one POST /v2/usage request body. A first-time full-history
// sync from one active member's local store can legitimately run tens of MiB (measured:
// ~120k records serialize to ~60 MiB); 128 MiB leaves headroom for that while still
// bounding worst-case abuse. Auth is checked before this cap is even reached -- see
// handleUsageV2 -- so this is a resource bound, not a security boundary by itself.
const maxUsageBodyBytes = 128 << 20

// Handler returns the Server's routes: usage ingestion, the served dashboard, and a
// liveness probe. Every request is logged minimally to stderr.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/usage", s.handleUsageV1Upgrade)
	mux.HandleFunc("POST /v2/usage", s.handleUsageV2)
	mux.HandleFunc("GET /{$}", s.handleDashboard)
	mux.HandleFunc("GET "+healthzPath, s.handleHealthz)
	return s.limit(logRequests(mux))
}

func (s *Server) handleUsageV1Upgrade(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "sync v1 is closed; upgrade the client, configure per-member tokens, and migrate the server store to v2", http.StatusUpgradeRequired)
}

// logRequests is the Server's minimal access log: method and path, to stderr. This MVP
// has no structured logging, status capture, or request IDs. Both fields are quoted via
// strconv.Quote before logging so a crafted method/path can't inject fake
// newline-delimited log lines.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		safeMethod, safePath := strconv.Quote(r.Method), strconv.Quote(r.URL.Path)
		log.Printf("%s %s", safeMethod, safePath)
		next.ServeHTTP(w, r)
	})
}

// healthzPath is the one route with no token and no rate limit: an orchestrator's liveness
// probe, which reads nothing and must not be able to fail because of unrelated traffic.
const healthzPath = "/healthz"

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

type usagePushV2 struct {
	Protocol     int            `json:"protocol"`
	MemberDigest string         `json:"memberDigest"`
	Records      []SyncRecordV2 `json:"records"`
}

type usagePushResultV2 struct {
	Protocol int `json:"protocol"`
	Inserted int `json:"inserted"`
	Received int `json:"received"`
}

func (s *Server) handleUsageV2(w http.ResponseWriter, r *http.Request) {
	presented, ok := bearer(r)
	if !ok || !s.authenticated(presented) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.members.Mode() != ServerDerived {
		http.Error(w, "sync v2 requires one token per member", http.StatusConflict)
		return
	}
	member, err := s.memberFor(presented)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := ValidateSyncMemberDigest(member); err != nil {
		http.Error(w, "configure server.members with keyed v2 digests", http.StatusConflict)
		return
	}
	version, err := s.store.SyncProtocolVersion(r.Context())
	if err != nil {
		log.Printf("read sync protocol: %v", err)
		http.Error(w, "failed to read sync protocol", http.StatusInternalServerError)
		return
	}
	if version != 2 {
		http.Error(w, "server store needs offline sync identity migration before v2 writes", http.StatusConflict)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUsageBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var push usagePushV2
	if err := decoder.Decode(&push); err != nil {
		http.Error(w, "malformed v2 request body", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(w, "malformed v2 request body", http.StatusBadRequest)
		return
	}
	if push.Protocol != 2 || push.MemberDigest != member {
		http.Error(w, "sync protocol or member digest does not match token", http.StatusForbidden)
		return
	}
	recs := make([]usage.Record, len(push.Records))
	for i := range push.Records {
		recs[i] = push.Records[i].usageRecord()
		if err := validateRecord(&recs[i]); err != nil {
			http.Error(w, fmt.Sprintf("invalid record %d: %v", i, err), http.StatusBadRequest)
			return
		}
		recs[i].Member = member
		recs[i].DedupeKey = member + ":" + recs[i].DedupeKey
	}
	inserted, err := s.store.InsertSynced(r.Context(), recs)
	if err != nil {
		log.Printf("insert v2 usage: %v", err)
		http.Error(w, "failed to store usage", http.StatusInternalServerError)
		return
	}
	writeJSON(w, usagePushResultV2{Protocol: 2, Inserted: inserted, Received: len(recs)})
}

// handleDashboard serves the aggregated Assay dashboard built fresh from the central store on
// every request -- this MVP has no caching. It requires a bearer token as of v0.24: the page
// carries a whole team's usage, and leaving the one route that renders it open while guarding
// the route that writes it protected the wrong direction. The error path still leaks no
// internal detail, because a caller holding a token is not thereby trusted with a schema.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedReader(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	data, err := s.buildDashboard(r.Context(), s.store)
	if err != nil {
		log.Printf("build dashboard: %v", err)
		http.Error(w, "failed to build dashboard", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboard.RenderHTML(w, data); err != nil {
		log.Printf("render dashboard: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
