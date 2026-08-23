package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// identityTurn is one turn identified by dedupe key "k1", carrying the identity columns a
// parser stamps on it.
func identityTurn(at time.Time, project, entrypoint, branch string) usage.Record {
	return usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: at, Model: "m",
		DedupeKey: "k1", Granularity: "turn", InputTokens: 10, OutputTokens: 5,
		Project: project, Entrypoint: entrypoint, GitBranch: branch,
	}
}

func storedIdentity(t *testing.T, st *Store) (ts, project, entrypoint, branch string) {
	t.Helper()
	row := st.db.QueryRowContext(context.Background(),
		`SELECT ts, project, entrypoint, git_branch FROM usage_record WHERE dedupe_key = 'k1'`)
	if err := row.Scan(&ts, &project, &entrypoint, &branch); err != nil {
		t.Fatal(err)
	}
	return ts, project, entrypoint, branch
}

// TestInsertLocalCorrectsTheIdentityColumns is B116: ts, project, entrypoint and git_branch
// were stamped by the first read and correctable by nothing, so a parser fix to any of them
// could not reach a single stored row -- including the one that decides which day a record
// counts toward.
func TestInsertLocalCorrectsTheIdentityColumns(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	first := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	corrected := time.Date(2026, 7, 3, 9, 0, 0, 0, time.UTC)

	if _, _, err := st.InsertLocal(ctx, []usage.Record{identityTurn(first, "wrong", "cli", "old-branch")}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.InsertLocal(ctx, []usage.Record{identityTurn(corrected, "right", "sdk-py", "new-branch")}); err != nil {
		t.Fatal(err)
	}

	ts, project, entrypoint, branch := storedIdentity(t, st)
	if ts != corrected.Format(time.RFC3339) {
		t.Fatalf("ts = %q, want the re-read's %q: a timestamp fix has to reach the day a record counts toward", ts, corrected.Format(time.RFC3339))
	}
	if project != "right" || entrypoint != "sdk-py" || branch != "new-branch" {
		t.Fatalf("identity = (%q, %q, %q), want the re-read's (right, sdk-py, new-branch)", project, entrypoint, branch)
	}
}

// TestInsertLocalNeverClearsAnIdentityItCannotSee is the other half of the same decision:
// correcting is overwriting with an answer, never with a silence. A sidecar not yet written,
// or a repository moved out from under a path, must not erase a name captured correctly.
func TestInsertLocalNeverClearsAnIdentityItCannotSee(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	at := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	if _, _, err := st.InsertLocal(ctx, []usage.Record{identityTurn(at, "acme", "cli", "main")}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.InsertLocal(ctx, []usage.Record{identityTurn(at, "", "", "")}); err != nil {
		t.Fatal(err)
	}

	_, project, entrypoint, branch := storedIdentity(t, st)
	if project != "acme" || entrypoint != "cli" || branch != "main" {
		t.Fatalf("identity = (%q, %q, %q), want the captured (acme, cli, main) kept", project, entrypoint, branch)
	}
}

// TestInsertLocalCountsADownwardRestatement is B116's canary. Assigned columns exist so a
// corrected attribution rule can reach history, which means the store cannot tell a fix landing
// from a parser regression erasing evidence -- so the move is counted and reported rather than
// left to be noticed as a figure that quietly shrank.
func TestInsertLocalCountsADownwardRestatement(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	at := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	rich := identityTurn(at, "acme", "cli", "main")
	rich.ToolCalls, rich.Edits, rich.LinesAdded = 40, 9, 300
	if _, lowered, err := st.InsertLocal(ctx, []usage.Record{rich}); err != nil || lowered != 0 {
		t.Fatalf("first insert: lowered = %d, err = %v; want 0, nil", lowered, err)
	}

	same := rich
	if _, lowered, err := st.InsertLocal(ctx, []usage.Record{same}); err != nil || lowered != 0 {
		t.Fatalf("identical re-read: lowered = %d, err = %v; want 0, nil", lowered, err)
	}

	higher := rich
	higher.Edits = 12
	if _, lowered, err := st.InsertLocal(ctx, []usage.Record{higher}); err != nil || lowered != 0 {
		t.Fatalf("upward restate: lowered = %d, err = %v; want 0, nil", lowered, err)
	}

	corrected := rich
	corrected.LinesAdded = 120
	inserted, lowered, err := st.InsertLocal(ctx, []usage.Record{corrected})
	if err != nil {
		t.Fatal(err)
	}
	if inserted != 0 || lowered != 1 {
		t.Fatalf("downward restate: inserted = %d, lowered = %d; want 0, 1", inserted, lowered)
	}

	var lines int64
	row := st.db.QueryRowContext(ctx, `SELECT lines_added FROM usage_record WHERE dedupe_key = 'k1'`)
	if err := row.Scan(&lines); err != nil {
		t.Fatal(err)
	}
	if lines != 120 {
		t.Fatalf("lines_added = %d, want the correction to have landed at 120", lines)
	}
}

// TestRestateMovesAStepWithItsRecord: both timestamps come from the same source line, so a
// timestamp fix that moves a usage row and leaves its steps behind splits one session across
// two days -- and PruneSteps reads the step's ts, so the horizon would cut the wrong ones.
func TestRestateMovesAStepWithItsRecord(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	first := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	corrected := time.Date(2026, 7, 3, 9, 0, 0, 0, time.UTC)

	step := usage.Step{
		Tool: "claude-code", SessionID: "s1", Timeline: "s1", DedupeKey: "k1",
		Timestamp: first, Ordinal: 1, Kind: usage.StepAssistant,
	}
	if _, _, err := st.InsertSteps(ctx, []usage.Step{step}); err != nil {
		t.Fatal(err)
	}
	step.Timestamp = corrected
	if _, _, err := st.InsertSteps(ctx, []usage.Step{step}); err != nil {
		t.Fatal(err)
	}

	var ts string
	row := st.db.QueryRowContext(ctx, `SELECT ts FROM session_step WHERE dedupe_key = 'k1'`)
	if err := row.Scan(&ts); err != nil {
		t.Fatal(err)
	}
	if ts != corrected.Format(time.RFC3339) {
		t.Fatalf("step ts = %q, want the re-read's %q so it stays in the same day as its record", ts, corrected.Format(time.RFC3339))
	}
}

// TestRestateCorrectsSubpathWithProject: one projectid.Resolve sets both, so correcting the
// root and keeping a path relative to the old one produces a subpath of a project it never
// belonged to -- which is exactly what the dashboard drill-down reads.
func TestRestateCorrectsSubpathWithProject(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	at := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	first := identityTurn(at, "old-root", "cli", "main")
	first.Subpath = "old/path"
	if _, _, err := st.InsertLocal(ctx, []usage.Record{first}); err != nil {
		t.Fatal(err)
	}
	corrected := identityTurn(at, "new-root", "cli", "main")
	corrected.Subpath = "new/path"
	if _, _, err := st.InsertLocal(ctx, []usage.Record{corrected}); err != nil {
		t.Fatal(err)
	}

	project, subpath := storedProjectPath(t, st)
	if project != "new-root" || subpath != "new/path" {
		t.Fatalf("stored = (%q, %q), want both corrected together", project, subpath)
	}
}

// TestRestateClearsSubpathWhenTheReReadResolvesARoot is the case guarding subpath on *itself*
// discards: "" is the correct subpath for a session at a repository root, so a re-read that
// resolves a deeper root -- a `git init` in a subdirectory, a submodule -- offers
// (deeper-project, "") and must move both. Guarded on its own emptiness, the project moved and
// the path stayed relative to a root it no longer belonged to, which is what the drill-down
// reads.
func TestRestateClearsSubpathWhenTheReReadResolvesARoot(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	at := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	nested := identityTurn(at, "mono", "cli", "main")
	nested.Subpath = "apps/mobile"
	if _, _, err := st.InsertLocal(ctx, []usage.Record{nested}); err != nil {
		t.Fatal(err)
	}
	// The same cwd, now its own repository root.
	rerooted := identityTurn(at, "mobile", "cli", "main")
	rerooted.Subpath = ""
	if _, _, err := st.InsertLocal(ctx, []usage.Record{rerooted}); err != nil {
		t.Fatal(err)
	}

	project, subpath := storedProjectPath(t, st)
	if project != "mobile" || subpath != "" {
		t.Fatalf("stored = (%q, %q), want (mobile, \"\") -- the path must not survive its project", project, subpath)
	}
}

// TestRestateStillKeepsAnIdentityTheReReadCannotSee holds the other half: a re-read with no
// project at all clears nothing.
func TestRestateStillKeepsAnIdentityTheReReadCannotSee(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	at := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	known := identityTurn(at, "mono", "cli", "main")
	known.Subpath = "apps/mobile"
	if _, _, err := st.InsertLocal(ctx, []usage.Record{known}); err != nil {
		t.Fatal(err)
	}
	blind := identityTurn(at, "", "", "")
	if _, _, err := st.InsertLocal(ctx, []usage.Record{blind}); err != nil {
		t.Fatal(err)
	}

	project, subpath := storedProjectPath(t, st)
	if project != "mono" || subpath != "apps/mobile" {
		t.Fatalf("stored = (%q, %q), want the captured pair kept", project, subpath)
	}
}

func storedProjectPath(t *testing.T, st *Store) (project, subpath string) {
	t.Helper()
	row := st.db.QueryRowContext(context.Background(),
		`SELECT project, subpath FROM usage_record WHERE dedupe_key = 'k1'`)
	if err := row.Scan(&project, &subpath); err != nil {
		t.Fatal(err)
	}
	return project, subpath
}
