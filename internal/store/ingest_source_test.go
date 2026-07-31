package store

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSourceHistoryEmptyStore(t *testing.T) {
	st := openTempStore(t)
	got, err := st.SourceHistory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty history, got %d tool(s)", len(got))
	}
}

func TestRecordSourceRunRoundTrip(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	run := SourceRun{
		Tool: "claude-code", RanAt: at,
		Discovered: 5742, Parsed: 37, Records: 812, Skipped: 3, ZeroToken: 1,
	}
	if err := st.RecordSourceRun(ctx, []SourceRun{run}); err != nil {
		t.Fatal(err)
	}
	got, err := st.SourceHistory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	runs := got["claude-code"]
	if len(runs) != 1 {
		t.Fatalf("want 1 run, got %d", len(runs))
	}
	g := runs[0]
	if g.Discovered != 5742 || g.Parsed != 37 ||
		g.Records != 812 || g.Skipped != 3 || g.ZeroToken != 1 {
		t.Errorf("round-trip mismatch: %+v", g)
	}
	if !g.RanAt.Equal(at) {
		t.Errorf("RanAt = %v, want %v", g.RanAt, at)
	}
}

// TestSourceHistoryIsOldestFirst pins the ordering the drift canaries depend on: the
// current run is the last element, every earlier run is its baseline.
func TestSourceHistoryIsOldestFirst(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	for i := range 3 {
		run := SourceRun{Tool: "codex", RanAt: at.Add(time.Duration(i) * time.Hour), Discovered: i}
		if err := st.RecordSourceRun(ctx, []SourceRun{run}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := st.SourceHistory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	runs := got["codex"]
	if len(runs) != 3 {
		t.Fatalf("want 3 runs, got %d", len(runs))
	}
	for i, r := range runs {
		if r.Discovered != i {
			t.Errorf("run %d: Discovered = %d, want %d (history must be oldest-first)", i, r.Discovered, i)
		}
	}
}

// TestSourceHistoryIsBoundedPerTool is the size discipline: this table must never grow
// with install age, and pruning one tool must not touch another's history.
func TestSourceHistoryIsBoundedPerTool(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	if err := st.RecordSourceRun(ctx, []SourceRun{{Tool: "codex", RanAt: at, Discovered: 7}}); err != nil {
		t.Fatal(err)
	}
	for i := range maxSourceRuns + 5 {
		run := SourceRun{Tool: "claude-code", RanAt: at.Add(time.Duration(i) * time.Hour), Discovered: i}
		if err := st.RecordSourceRun(ctx, []SourceRun{run}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := st.SourceHistory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	runs := got["claude-code"]
	if len(runs) != maxSourceRuns {
		t.Fatalf("want history capped at %d, got %d", maxSourceRuns, len(runs))
	}
	if first := runs[0].Discovered; first != 5 {
		t.Errorf("oldest surviving run = %d, want 5 (the %d newest kept)", first, maxSourceRuns)
	}
	if other := got["codex"]; len(other) != 1 {
		t.Errorf("pruning claude-code dropped codex history: %d run(s) left", len(other))
	}
}

func TestRecordSourceRunEmptyIsNoOp(t *testing.T) {
	st := openTempStore(t)
	if err := st.RecordSourceRun(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestRecordSourceRunWritesEveryToolInOneCall(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	err := st.RecordSourceRun(ctx, []SourceRun{
		{Tool: "claude-code", RanAt: at, Discovered: 5742},
		{Tool: "plugin:acme", RanAt: at, Records: 12},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.SourceHistory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 tools, got %d", len(got))
	}
	if got["plugin:acme"][0].Records != 12 {
		t.Errorf("plugin source not recorded: %+v", got["plugin:acme"])
	}
}

func TestProvenanceReportsNewestReadAndItsBuild(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	early := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	late := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	if err := st.RecordIngest(ctx, "v0.5.0", early, []IngestEntry{{Path: "/a", Tool: "claude-code"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordIngest(ctx, "v0.5.0", late, []IngestEntry{{Path: "/b", Tool: "codex"}}); err != nil {
		t.Fatal(err)
	}
	newest, build, err := st.Provenance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !newest.Equal(late) {
		t.Errorf("newest = %v, want %v", newest, late)
	}
	if build != "v0.5.0" {
		t.Errorf("build = %q, want v0.5.0", build)
	}
}

// TestProvenanceSaysWhenBuildsDisagree keeps an upgrade mid-history visible: part of the
// window may carry signals the older parser could not extract.
func TestProvenanceSaysWhenBuildsDisagree(t *testing.T) {
	st := openTempStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	for i, v := range []string{"v0.4.0", "v0.5.0"} {
		e := []IngestEntry{{Path: "/p" + string(rune('a'+i)), Tool: "claude-code"}}
		if err := st.RecordIngest(ctx, v, at, e); err != nil {
			t.Fatal(err)
		}
	}
	_, build, err := st.Provenance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(build, "mixed") {
		t.Errorf("build = %q, want it to disclose more than one parsing build", build)
	}
}

func TestProvenanceOnAnEmptyStore(t *testing.T) {
	st := openTempStore(t)
	newest, build, err := st.Provenance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !newest.IsZero() || build != "" {
		t.Errorf("empty store = (%v, %q), want unknown rather than a guess", newest, build)
	}
}
