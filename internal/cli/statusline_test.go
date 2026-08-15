package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/i18n"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// cliCatalog is the catalog under test; taken by pointer because it is a large value.
var cliCatalog = i18n.For("").CLI

// runStatuslineCmd executes the command against an isolated home and returns its output.
func runStatuslineCmd(t *testing.T, args ...string) string {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"statusline"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("statusline must never fail: %v", err)
	}
	return strings.TrimSpace(out.String())
}

// seedLocalStore writes recs into the default local store for an isolated XDG home.
func seedLocalStore(t *testing.T, recs []usage.Record) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dbPath := filepath.Join(dir, "assaio", "assaio.db")
	if err := ensureParent(dbPath); err != nil {
		t.Fatal(err)
	}
	seedStoreAt(t, dbPath, recs)
	return dbPath
}

func TestStatuslineEmptyStoreSaysSo(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := runStatuslineCmd(t); !strings.Contains(got, "no data") {
		t.Fatalf("statusline = %q, want the no-data hint", got)
	}
}

// TestStatuslineNeverCreatesTheStore keeps the command read-only: rendering a status line
// must not leave a database behind.
func TestStatuslineNeverCreatesTheStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runStatuslineCmd(t)
	if _, err := os.Stat(filepath.Join(dir, "assaio", "assaio.db")); err == nil {
		t.Fatal("statusline must not create the store")
	}
}

func TestStatuslineShowsTodaysWork(t *testing.T) {
	// Local midnight, not now-minus-a-minute: a run starting in the first minute of the
	// local day would place that record on yesterday and fail for the clock, not the code.
	now := startOfLocalDay(time.Now())
	seedLocalStore(t, []usage.Record{{
		Tool: "claude-code", SessionID: "s1", Timestamp: now.UTC(),
		Model: "claude-opus-4-5", InputTokens: 1000, OutputTokens: 500,
		Project: "web", DedupeKey: "1", LinesAdded: 120,
	}})
	got := runStatuslineCmd(t)
	for _, want := range []string{"assaio", "tok", "lines"} {
		if !strings.Contains(got, want) {
			t.Errorf("statusline = %q, want it to contain %q", got, want)
		}
	}
}

func TestStatuslineNothingToday(t *testing.T) {
	seedLocalStore(t, []usage.Record{{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().AddDate(0, 0, -9).UTC(),
		Model: "claude-opus-4-5", InputTokens: 1000, OutputTokens: 500, DedupeKey: "1", LinesAdded: 10,
	}})
	if got := runStatuslineCmd(t); !strings.Contains(got, "nothing today") {
		t.Fatalf("statusline = %q, want the nothing-today form", got)
	}
}

// TestStatuslineDayIsTheLocalDay is the B79 sidestep: the window is a timestamp range from
// local midnight, never the store's UTC day bucket, so work is counted on the day it
// actually happened wherever the machine is.
func TestStatuslineDayIsTheLocalDay(t *testing.T) {
	for _, tz := range []string{"UTC", "Pacific/Auckland", "America/Los_Angeles"} {
		t.Run(tz, func(t *testing.T) {
			loc, err := time.LoadLocation(tz)
			if err != nil {
				t.Skipf("timezone %s unavailable", tz)
			}
			// Noon local, so "just before" and "just after" midnight are unambiguous.
			now := time.Date(2026, 7, 27, 12, 0, 0, 0, loc)
			midnight := startOfLocalDay(now)

			dir := t.TempDir()
			t.Setenv("XDG_DATA_HOME", dir)
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			dbPath := filepath.Join(dir, "assaio", "assaio.db")
			if err := ensureParent(dbPath); err != nil {
				t.Fatal(err)
			}
			seedStoreAt(t, dbPath, []usage.Record{
				{
					Tool: "claude-code", SessionID: "before", Timestamp: midnight.Add(-time.Hour).UTC(),
					Model: "claude-opus-4-5", InputTokens: 5_000_000, DedupeKey: "before", LinesAdded: 999,
				},
				{
					Tool: "claude-code", SessionID: "after", Timestamp: midnight.Add(time.Hour).UTC(),
					Model: "claude-opus-4-5", InputTokens: 1000, DedupeKey: "after", LinesAdded: 7,
				},
			})

			st, err := store.Open(dbPath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = st.Close() }()
			totals, err := statuslineDay(context.Background(), st, midnight)
			if err != nil {
				t.Fatal(err)
			}
			if totals.lines != 7 {
				t.Errorf("lines = %d, want 7 (only the record after local midnight counts)", totals.lines)
			}
			if totals.tokens != 1000 {
				t.Errorf("tokens = %d, want 1000 (the pre-midnight record must be excluded)", totals.tokens)
			}
		})
	}
}

