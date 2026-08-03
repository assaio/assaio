package vcs

import (
	"context"
	"strings"
)

// Blame returns the hash of the commit that last touched each line of path at HEAD, one per
// line of the file. Hashes only: git reads the file's content, and nothing but an identifier
// crosses this boundary.
//
// A path git cannot blame -- deleted, renamed away, or binary -- is an error the caller
// decides about, because "its lines did not survive" and "the read failed" are different
// facts and only the caller knows which one its report means.
func Blame(ctx context.Context, root, path string) ([]string, error) {
	out, err := gitOutput(ctx, root, "blame", "--line-porcelain", "HEAD", "--", path)
	if err != nil {
		return nil, err
	}
	var hashes []string
	sc := newScanner(out)
	for sc.Scan() {
		// --line-porcelain prefixes each line's block with "<sha> <orig> <final> ...", then
		// keyword metadata and the content line. Only a block header starts with a full hash,
		// so matching the whole first token keeps keywords and content out.
		line := sc.Text()
		sp := strings.IndexByte(line, ' ')
		if sp <= 0 || !isHash(line[:sp]) {
			continue
		}
		hashes = append(hashes, line[:sp])
	}
	return hashes, sc.Err()
}

// isHash reports whether s is a full object name -- 40 hex digits for SHA-1, 64 for SHA-256.
func isHash(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
