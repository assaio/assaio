package report

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain gives every test in this package a hermetic data directory: Pseudonym now
// persists a per-install secret to disk (see installSecret), and without this, running
// `go test` would read and write the real user's data directory instead of a throwaway
// one.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "assaio-report-test")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("XDG_DATA_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestPseudonymDeterministic(t *testing.T) {
	a := Pseudonym("project", "acme-web")
	b := Pseudonym("project", "acme-web")
	if a != b {
		t.Fatalf("Pseudonym(%q) = %q, then %q: not deterministic", "acme-web", a, b)
	}
}

func TestPseudonymDiffersByInput(t *testing.T) {
	names := []string{"acme-web", "acme-infra", "playground", "billing-service"}
	seen := make(map[string]string, len(names))
	for _, n := range names {
		p := Pseudonym("project", n)
		if other, ok := seen[p]; ok {
			t.Fatalf("Pseudonym collision: %q and %q both produced %q", n, other, p)
		}
		seen[p] = n
	}
}

func TestPseudonymEmptyIsUnknown(t *testing.T) {
	if got := Pseudonym("project", ""); got != "unknown" {
		t.Fatalf("Pseudonym(%q, \"\") = %q, want %q", "project", got, "unknown")
	}
}

func TestPseudonymShape(t *testing.T) {
	got := Pseudonym("project", "acme-web")
	const wantLen = len("project-") + pseudonymHexLen
	if len(got) != wantLen {
		t.Fatalf("Pseudonym(%q) = %q, want length %d", "acme-web", got, wantLen)
	}
	if got[:len("project-")] != "project-" {
		t.Fatalf("Pseudonym(%q) = %q, want prefix %q", "acme-web", got, "project-")
	}
}

func TestPseudonymMemberKindGetsMemberPrefix(t *testing.T) {
	got := Pseudonym("member", "alice")
	const wantPrefix = "member-"
	if got[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("Pseudonym(%q, %q) = %q, want prefix %q", "member", "alice", got, wantPrefix)
	}
}

func TestPseudonymSameNameDiffersByKind(t *testing.T) {
	project := Pseudonym("project", "acme")
	member := Pseudonym("member", "acme")
	if project == member {
		t.Fatalf("Pseudonym(%q) and Pseudonym(%q) must not collide: both %q", "project, acme", "member, acme", project)
	}
}

// TestPseudonymNotReversibleWithoutTheKey is the finding-2 regression against the old
// vulnerability: the pre-fix scheme was a bare, unsalted SHA256("kind:name")[:n], so
// anyone who knew the convention could dictionary-attack any guessable name by hashing
// candidates and matching the displayed label. The HMAC-keyed scheme must never reproduce
// that bare hash.
func TestPseudonymNotReversibleWithoutTheKey(t *testing.T) {
	guess := "acme-web"
	bareHash := sha256.Sum256([]byte("project:" + guess))
	bareLabel := "project-" + hex.EncodeToString(bareHash[:])[:pseudonymHexLen]
	if got := Pseudonym("project", guess); got == bareLabel {
		t.Fatalf("Pseudonym(%q) = %q must not match the unsalted dictionary-attack hash %q", guess, got, bareLabel)
	}
}

// TestPseudonymWithSecretStableForSameSecret is "same input + same install -> stable" at
// the pure-function level: two calls under the same secret bytes (standing in for the
// same install's persisted key) must agree.
func TestPseudonymWithSecretStableForSameSecret(t *testing.T) {
	secret := bytes.Repeat([]byte{0x07}, pseudonymKeySize)
	a := pseudonymWithSecret("member", "alice", secret)
	b := pseudonymWithSecret("member", "alice", secret)
	if a != b {
		t.Fatalf("pseudonymWithSecret must be stable for the same secret: %q != %q", a, b)
	}
}

// TestPseudonymDiffersAcrossInstalls is "different installs -> different key -> different
// output": two distinct secrets standing in for two different machines' persisted keys
// must diverge on the identical (kind, name) input, which is what stops the same real
// name from correlating across installs.
func TestPseudonymDiffersAcrossInstalls(t *testing.T) {
	secretA := bytes.Repeat([]byte{0x01}, pseudonymKeySize)
	secretB := bytes.Repeat([]byte{0x02}, pseudonymKeySize)
	a := pseudonymWithSecret("project", "acme-web", secretA)
	b := pseudonymWithSecret("project", "acme-web", secretB)
	if a == b {
		t.Fatalf("the same name under two different install secrets must not match: both %q", a)
	}
}

func TestLoadOrCreateSecretPersistsAcrossCalls(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pseudonym.key")

	first, err := loadOrCreateSecret(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != pseudonymKeySize {
		t.Fatalf("secret length = %d, want %d", len(first), pseudonymKeySize)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("key file mode = %v, want 0600", got)
		}
	}

	// A second call against the same path is the same-install-later-process case: it
	// must read back the persisted secret rather than generating a new one.
	second, err := loadOrCreateSecret(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("loadOrCreateSecret must return the same persisted secret on a later call")
	}
}

func TestLoadOrCreateSecretDiffersAcrossFreshPaths(t *testing.T) {
	a, err := loadOrCreateSecret(filepath.Join(t.TempDir(), "pseudonym.key"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := loadOrCreateSecret(filepath.Join(t.TempDir(), "pseudonym.key"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("two fresh installs (no pre-existing key file) must not generate the same secret")
	}
}
