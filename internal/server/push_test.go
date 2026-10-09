package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestV2RejectsUnassignedMemberDigest(t *testing.T) {
	s, st := newV2TestServer(t)
	record := newValidRecord()
	input := NewSyncRecordV2(&record)
	rr := doV2Push(s, testToken, v2Body(t, "member-v2-33333333333333333333333333333333", input))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	recs, err := st.Export(context.Background(), time.Time{})
	if err != nil || len(recs) != 0 {
		t.Fatalf("rejected digest wrote rows=%+v err=%v", recs, err)
	}
}

// One invalid record rejects the whole batch, so no valid neighbor lands alone.
func TestV2PushRejectsInvalidRecordWholeBatch(t *testing.T) {
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
			s, st := newV2TestServer(t)
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
			if err != nil || len(recs) != 0 {
				t.Fatalf("invalid batch wrote rows=%+v err=%v", recs, err)
			}
		})
	}
}
