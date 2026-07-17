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

// TestBuildDashboardIncludesTeamSectionWithMembers is the SERVER-B proof that the served
// dashboard picks up the Team section automatically once the central store carries member
// data, and pseudonymizes those labels under this builder's hardcoded anonymize=true.
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
		t.Fatal("BuildDashboard Data.Team = nil, want a per-member breakdown when the central store has member data")
	}
	for _, s := range data.Team.Stats {
		if s.Member == "alice" || s.Member == "bob" {
			t.Fatalf("BuildDashboard must pseudonymize member labels by default: %+v", data.Team.Stats)
		}
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
