package ingest

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// TestUndatedRecordsAreSkippedNotStored: every report, validator and dashboard window is
// bounded by `ts >= ?`, so a record carrying the zero time is stored and then invisible to all
// of them while still counting toward the store's row total -- present in one place, absent in
// every other. Counting it as skipped is the honest word for evidence that could not be read,
// and it is the number the drift canaries already watch.
func TestUndatedRecordsAreSkippedNotStored(t *testing.T) {
	now := time.Now().UTC()
	stamped := usage.Record{Tool: "claude-code", DedupeKey: "k1", Timestamp: now}
	unstamped := usage.Record{Tool: "claude-code", DedupeKey: "k2"}
	epoch := usage.Record{Tool: "claude-code", DedupeKey: "k3", Timestamp: time.Unix(0, 0).UTC()}
	ahead := usage.Record{Tool: "claude-code", DedupeKey: "k4", Timestamp: now.Add(72 * time.Hour)}

	tests := []struct {
		name        string
		in          []usage.Record
		wantKept    int
		wantDropped int
	}{
		{"all dated", []usage.Record{stamped, stamped}, 2, 0},
		{"one undated", []usage.Record{stamped, unstamped}, 1, 1},
		{"all undated", []usage.Record{unstamped, unstamped}, 0, 2},
		{"implausibly early or far ahead", []usage.Record{stamped, epoch, ahead}, 1, 2},
		{"empty", nil, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kept, dropped := dated(tt.in, now)
			if len(kept) != tt.wantKept || dropped != tt.wantDropped {
				t.Errorf("dated = (%d kept, %d dropped), want (%d, %d)",
					len(kept), dropped, tt.wantKept, tt.wantDropped)
			}
		})
	}
}
