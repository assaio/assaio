package store

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

var markedAt = time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)

// labeledStore seeds two sessions on the same day and annotates only the first, which is
// the shape every assertion below rests on: some usage annotated, some not.
func labeledStore(t *testing.T) (*Store, context.Context, time.Time) {
	t.Helper()
	s, ctx := newStore(t), context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "aaa1", Timestamp: ts, Model: "m", Project: "web",
			InputTokens: 10, OutputTokens: 5, DedupeKey: "1", Granularity: "turn",
		},
		{
			Tool: "claude-code", SessionID: "bbb2", Timestamp: ts.Add(time.Hour), Model: "m", Project: "api",
			InputTokens: 20, OutputTokens: 7, DedupeKey: "2", Granularity: "turn",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	set := SessionLabel{Task: "refactor", Outcome: "done", Difficulty: "low", MarkedAt: markedAt}
	if err := s.Mark(ctx, SessionRef{SessionID: "aaa1"}, set); err != nil {
		t.Fatal(err)
	}
	return s, ctx, ts.Add(-time.Hour)
}

func TestMarkRoundTripAndReplace(t *testing.T) {
	s, ctx, _ := labeledStore(t)
	ref := SessionRef{SessionID: "aaa1"}

	got, ok, err := s.Label(ctx, ref)
	if err != nil || !ok {
		t.Fatalf("Label ok=%v err=%v", ok, err)
	}
	want := SessionLabel{Task: "refactor", Outcome: "done", Difficulty: "low", MarkedAt: markedAt}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Label = %+v want %+v", got, want)
	}

	// A second Mark replaces every axis rather than merging: merging is the caller's job.
	later := SessionLabel{Task: "bugfix", MarkedAt: markedAt.Add(time.Hour)}
	if err := s.Mark(ctx, ref, later); err != nil {
		t.Fatal(err)
	}
	got, _, _ = s.Label(ctx, ref)
	if got.Task != "bugfix" || got.Outcome != "" || got.Difficulty != "" {
		t.Fatalf("after replace Label = %+v", got)
	}
	if n, _ := s.LabelCount(ctx); n != 1 {
		t.Fatalf("LabelCount = %d want 1 (replace must not insert a second row)", n)
	}
}

func TestLabelAbsentAndUnmark(t *testing.T) {
	s, ctx, _ := labeledStore(t)

	if _, ok, err := s.Label(ctx, SessionRef{SessionID: "bbb2"}); err != nil || ok {
		t.Fatalf("Label(unmarked) ok=%v err=%v want false,nil", ok, err)
	}
	removed, err := s.Unmark(ctx, SessionRef{SessionID: "aaa1"})
	if err != nil || !removed {
		t.Fatalf("Unmark removed=%v err=%v", removed, err)
	}
	if removed, _ := s.Unmark(ctx, SessionRef{SessionID: "aaa1"}); removed {
		t.Fatal("Unmark twice reported a second removal")
	}
	if n, _ := s.LabelCount(ctx); n != 0 {
		t.Fatalf("LabelCount = %d want 0", n)
	}
}

func TestDeleteLabels(t *testing.T) {
	s, ctx, since := labeledStore(t)

	n, err := s.DeleteLabels(ctx, time.Time{}, "")
	if err != nil || n != 1 {
		t.Fatalf("DeleteLabels n=%d err=%v", n, err)
	}
	// Usage itself must be untouched: deleting annotations is not deleting data.
	rows, _ := s.Usage(ctx, since)
	if len(rows) != 2 {
		t.Fatalf("Usage after DeleteLabels = %d rows, want 2", len(rows))
	}
}