// TestStatuslineAgeReportsUnknownBeforeIngestTracking covers a store carried over from a
// build without ingest state: the age is genuinely unknown and must say so, not read as
// fresh.
func TestStatuslineAgeReportsUnknownBeforeIngestTracking(t *testing.T) {
	dbPath := seedLocalStore(t, []usage.Record{{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(),
		Model: "claude-opus-4-5", InputTokens: 1000, DedupeKey: "1", LinesAdded: 5,
	}})
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if got := statuslineAge(context.Background(), st, time.Now(), &cliCatalog); got != "age unknown" {
		t.Fatalf("age = %q, want %q", got, "age unknown")
	}
}

// TestStatuslineAgeFollowsIngestState proves the age tracks ingest, not usage timestamps.
func TestStatuslineAgeFollowsIngestState(t *testing.T) {
	dbPath := seedLocalStore(t, []usage.Record{{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(),
		Model: "claude-opus-4-5", InputTokens: 1000, DedupeKey: "1", LinesAdded: 5,
	}})
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	ctx := context.Background()
	now := time.Now()
	if err := st.RecordIngest(ctx, "v1", now.Add(-90*time.Minute), []store.IngestEntry{
		{Path: "/logs/a.jsonl", Tool: "claude-code", Size: 1, MtimeNS: 1},
	}); err != nil {
		t.Fatal(err)
	}
	if got := statuslineAge(ctx, st, now, &cliCatalog); got != "1h ago" {
		t.Fatalf("age = %q, want %q", got, "1h ago")
	}
}

// TestStatuslineSubscriptionShowsTwoRawNumbers is the honesty guard on the money segment:
// on a flat plan the API-equivalent figure is not spend, so it is shown month-to-date
// beside the plan price rather than as a percentage, which would read as "consumed" when
// a plan only pays off above its price.
func TestStatuslineSubscriptionShowsTwoRawNumbers(t *testing.T) {
	seedLocalStore(t, []usage.Record{{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(),
		Model: "claude-opus-4-5", InputTokens: 5_000_000, OutputTokens: 2_000_000,
		DedupeKey: "1", LinesAdded: 400,
	}})
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("pricing:\n  mode: subscription\n  monthly_subscription_cost: 200\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := runStatuslineCmd(t, "--config", cfgPath)
	if !strings.Contains(got, "/$200 mo") {
		t.Fatalf("statusline = %q, want the month-to-date vs plan comparison", got)
	}
	if strings.Contains(got, "%") {
		t.Errorf("statusline = %q, must not frame plan value as a percentage", got)
	}
	if strings.Contains(got, "API-equiv") {
		t.Errorf("statusline = %q, must not show the pay-as-you-go form on a subscription", got)
	}
}

// TestStatuslineWithholdsLinesFromASourceThatNeverRecordsThem is ADR 0011 on the most-read
// surface there is. Gemini CLI answers no line signal, so summing LinesAdded with no
// capability gate printed "+0 lines" -- an AI that wrote nothing -- every day, forever.
func TestStatuslineWithholdsLinesFromASourceThatNeverRecordsThem(t *testing.T) {
	now := startOfLocalDay(time.Now())
	seedLocalStore(t, []usage.Record{{
		Tool: "gemini-cli", SessionID: "s1", Timestamp: now.UTC(),
		Model: "gemini-2.5-pro", InputTokens: 1000, OutputTokens: 500, DedupeKey: "1",
	}})
	got := runStatuslineCmd(t)
	if !strings.Contains(got, "tok") {
		t.Fatalf("statusline = %q, want the token figure the source does answer", got)
	}
	if strings.Contains(got, "lines") {
		t.Fatalf("statusline = %q, must not state a line figure no source in it records", got)
	}
	if strings.Contains(got, "+0") {
		t.Fatalf("statusline = %q, a structural silence must never render as a zero", got)
	}
}
