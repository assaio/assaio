package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/report"
)

func TestDemoShowsSampleReports(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"demo"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"report -- cost", "effectiveness", "analyze", "web-app",
		"MODEL-FIT", report.CostEstimateDisclosure,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("demo output missing %q\n---\n%s", want, got)
		}
	}
}

// TestDemoRendersToolSignalPanels guards the demo's job as the README's 60-second first
// look: a panel reading "—" there says assaio has nothing to show, not that this sample
// lacks the signal, so the bundled data must exercise every one it ships.
func TestDemoRendersToolSignalPanels(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"demo"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, panel := range []string{"EXPLORE-PRODUCE", "FRICTION", "SKILL-ECONOMICS"} {
		line := panelHeader(t, out.String(), panel)
		if strings.Contains(line, "[—]") {
			t.Fatalf("%s reads as no-data on the sample: %q", panel, line)
		}
	}
}

// panelHeader returns the analyze section header line for panel, which carries its Read.
func panelHeader(t *testing.T, out, panel string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, panel+" · ") {
			return line
		}
	}
	t.Fatalf("demo output has no %s panel:\n%s", panel, out)
	return ""
}

// TestDemoRecordsHoldTheToolCallInvariant locks the sample to the contract both parsers'
// fuzz tests assert and the sync boundary now enforces: the purpose buckets split ToolCalls,
// so they sum to it, and Sidechain is a flag rather than a count.
func TestDemoRecordsHoldTheToolCallInvariant(t *testing.T) {
	recs := demoRecords(time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC))
	if len(recs) == 0 {
		t.Fatal("demoRecords produced no records")
	}
	for i := range recs {
		r := &recs[i]
		split := r.ToolReads + r.ToolSearches + r.ToolCommands + r.ToolWrites + r.ToolOther
		if split != r.ToolCalls {
			t.Fatalf("%s: purpose split sums to %d, tool_calls is %d", r.DedupeKey, split, r.ToolCalls)
		}
		if r.Sidechain != 0 && r.Sidechain != 1 {
			t.Fatalf("%s: sidechain %d is not 0 or 1", r.DedupeKey, r.Sidechain)
		}
	}
}

func TestDemoRecordsDeterministicAndSeeded(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	a, b := demoRecords(now), demoRecords(now)
	if len(a) == 0 {
		t.Fatal("demoRecords produced no records")
	}
	if len(a) != len(b) {
		t.Fatalf("demoRecords not deterministic: %d vs %d", len(a), len(b))
	}
}

func TestDemoDashboardWritesFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"demo", "--dashboard"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	// The path is unpredictable by design (a per-invocation temp dir, not a fixed name in
	// the shared temp root); read it back from the announcement the command prints.
	path := dashboardPathFromOutput(t, out.String())
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("demo --dashboard must write the announced file %s: %v", path, err)
	}
}

// dashboardPathFromOutput extracts the "Wrote sample dashboard to <path>" path from demo
// --dashboard's output.
func dashboardPathFromOutput(t *testing.T, out string) string {
	t.Helper()
	const marker = "sample dashboard to "
	i := strings.Index(out, marker)
	if i < 0 {
		t.Fatalf("demo --dashboard must announce the file: %q", out)
	}
	rest := out[i+len(marker):]
	if end := strings.Index(rest, " --"); end >= 0 {
		return rest[:end]
	}
	return strings.TrimSpace(rest)
}
