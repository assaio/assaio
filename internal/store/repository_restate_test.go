package store

import (
	"context"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

func storedRepo(t *testing.T, st *Store, dedupe string) (project string, repoID int64, conflict int) {
	t.Helper()
	err := st.db.QueryRowContext(context.Background(),
		`SELECT project, repo_id, project_conflict FROM usage_record WHERE dedupe_key = ?`, dedupe).
		Scan(&project, &repoID, &conflict)
	if err != nil {
		t.Fatal(err)
	}
	return project, repoID, conflict
}

func TestRestateRepository(t *testing.T) {
	agent := func(label, key string) usage.Record {
		r := repoRecord("claude-code", "s1", label, key, "agent:x")
		r.Granularity = "session"
		return r
	}
	turn := func(label, key string) usage.Record { return repoRecord("claude-code", "s1", label, key, "k") }
	guessed := func(label, key string) usage.Record {
		r := turn(label, key)
		r.ProjectGuessed = true
		return r
	}
	tests := []struct {
		name         string
		dedupe       string
		reads        []usage.Record
		wantProject  string
		wantKey      string // the repository the row must hold; "" when wantID holds instead
		wantID       int64
		wantConflict int
	}{
		{"an unresolved row gains its repository", "k", []usage.Record{turn("api", ""), turn("api", "v1:a")}, "api", "v1:a", 0, 0},
		{"a re-read with no repository keeps the one found", "k", []usage.Record{turn("api", "v1:a"), turn("api", "")}, "api", "v1:a", 0, 0},
		{"a guessed directory keeps the one found", "k", []usage.Record{turn("api", "v1:a"), guessed("elsewhere", "")}, "api", "v1:a", 0, 0},
		{"another label with no repository clears it", "k", []usage.Record{turn("api", "v1:a"), turn("home", "")}, "home", "", 0, 0},
		{"a turn moved to another repository follows it", "k", []usage.Record{turn("api", "v1:a"), turn("api", "v1:b")}, "api", "v1:b", 0, 0},
		{
			"a sub-agent claimed by two repositories of one name is ambiguous, keeping the name", "agent:x",
			[]usage.Record{agent("api", "v1:a"), agent("api", "v1:b"), agent("api", "v1:a")},
			"api", "", -1, 0,
		},
		{
			"a sub-agent claimed under two names is ambiguous and nameless", "agent:x",
			[]usage.Record{agent("api", "v1:a"), agent("web", "v1:w")},
			"", "", -1, 1,
		},
		{
			"an upgraded sub-agent row gains its repository", "agent:x",
			[]usage.Record{agent("api", ""), agent("api", "v1:a"), agent("api", "v1:a")},
			"api", "v1:a", 0, 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := openTempStore(t)
			for i := range tt.reads {
				insertLocal(t, st, tt.reads[i])
			}
			want := tt.wantID
			if tt.wantKey != "" {
				want = repositoryNamed(t, st, tt.wantProject, tt.wantKey).ID
			}
			project, id, conflict := storedRepo(t, st, tt.dedupe)
			if project != tt.wantProject || id != want || conflict != tt.wantConflict {
				t.Fatalf("stored = (%q, %d, %d), want (%q, %d, %d)", project, id, conflict, tt.wantProject, want, tt.wantConflict)
			}
		})
	}
}

// TestAConflictStoredBeforeRepositoriesStaysAmbiguous: a sub-agent row an older build already
// marked conflicted -- no project, no repository -- gains no repository from a later read, or a
// nameless row would keep that repository live and count as resolved.
func TestAConflictStoredBeforeRepositoriesStaysAmbiguous(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	r := repoRecord("claude-code", "s1", "api", "", "agent:x")
	r.Granularity = "session"
	insertLocal(t, st, r)
	if _, err := st.db.ExecContext(ctx,
		`UPDATE usage_record SET project = '', project_conflict = 1 WHERE dedupe_key = 'agent:x'`); err != nil {
		t.Fatal(err)
	}
	r.RepoKey = "v1:a"
	insertLocal(t, st, r)
	if project, id, conflict := storedRepo(t, st, "agent:x"); project != "" || id != -1 || conflict != 1 {
		t.Fatalf("stored = (%q, %d, %d), want (\"\", -1, 1)", project, id, conflict)
	}
}

// TestRestateReportsARepositoryMoveButNotItsFill: the first repository a row gains is the first
// read of it, a repository replaced under the same name is a restatement backfill reports.
func TestRestateReportsARepositoryMoveButNotItsFill(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	insertLocal(t, st, repoRecord("claude-code", "s1", "api", "", "k"))
	_, filled, err := st.InsertLocal(ctx, []usage.Record{repoRecord("claude-code", "s1", "api", "v1:a", "k")})
	if err != nil {
		t.Fatal(err)
	}
	if filled.Identity.Project != 0 {
		t.Fatalf("a first repository counted as %d project change(s), want 0", filled.Identity.Project)
	}
	_, moved, err := st.InsertLocal(ctx, []usage.Record{repoRecord("claude-code", "s1", "api", "v1:b", "k")})
	if err != nil {
		t.Fatal(err)
	}
	if moved.Identity.Project != 1 {
		t.Fatalf("a repository replaced under one name counted as %d project change(s), want 1", moved.Identity.Project)
	}
	b, err := st.RepositoryAt(ctx, "api", "v1:b")
	if err != nil {
		t.Fatal(err)
	}
	if _, id, _ := storedRepo(t, st, "k"); id != b.ID {
		t.Fatalf("repo_id = %d, want %d (v1:b)", id, b.ID)
	}
}

// TestAGuessedReadNeverMovesTheRepository: a read made after the directory was gone saw less
// than the read that resolved it, so whatever repository it offers is not an answer.
func TestAGuessedReadNeverMovesTheRepository(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	insertLocal(t, st, repoRecord("claude-code", "s1", "api", "v1:a", "k"))
	guessed := repoRecord("claude-code", "s1", "api", "v1:b", "k")
	guessed.ProjectGuessed = true
	insertLocal(t, st, guessed)
	a, err := st.RepositoryAt(ctx, "api", "v1:a")
	if err != nil {
		t.Fatal(err)
	}
	if _, id, _ := storedRepo(t, st, "k"); id != a.ID || a.ID == 0 {
		t.Fatalf("repo_id = %d, want %d (the repository found while the directory existed)", id, a.ID)
	}
}
