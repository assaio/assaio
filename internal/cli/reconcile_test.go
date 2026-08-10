package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// seedStore writes one priced day of local usage and returns the day it landed on.
func seedStore(t *testing.T, ts time.Time) {
	t.Helper()
	dbPath, err := paths.DBPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureParent(dbPath); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Insert(context.Background(), []usage.Record{{
		Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "claude-opus-4-5",
		InputTokens: 1_000_000, OutputTokens: 200_000, Project: "web", DedupeKey: "1",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeExport(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "export.csv")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReconcileReportsScopeDeltaAndRefusals(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ts := time.Now().UTC().Add(-24 * time.Hour)
	seedStore(t, ts)
	day := ts.Format("2006-01-02")
	path := writeExport(t, "Usage Date,Model,Cost USD,Currency\n"+day+",claude-opus-4-5,900.00,USD\n")

	out, err := runCLI(t, "reconcile", path, "--since", "7d")
	if err != nil {
		t.Fatalf("reconcile: %v (output %q)", err, out)
	}
	for _, want := range []string{
		"Scope — computed first",
		"vendor billed",
		"Unexplained delta",
		"assaio does not adjust either side",
		"flat-rate plan",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestReconcileJSONCarriesTheUnexplainedResidual(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ts := time.Now().UTC().Add(-24 * time.Hour)
	seedStore(t, ts)
	day := ts.Format("2006-01-02")
	path := writeExport(t, "date,amount\n"+day+",5.00\n")

	out, err := runCLI(t, "reconcile", path, "--since", "7d", "--format", "json")
	if err != nil {
		t.Fatalf("reconcile: %v (output %q)", err, out)
	}
	for _, want := range []string{`"unexplained"`, `"scope"`, `"refusals"`, `"binding"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("json missing %s:\n%s", want, out)
		}
	}
}

func TestReconcileRejectsAnUnbindableExport(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := writeExport(t, "widget,quantity\nfoo,2\n")

	_, err := runCLI(t, "reconcile", path)
	if err == nil {
		t.Fatal("an export with no date or cost column must fail rather than reconcile nothing")
	}
	if !strings.Contains(err.Error(), "--map") {
		t.Fatalf("the error must say how to fix it: %v", err)
	}
}
