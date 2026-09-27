package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// watchBase carries a distinct non-empty value in every watched column, so a swapped argument
// between two of them shows up as a change nobody offered.
func watchBase() usage.Record {
	r := identityTurn(time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC), "acme", "cli", "main")
	r.Subpath = "apps/api"
	r.Skill, r.Agent, r.CacheMissReason = "review", "planner", "ttl"
	return r
}

type watchedRow struct {
	ts, model, project, subpath, entrypoint, branch, skill, agent, missReason string
}

func readWatched(t *testing.T, st *Store, key string) watchedRow {
	t.Helper()
	var w watchedRow
	err := st.db.QueryRowContext(context.Background(), `
		SELECT ts, model, project, subpath, entrypoint, git_branch, skill, agent, cache_miss_reason
		FROM usage_record WHERE dedupe_key = ?`, key).Scan(
		&w.ts, &w.model, &w.project, &w.subpath, &w.entrypoint, &w.branch, &w.skill, &w.agent, &w.missReason)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// replaced reports whether any column went from a stated value to a different stated one --
// what IdentityChanges.Rows claims to count, read back from what the UPDATE actually did.
func replaced(before, after *watchedRow) bool {
	pairs := [][2]string{
		{before.ts, after.ts},
		{before.model, after.model},
		{before.project, after.project},
		{before.entrypoint, after.entrypoint},
		{before.branch, after.branch},
		{before.skill, after.skill},
		{before.agent, after.agent},
		{before.missReason, after.missReason},
	}
	for _, p := range pairs {
		if p[0] != "" && p[1] != "" && p[0] != p[1] {
			return true
		}
	}
	return before.project != "" && before.project == after.project && before.subpath != after.subpath
}

func TestRestateWatchCountsWhatTheReReadReplaced(t *testing.T) {
	earlier := time.Date(2026, 6, 30, 9, 0, 0, 0, time.UTC)
	later := time.Date(2026, 7, 2, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		first       func(*usage.Record)
		offer       func(*usage.Record)
		want        IdentityChanges
		wantModel   string
		wantProject string
	}{
		{"an identical re-read changes nothing", nil, nil, IdentityChanges{}, "m", "acme"},
		{
			"a corrected model replaces the stored one", nil, func(r *usage.Record) { r.Model = "m2" },
			IdentityChanges{Rows: 1, Model: 1},
			"m2", "acme",
		},
		{
			"a blank model keeps the stored one and is counted as kept", nil, func(r *usage.Record) { r.Model = "" },
			IdentityChanges{Kept: 1},
			"m", "acme",
		},
		{
			"filling a blank model is not a change", func(r *usage.Record) { r.Model = "" }, nil,
			IdentityChanges{},
			"m", "acme",
		},
		{
			"an earlier timestamp", nil, func(r *usage.Record) { r.Timestamp = earlier },
			IdentityChanges{Rows: 1, TS: 1},
			"m", "acme",
		},
		{
			"a later timestamp is another carrier, not a correction", nil, func(r *usage.Record) { r.Timestamp = later },
			IdentityChanges{},
			"m", "acme",
		},
		{
			"a changed entrypoint", nil, func(r *usage.Record) { r.Entrypoint = "sdk-py" },
			IdentityChanges{Rows: 1, Entrypoint: 1},
			"m", "acme",
		},
		{
			"a changed branch", nil, func(r *usage.Record) { r.GitBranch = "feature" },
			IdentityChanges{Rows: 1, Branch: 1},
			"m", "acme",
		},
		{
			"a changed label", nil, func(r *usage.Record) { r.Skill = "deploy" },
			IdentityChanges{Rows: 1, Label: 1},
			"m", "acme",
		},
		{
			"a changed project", nil, func(r *usage.Record) { r.Project = "beta" },
			IdentityChanges{Rows: 1, Project: 1},
			"m", "beta",
		},
		{
			"a changed subpath alone", nil, func(r *usage.Record) { r.Subpath = "apps/web" },
			IdentityChanges{Rows: 1, Project: 1},
			"m", "acme",
		},
		{
			"a guessed project is no answer", nil, func(r *usage.Record) { r.Project, r.Subpath, r.ProjectGuessed = "api", "", true },
			IdentityChanges{},
			"m", "acme",
		},
		{
			"a blank project is kept", nil, func(r *usage.Record) { r.Project, r.Subpath = "", "" },
			IdentityChanges{Kept: 1},
			"m", "acme",
		},
		{
			"an aggregate's competing project is not a replaced answer", aggregate, func(r *usage.Record) { aggregate(r); r.Project = "beta" },
			IdentityChanges{},
			"m", "",
		},
		{
			"an aggregate's later time is another carrier", aggregate, func(r *usage.Record) { aggregate(r); r.Timestamp = later },
			IdentityChanges{},
			"m", "acme",
		},
		{
			"an aggregate's earlier time is watched", aggregate, func(r *usage.Record) { aggregate(r); r.Timestamp = earlier },
			IdentityChanges{Rows: 1, TS: 1},
			"m", "acme",
		},
		{
			"another source's later time is a correction", codex, func(r *usage.Record) { codex(r); r.Timestamp = later },
			IdentityChanges{Rows: 1, TS: 1},
			"m", "acme",
		},
		{
			"an aggregate's model is watched", aggregate, func(r *usage.Record) { aggregate(r); r.Model = "m2" },
			IdentityChanges{Rows: 1, Model: 1},
			"m2", "acme",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			st := openTempStore(t)
			first, offer := watchBase(), watchBase()
			if tt.first != nil {
				tt.first(&first)
			}
			if tt.offer != nil {
				tt.offer(&offer)
			}
			if _, _, err := st.InsertLocal(ctx, []usage.Record{first}); err != nil {
				t.Fatal(err)
			}
			before := readWatched(t, st, first.DedupeKey)
			_, w, err := st.InsertLocal(ctx, []usage.Record{offer})
			if err != nil {
				t.Fatal(err)
			}
			after := readWatched(t, st, offer.DedupeKey)
			if w.Identity != tt.want {
				t.Fatalf("Identity = %+v, want %+v", w.Identity, tt.want)
			}
			if after.model != tt.wantModel || after.project != tt.wantProject {
				t.Fatalf("stored (model %q, project %q), want (%q, %q)", after.model, after.project, tt.wantModel, tt.wantProject)
			}
			if !isAgentAggregate(offer.Tool, offer.DedupeKey) && replaced(&before, &after) != (w.Identity.Rows == 1) {
				t.Fatalf("the watch and the update disagree: replaced=%v, Rows=%d", replaced(&before, &after), w.Identity.Rows)
			}
		})
	}
}

func aggregate(r *usage.Record) { r.DedupeKey = "agent:x1" }

func codex(r *usage.Record) { r.Tool = "codex" }