func TestMatchSessions(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "t", SessionID: "abc1", Timestamp: ts, Model: "m", DedupeKey: "1"},
		{Tool: "t", SessionID: "abc2", Timestamp: ts, Model: "m", DedupeKey: "2"},
		{Tool: "t", SessionID: "zz_9", Timestamp: ts, Model: "m", DedupeKey: "3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, prefix string
		want         []string
	}{
		{"unique", "abc1", []string{"abc1"}},
		{"ambiguous", "abc", []string{"abc1", "abc2"}},
		{"none", "qqq", nil},
		// _ is a LIKE wildcard; a prefix must match it literally or "zz_9" would also be
		// reachable by typing "zzX9".
		{"underscore is literal", "zz_", []string{"zz_9"}},
		{"wildcard does not match", "zzX", nil},
		{"percent is literal", "%", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			refs, err := s.MatchSessions(ctx, tc.prefix)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, r := range refs {
				got = append(got, r.SessionID)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("MatchSessions(%q) = %v want %v", tc.prefix, got, tc.want)
			}
		})
	}
}

func TestLatestSession(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	_, err := s.Insert(ctx, []usage.Record{
		{Tool: "t", SessionID: "old", Timestamp: ts, Model: "m", Project: "web", DedupeKey: "1"},
		{Tool: "t", SessionID: "new", Timestamp: ts.Add(2 * time.Hour), Model: "m", Project: "api", DedupeKey: "2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, project, want string
		wantOK              bool
	}{
		{name: "any project", want: "new", wantOK: true},
		{name: "scoped to project", project: "web", want: "old", wantOK: true},
		{name: "project with no sessions", project: "ghost"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ref, ok, err := s.LatestSession(ctx, tc.project)
			if err != nil {
				t.Fatal(err)
			}
			if ok != tc.wantOK || ref.SessionID != tc.want {
				t.Fatalf("LatestSession(%q) = %q,%v want %q,%v", tc.project, ref.SessionID, ok, tc.want, tc.wantOK)
			}
		})
	}
}

// TestEmptyFilterMatchesUnfiltered is the guarantee the whole feature rests on: adding
// annotations changes nothing for anyone who does not use them.
func TestEmptyFilterMatchesUnfiltered(t *testing.T) {
	s, ctx, since := labeledStore(t)

	plain, err := s.Usage(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := s.UsageFiltered(ctx, since, LabelFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plain, filtered) {
		t.Fatalf("UsageFiltered(empty) = %+v want %+v", filtered, plain)
	}

	sessions, err := s.Sessions(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	sessionsFiltered, err := s.SessionsFiltered(ctx, since, LabelFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sessions, sessionsFiltered) {
		t.Fatalf("SessionsFiltered(empty) = %+v want %+v", sessionsFiltered, sessions)
	}
}

func TestUsageFilteredNarrows(t *testing.T) {
	s, ctx, since := labeledStore(t)

	for _, tc := range []struct {
		name    string
		filter  LabelFilter
		wantIn  int64
		wantLen int
	}{
		{"by task", LabelFilter{Task: "refactor"}, 10, 1},
		{"by every axis", LabelFilter{Task: "refactor", Outcome: "done", Difficulty: "low"}, 10, 1},
		{"axis that matches nothing", LabelFilter{Task: "docs"}, 0, 0},
		{"one axis mismatched", LabelFilter{Task: "refactor", Outcome: "abandoned"}, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := s.UsageFiltered(ctx, since, tc.filter)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != tc.wantLen {
				t.Fatalf("UsageFiltered(%+v) = %d rows want %d", tc.filter, len(rows), tc.wantLen)
			}
			if tc.wantLen > 0 && rows[0].In != tc.wantIn {
				t.Fatalf("In = %d want %d", rows[0].In, tc.wantIn)
			}
		})
	}
}

func TestUsageByLabelKeepsUnannotatedUsage(t *testing.T) {
	s, ctx, since := labeledStore(t)

	rows, err := s.UsageByLabel(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("UsageByLabel = %d rows want 2", len(rows))
	}
	byTask := map[string]int64{}
	for _, r := range rows {
		byTask[r.Task] = r.In
	}
	// The unannotated session is grouped under the empty task, never dropped.
	if byTask["refactor"] != 10 || byTask[""] != 20 {
		t.Fatalf("UsageByLabel grouping = %v want refactor:10 and unannotated:20", byTask)
	}
	// Every other query leaves the annotation fields empty, so no surface can accidentally
	// read them without asking.
	plain, _ := s.Usage(ctx, since)
	for _, r := range plain {
		if r.Task != "" || r.Outcome != "" || r.Difficulty != "" {
			t.Fatalf("Usage returned annotations %+v", r)
		}
	}
}

func TestSessionsCarryAnnotations(t *testing.T) {
	s, ctx, since := labeledStore(t)

	rows, err := s.Sessions(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("Sessions = %d rows want 2", len(rows))
	}
	for _, r := range rows {
		switch r.SessionID {
		case "aaa1":
			if r.Task != "refactor" || r.Outcome != "done" || r.Difficulty != "low" {
				t.Fatalf("annotated session = %+v", r)
			}
		case "bbb2":
			if r.Task != "" || r.Outcome != "" || r.Difficulty != "" {
				t.Fatalf("unannotated session carried labels: %+v", r)
			}
		}
	}

	filtered, err := s.SessionsFiltered(ctx, since, LabelFilter{Task: "refactor"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].SessionID != "aaa1" {
		t.Fatalf("SessionsFiltered = %+v want only aaa1", filtered)
	}
}

func TestWindowQueriesFilterConsistently(t *testing.T) {
	s, ctx, since := labeledStore(t)
	only := LabelFilter{Task: "refactor"}

	_, total, err := s.DelegationFiltered(ctx, since, only)
	if err != nil {
		t.Fatal(err)
	}
	if total != 15 {
		t.Fatalf("DelegationFiltered total = %d want 15 (the annotated session's tokens only)", total)
	}

	turns, err := s.TurnSizingFiltered(ctx, since, 1000, only)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || turns[0].Turns != 1 {
		t.Fatalf("TurnSizingFiltered = %+v want one model with one turn", turns)
	}

	// Attribution has no skill/agent data in this fixture; the assertion that matters is
	// that the filtered query is valid SQL and returns nothing rather than erroring.
	skills, agents, err := s.AttributionFiltered(ctx, since, only)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 || len(agents) != 0 {
		t.Fatalf("AttributionFiltered = %v/%v want empty", skills, agents)
	}
}

// A label belongs to (session_id, member), which is its own primary key and how the grouped
// and joined queries read it. The filter used to match on session_id alone, so on a central
// store one member's annotation selected another member's identically-keyed usage.
func TestLabelFilterDoesNotCrossMembers(t *testing.T) {
	s, ctx := newStore(t), context.Background()
	ts := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	if _, err := s.Insert(ctx, []usage.Record{
		{
			Tool: "claude-code", SessionID: "shared", Member: "alice", Timestamp: ts, Model: "m",
			Project: "web", InputTokens: 10, DedupeKey: "a", Granularity: "turn",
		},
		{
			Tool: "claude-code", SessionID: "shared", Member: "bob", Timestamp: ts, Model: "m",
			Project: "web", InputTokens: 20, DedupeKey: "b", Granularity: "turn",
		},
	}); err != nil {
		t.Fatal(err)
	}
	set := SessionLabel{Task: "refactor", MarkedAt: markedAt}
	if err := s.Mark(ctx, SessionRef{SessionID: "shared", Member: "alice"}, set); err != nil {
		t.Fatal(err)
	}

	rows, err := s.UsageFiltered(ctx, ts.Add(-time.Hour), LabelFilter{Task: "refactor"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Member != "alice" {
		t.Fatalf("rows = %+v, want only alice's usage", rows)
	}

	sessions, err := s.SessionsFiltered(ctx, ts.Add(-time.Hour), LabelFilter{Task: "refactor"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].Member != "alice" {
		t.Fatalf("sessions = %+v, want only alice's session", sessions)
	}
}
