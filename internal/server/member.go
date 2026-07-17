package server

import (
	"errors"
	"fmt"
	"regexp"
)

// maxMemberLen bounds a member identifier so it can't smuggle an oversized value into
// the store or the dashboard it feeds.
const maxMemberLen = 64

// memberPattern is the charset a member identifier must match. It deliberately excludes
// ':' -- handleUsage composes "<member>:<dedupe_key>" as the store's unique key, so a
// colon in member would let two distinct (member, dedupe_key) pairs collide (e.g. member
// "team:alice" key "x" vs member "team" key "alice:x" both compose to "team:alice:x")
// and silently drop one side. It also excludes '/', whitespace, and control characters.
var memberPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateMember reports whether member is safe to compose into the store's
// "<member>:<dedupe_key>" unique key: non-empty, bounded length, and restricted to a
// charset that cannot collide across members or carry control/path characters. Called
// both server-side (handleUsage, on untrusted input) and client-side (sync's --member
// flag, to fail fast before a round trip -- see internal/cli/sync.go).
func ValidateMember(member string) error {
	if member == "" {
		return errors.New("member must not be empty")
	}
	if len(member) > maxMemberLen {
		return fmt.Errorf("member exceeds %d bytes", maxMemberLen)
	}
	if !memberPattern.MatchString(member) {
		return fmt.Errorf("member %q must match %s", member, memberPattern.String())
	}
	return nil
}
