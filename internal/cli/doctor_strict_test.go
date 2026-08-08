package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
