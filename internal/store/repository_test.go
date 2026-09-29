package store

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// repoRecord is one turn from the repository key names, labelled label. A key of "" is a row
// whose repository was not resolved.
func repoRecord(tool, session, label, key, dedupe string) usage.Record {
	return usage.Record{
		Tool: tool, SessionID: session, Timestamp: time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC),
		Model: "m", DedupeKey: dedupe, Granularity: "turn", InputTokens: 10,
		Project: label, RepoKey: key,
	}
}

// shownTokens sums input tokens per shown project name over every stored row.
func shownTokens(t *testing.T, st *Store) map[string]int64 {
	t.Helper()
	rows, err := st.Usage(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int64{}
	for i := range rows {
		out[rows[i].Project] += rows[i].In
	}
	return out
}

func insertLocal(t *testing.T, st *Store, recs ...usage.Record) {
	t.Helper()
	if _, _, err := st.InsertLocal(context.Background(), recs); err != nil {
		t.Fatal(err)
	}
}

// TestInsertWithoutAKeyRegistersNothing: a record from a member's push or a parser plugin
// carries no key, so it registers nothing and stays unresolved.
func TestInsertWithoutAKeyRegistersNothing(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	if _, err := st.Insert(ctx, []usage.Record{repoRecord("plugin:x", "s1", "api", "", "1")}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertSynced(ctx, []usage.Record{repoRecord("claude-code", "s2", "api", "", "alice:2")}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM repository`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("repository rows = %d, %v; want 0", n, err)
	}
}

func TestRepositorySaltIsStoredOnce(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/s.db"
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := first.RepositorySalt(ctx)
	if err != nil || len(a) != 32 {
		t.Fatalf("salt = %d bytes, %v; want 32", len(a), err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = again.Close() }()
	b, err := again.RepositorySalt(ctx)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatalf("salt changed across a reopen: %x then %x (%v)", a, b, err)
	}
	other := openTempStore(t)
	c, err := other.RepositorySalt(ctx)
	if err != nil || bytes.Equal(a, c) {
		t.Fatalf("two stores share one salt: %x (%v)", c, err)
	}
}

func TestSplitLabelsAreSorted(t *testing.T) {
	st := openTempStore(t)
	insertLocal(t, st,
		repoRecord("claude-code", "s1", "web", "v1:w1", "1"),
		repoRecord("claude-code", "s2", "web", "v1:w2", "2"),
		repoRecord("claude-code", "s3", "api", "v1:a1", "3"),
		repoRecord("claude-code", "s4", "api", "v1:a2", "4"),
		repoRecord("claude-code", "s5", "docs", "v1:d", "5"))
	got, err := st.SplitLabels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"api", "web"}; !reflect.DeepEqual(got, want) || !sort.StringsAreSorted(got) {
		t.Fatalf("SplitLabels = %v, want %v", got, want)
	}
}

func TestRepositoriesCountEachSessionOnceWithItsUnresolvedShare(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	insertLocal(t, st,
		repoRecord("claude-code", "one", "api", "v1:a", "1"),
		repoRecord("claude-code", "one", "api", "", "2"),
		repoRecord("claude-code", "two", "api", "v1:b", "3"),
		repoRecord("claude-code", "three", "api", "", "4"),
		repoRecord("claude-code", "four", "web", "", "5"))
	pushed := repoRecord("claude-code", "team", "api", "", "alice:6")
	pushed.Member = "alice"
	if _, err := st.InsertSynced(ctx, []usage.Record{pushed}); err != nil {
		t.Fatal(err)
	}
	got, err := st.Repositories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	type counts struct{ sessions, unresolved int64 }
	names := map[string]counts{}
	for _, r := range got {
		names[r.Name] = counts{r.Sessions, r.Unresolved}
		if r.LastTs.IsZero() {
			t.Errorf("%s: no last activity", r.Name)
		}
	}
	want := map[string]counts{"api": {1, 0}, "api (2)": {1, 0}, "api (?)": {1, 1}, "web": {1, 1}}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("Repositories = %v, want %v (member rows left out)", names, want)
	}
	c, err := st.IdentityCoverage(ctx)
	if want := (Coverage{Sessions: 4, SessionsResolved: 2, Rows: 5, RowsResolved: 2}); err != nil || c != want {
		t.Fatalf("IdentityCoverage = %+v, %v; want %+v (member rows left out)", c, err, want)
	}
}

// TestClearingEverythingForgetsRepositories: with no usage row left, the store keeps no
// repository name and no hash of a path; a scoped clear keeps them so numbering stays stable.
func TestClearingEverythingForgetsRepositories(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	insertLocal(t, st, repoRecord("claude-code", "s1", "api", "v1:a", "1"), repoRecord("codex", "s2", "web", "v1:w", "2"))
	before, err := st.RepositorySalt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	registered := func() int {
		var n int
		if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM repository`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if _, err := st.Clear(ctx, time.Time{}, "codex"); err != nil {
		t.Fatal(err)
	}
	if n := registered(); n != 2 {
		t.Fatalf("after a scoped clear: %d repositories, want both kept", n)
	}
	if _, err := st.Clear(ctx, time.Time{}, ""); err != nil {
		t.Fatal(err)
	}
	after, err := st.RepositorySalt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n := registered(); n != 0 || bytes.Equal(before, after) || len(after) != 32 {
		t.Fatalf("after clearing everything: %d repositories, salt changed %v (%d bytes); want none and a new 32-byte salt",
			n, !bytes.Equal(before, after), len(after))
	}
}
