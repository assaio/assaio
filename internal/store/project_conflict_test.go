package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func projectRecord(project, subpath string) usage.Record {
	return usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC),
		Model: "m", InputTokens: 10, DedupeKey: "agent:shared", Granularity: "session",
		Project: project, Subpath: subpath,
	}
}

func storedProjectConflict(t *testing.T, st *Store) (project, subpath string, conflict int) {
	t.Helper()
	err := st.db.QueryRowContext(context.Background(),
		`SELECT project, subpath, project_conflict FROM usage_record WHERE dedupe_key = 'agent:shared'`).
		Scan(&project, &subpath, &conflict)
	if err != nil {
		t.Fatal(err)
	}
	return project, subpath, conflict
}

func TestProjectConflictAbstainsRegardlessOfInputOrder(t *testing.T) {
	for _, projects := range [][]string{{"alpha", "beta"}, {"beta", "alpha"}} {
		t.Run(projects[0]+"-then-"+projects[1], func(t *testing.T) {
			st := openTempStore(t)
			for _, project := range projects {
				if _, _, err := st.InsertLocal(context.Background(), []usage.Record{
					projectRecord(project, "apps/api"),
				}); err != nil {
					t.Fatal(err)
				}
			}

			project, subpath, conflict := storedProjectConflict(t, st)
			if project != "" || subpath != "" || conflict != 1 {
				t.Fatalf("stored = (%q, %q, %d), want (\"\", \"\", 1)", project, subpath, conflict)
			}

			if _, _, err := st.InsertLocal(context.Background(), []usage.Record{
				projectRecord(projects[0], "apps/api"),
			}); err != nil {
				t.Fatal(err)
			}
			project, subpath, conflict = storedProjectConflict(t, st)
			if project != "" || subpath != "" || conflict != 1 {
				t.Fatalf("re-import stored = (%q, %q, %d), want sticky ambiguity", project, subpath, conflict)
			}
		})
	}
}

func TestUnknownProjectCanStillBeRestatedWithEvidence(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	if _, _, err := st.InsertLocal(ctx, []usage.Record{projectRecord("", "")}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.InsertLocal(ctx, []usage.Record{projectRecord("alpha", "apps/api")}); err != nil {
		t.Fatal(err)
	}

	project, subpath, conflict := storedProjectConflict(t, st)
	if project != "alpha" || subpath != "apps/api" || conflict != 0 {
		t.Fatalf("stored = (%q, %q, %d), want (alpha, apps/api, 0)", project, subpath, conflict)
	}
}

func TestSessionsWithConflictingProjectsStayUnattributed(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	if _, _, err := st.InsertLocal(ctx, []usage.Record{
		projectRecord("alpha", "apps/api"),
		projectRecord("beta", "apps/api"),
	}); err != nil {
		t.Fatal(err)
	}

	sessions, err := st.Sessions(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].Project != "" {
		t.Fatalf("sessions = %+v, want one session with no asserted project", sessions)
	}
}

func TestSessionsWithMultipleProjectClaimsStayUnattributed(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	alpha := projectRecord("alpha", "apps/api")
	beta := projectRecord("beta", "apps/api")
	alpha.DedupeKey = "turn:1"
	beta.DedupeKey = "turn:2"
	if _, _, err := st.InsertLocal(ctx, []usage.Record{alpha, beta}); err != nil {
		t.Fatal(err)
	}

	sessions, err := st.Sessions(ctx, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].Project != "" {
		t.Fatalf("sessions = %+v, want one session with no asserted project", sessions)
	}
}
