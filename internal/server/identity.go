package server

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"regexp"
)

// ErrUnknownToken is returned when a presented token matches no configured member and no
// shared secret.
var ErrUnknownToken = errors.New("unauthorized")

var syncMemberDigestPattern = regexp.MustCompile(`^member-v2-[0-9a-f]{32}$`)

// ValidateSyncMemberDigest accepts only the keyed member label used by sync v2.
func ValidateSyncMemberDigest(member string) error {
	if !syncMemberDigestPattern.MatchString(member) {
		return errors.New("member digest must be member-v2- followed by 32 lowercase hex digits")
	}
	return nil
}

// Identity is how a request's member is decided. The two modes are a real security difference
// and they are named rather than inferred, so a deployment's own `doctor` can say which one it
// is running in.
type Identity int

const (
	// ClientAsserted identifies a legacy shared-token configuration. It may still
	// read the dashboard; sync v2 refuses writes because no token owns one member.
	ClientAsserted Identity = iota
	// ServerDerived is per-member tokens: the member is whoever holds the secret, decided here
	// and never read from the body. A member cannot then write another member's rows, which is
	// what the dedupe-key prefix has always assumed and never enforced.
	ServerDerived
)

func (i Identity) String() string {
	if i == ServerDerived {
		return "server-derived (per-member tokens)"
	}
	return "shared token (sync v2 requires per-member tokens)"
}

// Members maps a keyed member digest to that member's bearer secret for v2 writes.
// Empty means the legacy shared-token mode, which may read but not sync.
type Members map[string]string

// Mode reports how this server decides who a request is.
func (m Members) Mode() Identity {
	if len(m) > 0 {
		return ServerDerived
	}
	return ClientAsserted
}

// Validate rejects a member set that cannot be used safely: a name the store cannot key on, an
// empty secret, or two members sharing one -- the last being the failure that looks like it
// works, since both would authenticate and one would silently own the other's rows.
func (m Members) Validate() error {
	seen := make(map[string]string, len(m))
	for name, token := range m {
		if err := ValidateMember(name); err != nil {
			return fmt.Errorf("member %q: %w", name, err)
		}
		if len(token) < MinTokenBytes {
			return fmt.Errorf("member %q: token must be at least %d characters", name, MinTokenBytes)
		}
		if other, dup := seen[token]; dup {
			return fmt.Errorf("members %q and %q share a token, so neither can be told from the other", other, name)
		}
		seen[token] = name
	}
	return nil
}

// MinTokenBytes is the shortest secret this server accepts. Short enough to type, long enough
// that guessing it is not the easiest way in.
const MinTokenBytes = 16

// authenticated reports whether a presented secret is one this server knows. It is separate
// from memberFor because the two answers are needed at different moments: whether to read the
// body at all, and -- once read -- whose rows it holds.
func (s *Server) authenticated(presented string) bool {
	if s.members.Mode() == ClientAsserted {
		return constantTimeEqual(presented, s.token)
	}
	for _, token := range s.members {
		if constantTimeEqual(presented, token) {
			return true
		}
	}
	return false
}

// memberFor resolves the keyed digest assigned to a per-member token.
func (s *Server) memberFor(presented string) (string, error) {
	for name, token := range s.members {
		if constantTimeEqual(presented, token) {
			return name, nil
		}
	}
	return "", ErrUnknownToken
}

// constantTimeEqual compares two secrets without leaking how many bytes matched. An empty
// expected secret never matches: a misconfigured server must not become an open one.
func constantTimeEqual(got, want string) bool {
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
