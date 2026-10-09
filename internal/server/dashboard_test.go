package server

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func TestBuildDashboardAnonymizedByDefault(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	_, err = st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m",
			InputTokens: 10, Project: "web", DedupeKey: "a1", LinesAdded: 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := BuildDashboard(context.Background(), st)
	if err != nil {
		t.Fatal(err)
	}
	if !data.Anonymized {
		t.Fatal("BuildDashboard Data.Anonymized = false, want true (anonymized by default)")
	}
}

// TestBuildDashboardIncludesTeamSectionWithMembers keeps the section and team totals
// visible while suppressing per-member rows below the minimum cohort.
func TestBuildDashboardIncludesTeamSectionWithMembers(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	_, err = st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m",
			InputTokens: 10, Project: "web", DedupeKey: "a1", LinesAdded: 5, Member: "alice",
		},
		{
			Tool: "claude-code", SessionID: "s2", Timestamp: time.Now(), Model: "m",
			InputTokens: 10, Project: "web", DedupeKey: "a2", LinesAdded: 5, Member: "bob",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := BuildDashboard(context.Background(), st)
	if err != nil {
		t.Fatal(err)
	}
	if data.Team == nil {
		t.Fatal("BuildDashboard Data.Team = nil, want a team section when the central store has member data")
	}
	if !data.Team.Suppressed || data.Team.MemberCount != 2 || len(data.Team.Stats) != 0 {
		t.Fatalf("BuildDashboard Team = %+v, want no per-member rows for a two-member cohort", data.Team)
	}
}

// TestBuildDashboardOmitsTeamSectionForMemberlessStore keeps the "no empty section"
// promise: a central store with no synced member data yet must not show a Team section.
func TestBuildDashboardOmitsTeamSectionForMemberlessStore(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	_, err = st.Insert(context.Background(), []usage.Record{
		{
			Tool: "claude-code", SessionID: "s1", Timestamp: time.Now(), Model: "m",
			InputTokens: 10, Project: "web", DedupeKey: "a1", LinesAdded: 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := BuildDashboard(context.Background(), st)
	if err != nil {
		t.Fatal(err)
	}
	if data.Team != nil {
		t.Fatalf("BuildDashboard Data.Team = %+v, want nil when no record carries a member", data.Team)
	}
}

func TestBuildDashboardEmptyStoreNoError(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	data, err := BuildDashboard(context.Background(), st)
	if err != nil {
		t.Fatal(err)
	}
	if data.Verdicts == nil {
		t.Fatal("BuildDashboard on empty store returned nil Verdicts, want the full validator set with honest empty-state results")
	}
}
