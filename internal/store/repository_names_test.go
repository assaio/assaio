package store

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestShownNames(t *testing.T) {
	tests := []struct {
		name string
		recs []usage.Record
		want map[string]int64
	}{
		{
			name: "one repository: unresolved rows keep its name",
			recs: []usage.Record{
				repoRecord("claude-code", "s1", "api", "v1:a", "1"),
				repoRecord("claude-code", "s2", "api", "", "2"),
			},
			want: map[string]int64{"api": 20},
		},
		{
			name: "two repositories: the first keeps the name, the rest are numbered, unresolved rows apart",
			recs: []usage.Record{
				repoRecord("claude-code", "s1", "api", "v1:a", "1"),
				repoRecord("claude-code", "s2", "api", "v1:b", "2"),
				repoRecord("claude-code", "s3", "api", "", "3"),
			},
			want: map[string]int64{"api": 10, "api (2)": 10, "api (?)": 10},
		},
		{
			name: "order of first registration, not of the key",
			recs: []usage.Record{
				repoRecord("claude-code", "s1", "api", "v1:z", "1"),
				repoRecord("claude-code", "s2", "api", "v1:a", "2"),
				repoRecord("claude-code", "s3", "api", "v1:a", "3"),
			},
			want: map[string]int64{"api": 10, "api (2)": 20},
		},
		{
			name: "a generated name another label already carries is marked",
			recs: []usage.Record{
				repoRecord("claude-code", "s1", "api", "v1:a", "1"),
				repoRecord("claude-code", "s2", "api", "v1:b", "2"),
				repoRecord("claude-code", "s3", "api (2)", "", "3"),
			},
			want: map[string]int64{"api": 10, "api (#2)": 10, "api (2)": 10},
		},
		{
			name: "no key, no split: other labels are untouched",
			recs: []usage.Record{
				repoRecord("claude-code", "s1", "web", "", "1"),
				repoRecord("claude-code", "s2", "", "", "2"),
			},
			want: map[string]int64{"web": 10, "": 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := openTempStore(t)
			for i := range tt.recs {
				insertLocal(t, st, tt.recs[i])
			}
			if got := shownTokens(t, st); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("shown = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRepositoryWithNoRowsLeftStopsSplitting: a label is split on evidence of two
// repositories in the store, not on a registry entry whose rows were cleared.
func TestRepositoryWithNoRowsLeftStopsSplitting(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	insertLocal(t, st,
		repoRecord("claude-code", "s1", "api", "v1:a", "1"),
		repoRecord("codex", "s2", "api", "v1:b", "2"),
		repoRecord("claude-code", "s3", "api", "", "3"))
	if got := shownTokens(t, st); len(got) != 3 {
		t.Fatalf("before clear: shown = %v, want three names", got)
	}
	if _, err := st.Clear(ctx, time.Time{}, "codex"); err != nil {
		t.Fatal(err)
	}
	if got, want := shownTokens(t, st), map[string]int64{"api": 20}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after clear: shown = %v, want %v", got, want)
	}
	split, err := st.SplitLabels(ctx)
	if err != nil || len(split) != 0 {
		t.Fatalf("SplitLabels = %v, %v; want none", split, err)
	}
	// The ordinal is not reused: the cleared repository comes back under its old number.
	insertLocal(t, st, repoRecord("codex", "s4", "api", "v1:c", "4"), repoRecord("codex", "s5", "api", "v1:b", "5"))
	if got, want := shownTokens(t, st), map[string]int64{"api": 10, "api (2)": 10, "api (3)": 10, "api (?)": 10}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after re-import: shown = %v, want %v", got, want)
	}
}

func TestRepositoryAt(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	insertLocal(t, st,
		repoRecord("claude-code", "s1", "api", "v1:a", "1"),
		repoRecord("claude-code", "s2", "api", "v1:b", "2"),
		repoRecord("claude-code", "s3", "web", "v1:w", "3"),
		repoRecord("claude-code", "s4", "docs", "", "4"))
	tests := []struct {
		label, key                string
		wantLabel, name, unplaced string
		resolved                  bool
		others                    int
	}{
		{"api", "v1:a", "api", "api", "api (?)", true, 1},
		{"api", "v1:b", "api", "api (2)", "api (?)", true, 1},
		{"api", "v1:elsewhere", "api", "", "api (?)", false, 2},
		{"api", "", "api", "", "api (?)", false, 2},
		{"API", "v1:a", "api", "api", "api (?)", true, 1},
		{"web", "v1:w", "web", "web", "web", true, 0},
		{"docs", "v1:d", "docs", "", "docs", false, 0},
		{"new", "v1:n", "new", "", "new", false, 0},
		{"", "", "", "", "", false, 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s", tt.label, tt.key), func(t *testing.T) {
			got, err := st.RepositoryAt(ctx, tt.label, tt.key)
			if err != nil {
				t.Fatal(err)
			}
			if got.Label != tt.wantLabel || got.Name != tt.name || got.Unplaced != tt.unplaced ||
				(got.ID > 0) != tt.resolved || got.Others != tt.others {
				t.Fatalf("RepositoryAt = %+v, want label %q name %q unplaced %q resolved %v others %d",
					got, tt.wantLabel, tt.name, tt.unplaced, tt.resolved, tt.others)
			}
		})
	}
}

func TestSessionsNameTheirRepository(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	insertLocal(t, st,
		repoRecord("claude-code", "one", "api", "v1:a", "1"),
		repoRecord("claude-code", "two", "api", "v1:b", "2"),
		repoRecord("claude-code", "two", "api", "", "3"),
		repoRecord("claude-code", "both", "api", "v1:a", "4"),
		repoRecord("claude-code", "both", "api", "v1:b", "5"),
		repoRecord("claude-code", "none", "api", "", "6"))
	rows, err := st.Sessions(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	a, b := repositoryNamed(t, st, "api", "v1:a"), repositoryNamed(t, st, "api", "v1:b")
	want := map[string]struct {
		name string
		id   int64
	}{
		"one":  {"api", a.ID},
		"two":  {"api (2)", b.ID},
		"both": {"api (?)", 0},
		"none": {"api (?)", 0},
	}
	for i := range rows {
		w := want[rows[i].SessionID]
		if rows[i].Project != w.name || rows[i].RepoID != w.id {
			t.Errorf("session %s = (%q, %d), want (%q, %d)", rows[i].SessionID, rows[i].Project, rows[i].RepoID, w.name, w.id)
		}
	}
	latest, ok, err := st.LatestSessionIn(ctx, b.ID)
	if err != nil || !ok || latest.SessionID != "both" || latest.Project != "api (?)" {
		t.Fatalf("LatestSessionIn(api (2)) = %+v, %v, %v; want session both, the newest with a row in it", latest, ok, err)
	}
	lines, unplaced, err := st.RepositoryLines(ctx, b, time.Time{})
	if err != nil || lines != 0 || unplaced != 0 {
		t.Fatalf("RepositoryLines(api (2)) = %d, %d, %v; want 0, 0 (no row records lines)", lines, unplaced, err)
	}
	subpaths, err := st.Subpaths(ctx, "api (?)", time.Time{})
	if err != nil || len(subpaths) != 1 || subpaths[0].Sessions != 2 {
		t.Fatalf("Subpaths(api (?)) = %+v, %v; want the two unresolved sessions' rows", subpaths, err)
	}
}

// storedRepo reads one row's label, repository id and conflict bit.
// TestTimelinesNameTheirRepository: a step sequence carries the name its session is shown under,
// so a detector scoped to a project reads one repository, never two that share a name.
func TestTimelinesNameTheirRepository(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	insertLocal(t, st,
		repoRecord("claude-code", "one", "api", "v1:a", "1"),
		repoRecord("claude-code", "two", "api", "v1:b", "2"),
		repoRecord("claude-code", "two", "api", "", "3"))
	at := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	steps := []usage.Step{
		{Tool: "claude-code", SessionID: "one", DedupeKey: "a", Timestamp: at, Ordinal: 1, Kind: usage.StepAssistant},
		{Tool: "claude-code", SessionID: "two", DedupeKey: "b", Timestamp: at, Ordinal: 1, Kind: usage.StepAssistant},
	}
	if _, err := st.InsertSteps(ctx, steps); err != nil {
		t.Fatal(err)
	}
	got, err := st.Timelines(ctx, at.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for i := range got {
		names[got[i].SessionID] = got[i].Project
	}
	if want := map[string]string{"one": "api", "two": "api (2)"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("timeline projects = %v, want %v", names, want)
	}
}

// TestAmbiguousRowsUnderASplitNameAreUnplaced: a sub-agent aggregate two checkouts of one name
// both claimed belongs to neither, so under that name it is shown apart, with "?".
func TestAmbiguousRowsUnderASplitNameAreUnplaced(t *testing.T) {
	st := openTempStore(t)
	agent := func(key string) usage.Record {
		r := repoRecord("claude-code", "s1", "api", key, "agent:x")
		r.Granularity = "session"
		return r
	}
	insertLocal(t, st,
		repoRecord("claude-code", "s2", "api", "v1:a", "1"),
		repoRecord("claude-code", "s3", "api", "v1:b", "2"),
		agent("v1:a"), agent("v1:b"))
	if got, want := shownTokens(t, st), map[string]int64{"api": 10, "api (2)": 10, "api (?)": 10}; !reflect.DeepEqual(got, want) {
		t.Fatalf("shown = %v, want %v", got, want)
	}
}

// repositoryNamed is the stored repository for label and key, failing the test when none is.
func repositoryNamed(t *testing.T, st *Store, label, key string) Repository {
	t.Helper()
	r, err := st.RepositoryAt(context.Background(), label, key)
	if err != nil || r.ID == 0 {
		t.Fatalf("RepositoryAt(%s, %s) = %+v, %v; want a stored repository", label, key, r, err)
	}
	return r
}

// TestRepositoryLinesCountOnlyItsOwnRows: the lines resolved to one repository exclude another
// repository with its name, and the lines nothing places are returned apart -- a join from a
// directory never adds either to its figure.
func TestRepositoryLinesCountOnlyItsOwnRows(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	rec := func(label, key, dedupe string, lines int64) usage.Record {
		r := repoRecord("claude-code", dedupe, label, key, dedupe)
		r.LinesAdded = lines
		return r
	}
	insertLocal(t, st, rec("api", "v1:a", "1", 7), rec("api", "v1:b", "2", 100), rec("api", "", "3", 30),
		rec("web", "v1:w", "4", 1000))
	for _, tt := range []struct {
		key                     string
		wantLines, wantUnplaced int64
	}{
		{"v1:a", 7, 30},
		{"v1:b", 100, 30},
		{"v1:missing", 0, 30},
	} {
		r, err := st.RepositoryAt(ctx, "api", tt.key)
		if err != nil {
			t.Fatal(err)
		}
		lines, unplaced, err := st.RepositoryLines(ctx, r, time.Time{})
		if err != nil || lines != tt.wantLines || unplaced != tt.wantUnplaced {
			t.Errorf("%s: lines %d unplaced %d (%v), want %d and %d", tt.key, lines, unplaced, err, tt.wantLines, tt.wantUnplaced)
		}
	}
}
