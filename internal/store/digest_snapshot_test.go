package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestDigestSnapshotsAreBounded(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	total := DigestSnapshotsKept + 5
	for i := range total {
		snap := DigestSnapshot{
			TakenAt: base.AddDate(0, 0, 7*i), ParsedBy: "v0.17.0", Window: "7d",
			Payload: []byte(fmt.Sprintf(`{"version":1,"tokens":%d}`, i)),
		}
		if err := s.SaveDigestSnapshot(ctx, &snap); err != nil {
			t.Fatalf("SaveDigestSnapshot(%d): %v", i, err)
		}
	}

	var kept int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM digest_snapshot`).Scan(&kept); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if kept != DigestSnapshotsKept {
		t.Errorf("kept %d snapshots after writing %d, want the bound of %d", kept, total, DigestSnapshotsKept)
	}
	// The ones dropped must be the oldest, or "since last time" would start reaching further
	// back the longer the tool is used.
	got, ok, err := s.PreviousDigestSnapshot(ctx, base.AddDate(0, 0, 7*total), "7d")
	if err != nil || !ok {
		t.Fatalf("PreviousDigestSnapshot = %v, %v", ok, err)
	}
	if want := base.AddDate(0, 0, 7*(total-1)); !got.TakenAt.Equal(want) {
		t.Errorf("newest kept = %s, want %s", got.TakenAt, want)
	}
}

func TestPreviousDigestSnapshotExcludesThisRun(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	first := time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC)
	second := first.AddDate(0, 0, 7)
	for _, at := range []time.Time{first, second} {
		if err := s.SaveDigestSnapshot(ctx, &DigestSnapshot{TakenAt: at, Window: "7d", Payload: []byte(`{}`)}); err != nil {
			t.Fatalf("SaveDigestSnapshot: %v", err)
		}
	}
	got, ok, err := s.PreviousDigestSnapshot(ctx, second, "7d")
	if err != nil || !ok {
		t.Fatalf("PreviousDigestSnapshot = %v, %v", ok, err)
	}
	if !got.TakenAt.Equal(first) {
		t.Errorf("got the run at %s, want the one before it at %s", got.TakenAt, first)
	}
}

func TestFirstRunHasNoPreviousSnapshot(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	_, ok, err := s.PreviousDigestSnapshot(ctx, time.Now(), "7d")
	if err != nil {
		t.Fatalf("PreviousDigestSnapshot: %v", err)
	}
	if ok {
		t.Error("an empty store reported a previous snapshot")
	}
}

// A row this build cannot read must read as no basis rather than fail a cron job.
func TestAnUnreadableTimestampIsNoBasis(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO digest_snapshot(taken_at, parsed_by, window, payload) VALUES ('not a time','','7d','{}')`); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	_, ok, err := s.PreviousDigestSnapshot(ctx, time.Now(), "7d")
	if err != nil {
		t.Errorf("PreviousDigestSnapshot returned an error for an unreadable row: %v", err)
	}
	if ok {
		t.Error("an unreadable row was accepted as a comparison basis")
	}
}

// A digest compares this window against what the last run reported; rows deleted by `clear`
// are not a change in how the tools were used, and nothing downstream can tell the
// difference -- so the basis has to go with the records it described.
func TestClearDropsTheDigestBasisWithTheRecordsItDescribed(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	if err := s.SaveDigestSnapshot(ctx, &DigestSnapshot{
		TakenAt: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC), Window: "7d", Payload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("SaveDigestSnapshot: %v", err)
	}
	if _, err := s.Clear(ctx, time.Time{}, ""); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	_, ok, err := s.PreviousDigestSnapshot(ctx, time.Now(), "7d")
	if err != nil {
		t.Fatalf("PreviousDigestSnapshot: %v", err)
	}
	if ok {
		t.Error("a snapshot survived the clear that removed the records it described")
	}
}

// The read is per window, so the bound must be too: twelve daily runs must not evict the
// monthly basis and turn that digest silently into a first run.
func TestTheBoundIsPerWindow(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	base := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	monthly := DigestSnapshot{TakenAt: base, Window: "30d", Payload: []byte(`{}`)}
	if err := s.SaveDigestSnapshot(ctx, &monthly); err != nil {
		t.Fatalf("SaveDigestSnapshot: %v", err)
	}
	for i := 1; i <= DigestSnapshotsKept+3; i++ {
		snap := DigestSnapshot{TakenAt: base.AddDate(0, 0, i), Window: "7d", Payload: []byte(`{}`)}
		if err := s.SaveDigestSnapshot(ctx, &snap); err != nil {
			t.Fatalf("SaveDigestSnapshot(%d): %v", i, err)
		}
	}
	got, ok, err := s.PreviousDigestSnapshot(ctx, base.AddDate(0, 1, 0), "30d")
	if err != nil {
		t.Fatalf("PreviousDigestSnapshot: %v", err)
	}
	if !ok || !got.TakenAt.Equal(base) {
		t.Errorf("the 30d basis was evicted by 7d runs (ok=%v)", ok)
	}
}
