package cli

import (
	"bytes"
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

// corruptStoreHome points every path assaio resolves at a fresh temp tree and plants a
// file that is not a SQLite database where the store belongs.
func corruptStoreHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	data := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := filepath.Join(data, "assaio")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assaio.db"), []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runDoctor(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"doctor"}, args...))
	err := root.Execute()
	return out.String(), err
}

// TestDoctorStrictFailsOnUnreadableStore is B123: an unopenable store printed ERROR and
// returned nil, so a cron job with a corrupt database reported green -- and the drift
// canaries, which the whole --strict promise rests on, never ran at all.
func TestDoctorStrictFailsOnUnreadableStore(t *testing.T) {
	corruptStoreHome(t)
	out, err := runDoctor(t, "--strict")
	if err == nil {
		t.Fatalf("doctor --strict err = nil on an unreadable store, want non-zero exit: %q", out)
	}
	if !strings.Contains(out, "store:        ERROR") {
		t.Fatalf("doctor must still report what broke: %q", out)
	}
}

// TestDoctorStrictSaysCanariesDidNotRun covers the second half: without a store there is
// no ingest history to judge, so the drift line must not read as "no canary fired".
func TestDoctorStrictSaysCanariesDidNotRun(t *testing.T) {
	corruptStoreHome(t)
	out, _ := runDoctor(t)
	if strings.Contains(out, "no canary fired") {
		t.Fatalf("drift line claims a clean canary run that never happened: %q", out)
	}
	if !strings.Contains(out, "drift:") {
		t.Fatalf("doctor dropped the drift line entirely: %q", out)
	}
}

// seedPricedStore plants a store whose usage splits between a model the vendored table
// prices and one it does not, in the given token proportion.
func seedPricedStore(t *testing.T, pricedTokens, unpricedTokens int64) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	ts := time.Now().UTC()
	recs := []usage.Record{
		{Tool: "claude-code", SessionID: "s1", Timestamp: ts, Model: "claude-opus-4-5", InputTokens: pricedTokens, DedupeKey: "priced"},
	}
	if unpricedTokens > 0 {
		recs = append(recs, usage.Record{
			Tool: "claude-code", SessionID: "s2", Timestamp: ts,
			Model: "model-no-table-row-has", InputTokens: unpricedTokens, DedupeKey: "unpriced",
		})
	}
	if _, err := st.Insert(context.Background(), recs); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
}

// B139: a price table that has fallen behind the models in use is indistinguishable from a
// complete one from the inside. `pricing: N models, snapshot <date>` is a fact about the
// table, never about the reader's store, which is how 45.5% of the maintainer's own tokens
// went unpriced for five weeks with every surface reporting normally.
func TestDoctorStrictFailsWhenTooMuchOfTheStoreHasNoPrice(t *testing.T) {
	seedPricedStore(t, 100, 900)
	out, err := runDoctor(t, "--strict")
	if err == nil {
		t.Fatalf("doctor --strict err = nil with 90%% of tokens unpriced, want non-zero exit: %q", out)
	}
	if !strings.Contains(out, "unpriced:     90.0% of the last 30d") {
		t.Fatalf("doctor must state the share of the window it cannot price: %q", out)
	}
	if !strings.Contains(out, "model-no-table-row-has") {
		t.Fatalf("doctor must name the model a refresh has to cover: %q", out)
	}
}

// The gate must stay quiet on a store the table covers, or a cron job learns to ignore it.
func TestDoctorStrictPassesOnAFullyPricedStore(t *testing.T) {
	seedPricedStore(t, 1000, 0)
	out, err := runDoctor(t, "--strict")
	if err != nil {
		t.Fatalf("doctor --strict err = %v on a fully priced store: %q", err, out)
	}
	if !strings.Contains(out, "unpriced:     none") {
		t.Fatalf("doctor must say the table covers this store: %q", out)
	}
}

// A share under the ceiling is disclosed, never failed on: a single afternoon's experiment
// with a model nobody has priced yet is not a broken price table.
func TestDoctorStrictToleratesAShareUnderTheCeiling(t *testing.T) {
	seedPricedStore(t, 1000, 10)
	out, err := runDoctor(t, "--strict")
	if err != nil {
		t.Fatalf("doctor --strict err = %v at ~1%% unpriced: %q", err, out)
	}
	if !strings.Contains(out, "unpriced:     1.0% of the last 30d") {
		t.Fatalf("doctor must still disclose the share it tolerates: %q", out)
	}
}

// TestDoctorWithoutStrictStillSucceedsOnUnreadableStore keeps the diagnostic posture: a
// plain doctor reports problems, it does not fail on them.
func TestDoctorWithoutStrictStillSucceedsOnUnreadableStore(t *testing.T) {
	corruptStoreHome(t)
	out, err := runDoctor(t)
	if err != nil {
		t.Fatalf("doctor (no --strict) err = %v, want nil: %q", err, out)
	}
	if !strings.Contains(out, "caveats:") {
		t.Fatalf("doctor stopped before its caveats: %q", out)
	}
}

// The store's zero-token unpriced rows are the middle case report and check both print --
// Claude writes its locally-generated turns as model "<synthetic>" with an all-zero usage
// block. doctor saying "every model has a price" there contradicts the "*" the same store's
// tables show, on the one surface a reader consults to settle which of them is right.
func TestDoctorSeparatesUnpricedRowsFromMissingCost(t *testing.T) {
	seedPricedStore(t, 1000, 0)
	dbPath, err := paths.DBPath()
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Insert(context.Background(), []usage.Record{{
		Tool: "claude-code", SessionID: "s3", Timestamp: time.Now().UTC(),
		Model: "<synthetic>", DedupeKey: "synthetic",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := runDoctor(t, "--strict")
	if err != nil {
		t.Fatalf("doctor --strict err = %v: a row carrying no token cannot make the cost short: %q", err, out)
	}
	if !strings.Contains(out, "no tokens — 1 row(s)") {
		t.Fatalf("doctor must name the rows it cannot price without claiming the cost is short: %q", out)
	}
}
