// Package parser holds helpers shared by the per-tool log parsers.
package parser

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
)

// MaxLineBytes caps a single log line; a longer line aborts the file scan.
const MaxLineBytes = 16 * 1024 * 1024

// NewScanner returns a line scanner over r, buffered up to MaxLineBytes.
func NewScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1024*1024), MaxLineBytes)
	return sc
}

// NonNeg clamps n to zero, dropping negative deltas rather than propagating them.
func NonNeg(n int64) int64 {
	if n < 0 {
		return 0
	}
	return n
}

// FileFingerprint returns a short deterministic digest of b. Parsers that build a
// DedupeKey from a positional counter (e.g. "session:index") prefix it with the
// fingerprint of the first line they read, so the same session/task id reused across two
// different files (e.g. a resumed session logged to a new file) never collides.
func FileFingerprint(b []byte) string {
	h := fnv.New32a()
	_, _ = h.Write(b)
	return fmt.Sprintf("%08x", h.Sum32())
}
