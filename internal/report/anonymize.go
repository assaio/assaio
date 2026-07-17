package report

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/assaio/assaio/internal/paths"
)

// pseudonymHexLen is the number of hex characters kept from the HMAC, e.g.
// "a1b2c3d4e5". 10 hex chars (40 bits) makes collisions negligible well past any single
// install's or team's project/member count; the prior 4-char (16-bit) length collided
// around ~256 labels.
const pseudonymHexLen = 10

// pseudonymKeyFilename is the per-install secret's name under paths.DataDir().
const pseudonymKeyFilename = "pseudonym.key"

// pseudonymKeySize is the persisted secret's length in bytes: 256 bits, sized for the
// HMAC-SHA256 it keys, not for the (much shorter) truncated output.
const pseudonymKeySize = 32

// Pseudonym derives a short, stable label for name under kind (e.g. "project", "member"),
// so a shareable export can carry an identity without exposing the real value, e.g.
// Pseudonym("member", "alice") -> "member-a1b2c3d4e5". Deterministic per install: the
// same kind and name always yield the same label on this machine, because the HMAC is
// keyed by a secret generated once and persisted under this install's data directory
// (see installSecret). Guarantee: without that secret, a dictionary attack that just
// hashes guessed names (as a bare hash of "kind:name" would allow) cannot reproduce the
// label, and the same real name pseudonymizes differently on a different install, closing
// the cross-install correlation an unsalted hash would otherwise allow. Limitation: this
// is not encryption -- a holder of the secret (or of the real names) can still map labels
// back, so it defends against an outside dictionary/correlation attack, not against
// whoever already has the install's key or the underlying names. An empty name yields
// "unknown" rather than a hash of nothing.
func Pseudonym(kind, name string) string {
	if name == "" {
		return "unknown"
	}
	return pseudonymWithSecret(kind, name, installSecret())
}

// pseudonymWithSecret is Pseudonym's pure HMAC step, parameterized on the secret so it is
// directly testable (same input across two different secrets must diverge) without going
// through the process-wide cached installSecret.
func pseudonymWithSecret(kind, name string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	_, _ = h.Write([]byte(kind + ":" + name))
	return kind + "-" + hex.EncodeToString(h.Sum(nil))[:pseudonymHexLen]
}

var (
	secretOnce sync.Once
	secret     []byte
)

// installSecret returns this install's persisted HMAC key, generating and saving one on
// first use. Loaded once per process. A failure to resolve or persist it (e.g. an
// unwritable or undeterminable data directory) falls back to a random in-memory secret
// for this process only, so Pseudonym degrades to per-run rather than per-install
// stability instead of failing outright.
func installSecret() []byte {
	secretOnce.Do(func() {
		path, err := pseudonymKeyPath()
		if err != nil {
			secret = randomSecret()
			return
		}
		k, err := loadOrCreateSecret(path)
		if err != nil {
			secret = randomSecret()
			return
		}
		secret = k
	})
	return secret
}

func pseudonymKeyPath() (string, error) {
	dir, err := paths.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, pseudonymKeyFilename), nil
}

// loadOrCreateSecret reads the pseudonymKeySize-byte secret at path, generating and
// persisting (mode 0600) a fresh random one the first time path does not exist. A second
// call against the same path -- the same install, a later process -- returns the same
// bytes, which is what makes Pseudonym's output stable across separate CLI invocations.
func loadOrCreateSecret(path string) ([]byte, error) {
	existing, err := os.ReadFile(path) //nolint:gosec // path is this install's own data directory, not user input
	if err == nil && len(existing) == pseudonymKeySize {
		return existing, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	k := randomSecret()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, k, 0o600); err != nil {
		return nil, err
	}
	return k, nil
}

// randomSecret generates a fresh pseudonymKeySize-byte secret from a CSPRNG -- never
// math/rand, since this key's whole purpose is to resist an outside guesser.
func randomSecret() []byte {
	k := make([]byte, pseudonymKeySize)
	if _, err := rand.Read(k); err != nil {
		// The OS CSPRNG failing has no safe fallback source of randomness.
		panic("report: crypto/rand unavailable: " + err.Error())
	}
	return k
}
