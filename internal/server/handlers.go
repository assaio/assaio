package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/assaio/assaio/internal/dashboard"
	"github.com/assaio/assaio/internal/usage"
)

// maxUsageBodyBytes caps one POST /v1/usage request body. A first-time full-history
// sync from one active member's local store can legitimately run tens of MiB (measured:
// ~120k records serialize to ~60 MiB); 128 MiB leaves headroom for that while still
// bounding worst-case abuse. Auth is checked before this cap is even reached -- see
// handleUsage -- so this is a resource bound, not a security boundary by itself.
const maxUsageBodyBytes = 128 << 20

// Handler returns the Server's routes: usage ingestion, the served dashboard, and a
// liveness probe. Every request is logged minimally to stderr.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/usage", s.handleUsage)
	mux.HandleFunc("GET /{$}", s.handleDashboard)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	return logRequests(mux)
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

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// usagePush is POST /v1/usage's request body: one member's batch of local records.
type usagePush struct {
	Member  string         `json:"member"`
	Records []usage.Record `json:"records"`
}

// usagePushResult is POST /v1/usage's response body.
type usagePushResult struct {
	Inserted int `json:"inserted"`
	Received int `json:"received"`
}

// handleUsage authenticates, decodes a usagePush, validates the member label and every
// record, tags each record with its member, and inserts them. A record's DedupeKey is
// prefixed with "<member>:" before insert so two members' records can never collide
// under the store's unique(tool, dedupe_key) constraint -- see AGENTS.md's
// member-dimension note. A push that fails validation is rejected whole (nothing is
// inserted, not even its otherwise-valid records) rather than silently dropping the bad
// rows, so a buggy or malicious client can never partially poison the shared dashboard.
// Error responses describe the violated rule in the client's own submitted data, never
// an internal (DB/schema) detail -- see logging below for that.
func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	if !authorized(r, s.token) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUsageBodyBytes)
	var push usagePush
	if err := json.NewDecoder(r.Body).Decode(&push); err != nil {
		log.Printf("decode usage push: %v", err)
		http.Error(w, "malformed request body", http.StatusBadRequest)
		return
	}

	if err := ValidateMember(push.Member); err != nil {
		http.Error(w, "invalid member: "+err.Error(), http.StatusBadRequest)
		return
	}
	for i := range push.Records {
		if err := validateRecord(&push.Records[i]); err != nil {
			http.Error(w, fmt.Sprintf("invalid record %d: %v", i, err), http.StatusBadRequest)
			return
		}
	}

	for i := range push.Records {
		push.Records[i].Member = push.Member
		push.Records[i].DedupeKey = push.Member + ":" + push.Records[i].DedupeKey
	}
	inserted, err := s.store.Insert(r.Context(), push.Records)
	if err != nil {
		log.Printf("insert usage: %v", err)
		http.Error(w, "failed to store usage", http.StatusInternalServerError)
		return
	}

	writeJSON(w, usagePushResult{Inserted: inserted, Received: len(push.Records)})
}

// handleDashboard serves the aggregated Assay dashboard built fresh from the central
// store on every request -- this MVP has no caching. GET / is unauthenticated (see
// Server's doc comment), so its error path must never leak internal (DB/schema) detail
// to what may be an untrusted caller.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
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
