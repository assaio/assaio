package projectid

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// KeyVersion prefixes every key, so a later derivation can coexist with this one and a store
// can tell which rows it must re-derive rather than compare keys of two derivations.
const KeyVersion = "v1:"

// keyHexLen keeps 128 bits of the HMAC: collisions stay negligible for any number of
// repositories one machine can hold.
const keyHexLen = 32

// keyDomain separates this HMAC from any other use of the same salt.
const keyDomain = "assaio.repository.v1"

// Key is the opaque local identity of the repository rooted at root: KeyVersion and keyHexLen
// hex characters of an HMAC keyed by the store's salt over the root's canonical spelling. It
// names a root on this machine, not an upstream project: two clones are two keys, and a checkout
// moved to a new path is a new key. A symlinked alias, or another case or Unicode form of the same
// path, shares the key. It is a keyed hash of a path, so it serves joins inside the store and
// never leaves it. There is no key without a salt, or for a root that cannot be resolved now --
// a key of an unresolved spelling would name a repository that does not exist.
func Key(salt []byte, root string) string {
	if len(salt) == 0 || root == "" {
		return ""
	}
	path, ok := canonical(root)
	if !ok {
		return ""
	}
	h := hmac.New(sha256.New, salt)
	_, _ = h.Write([]byte(keyDomain))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(path))
	return KeyVersion + hex.EncodeToString(h.Sum(nil))[:keyHexLen]
}
