package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestHandleUsageMemberWithColonRejected(t *testing.T) {
	s, st := newTestServer(t)
	body := pushBody(t, "team:alice", []usage.Record{newValidRecord()})

	rr := doUsagePush(s, testToken, body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a member containing ':'", rr.Code)
	}
	recs, err := st.Export(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Fatalf("stored records = %+v, want none inserted for a rejected push", recs)
	}
}

// TestHandleUsageMemberColonCannotCauseCollision is the exact regression case from the
// bug report: composing "<member>:<dedupe_key>" naively, member "team:alice" key "x" and
// member "team" key "alice:x" both produce the store key "team:alice:x" and collide,
// silently dropping one member's record. Rejecting the colon-bearing member makes the
// first push impossible, so only the second's data ever lands.
func TestHandleUsageMemberColonCannotCauseCollision(t *testing.T) {
	s, st := newTestServer(t)

	poisoned := newValidRecord()
	poisoned.DedupeKey = "x"
	rejected := doUsagePush(s, testToken, pushBody(t, "team:alice", []usage.Record{poisoned}))
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("member with ':' status = %d, want 400: %s", rejected.Code, rejected.Body.String())
	}

	legit := newValidRecord()
	legit.DedupeKey = "alice:x"
	accepted := doUsagePush(s, testToken, pushBody(t, "team", []usage.Record{legit}))
	if accepted.Code != http.StatusOK {
		t.Fatalf("member without ':' status = %d, want 200: %s", accepted.Code, accepted.Body.String())
	}

	recs, err := st.Export(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Member != "team" || recs[0].DedupeKey != "team:alice:x" {
		t.Fatalf("stored records = %+v, want exactly one record member=team dedupe_key=team:alice:x", recs)
	}
}

// TestHandleUsagePushRejectsInvalidRecordWholeBatch covers the push-path validation fix:
// a negative token count, an unknown granularity, or a forged/unknown tool must each
// reject the request with 400 -- and must not insert anything, including a perfectly
// valid record riding along in the same batch (fail-closed, not a partial success that
// could still poison the shared dashboard).
func TestHandleUsagePushRejectsInvalidRecordWholeBatch(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*usage.Record)
	}{
		{"negative tokens", func(r *usage.Record) { r.InputTokens = -1 }},
		{"unknown granularity", func(r *usage.Record) { r.Granularity = "weekly" }},
		{"forged tool", func(r *usage.Record) { r.Tool = "not-a-real-tool" }},
		{"forged plugin namespace", func(r *usage.Record) { r.Tool = "plugin:" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, st := newTestServer(t)
			good := newValidRecord()
			good.DedupeKey = "good"
			bad := newValidRecord()
			bad.DedupeKey = "bad"
			tt.mut(&bad)

			rr := doUsagePush(s, testToken, pushBody(t, "alice", []usage.Record{good, bad}))
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rr.Code, rr.Body.String())
			}

			recs, err := st.Export(context.Background(), time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			if len(recs) != 0 {
				t.Fatalf("stored records = %+v, want none inserted (fail-closed on an invalid batch)", recs)
			}
		})
	}
}
